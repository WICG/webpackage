package bigendian

import (
	"errors"
)

var ErrOutOfRange = errors.New("bigendian: Given integer is out of encodable range.")

func EncodeBytesUint(n int64, size int) ([]byte, error) {
	if n < 0 {
		return nil, ErrOutOfRange
	}
	if size < 7 && n > int64(1)<<uint(size*8) {
		return nil, ErrOutOfRange
	}

	bs := make([]byte, size)
	for i := range bs {
		bs[i] = byte(n >> uint((size-i-1)*8) & 0xff)
	}
	return bs, nil
}

func Decode3BytesUint(b [3]byte) int {
	return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}
