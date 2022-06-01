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
