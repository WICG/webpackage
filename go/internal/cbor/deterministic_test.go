package cbor

import (
	"io/ioutil"
	"os"
	"testing"
)

var (
	// Deterministically encoded unsigned integers.
	uint0          = []byte{0x00}
	uint10         = []byte{0x0A}
	uint23         = []byte{0x17}
	uint24         = []byte{0x18, 0x18}
	uint45         = []byte{0x18, 0x2D}
	uint255        = []byte{0x18, 0xFF}
	uint256        = []byte{0x19, 0x01, 0x00}
	uint5000       = []byte{0x19, 0x13, 0x88}
	uint65535      = []byte{0x19, 0xFF, 0xFF}
	uint65536      = []byte{0x1A, 0x00, 0x01, 0x00, 0x00}
	uint4294967    = []byte{0x1A, 0x00, 0x41, 0x89, 0x37}
	uint4294967295 = []byte{0x1A, 0xFF, 0xFF, 0xFF, 0xFF}
	uint4294967296 = []byte{0x1B, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}

	// Long.MAX_VALUE aka 0x7F 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF.
	uint9223372036854775807 = []byte{0x1B, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	// Max CBOR supported value 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF.
	uint18446744073709551615 = []byte{0x1B, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	uint64TestCasesAsBytes = [][]byte{uint0, uint10, uint23, uint24, uint45, uint255, uint256, uint5000, uint65535, uint65536, uint4294967, uint4294967295, uint4294967296, uint9223372036854775807, uint18446744073709551615}
	uint64Values           = []uint64{0, 10, 23, 24, 45, 255, 256, 5000, 65535, 65536, 4294967, 4294967295, 4294967296, 9223372036854775807, 18446744073709551615}
)

func TestUintDeterministic(t *testing.T) {
	for _, testCase := range uint64TestCasesAsBytes {
		if err := Deterministic(testCase); err != nil {
			t.Error("Deterministically encoded unsigned integers should not return error.")
		}
	}
}

func TestUintCborSequenceDeterministic(t *testing.T) {
	concatenatedTestCases := multiappend(uint64TestCasesAsBytes...)
	if err := Deterministic(concatenatedTestCases); err != nil {
		t.Error("Deterministically encoded unsigned integers should not return false for deterministicy.")
	}
}

// TestNotDeterministic tests that if the value of the unsigned integer is kept the same but there are
// additional non-necessary empty bytes added in-front of the actual value's bytes, it is no longer
// considered as deterministically encoded CBOR.
func TestUintNotDeterministic(t *testing.T) {
	// The additional information values for representing the number with 1, 2, 4 or 8 bytes.
	firstBytes := []byte{0x18, 0x19, 0x1A, 0x1B}

	for _, testCase := range uint64TestCasesAsBytes {
		for _, firstByte := range firstBytes {
			ainfo := convertToAdditionalInfo(firstByte)
			newLength := ainfo.getAdditionalInfoLength()

			// Cannot represent too big number with too little amount of bytes.
			if len(testCase) >= newLength {
				continue
			}

			nonDeterministicBytes := convertToNonDeterministicUintHelper(testCase, firstByte)
			valueOfOriginalByteArr := getUnsignedIntegerValue(testCase, convertToAdditionalInfo(testCase[0]))
			valueOfNonDeterministicByteArr := getUnsignedIntegerValue(nonDeterministicBytes, convertToAdditionalInfo(nonDeterministicBytes[0]))

			if valueOfOriginalByteArr != valueOfNonDeterministicByteArr {
				t.Error("valueOfOriginalByteArr and valueOfNonDeterministicByteArr should match.")
			}

			if err := Deterministic(nonDeterministicBytes); err == nil {
				t.Error("Non-deterministically encoded unsigned integers should not return true for deterministicy.")
			}
		}
	}
}

func TestEmptyByteArrayIsDeterministic(t *testing.T) {
	if err := Deterministic([]byte{}); err != nil {
		t.Error("Empty byte array should return true.")
	}
}

func TestUnsupportedAdditionalInformationValues(t *testing.T) {
	notSupportedStartBytes := []byte{0x1C, 0x1D, 0x1E /*AdditionalInfoReserved*/, 0x1F /*AdditionalInfoInfinite*/}

	for _, b := range notSupportedStartBytes {
		ainfo := convertToAdditionalInfo(b)

		shouldPanic(t, func() {
			ainfo.getAdditionalInfoLength()
		})

		if err := Deterministic([]byte{b}); err == nil {
			t.Error("Using AdditionalInfoReserved and AdditionalInfoInfinite should not return true for deterministicy.")
		}
	}
}

func TestGetUnsignedIntegerValue(t *testing.T) {
	for i, testCase := range uint64TestCasesAsBytes {
		res := getUnsignedIntegerValue(testCase, convertToAdditionalInfo(testCase[0]))

		if res != uint64Values[i] {
			t.Errorf("deterministic: getUnsignedIntegerValue, got %v, wanted %v", res, uint64Values[i])
		}
	}
}

func TestAdditionalInfoConversion(t *testing.T) {
	for i := 0; i <= 23; i++ {
		if got := convertToAdditionalInfo(byte(i)); got != AdditionalInfoDirect {
			t.Errorf("deterministic: convertToAdditionalInfo, got %v, wanted %v", got, AdditionalInfoDirect)
		}
	}

	for b, wanted := range map[byte]AdditionalInfo{
		24: AdditionalInfoOneByte,
		25: AdditionalInfoTwoBytes,
		26: AdditionalInfoFourBytes,
		27: AdditionalInfoEightBytes,
		28: AdditionalInfoReserved,
		29: AdditionalInfoReserved,
		30: AdditionalInfoReserved,
		31: AdditionalInfoIndefinite,
	} {
		if got := convertToAdditionalInfo(b); got != wanted {
			t.Errorf("deterministic: convertToAdditionalInfo, got %v, wanted %v", got, wanted)
		}
	}
}

func TestByteStringIsDeterministic(t *testing.T) {
	testBytes := []byte{toCborHeader(TypeBytes, 3), 0x1A, 0x2B, 0x3C}
	testBytesAsCborSequence := multiappend(testBytes, testBytes)

	for _, testCase := range [][]byte{testBytes, testBytesAsCborSequence} {
		if err := Deterministic(testCase); err != nil {
			t.Error("Deterministically encoded byte string should not return false for deterministicy.")
		}
	}
}

func TestLongTextAndByteStringIsDeterministic(t *testing.T) {
	longStr := "olipakerrankilpikonnajakissajotkajuoksivatkilpaajakilpikonnavoitti"
	testBytesAsByteString := multiappend([]byte{toCborHeader(TypeBytes, 24)}, []byte{byte(len(longStr))}, []byte(longStr))
	testBytesAsTextString := multiappend([]byte{toCborHeader(TypeText, 24)}, []byte{byte(len(longStr))}, []byte(longStr))

	for _, testCase := range [][]byte{testBytesAsByteString, testBytesAsTextString} {
		if err := Deterministic(testCase); err != nil {
			t.Error("Deterministically encoded text or byte string should not return false for deterministicy.")
		}
	}
}

func TestTextStringIsDeterministic(t *testing.T) {
	hello := multiappend([]byte{toCborHeader(TypeText, 5)}, []byte("hello"))
	hello2 := multiappend([]byte{toCborHeader(TypeText, 6)}, []byte("hello2"))
	combinedHellos := multiappend(hello, hello2)

	for _, testCase := range [][]byte{hello, hello2, combinedHellos} {
		if err := Deterministic(testCase); err != nil {
			t.Error("Deterministically encoded text string should not return false for deterministicy.")
		}
	}
}

func TestByteStringIsNotDeterministic(t *testing.T) {
	// Claiming it would be followed by 3 bytes but there are only 2.
	testBytes := []byte{toCborHeader(TypeBytes, 3), 0x1A, 0x2B}

	shouldPanic(t, func() {
		textOrByteStringDeterministic(testBytes)
	})
}

func TestArrayIsDeterministic(t *testing.T) {
	arr := multiappend([]byte{toCborHeader(TypeArray, 5)}, uint23, uint24, uint45, uint4294967295, uint18446744073709551615)
	cborSequenceArray := multiappend(arr, arr)

	for _, testCase := range [][]byte{arr, cborSequenceArray} {
		if err := Deterministic(testCase); err != nil {
			t.Error("Deterministically encoded CBOR array should not return false for deterministicy.")
		}
	}
}

func TestEmptyArrayIsDeterministic(t *testing.T) {
	if err := Deterministic([]byte{toCborHeader(TypeArray, 0)}); err != nil {
		t.Error("Empty array should not return false for deterministicy.")
	}
}

func TestLongArrayIsDeterministic(t *testing.T) {
	const numOfItems = 24
	longArr := make([]byte, 2 /*=num of startBytes*/ +len(uint45)*numOfItems)
	longArr[0] = toCborHeader(TypeArray, 24) // ARR (4) + ONE_BYTE (24)
	longArr[1] = byte(numOfItems)

	// Fill in the byte array with 24 times uint45 bytes.
	for i := 2; i < len(longArr); i += len(uint45) {
		for j := 0; j < len(uint45); j++ {
			longArr[i+j] = uint45[j]
		}
	}

	cborSequenceArray := multiappend(longArr, longArr)

	for _, testCase := range [][]byte{longArr, cborSequenceArray} {
		if err := Deterministic(testCase); err != nil {
			t.Error("Deterministically encoded CBOR array should not return false for deterministicy.")
		}
	}
}

func TestArraysNumberOfItemsIsWrong(t *testing.T) {
	// There is one item too little. The additional information claims that there are 5 but there are only 4.
	arr := multiappend([]byte{toCborHeader(TypeArray, 5)}, uint23, uint24, uint45, uint4294967295)
	shouldPanic(t, func() {
		Deterministic(arr)
	})
}

func TestArrayContainsNonDeterministicCbor(t *testing.T) {
	arr := multiappend([]byte{toCborHeader(TypeArray, 5)}, uint23, uint24, uint45, convertToNonDeterministicUintHelper(uint255, 0x19), uint4294967295)
	if err := Deterministic(arr); err == nil {
		t.Error("Deterministically encoded CBOR array should not return true for deterministicy.")
	}
}

func TestMapIsDeterministic(t *testing.T) {
	hello := multiappend([]byte{toCborHeader(TypeBytes, 5)}, []byte("hello"))
	hello2 := multiappend([]byte{toCborHeader(TypeBytes, 6)}, []byte("hello2"))
	testMap := multiappend([]byte{toCborHeader(TypeMap, 2)}, hello, uint24, hello2, uint4294967295)
	cborSequenceMap := multiappend(testMap, testMap)

	for _, testCase := range [][]byte{testMap, cborSequenceMap} {
		if err := Deterministic(testCase); err != nil {
			t.Error("Deterministically encoded CBOR map should not return false for deterministicy.")
		}
	}
}

func TestMapWithDuplicateKeysIsNotDeterministic(t *testing.T) {
	hello := multiappend([]byte{toCborHeader(TypeBytes, 5)}, []byte("hello"))
	testMap := multiappend([]byte{toCborHeader(TypeMap, 2)}, hello, uint24, hello, uint4294967295)

	if err := Deterministic(testMap); err == nil {
		t.Error("CBOR map with duplicate keys should not return true for deterministicy.")
	}
}

func TestMapWithUnorderedKeysIsNotDeterministic(t *testing.T) {
	hello := multiappend([]byte{toCborHeader(TypeBytes, 5)}, []byte("hello"))
	hello2 := multiappend([]byte{toCborHeader(TypeBytes, 6)}, []byte("hello2"))

	// 0b01100101 (0x65) should become before 0b01100110 (0x66), so this doesn't follow the lexicographical order.
	testMap := multiappend([]byte{toCborHeader(TypeMap, 2)}, hello2, uint24, hello, uint4294967295)

	if err := Deterministic(testMap); err == nil {
		t.Error("CBOR map with unordered keys should not return true for deterministicy.")
	}
}

func TestMapWithDuplicateNotSequentialKeysIsNotDeterministic(t *testing.T) {
	hello := multiappend([]byte{toCborHeader(TypeBytes, 5)}, []byte("hello"))
	hello2 := multiappend([]byte{toCborHeader(TypeBytes, 6)}, []byte("hello2"))
	testMap := multiappend([]byte{toCborHeader(TypeMap, 3)}, hello, uint24, hello2, uint4294967295, hello, uint18446744073709551615)

	if err := Deterministic(testMap); err == nil {
		t.Error("CBOR map with duplicate keys (which are not following each other) should not return true for deterministicy.")
	}
}

func TestMapsNumberOfItemsIsWrong(t *testing.T) {
	// There is one item too little. The additional information claims that there are
	// 2 pairs (=4 items) but there are only 1 pair and 1 item (3 items).
	testMap := multiappend([]byte{toCborHeader(TypeMap, 2)}, uint23, uint24, uint45)
	shouldPanic(t, func() {
		Deterministic(testMap)
	})
}

// The test web bundle have generated only contains deterministic CBOR. This test is to make
// sure that we don't accidentally break our deterministic checker. It contains a lot of
// nested CBOR types so it might catch something that has not been covered with other tests.
func TestWebBundleHasDeterministicCbor(t *testing.T) {
	bundleFile, err := os.Open("../../integrityblock/testfile.wbn")
	if err != nil {
		t.Error("Failed to open the test file")
	}
	defer bundleFile.Close()
	bundleBytes, err := ioutil.ReadAll(bundleFile)
	if err != nil {
		t.Error("Failed to read the test file")
	}

	if err := Deterministic(bundleBytes); err != nil {
		t.Error("testfile.wbn only contains deterministic CBOR and this shouldn't fail.")
	}
}

// Helper functions:

func convertToNonDeterministicUintHelper(deterministicUintBytes []byte, firstByte byte) []byte {
	newLength := convertToAdditionalInfo(firstByte).getAdditionalInfoLength()
	shift := 0
	if convertToAdditionalInfo(deterministicUintBytes[0]) != AdditionalInfoDirect {
		// If it's a direct value, the first byte is not part of the actual value.
		shift = 1
	}

	emptyExtraBytes := make([]byte, newLength-len(deterministicUintBytes)+shift)

	return multiappend([]byte{firstByte}, emptyExtraBytes, deterministicUintBytes[shift:])
}

func multiappend(inputs ...[]byte) []byte {
	var res []byte
	for _, input := range inputs {
		res = append(res, input...)
	}
	return res
}

func shouldPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() { _ = recover() }()
	f()
	t.Errorf("should have panicked")
}

func toCborHeader(majorType Type, ainfoValue int) byte {
	if ainfoValue < 0 || ainfoValue > 27 {
		panic("The value of the additional information should be between 0 and 27 inclusive.")
	}
	return byte(majorType) | byte(ainfoValue)
}
