// Package cbor defines a parser and encoder for a subset of CBOR, RFC7049.
//
// Supported:
//  * Major types:
//     * 0: Unsigned integers, with both minimal and fixed-length 64-bit
//          (additional information 27) encodings.
//     * 1: Negative integers, with minimal encoding for integers.
//     * 2 & 3: Byte and UTF-8 strings, with minimal encoding for lengths.
//     * 4 & 5: Arrays and maps, with the number of elements known at the start
//              of the container, encoded minimally.
//  * Pre-encoded items, by copying from a Reader.
//  * Retrieval of the current byte offset within an array or map.
//  * Items that don't fit in memory.
//
// Unsupported:
//  * Negative integers (major type 1) between -2^63-1 and -2^64 inclusive,
//    since they don't fit in a 2's-complement int64.
//  * Floating-point numbers
//  * Indefinite-length encodings.
//  * Parsing
package cbor

import "bytes"

type Type byte

const (
	TypePosInt Type = iota << 5
	TypeNegInt
	TypeBytes
	TypeText
	TypeArray
	TypeMap
	TypeTag
	TypeOther
)

const (
	TagTime uint64 = 1
	TagURI         = 32
)

// cbor.CanonicalLessStrings compares strings in the order needed for map keys
// in Canonical CBOR: https://tools.ietf.org/html/rfc7049#section-3.9
func CanonicalLessStrings(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// cbor.CanonicalLessBytes compares []bytes in the order needed for map keys
// in Canonical CBOR: https://tools.ietf.org/html/rfc7049#section-3.9
func CanonicalLessBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return bytes.Compare(a, b) < 0
}
