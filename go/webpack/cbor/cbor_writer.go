package cbor

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

// Encoded returns the array of bytes that prefix a CBOR item of type t with
// either value or length "value", depending on the type.
func Encoded(t Type, value int) []byte {
	var buffer bytes.Buffer
	item := New(&buffer)
	item.encodeInt(t, value)
	item.Finish()
	return buffer.Bytes()
}

// EncodedFixedLen is like Encoded(), but always uses the size-byte encoding of
// value.
func EncodedFixedLen(size int, t Type, value int) []byte {
	var buffer bytes.Buffer
	item := New(&buffer)
	item.encodeSizedInt64(size, t, uint64(value))
	item.Finish()
	return buffer.Bytes()
}

type countingWriter struct {
	w *bufio.Writer
	// bytes counts the total number of bytes written to w.
	bytes uint64
}

func newCountingWriter(to io.Writer) *countingWriter {
	return &countingWriter{w: bufio.NewWriter(to)}
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	nn, err := cw.w.Write(p)
	cw.bytes += uint64(nn)
	return nn, err
}

func (cw *countingWriter) Flush() error {
	return cw.w.Flush()
}

type compoundItem struct {
	*countingWriter
	// nil for the root.
	parent *compoundItem
	// nil for the leaf-most active child.
	activeChild *compoundItem
	// How many elements have been added to this item so far.
	elements uint64
	// The byte offset within the buffer at which this item starts.
	startOffset uint64
}

type TopLevel struct {
	compoundItem
}

// New returns a new CBOR top-level item for the caller to write into. Call
// .Finish() when serialization is complete.
func New(to io.Writer) *TopLevel {
	result := &TopLevel{}
	result.countingWriter = newCountingWriter(to)
	return result
}

// Finish checks for well-formed-ness and flushes the serialization to the
// Writer passed to New.
func (c *TopLevel) Finish() error {
	if c.activeChild != nil {
		panic(fmt.Sprintf("Must finish child %v before its parent %v.",
			c.activeChild, c))
	}
	err := c.Flush()
	c.countingWriter = nil
	return err
}

func encodedSize(i uint64) int {
	if i < 24 {
		return 0
	}
	if i < (1 << 8) {
		return 1
	}
	if i < (1 << 16) {
		return 2
	}
	if i < (1 << 32) {
		return 4
	}
	return 8
}

func (item *compoundItem) encodeInt(t Type, i int) {
	item.encodeInt64(t, uint64(i))
}
func (item *compoundItem) encodeInt64(t Type, i uint64) {
	item.encodeSizedInt64(encodedSize(i), t, i)
}
func (item *compoundItem) encodeSizedInt64(size int, t Type, i uint64) {
	item.elements++

	switch size {
	case 0:
		item.Write([]byte{byte(t) | byte(i)})
	case 1:
		item.Write([]byte{byte(t) | 24, byte(i)})
	case 2:
		item.Write([]byte{byte(t) | 25, byte(i >> 8), byte(i)})
	case 4:
		item.Write([]byte{byte(t) | 26,
			byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
	case 8:
		item.Write([]byte{byte(t) | 27,
			byte(i >> 56), byte(i >> 48), byte(i >> 40), byte(i >> 32),
			byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
	default:
		panic(fmt.Sprintf("Unexpected CBOR item size: %v", size))
	}
}

func (item *compoundItem) AppendUint64(i uint64) {
	item.encodeInt64(TypePosInt, i)
}

// AppendFixedSizeUint64 always uses the 8-byte encoding for this uint64.
func (item *compoundItem) AppendFixedSizeUint64(i uint64) {
	item.encodeSizedInt64(8, TypePosInt, i)
}
func (item *compoundItem) AppendInt64(i int64) {
	if i < 0 {
		item.encodeInt64(TypeNegInt, uint64(-1-i))
	} else {
		item.encodeInt64(TypePosInt, uint64(i))
	}
}

func (item *compoundItem) AppendBytes(bs []byte) {
	item.encodeInt(TypeBytes, len(bs))
	item.Write(bs)
}

// AppendUtf8 checks that bs holds valid UTF-8.
func (item *compoundItem) AppendUtf8(bs []byte) {
	if !utf8.Valid(bs) {
		panic(fmt.Sprintf("Invalid UTF-8 in %q.", bs))
	}
	item.encodeInt(TypeText, len(bs))
	item.Write(bs)
}

func (item *compoundItem) AppendUtf8S(str string) {
	item.AppendUtf8([]byte(str))
}

// ByteLenSoFar returns the number of bytes from the start of item's encoding.
func (item *compoundItem) ByteLenSoFar() uint64 {
	return item.bytes - item.startOffset
}

func (item *compoundItem) AppendSerializedItem(r io.Reader) {
	item.elements++
	io.Copy(item, r)
}

type Array struct {
	compoundItem
	expectedSize uint64
}

func (item *compoundItem) AppendArray(expectedSize uint64) *Array {
	startOffset := item.bytes
	item.encodeInt64(TypeArray, expectedSize)
	a := &Array{
		compoundItem: compoundItem{
			countingWriter: item.countingWriter,
			parent:         item,
			elements:       0,
			startOffset:    startOffset,
		},
		expectedSize: expectedSize,
	}
	item.activeChild = &a.compoundItem
	return a
}

func (a *Array) Finish() {
	if a.activeChild != nil {
		panic(fmt.Sprintf("Must finish child %v before its parent %v.",
			a.activeChild, a))
	}
	if a.elements != a.expectedSize {
		panic(fmt.Sprintf("Array has size %v but was initialized with size %v",
			a.elements, a.expectedSize))
	}
	a.parent.activeChild = nil
	a.countingWriter = nil
}

type Map struct {
	compoundItem
	expectedSize uint64
}

func (item *compoundItem) AppendMap(expectedSize uint64) *Map {
	startOffset := item.bytes
	item.encodeInt64(TypeMap, expectedSize)
	m := &Map{
		compoundItem: compoundItem{
			countingWriter: item.countingWriter,
			parent:         item,
			elements:       0,
			startOffset:    startOffset,
		},
		expectedSize: expectedSize,
	}
	item.activeChild = &m.compoundItem
	return m
}

func (m *Map) Finish() {
	if m.activeChild != nil {
		panic(fmt.Sprintf("Must finish child %v before its parent %v.",
			m.activeChild, m))
	}
	if m.elements%2 != 0 {
		panic("Map's last key is missing a value.")
	}
	if m.elements != m.expectedSize*2 {
		panic(fmt.Sprintf("Map has size %v but was initialized with size %v",
			m.elements/2, m.expectedSize))
	}
	m.parent.activeChild = nil
	m.countingWriter = nil
}
