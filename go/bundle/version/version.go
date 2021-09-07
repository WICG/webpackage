package version

import (
	"bytes"
	"errors"
	"io"

	"github.com/WICG/webpackage/go/signedexchange/mice"
)

type Version string

const (
	VersionB1   Version = "b1"
	VersionB2   Version = "b2"
)

var AllVersions = []Version{
	VersionB1,
	VersionB2,
}

// HeaderMagicBytesB1 is the CBOR encoding of the 6-item array initial byte and 8-byte bytestring initial byte, followed by 🌐📦 in UTF-8.
// These bytes are for the header of b1 version.
var HeaderMagicBytesB1 = []byte{0x86, 0x48, 0xf0, 0x9f, 0x8c, 0x90, 0xf0, 0x9f, 0x93, 0xa6}

// HeaderMagicBytesB2 is the CBOR encoding of the 5-item array initial byte and 8-byte bytestring initial byte, followed by 🌐📦 in UTF-8.
// These bytes are for the header of b2 version.
var HeaderMagicBytesB2 = []byte{0x85, 0x48, 0xf0, 0x9f, 0x8c, 0x90, 0xf0, 0x9f, 0x93, 0xa6}

// VersionMagicBytesB1 is the CBOR encoding of a 4-byte byte string holding an ASCII "b1" followed by two 0 bytes
var VersionMagicBytesB1 = []byte{0x44, 0x62, 0x31, 0x00, 0x00}

// VersionMagicBytesB2 is the CBOR encoding of a 4-byte byte string holding an ASCII "b2" followed by two 0 bytes
var VersionMagicBytesB2 = []byte{0x44, 0x62, 0x32, 0x00, 0x00}

func Parse(str string) (Version, bool) {
	switch Version(str) {
	case VersionB1:
		return VersionB1, true
	case VersionB2:
		return VersionB2, true
	}
	return "", false
}

func (v Version) HeaderMagicBytes() []byte {
	switch v {
	case VersionB1:
		return append(HeaderMagicBytesB1, VersionMagicBytesB1...)
	case VersionB2:
		return append(HeaderMagicBytesB2, VersionMagicBytesB2...)
	default:
		panic("not reached")
	}
}

func ParseMagicBytes(r io.Reader) (Version, error) {
	hdrMagic := make([]byte, len(HeaderMagicBytesB1))
	if _, err := io.ReadFull(r, hdrMagic); err != nil {
		return "", err
	}
	if bytes.Compare(hdrMagic, HeaderMagicBytesB1) != 0 && bytes.Compare(hdrMagic, HeaderMagicBytesB2) != 0 {
		return "", errors.New("bundle: unrecognized header magic")
	}

	verMagic := make([]byte, len(VersionMagicBytesB1))
	if _, err := io.ReadFull(r, verMagic); err != nil {
		return "", err
	}
	if bytes.Compare(verMagic, VersionMagicBytesB1) == 0 {
		if bytes.Compare(hdrMagic, HeaderMagicBytesB1) != 0 {
			return "", errors.New("bundle: header magic bytes does not match version magic bytes")
		}
		return VersionB1, nil
	}
	if bytes.Compare(verMagic, VersionMagicBytesB2) == 0 {
		if bytes.Compare(hdrMagic, HeaderMagicBytesB2) != 0 {
			return "", errors.New("bundle: header magic bytes does not match version magic bytes")
		}
		return VersionB2, nil
	}
	return "", errors.New("bundle: unrecognized version magic")
}

func (v Version) MiceEncoding() mice.Encoding {
	switch v {
	case VersionB1:
		return mice.Draft03Encoding
	default:
		panic("not reached")
	}
}

func (v Version) SignatureContextString() string {
	switch v {
	case VersionB1:
		return "Web Package 1 b1"
	case VersionB2:
		return "Web Package 1 b2"
	default:
		panic("not reached")
	}
}

func (v Version) HasPrimaryURLFieldInHeader() bool {
	return v == VersionB1
}

// TODO(myrzakereyms): change this to 'v == VersionB1' like above
// as B1 is the only version that still supports variants, leaving
// it like this for now until the proper removal of the variant
// support.
func (v Version) SupportsVariants() bool {
	return true
}

// TODO: consider changing this also to only version B1, as the
// signatures section is not a part of the main spec anymore.
// Currently returns true as both B1 and B2 can have signatures
// (temporarily).
func (v Version) SupportsSignatures() bool {
	return true
}
