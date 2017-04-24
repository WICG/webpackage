// Defines a parser and encoder for a subset of CBOR, RFC7049.
//
// Supported:
//  * Major types 0-3, with minimal encodings for the integers and lengths.
//  * Major type 0, with a fixed 64-bit encoding.
//  * Major types 4-5 (arrays and maps), with known lengths at the start of the
//    container, encoded minimally.
//  * Unsigned integers whose value isn't known when they're first inserted.
//  * Retrieval of the current offset within an array or map.
//
// Unsupported:
//  * Negative integers that don't fit in a 2's-complement int64.
//  * Floating-point numbers
//  * Indefinite-length encodings.
//  * Parsing
package cbor

type Type byte

const (
	TypeUint Type = iota << 5
	TypeSint
	TypeBytes
	TypeText
	TypeArray
	TypeMap
	TypeTag
	TypeOther
)
