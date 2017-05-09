package cbor

import (
	"fmt"
	"unicode/utf8"

	"github.com/google/btree"
)

// Returns the array of bytes that prefix a CBOR item of type t with either
// value or length "value", depending on the type.
func Encoded(t Type, value int) []byte {
	item := New()
	item.encodeInt(t, value)
	return item.buffer.bytes
}

// Like Encoded(), but always uses the size-byte encoding of value.
func EncodedFixedLen(size int, t Type, value int) []byte {
	item := New()
	item.encodeSizedInt64(size, t, uint64(value))
	return item.buffer.bytes
}

type buffer struct {
	bytes []byte
	root  *compoundItem
	// Holds *PendingInt instances.
	pending *btree.BTree
}

func newBuffer() *buffer {
	return &buffer{pending: btree.New(32)}
}

type PendingInt struct {
	buf    *buffer
	offset int
	t      Type
}

// btree.Item interface. This sorts the PendingInts by their offset within the
// btree, which lets us efficiently traverse the PendingInts within a given
// range of the buffer.
func (p *PendingInt) Less(than btree.Item) bool {
	return p.offset < than.(*PendingInt).offset
}

func (p *PendingInt) Complete(value uint64) {
	if p == nil {
		panic("nil.Complete() isn't allowed")
	}
	p.buf.pending.Delete(p)
	size := encodedSize(value)
	p.buf.insertHole(int(p.offset+1), size)
	p.buf.encodeSizedIntAt(int(p.offset), size, p.t, value)
}

type compoundItem struct {
	*buffer
	// nil for the root.
	parent *compoundItem
	// nil for the leaf-most active child.
	activeChild *compoundItem
	// How many elements have been added to this item so far.
	count int
	// The byte offset within the buffer at which this item starts.
	startOffset int
}

type TopLevel struct {
	compoundItem
}

// Returns a new CBOR top-level item for the caller to write into. Call
// .Finish() to check for well-formed-ness and return the bytes representing the
// written structure.
func New() *TopLevel {
	result := &TopLevel{compoundItem: compoundItem{buffer: newBuffer()}}
	result.root = &result.compoundItem
	return result
}

func (c *TopLevel) Finish() []byte {
	if c.activeChild != nil {
		panic(fmt.Sprintf("Must finish child %v before its parent %v.",
			c.activeChild, c))
	}
	result := c.bytes
	c.buffer = nil
	return result
}

// Creates a hole in the buffer from [start:start+n], and shifts later data to
// make room for it. Returns a slice of the hole.
func (buf *buffer) insertHole(start, n int) []byte {
	// Add n bytes to the buffer.
	buf.bytes = append(buf.bytes, make([]byte, n)...)
	// Shift data to make the hole.
	copy(buf.bytes[start+n:], buf.bytes[start:])
	// Shift pending values to point to their original bytes.
	buf.pending.AscendGreaterOrEqual(&PendingInt{offset: start},
		func(i btree.Item) bool {
			pending := i.(*PendingInt)
			pending.offset += n
			return true
		})
	// Shift active children to point to their original bytes.
	for item := buf.root; item != nil; item = item.activeChild {
		if item.startOffset >= start {
			item.startOffset += n
		}
	}

	return buf.bytes[start : start+n]
}

func (buf *buffer) append(bytes ...byte) {
	buf.bytes = append(buf.bytes, bytes...)
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
	originalLen := len(item.bytes)
	item.insertHole(originalLen, size+1)
	item.encodeSizedIntAt(originalLen, size, t, i)
	item.count++
}

