package cbor

import (
	"fmt"
	"io"
	"unicode/utf8"
)

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r}
}

func (d *Decoder) ReadByte() (byte, error) {
	b := make([]byte, 1)
	if _, err := io.ReadFull(d.r, b); err != nil {
		return 0, err
	}
	return b[0], nil
}

func (d *Decoder) decodeTypedUInt() (Type, uint64, error) {
	const (
		maskType                  = 0xe0
		maskAdditionalInformation = 0x1f
	)

	b, err := d.ReadByte()
	if err != nil {
		return TypeOther, 0, err
	}

	t := Type(b & maskType)
	ai := b & maskAdditionalInformation
	nfollow := 0
	switch ai {
	case 24:
		nfollow = 1
	case 25:
		nfollow = 2
	case 26:
		nfollow = 4
	case 27:
		nfollow = 8
	default:
		nfollow = 0
	}

	n := uint64(0)

	var follow []byte
	if nfollow > 0 {
		follow = make([]byte, nfollow)
		if _, err := io.ReadFull(d.r, follow); err != nil {
			return t, 0, fmt.Errorf("cbor: Failed to read %d bytes following the tag byte: %v", nfollow, err)
		}
		for i := 0; i < nfollow; i++ {
			n = n<<8 | uint64(follow[i])
		}
	} else {
		n = uint64(ai)
	}

	return t, n, nil
}

func (d *Decoder) decodeUintOfType(expected Type) (uint64, error) {
	t, n, err := d.decodeTypedUInt()
	if err != nil {
		return 0, err
	}
	if t != expected {
		return 0, fmt.Errorf("cbor: Expected type %v, got type %v", expected, t)
	}
	return n, nil
}

func (d *Decoder) DecodeUInt() (uint64, error) {
	return d.decodeUintOfType(TypePosInt)
}

func (d *Decoder) DecodeArrayHeader() (uint64, error) {
	return d.decodeUintOfType(TypeArray)
}

func (d *Decoder) DecodeMapHeader() (uint64, error) {
	return d.decodeUintOfType(TypeMap)
}

func (d *Decoder) decodeBytesOfType(expected Type) ([]byte, error) {
	n, err := d.decodeUintOfType(expected)
	if err != nil {
		return nil, err
	}
	bs := make([]byte, n)
	if _, err := io.ReadFull(d.r, bs); err != nil {
		return nil, err
	}
	return bs, nil
}

func (d *Decoder) DecodeTextString() (string, error) {
	bs, err := d.decodeBytesOfType(TypeText)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(bs) {
		return "", ErrInvalidUTF8
	}
	return string(bs), nil
}

func (d *Decoder) DecodeByteString() ([]byte, error) {
	return d.decodeBytesOfType(TypeBytes)
}
