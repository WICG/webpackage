package signedexchange

import (
	"errors"
)

var ErrOutOfRange = errors.New("signedexchange: Given integer is out of encodable range.")

func Encode3BytesBigEndianUint(n int) ([3]byte, error) {
	if n < 0 || n > 0xffffff {
		return [3]byte{}, ErrOutOfRange
	}

	return [3]byte{
		byte((n >> 16) & 0xff),
		byte((n >> 8) & 0xff),
		byte(n & 0xff),
	}, nil
}

func Decode3BytesBigEndianUint(b []byte) int {
	if len(b) != 3 {
		panic("len(b) must be 3")
	}
	return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}