func (b *buffer) encodeSizedIntAt(offset int, size int, t Type, i uint64) {
	switch size {
	case 0:
		b.bytes[offset] = byte(t) | byte(i)
	case 1:
		b.bytes[offset] = byte(t) | 24
		b.bytes[offset+1] = byte(i)
	case 2:
		b.bytes[offset] = byte(t) | 25
		b.bytes[offset+1] = byte(i >> 8)
		b.bytes[offset+2] = byte(i)
	case 4:
		copy(b.bytes[offset:offset+5],
			[]byte{byte(t) | 26,
				byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
	case 8:
		copy(b.bytes[offset:offset+9],
			[]byte{byte(t) | 27,
				byte(i >> 56), byte(i >> 48), byte(i >> 40), byte(i >> 32),
				byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
	default:
		panic(fmt.Sprintf("Unexpected CBOR item size: %v", size))
	}
}

func (item *compoundItem) AppendUint64(i uint64) {
	item.encodeInt64(TypePosInt, i)
}

// Always uses the 8-byte encoding for this uint64.
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

// Adds an unsigned integer with unknown value to the end of the item. Call
// .Complete() to fill in the value. Depending on the length of a pending
// integer, usually via ByteLenSoFar(), causes a panic.
func (item *compoundItem) AppendPendingUint() *PendingInt {
	item.encodeInt(TypePosInt, 0)
	result := &PendingInt{
		buf:    item.buffer,
		offset: len(item.bytes) - 1,
		t:      TypePosInt,
	}
	if old := item.pending.ReplaceOrInsert(result); old != nil {
		panic(fmt.Sprintf("Two pending values at the same offset: %#v", old))
	}
	return result
}

func (item *compoundItem) AppendBytes(bs []byte) {
	item.encodeInt(TypeBytes, len(bs))
	item.append(bs...)
}

// This function checks that bs holds valid UTF-8.
func (item *compoundItem) AppendUtf8(bs []byte) {
	if !utf8.Valid(bs) {
		panic(fmt.Sprintf("Invalid UTF-8 in %q.", bs))
	}
	item.encodeInt(TypeText, len(bs))
	item.append(bs...)
}

func (item *compoundItem) AppendUtf8S(str string) {
	item.AppendUtf8([]byte(str))
}

// ByteLenSoFar() returns the number of bytes from the start of item's encoding.
// It can only be called on an active item with no pending ints within its
// bounds.
func (item *compoundItem) ByteLenSoFar() int {
	var pending *PendingInt = nil
	item.pending.AscendGreaterOrEqual(&PendingInt{offset: item.startOffset},
		func(i btree.Item) bool {
			pending = i.(*PendingInt)
			return false // Only find one item.
		})
	if pending != nil {
		panic(fmt.Sprintf("Can't compute byte length of %T starting at %v "+
			"because it has a pending value of type %X at %v",
			*item, item.startOffset, pending.t, pending.offset))
	}
	return len(item.bytes) - item.startOffset
}

type Array struct {
	compoundItem
	expectedSize int
}

func (item *compoundItem) AppendArray(expectedSize int) (a *Array) {
	startOffset := len(item.bytes)
	item.encodeInt(TypeArray, expectedSize)
	a = &Array{
		compoundItem: compoundItem{
			buffer:      item.buffer,
			parent:      item,
			count:       0,
			startOffset: startOffset,
		},
		expectedSize: expectedSize,
	}
	item.activeChild = &a.compoundItem
	return
}

func (a *Array) Finish() {
	if a.activeChild != nil {
		panic(fmt.Sprintf("Must finish child %v before its parent %v.",
			a.activeChild, a))
	}
	if a.count != a.expectedSize {
		panic(fmt.Sprintf("Array has size %v but was initialized with size %v",
			a.count, a.expectedSize))
	}
	a.parent.activeChild = nil
	a.buffer = nil
}

type Map struct {
	compoundItem
	expectedSize int
}

func (item *compoundItem) AppendMap(expectedSize int) *Map {
	startOffset := len(item.bytes)
	item.encodeInt(TypeMap, expectedSize)
	m := &Map{
		compoundItem: compoundItem{
			buffer:      item.buffer,
			parent:      item,
			count:       0,
			startOffset: startOffset,
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
	if m.count%2 != 0 {
		panic("Map's last key is missing a value.")
	}
	if m.count != m.expectedSize*2 {
		panic(fmt.Sprintf("Map has size %v but was initialized with size %v",
			m.count/2, m.expectedSize))
	}
	m.parent.activeChild = nil
	m.buffer = nil
}
