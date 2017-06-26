package webpack

import (
	"bytes"
	"fmt"
	"testing"

	"golang.org/x/net/http2/hpack"

	"github.com/WICG/webpackage/go/webpack/cbor"
	"github.com/stretchr/testify/assert"
)

func TestParseCBOR(t *testing.T) {
}

func hpackByteArray(headersAndValues ...string) []byte {
	var buf bytes.Buffer
	encoder := hpack.NewEncoder(&buf)
	if len(headersAndValues)%2 != 0 {
		panic(fmt.Sprintf("Header without value: %v", headersAndValues))
	}
	for i := 0; i < len(headersAndValues); i += 2 {
		encoder.WriteField(httpHeader(headersAndValues[i], headersAndValues[i+1]))
	}
	result := buf.Bytes()
	return append(cbor.Encoded(cbor.TypeBytes, len(result)), result...)
}

func TestWriteCBOR(t *testing.T) {
	pack := Package{
		parts: []*PackPart{
			&PackPart{
				requestHeaders: HTTPHeaders{
					httpHeader(":method", "GET"),
					httpHeader(":scheme", "https"),
					httpHeader(":authority", "example.com"),
					httpHeader(":path", "/index.html?query"),
				},
				responseHeaders: HTTPHeaders{
					httpHeader(":status", "200"),
					httpHeader("Content-Type", "text/html"),
					httpHeader("Expires", "Mon, 1 Jan 2018 01:00:00 GMT"),
				},
				content: []byte("I am example.com's index.html\n"),
			},
		},
	}

	var cborPack bytes.Buffer
	err := WriteCBOR(&pack, &cborPack)
	assert.NoError(t, err)

	assert.Equal(t, bytes.Join([][]byte{
		// Outer array.
		cbor.Encoded(cbor.TypeArray, 5),
		// magic1.
		cbor.Encoded(cbor.TypeBytes, 8), []byte("🌐📦"),
		// section-offsets.
		cbor.Encoded(cbor.TypeMap, 1),
		cbor.Encoded(cbor.TypeText, 15), []byte("indexed-content"),
		cbor.Encoded(cbor.TypePosInt, 1),
		// sections.
		cbor.Encoded(cbor.TypeMap, 1),
		cbor.Encoded(cbor.TypeText, 15), []byte("indexed-content"),
		[]byte{}, cbor.Encoded(cbor.TypeArray, 2),
		[]byte{}, // index.
		[]byte{}, cbor.Encoded(cbor.TypeArray, 1),
		[]byte{}, cbor.Encoded(cbor.TypeArray, 2),
		[]byte{}, hpackByteArray(
			":method", "GET",
			":scheme", "https",
			":authority", "example.com",
			":path", "/index.html?query"),
		[]byte{}, cbor.Encoded(cbor.TypePosInt, 1),
		[]byte{}, // responses.
		[]byte{}, cbor.Encoded(cbor.TypeArray, 1),
		[]byte{}, cbor.Encoded(cbor.TypeArray, 2),
		[]byte{}, hpackByteArray(
			":status", "200",
			"content-type", "text/html",
			"expires", "Mon, 1 Jan 2018 01:00:00 GMT"),
		[]byte{}, cbor.Encoded(cbor.TypeBytes, 30),
		[]byte{}, []byte("I am example.com's index.html\n"),
		// length.
		cbor.EncodedFixedLen(8, cbor.TypePosInt, len(cborPack.Bytes())),
		// magic2.
		cbor.Encoded(cbor.TypeBytes, 8), []byte("🌐📦"),
	}, []byte{}), cborPack.Bytes())
}
