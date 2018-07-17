package bigendian_test

import (
	"bytes"
	"testing"

	"github.com/WICG/webpackage/go/signedexchange/internal/bigendian"
)

func TestEncode3BytesBigEndianUint(t *testing.T) {
	b, err := bigendian.Encode3BytesBigEndianUint(0x123456)
	if !bytes.Equal(b[:], []byte{0x12, 0x34, 0x56}) {
		t.Errorf("unexpected bytes")
		return
	}
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}

	if _, err = bigendian.Encode3BytesBigEndianUint(0x12345678); err != bigendian.ErrOutOfRange {
		t.Errorf("unexpected err: %v", err)
	}
}

func TestDecode3BytesBigEndianUint(t *testing.T) {
	expected := 0xabcdef
	actual := bigendian.Decode3BytesBigEndianUint([...]byte{0xab, 0xcd, 0xef})
	if expected != actual {
		t.Errorf("expected decoded value %v but got %v", expected, actual)
	}
}
