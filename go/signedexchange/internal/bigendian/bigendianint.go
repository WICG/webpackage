package bigendian

import (
	"errors"
)

var ErrOutOfRange = errors.New("bigendian: Given integer is out of encodable range.")

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

func Decode3BytesBigEndianUint(b [3]byte) int {
	return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}
