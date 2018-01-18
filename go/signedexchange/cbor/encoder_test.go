package cbor

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

// fromHex converts strings of the form "12 34  5678 9a" to byte slices.
func fromHex(h string) []byte {
	bytes, err := hex.DecodeString(strings.Replace(h, " ", "", -1))
	if err != nil {
		panic(err)
	}
	return bytes
}

func TestIntegers(t *testing.T) {
	var inttests = []struct {
		i        int64
		encoding string
	}{
		{0, "00"},
		{1, "01"},
		{10, "0a"},
		{23, "17"},
		{24, "1818"},
		{25, "1819"},
		{100, "1864"},
		{255, "18ff"},
		{256, "190100"},
		{1000, "1903e8"},
		{1000000, "1a000f4240"},
		{1000000000000, "1b000000e8d4a51000"},
		{-1, "20"},
		{-10, "29"},
		{-100, "3863"},
		{-1000, "3903e7"},
		{-9223372036854775808, "3b7fffffffffffffff"},
	}
	for _, test := range inttests {
		var b bytes.Buffer
		e := &Encoder{&b}

		if err := e.EncodeInt(test.i); err != nil {
			t.Errorf("Encode. err: %v", err)
		}
		expected := fromHex(test.encoding)

		if !bytes.Equal(expected, b.Bytes()) {
			t.Errorf("%d expected to encode to %v, actual %v", test.i, expected, b.Bytes())
		}
	}
}
