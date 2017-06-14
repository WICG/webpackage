package cbor

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// A Decoder helps callers iterate through a CBOR buffer, extracting item
// headers and string bodies as they go.
type Decoder struct {
	// Holds the full CBOR item being decoded, not just the remaining suffix.
	cborBuffer []byte
	Pos        int
}

func NewDecoder(buf []byte) *Decoder {
	return &Decoder{buf, 0}
}

// Decode returns the type of the current item and its value (for integers,
// tags, and 'other' types) or length (for strings, arrays, and maps). It
// advances the Decoder's current position past the item header.
func (d *Decoder) Decode() (typ Type, value uint64, err error) {
	if d.Pos == len(d.cborBuffer) {
		return 0, 0, io.EOF
	}
	typ = Type(d.cborBuffer[d.Pos] & 0xe0)
	value = uint64(d.cborBuffer[d.Pos] & 0x1f)
	pos := d.Pos + 1
	var extraBytes int
	switch {
	case value <= 23:
		d.Pos = pos
		return typ, value, nil
	case 24 <= value && value <= 27:
		extraBytes = 1 << (value - 24)
	case 28 <= value && value <= 30:
		return 0, 0, fmt.Errorf("unexpected additional information %d", value)
	case value == 31:
		return 0, 0, errors.New("indefinite-length items aren't supported")
	}
	if len(d.cborBuffer) < pos+extraBytes {
		return 0, 0, fmt.Errorf("data too short: 0x%X wants %d more bytes from %d available",
			value, extraBytes+1, len(d.cborBuffer)-d.Pos)
	}
	switch extraBytes {
	case 1:
		value = uint64(d.cborBuffer[pos])
	case 2:
		value = uint64(binary.BigEndian.Uint16(d.cborBuffer[pos:]))
	case 4:
		value = uint64(binary.BigEndian.Uint32(d.cborBuffer[pos:]))
	case 8:
		value = binary.BigEndian.Uint64(d.cborBuffer[pos:])
	}
	d.Pos = pos + extraBytes
	return typ, value, nil
}

// Read returns a slice referring to the next n bytes from the Decoder,
// advancing past them. This operation only makes sense if a byte or text
// string's header was just read.
func (d *Decoder) Read(n int) ([]byte, error) {
	if d.Pos+n > len(d.cborBuffer) {
		return nil, io.EOF
	}
	result := d.cborBuffer[d.Pos : d.Pos+n]
	d.Pos += n
	return result, nil
}
