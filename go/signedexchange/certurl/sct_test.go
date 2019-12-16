package certurl_test

import (
	"bytes"
	"testing"

	. "github.com/WICG/webpackage/go/signedexchange/certurl"
)

func TestSerializeSCTList(t *testing.T) {
	expected := []byte{
		0x00, 0x0a, // length
		0x00, 0x03, 0x01, 0x02, 0x03, // length + first SCT
		0x00, 0x03, 0x04, 0x05, 0x06, // length + second SCT
	}
	serialized, err := SerializeSCTList([][]byte{{1, 2, 3}, {4, 5, 6}})
	if err != nil {
		t.Errorf("SerializeSCTList failed: %v", err)
		return
	}
	if !bytes.Equal(expected, serialized) {
		t.Errorf("The SCTs expected to serialize to %v, actual %v", expected, serialized)
	}
}

func TestSerializeSCTListTooLarge(t *testing.T) {
	_, err := SerializeSCTList([][]byte{make([]byte, 65536)})
	if err == nil {
		t.Errorf("SerializeSCTList didn't fail with too large SCT")
	}

	// (32766 + 2) * 2 = 65536
	_, err = SerializeSCTList([][]byte{make([]byte, 32766), make([]byte, 32766)})
	if err == nil {
		t.Errorf("SerializeSCTList didn't fail with too large SCT list")
	}
}
