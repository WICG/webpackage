package cbor_test

import (
	"bytes"
	"testing"

	. "github.com/WICG/webpackage/go/signedexchange/cbor"
)

func TestDecodeByteString(t *testing.T) {
	var bytesTests = []struct {
		in  []byte
		out []byte
	}{
		{
			in:  []byte{0x40},
			out: []byte{},
		},
		{
			in:  []byte{0x41, 0xab},
			out: []byte{0xab},
		},
		{
			in: []byte{
				0x58, 0x19, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
				0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d,
				0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
				0x16, 0x17, 0x18,
			},
			out: []byte{
				0, 1, 2, 3, 4, 5, 6, 7,
				8, 9, 10, 11, 12, 13, 14, 15,
				16, 17, 18, 19, 20, 21, 22, 23,
				24,
			},
		},
	}

	for _, test := range bytesTests {
		e := NewDecoder(bytes.NewReader(test.in))
		got, err := e.DecodeByteString()
		if err != nil {
			t.Errorf("Encode. err: %v", err)
		}

		if !bytes.Equal(test.out, got) {
			t.Errorf("%v expected to encode to %v, actual %v", test.in, test.out, got)
		}
	}
}

func TestDecodeByteStringNotCrashing(t *testing.T) {
	var in = []byte{0x5b, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	e := NewDecoder(bytes.NewReader(in))
	_, err := e.DecodeByteString()
	if err == nil {
		t.Error("got success, want error")
	}
}
