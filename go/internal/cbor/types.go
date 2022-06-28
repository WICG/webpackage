// Package cbor defines a parser and encoder for a subset of CBOR, RFC7049.
package cbor

type Type byte

const (
	TypePosInt Type = 0
	TypeNegInt      = 0x20
	TypeBytes       = 0x40
	TypeText        = 0x60
	TypeArray       = 0x80
	TypeMap         = 0xa0
	TypeTag         = 0xc0
	TypeOther       = 0xe0
)

// getMajorType returns the first 3 bits of the first byte representing cbor's major type.
// https://www.rfc-editor.org/rfc/rfc8949.html#name-major-types
func getMajorType(b byte) Type {
	return Type(b & 0b11100000)
}
