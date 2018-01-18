package cbor

import (
	"io"
)

type Encoder struct {
	w io.Writer
}

func (e *Encoder) encodeTypedUInt(t Type, n uint64) error {
	/*
	   Major type 0:  an unsigned integer.  The 5-bit additional information
	       is either the integer itself (for additional information values 0
	       through 23) or the length of additional data.  Additional
	       information 24 means the value is represented in an additional
	       uint8_t, 25 means a uint16_t, 26 means a uint32_t, and 27 means a
	       uint64_t.  For example, the integer 10 is denoted as the one byte
	       0b000_01010 (major type 0, additional information 10).  The
	       integer 500 would be 0b000_11001 (major type 0, additional
	       information 25) followed by the two bytes 0x01f4, which is 500 in
	       decimal.
	*/
	var ai byte     // "additional information"
	var nfollow int // length of the following bytes
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

	/*
	   Major type 1:  a negative integer.  The encoding follows the rules
	       for unsigned integers (major type 0), except that the value is
	       then -1 minus the encoded unsigned integer.  For example, the
	       integer -500 would be 0b001_11001 (major type 1, additional
	       information 25) followed by the two bytes 0x01f3, which is 499 in
	       decimal.
	*/
	return e.encodeTypedUInt(TypeNegInt, uint64(-n)-1)
}
