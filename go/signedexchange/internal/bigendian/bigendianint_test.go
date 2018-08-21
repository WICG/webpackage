package bigendian_test

import (
	"bytes"
	"testing"

	"github.com/WICG/webpackage/go/signedexchange/internal/bigendian"
)

func TestEncodeBytesUint(t *testing.T) {
	b, err := bigendian.EncodeBytesUint(0x123456, 3)
	if !bytes.Equal(b[:], []byte{0x12, 0x34, 0x56}) {
		t.Errorf("unexpected bytes: got %v", b)
		return
	}
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}

	if _, err = bigendian.EncodeBytesUint(0x12345678, 3); err != bigendian.ErrOutOfRange {
		t.Errorf("unexpected err: %v", err)
	}

	b, err = bigendian.EncodeBytesUint(0x123456, 8)
	if !bytes.Equal(b[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56}) {
		t.Errorf("unexpected bytes: got %v", b)
		return
	}

	b, err = bigendian.EncodeBytesUint(0x7ffffffffffffffe, 8)
	if !bytes.Equal(b[:], []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe}) {
		t.Errorf("unexpected bytes: got %v", b)
		return
	}
}

func TestDecode3BytesUint(t *testing.T) {
	expected := 0xabcdef
	actual := bigendian.Decode3BytesUint([...]byte{0xab, 0xcd, 0xef})
	if expected != actual {
		t.Errorf("expected decoded value %v but got %v", expected, actual)
	}
}
