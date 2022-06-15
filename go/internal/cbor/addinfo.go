package cbor

import "fmt"

// 5 bits of the first byte on CBOR contain the additional information which can be either the
// actual value or a value telling us how to get the value.

// The meaning of additional information depends on the major type. For example, in major type 0,
// the argument is the value of the data item itself (and in major type 1, the value of the data
// item is computed from the argument); in major type 2 and 3, it gives the length of the string
// data in bytes that follow; and in major types 4 and 5, it is used to determine the number of data
// items enclosed.

// https://www.rfc-editor.org/rfc/rfc8949.html#name-specification-of-the-cbor-e

type AdditionalInfo int

const (
	// The integers match with the length in bytes apart from reserved and indefinite.
	AdditionalInfoDirect     AdditionalInfo = iota // 0->23
	AdditionalInfoOneByte                          // 24
	AdditionalInfoTwoBytes                         // 25
	AdditionalInfoFourBytes                        // 26
	AdditionalInfoEightBytes                       // 27
	AdditionalInfoReserved                         // 28->30
	AdditionalInfoIndefinite                       // 31
)

func convertToAdditionalInfo(b byte) AdditionalInfo {
	switch b & 0b00011111 {
	case 0x18:
		return AdditionalInfoOneByte
	case 0x19:
		return AdditionalInfoTwoBytes
	case 0x1A:
		return AdditionalInfoFourBytes
	case 0x1B:
		return AdditionalInfoEightBytes
	case 0x1C, 0x1D, 0x1E:
		return AdditionalInfoReserved
	case 0x1F:
		return AdditionalInfoIndefinite
	default:
		return AdditionalInfoDirect
	}
}

// getAdditionalInfoDirectValue is used for AdditionalInfoDirect to read the value following the additional info.
func getAdditionalInfoDirectValue(b byte) byte {
	return b & 0b00011111
}

// getAdditionalInfoLength returns the length of the byte array following the additional info byte to read the unsigned integer from.
func (ainfo AdditionalInfo) getAdditionalInfoLength() int {
	switch ainfo {
	case AdditionalInfoDirect:
		return 0
	case AdditionalInfoOneByte:
		return 1
	case AdditionalInfoTwoBytes:
		return 2
	case AdditionalInfoFourBytes:
		return 4
	case AdditionalInfoEightBytes:
		return 8
	default:
		panic("getAdditionalInfoLength() should never be called with: " + fmt.Sprint(ainfo))
	}
}

// getAdditionalInfoValueLowerLimit returns the unsigned integer limit which is the lowest that should
// be using the AdditionalInfo in question. If the unsigned integer is smaller than the limit, it should
// use less bytes than it is currently using meaning it is not following the deterministic principles.
func (ainfo AdditionalInfo) getAdditionalInfoValueLowerLimit() uint64 {
	switch ainfo {
	case AdditionalInfoDirect:
		return 0
	case AdditionalInfoOneByte:
		// This is a special case and defined by the deterministic CBOR standard. Anything <= 23L
		// should be directly the value of the additional information.
		return 24
	case AdditionalInfoTwoBytes:
		return 256 // Aka 0b11111111 +1
	case AdditionalInfoFourBytes:
		return 1 << 16 // Aka 0b11111111_11111111 +1
	case AdditionalInfoEightBytes:
		return 1 << 32 // Aka 0b11111111_11111111_11111111_11111111 +1
	default:
		panic("Invalid additional information value: " + fmt.Sprint(ainfo))
	}
}
