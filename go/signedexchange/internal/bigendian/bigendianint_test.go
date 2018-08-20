package bigendian_test

import (
	"bytes"
	"testing"

	"github.com/WICG/webpackage/go/signedexchange/internal/bigendian"
)

func TestEncodeBytesUint(t *testing.T) {
	b, err := bigendian.EncodeBytesUint(0x123456, 3)
	if !bytes.Equal(b[:], []byte{0x12, 0x34, 0x56}) {
		t.Errorf("unexpected bytes")
		return
	}
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}

	if _, err = bigendian.EncodeBytesUint(0x12345678, 3); err != bigendian.ErrOutOfRange {
		t.Errorf("unexpected err: %v", err)
	}
}

func TestDecode3BytesUint(t *testing.T) {
	expected := 0xabcdef
	actual := bigendian.Decode3BytesUint([...]byte{0xab, 0xcd, 0xef})
	if expected != actual {
		t.Errorf("expected decoded value %v but got %v", expected, actual)
	}
}
