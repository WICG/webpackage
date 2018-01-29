package cbor_test

import (
	"bytes"
	"encoding/hex"
	"sort"
	"strings"
	"testing"

	. "github.com/WICG/webpackage/go/signedexchange/cbor"
)

// fromHex converts strings of the form "12 34  5678 9a" to byte slices.
func fromHex(h string) []byte {
	bytes, err := hex.DecodeString(strings.Replace(h, " ", "", -1))
	if err != nil {
		panic(err)
	}
	return bytes
}

func TestEncodeInt(t *testing.T) {
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
		e := NewEncoder(&b)

		if err := e.EncodeInt(test.i); err != nil {
			t.Errorf("Encode. err: %v", err)
		}
		exp := fromHex(test.encoding)

		if !bytes.Equal(exp, b.Bytes()) {
			t.Errorf("%d expected to encode to %v, actual %v", test.i, exp, b.Bytes())
		}
	}
}

func TestEncodeByteString(t *testing.T) {
	var bytesTests = []struct {
		bs       []byte
		encoding string
	}{
		{[]byte{}, "40"},
		{[]byte{0xab}, "41ab"},
		{
			[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
			"5819 00 01 02 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f 10 11 12 13 14 15 16 17 18",
		},
	}
	// This doesn't test every size of integer that might encode the length
	// of the byte string.

	for _, test := range bytesTests {
		var b bytes.Buffer
		e := NewEncoder(&b)

		if err := e.EncodeByteString(test.bs); err != nil {
			t.Errorf("Encode. err: %v", err)
		}
		exp := fromHex(test.encoding)

		if !bytes.Equal(exp, b.Bytes()) {
			t.Errorf("%v expected to encode to %v, actual %v", test.bs, exp, b.Bytes())
		}
	}
}

func TestEncodeTextString(t *testing.T) {
	var textTests = []struct {
		s        string
		encoding string
	}{
		{"", "60"},
		{"a", "6161"},
		{"IETF", "6449455446"},
		{`"\`, "62225c"},
		{"\u00fc", "62c3bc"},
		{"\u6c34", "63e6b0b4"},
		{"\U00010151", "64f0908591"},
	}
	for _, test := range textTests {
		var b bytes.Buffer
		e := NewEncoder(&b)

		if err := e.EncodeTextString(test.s); err != nil {
			t.Errorf("Encode. err: %v", err)
		}
		exp := fromHex(test.encoding)

		if !bytes.Equal(exp, b.Bytes()) {
			t.Errorf("\"%s\" expected to encode to %v, actual %v", test.s, exp, b.Bytes())
		}
	}
}

func TestMapEncoderSorter(t *testing.T) {
	// The keys in every map MUST be sorted in the bytewise lexicographic order of
	// their canonical encodings. For example, the following keys are correctly
	// sorted:
	// 1. 10, encoded as 0A.
	// 2. 100, encoded as 18 64.
	// 3. -1, encoded as 20.
	// 4. “z”, encoded as 61 7A.
	// 5. “aa”, encoded as 62 61 61.
	// 6. [100], encoded as 81 18 64.
	// 7. [-1], encoded as 81 20.
	// 8. false, encoded as F4.
	es := []*MapEntryEncoder{
		GenerateMapEntry(func(keyE *Encoder, valueE *Encoder) {
			keyE.EncodeInt(10)
			valueE.EncodeTextString("int 10")
		}),
		GenerateMapEntry(func(keyE *Encoder, valueE *Encoder) {
			keyE.EncodeInt(100)
			valueE.EncodeTextString("int 100")
		}),
		GenerateMapEntry(func(keyE *Encoder, valueE *Encoder) {
			keyE.EncodeInt(-1)
			valueE.EncodeTextString("int -1")
		}),
		GenerateMapEntry(func(keyE *Encoder, valueE *Encoder) {
			keyE.EncodeTextString("z")
			valueE.EncodeTextString("string \"z\"")
		}),
		GenerateMapEntry(func(keyE *Encoder, valueE *Encoder) {
			keyE.EncodeTextString("aa")
			valueE.EncodeTextString("string \"aa\"")
		}),
		GenerateMapEntry(func(keyE *Encoder, valueE *Encoder) {
			keyE.EncodeArrayHeader(1)
			keyE.EncodeInt(100)
			valueE.EncodeTextString("array [100]")
		}),
		GenerateMapEntry(func(keyE *Encoder, valueE *Encoder) {
			keyE.EncodeArrayHeader(1)
			keyE.EncodeInt(-1)
			valueE.EncodeTextString("array [-1]")
		}),
		GenerateMapEntry(func(keyE *Encoder, valueE *Encoder) {
			keyE.EncodeBool(false)
			valueE.EncodeTextString("false")
		}),
	}

	keyExp := [][]byte{
		fromHex("0A"),
		fromHex("18 64"),
		fromHex("20"),
		fromHex("61 7A"),
		fromHex("62 61 61"),
		fromHex("81 18 64"),
		fromHex("81 20"),
		fromHex("F4"),
	}
	for i, exp := range keyExp {
		if !bytes.Equal(exp, es[i].KeyBytes()) {
			t.Errorf("FAIL key %d expected to encode to %v, actual %v ", i, exp, es[i].KeyBytes())
		}
	}

	if !sort.SliceIsSorted(es, func(i, j int) bool {
		return bytes.Compare(es[i].KeyBytes(), es[j].KeyBytes()) < 0
	}) {
		t.Errorf("FAIL")
	}
}
