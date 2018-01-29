package cbor

import (
	"bytes"
	"errors"
	"io"
	"unicode/utf8"
)

var (
	ErrInvalidUTF8 = errors.New("Cannot encode invalid UTF-8.")
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w}
}

func (e *Encoder) encodeTypedUInt(t Type, n uint64) error {
	// Major type 0:  an unsigned integer.  The 5-bit additional information
	//   is either the integer itself (for additional information values 0
	//   through 23) or the length of additional data.  Additional
	//   information 24 means the value is represented in an additional
	//   uint8_t, 25 means a uint16_t, 26 means a uint32_t, and 27 means a
	//   uint64_t.  For example, the integer 10 is denoted as the one byte
	//   0b000_01010 (major type 0, additional information 10).  The
	//   integer 500 would be 0b000_11001 (major type 0, additional
	//   information 25) followed by the two bytes 0x01f4, which is 500 in
	//   decimal.
	//
	// https://tools.ietf.org/html/rfc7049#section-2.1
	ai := byte(0) // "additional information"
	nfollow := 0  // length of the following bytes
	switch {
	case n < 24:
		ai = byte(n)
		nfollow = 0
	case n < (1 << 8):
		ai = 24
		nfollow = 1
	case n < (1 << 16):
		ai = 25
		nfollow = 2
	case n < (1 << 32):
		ai = 26
		nfollow = 4
	default:
		ai = 27
		nfollow = 8
	}

	encoded := make([]byte, 1+nfollow)
	encoded[0] = byte(t) | ai
	for i := nfollow - 1; i >= 0; i-- {
		encoded[i+1] = byte(n)
		n >>= 8
	}

	if _, err := e.w.Write(encoded); err != nil {
		return err
	}
	return nil
}

func (e *Encoder) EncodeUInt(n uint64) error {
	return e.encodeTypedUInt(TypePosInt, n)
}

func (e *Encoder) EncodeInt(n int64) error {
	if n >= 0 {
		return e.encodeTypedUInt(TypePosInt, uint64(n))
	}

	// Major type 1:  a negative integer.  The encoding follows the rules
	//   for unsigned integers (major type 0), except that the value is
	//   then -1 minus the encoded unsigned integer.  For example, the
	//   integer -500 would be 0b001_11001 (major type 1, additional
	//   information 25) followed by the two bytes 0x01f3, which is 499 in
	//   decimal.
	//
	// https://tools.ietf.org/html/rfc7049#section-2.1
	return e.encodeTypedUInt(TypeNegInt, uint64(-n)-1)
}

func (e *Encoder) encodeBytes(t Type, bs []byte) error {
	if err := e.encodeTypedUInt(t, uint64(len(bs))); err != nil {
		return err
	}
	if _, err := e.w.Write(bs); err != nil {
		return err
	}
	return nil
}

func (e *Encoder) EncodeByteString(bs []byte) error {
	// Major type 2:  a byte string.  The string's length in bytes is
	//   represented following the rules for positive integers (major type
	//   0).  For example, a byte string whose length is 5 would have an
	//   initial byte of 0b010_00101 (major type 2, additional information
	//   5 for the length), followed by 5 bytes of binary content.  A byte
	//   string whose length is 500 would have 3 initial bytes of
	//   0b010_11001 (major type 2, additional information 25 to indicate a
	//   two-byte length) followed by the two bytes 0x01f4 for a length of
	//   500, followed by 500 bytes of binary content.
	//
	// https://tools.ietf.org/html/rfc7049#section-2.1
	return e.encodeBytes(TypeBytes, bs)
}

func (e *Encoder) EncodeTextString(s string) error {
	// Major type 3:  a text string, specifically a string of Unicode
	//   characters that is encoded as UTF-8 [RFC3629].  The format of this
	//   type is identical to that of byte strings (major type 2), that is,
	//   as with major type 2, the length gives the number of bytes.  This
	//   type is provided for systems that need to interpret or display
	//   human-readable text, and allows the differentiation between
	//   unstructured bytes and text that has a specified repertoire and
	//   encoding.  In contrast to formats such as JSON, the Unicode
	//   characters in this type are never escaped.  Thus, a newline
	//   character (U+000A) is always represented in a string as the byte
	//   0x0a, and never as the bytes 0x5c6e (the characters "\" and "n")
	//   or as 0x5c7530303061 (the characters "\", "u", "0", "0", "0", and
	//   "a").
	//
	// https://tools.ietf.org/html/rfc7049#section-2.1
	bs := []byte(s)
	if !utf8.Valid(bs) {
		return ErrInvalidUTF8
	}

	return e.encodeBytes(TypeText, bs)
}

func (e *Encoder) EncodeArrayHeader(n int) error {
	// Major type 4:  an array of data items.  Arrays are also called lists,
	//   sequences, or tuples.  The array's length follows the rules for
	//   byte strings (major type 2), except that the length denotes the
	//   number of data items, not the length in bytes that the array takes
	//   up.  Items in an array do not need to all be of the same type.
	//   For example, an array that contains 10 items of any type would
	//   have an initial byte of 0b100_01010 (major type of 4, additional
	//   information of 10 for the length) followed by the 10 remaining
	//   items.
	//
	// https://tools.ietf.org/html/rfc7049#section-2.1
	return e.encodeTypedUInt(TypeArray, uint64(n))
}

func (e *Encoder) EncodeBool(b bool) error {
	ai := byte(0)
	if b {
		// True (major type 7, additional information 21)
		ai = 21
	} else {
		// False (major type 7, additional information 20)
		ai = 20
	}

	bs := []byte{TypeOther | ai}
	if _, err := e.w.Write(bs); err != nil {
		return err
	}
	return nil
}

type MapEntryEncoder struct {
	keyBuf   bytes.Buffer
	valueBuf bytes.Buffer

	keyE   *Encoder
	valueE *Encoder
}

func NewMapEntry() *MapEntryEncoder {
	e := &MapEntryEncoder{}
	e.keyE = &Encoder{&e.keyBuf}
	e.valueE = &Encoder{&e.valueBuf}
	return e
}

func (e *MapEntryEncoder) KeyBytes() []byte {
	return e.keyBuf.Bytes()
}

func GenerateMapEntry(f func(keyE *Encoder, valueE *Encoder)) *MapEntryEncoder {
	e := NewMapEntry()
	f(e.keyE, e.valueE)
	return e
}

func (e *Encoder) EncodeMap(mes []*MapEntryEncoder) error {
	if err := e.encodeTypedUInt(TypeMap, uint64(len(mes))); err != nil {
		return err
	}
	for _, me := range mes {
		if _, err := io.Copy(e.w, &me.keyBuf); err != nil {
			return err
		}
		if _, err := io.Copy(e.w, &me.valueBuf); err != nil {
			return err
		}
	}
	return nil
}
