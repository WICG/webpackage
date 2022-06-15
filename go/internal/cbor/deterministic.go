package cbor

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Deterministic function loops through the given byte array containing CBOR sequence
// and checks that they follow the deterministic principles described here:
// https://www.rfc-editor.org/rfc/rfc8949.html#name-deterministically-encoded-c.
func Deterministic(input []byte) error {
	index := 0

	for index < len(input) {
		// TODO(sonkkeli): Remove debug print.
		fmt.Printf("index=%v\n", index)

		length, err := deterministicRec(input[index:])
		if err != nil {
			return err
		}
		index += length
	}
	return nil
}

// deterministicRec is a recursively called helper function to check the deterministicy of a bytes array.
// TODO(sonkkeli): Recursively called parts will be the maps and arrays, which contain other types, meaning
// unsigned integers, byte strings, text strings.
func deterministicRec(input []byte) (int, error) {
	b := input[0]

	switch getMajorType(b) {
	case TypePosInt:
		length, _, err := unsignedIntegerDeterministic(input)
		if err != nil {
			return 0, err
		}
		return length + 1, nil

	case TypeBytes, TypeText:
		// TODO(sonkkeli):
		// length, err := textOrBytesDeterministic(input[index:])

		return 0, nil

	case TypeArray:
		// TODO(sonkkeli):
		// length, err := arrayDeterministic(input[index:])

		return 0, nil

	case TypeMap:
		// TODO(sonkkeli):
		// length, err := mapDeterministic(input[index:])

		return 0, nil

	default:
		return 0, errors.New("Missing implementation for major type.")
	}

}

// unsignedIntegerDeterministic calculates the value of the unsigned integer to ensure that
// the right amount of bytes is used in the CBOR encoding. It returns both,
// the length of the bytes array and the value of the unsigned integer.
func unsignedIntegerDeterministic(input []byte) (int, uint64, error) {
	ainfo := convertToAdditionalInfo(input[0])

	if ainfo == AdditionalInfoIndefinite || ainfo == AdditionalInfoReserved {
		return -1, 0, fmt.Errorf("%v should not be in use in deterministic CBOR.", ainfo)
	}

	lengthInBytes := ainfo.getAdditionalInfoLength()
	limit := ainfo.getAdditionalInfoValueLowerLimit()
	value := getUnsignedIntegerValue(input[0:], ainfo)

	if value < limit {
		return -1, value, fmt.Errorf("When following CBOR deterministic principles, %v should not be represented with %v bytes.", value, len(input)-1)
	}

	return lengthInBytes, value, nil
}

// getUnsignedIntegerValue converts the byte array containing the CBOR encoded unsigned integer into an actual value.
func getUnsignedIntegerValue(input []byte, ainfo AdditionalInfo) uint64 {
	switch ainfo {
	case AdditionalInfoDirect:
		// Get the value from the last 5 bits of the byte.
		return uint64(getAdditionalInfoDirectValue(input[0]))

	case AdditionalInfoOneByte:
		return uint64(input[1])

	case AdditionalInfoTwoBytes:
		return uint64(binary.BigEndian.Uint16(input[1:]))

	case AdditionalInfoFourBytes:
		return uint64(binary.BigEndian.Uint32(input[1:]))

	case AdditionalInfoEightBytes:
		return binary.BigEndian.Uint64(input[1:])

	default:
		panic("getUnsignedIntegerValue() should never be called with: " + fmt.Sprint(ainfo))
	}
}
