package version

import (
	"bytes"
	"fmt"

	"github.com/WICG/webpackage/go/signedexchange/mice"
)

type Version string

const (
	Version1b1 Version = "1b1"
	Version1b2 Version = "1b2"
	Version1b3 Version = "1b3"
)

const HeaderMagicBytesLen = 8

var AllVersions = []Version{
	Version1b1,
	Version1b2,
	Version1b3,
}

func Parse(str string) (Version, bool) {
	switch Version(str) {
	case Version1b1:
		return Version1b1, true
	case Version1b2:
		return Version1b2, true
	case Version1b3:
		return Version1b3, true
	}
	return "", false
}

func (v Version) HeaderMagicBytes() []byte {
	switch v {
	case Version1b1:
		return []byte("sxg1-b1\x00")
	case Version1b2:
		return []byte("sxg1-b2\x00")
	case Version1b3:
		return []byte("sxg1-b3\x00")
	default:
		panic("not reached")
	}
}

func (v Version) MimeType() string {
	return fmt.Sprintf("application/signed-exchange;v=%s", v[1:])
}

func FromMagicBytes(bs []byte) (Version, error) {
	if bytes.Equal(bs, Version1b1.HeaderMagicBytes()) {
		return Version1b1, nil
	} else if bytes.Equal(bs, Version1b2.HeaderMagicBytes()) {
		return Version1b2, nil
	} else if bytes.Equal(bs, Version1b3.HeaderMagicBytes()) {
		return Version1b3, nil
	} else {
		return Version(""), fmt.Errorf("signedexchange: unknown magic bytes: %v", bs)
	}
}

func (v Version) MiceEncoding() mice.Encoding {
	switch v {
	case Version1b1:
		return mice.Draft02Encoding
	case Version1b2, Version1b3:
		return mice.Draft03Encoding
	default:
		panic("not reached")
	}
}
