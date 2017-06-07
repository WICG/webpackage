package webpack

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"golang.org/x/net/http2/hpack"

	"github.com/WICG/webpackage/go/webpack/cbor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCBOR(t *testing.T) {
}

func concat(bs ...[]byte) []byte {
	return bytes.Join(bs, []byte{})
}

func encodedBytes(s string) []byte {
	return append(cbor.Encoded(cbor.TypeBytes, len(s)), []byte(s)...)
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

func hpackDecode(t *testing.T, encoded []byte) HTTPHeaders {
	var result HTTPHeaders
	dec := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		result = append(result, f)
	})
	_, err := dec.Write(encoded)
	require.NoError(t, err)
	require.NoError(t, dec.Close())
	return result
}

func TestWriteCBOR(t *testing.T) {
	pack := Package{
		Parts: []*PackPart{
			&PackPart{
				RequestHeaders: HTTPHeaders{
					httpHeader(":method", "GET"),
					httpHeader(":scheme", "https"),
					httpHeader(":authority", "example.com"),
					httpHeader(":path", "/index.html?query"),
				},
				ResponseHeaders: HTTPHeaders{
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

func mustReadFile(filename string) []byte {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return content
}

type testDecoder struct {
	d cbor.Decoder
	t *testing.T
}

func (td *testDecoder) pos() int { return td.d.Pos }

func (td *testDecoder) decodeType(expectedType cbor.Type) uint64 {
	typ, value, err := td.d.Decode()
	require.NoError(td.t, err)
	assert.Equal(td.t, expectedType, typ)
	return value
}

func (td *testDecoder) read(n uint64) []byte {
	value, err := td.d.Read(int(n))
	require.NoError(td.t, err)
	return value
}

func TestWriteCBORManifest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	intermediateCert := mustLoadCertificate("testdata/pki/intermediate1.cert")

	signWithServerCert, err := LoadSignWith("testdata/pki/example.com.cert", "testdata/pki/example.com.key")
	require.NoError(err)
	require.NoError(signWithServerCert.GivePassword(bytes.TrimSpace(mustReadFile("testdata/pki/example.com.password"))))

	pack := Package{
		Manifest: Manifest{
			Metadata: Metadata{
				Date:   time.Date(2017, 5, 20, 10, 0, 0, 0, time.UTC),
				Origin: staticUrl("https://example.com"),
			},
			Signatures:   []SignWith{signWithServerCert},
			Certificates: []*x509.Certificate{intermediateCert},
			HashTypes:    []crypto.Hash{crypto.SHA256, crypto.SHA512},
		},
		Parts: []*PackPart{
			&PackPart{
				RequestHeaders: HTTPHeaders{
					httpHeader(":method", "GET"),
					httpHeader(":scheme", "https"),
					httpHeader(":authority", "example.com"),
					httpHeader(":path", "/index.html?query"),
				},
				ResponseHeaders: HTTPHeaders{
					httpHeader(":status", "200"),
					httpHeader("Content-Type", "text/html"),
					httpHeader("Expires", "Mon, 1 Jan 2018 01:00:00 GMT"),
				},
				content: []byte("I am example.com's index.html\n"),
			},
		},
	}

	var cborPack bytes.Buffer
	require.NoError(WriteCBOR(&pack, &cborPack))

	d := testDecoder{*cbor.NewDecoder(cborPack.Bytes()), t}

	// Outer array.
	assert.EqualValues(5, d.decodeType(cbor.TypeArray))
	// magic1.
	assert.EqualValues("🌐📦", d.read(d.decodeType(cbor.TypeBytes)))

	// section-offsets.
	assert.EqualValues(2, d.decodeType(cbor.TypeMap))
	assert.EqualValues("manifest", d.read(d.decodeType(cbor.TypeText)))
	manifestOffset := int(d.decodeType(cbor.TypePosInt))

	assert.EqualValues("indexed-content", d.read(d.decodeType(cbor.TypeText)))
	indexedContentOffset := int(d.decodeType(cbor.TypePosInt))

	sectionsStart := d.pos()
	// sections.
	assert.EqualValues(2, d.decodeType(cbor.TypeMap))
	assert.Equal(sectionsStart+manifestOffset, d.pos(),
		"section-offsets should encode the position of the manifest relative to the start of the sections item.")

	assert.EqualValues("manifest", d.read(d.decodeType(cbor.TypeText)))

	assert.EqualValues(3, d.decodeType(cbor.TypeMap))
	assert.EqualValues("manifest", d.read(d.decodeType(cbor.TypeText)))
	manifestStart := d.pos()
	assert.EqualValues(2, d.decodeType(cbor.TypeMap))
	// manifest-metadata
	assert.EqualValues("metadata", d.read(d.decodeType(cbor.TypeText)))
	assert.EqualValues(2, d.decodeType(cbor.TypeMap))

	assert.EqualValues("date", d.read(d.decodeType(cbor.TypeText)))
	assert.EqualValues(cbor.TagTime, d.decodeType(cbor.TypeTag))
	assert.EqualValues(time.Date(2017, 5, 20, 10, 0, 0, 0, time.UTC).Unix(), d.decodeType(cbor.TypePosInt))

	assert.EqualValues("origin", d.read(d.decodeType(cbor.TypeText)))

	assert.EqualValues(cbor.TagURI, d.decodeType(cbor.TypeTag))
	assert.EqualValues("https://example.com", d.read(d.decodeType(cbor.TypeText)))

	// resource-hashes
	assert.EqualValues("resource-hashes", d.read(d.decodeType(cbor.TypeText)))

	hashedIndexHtml := concat(
		cbor.Encoded(cbor.TypeArray, 3),
		cbor.Encoded(cbor.TypeArray, 8),
		encodedBytes(":method"), encodedBytes("GET"),
		encodedBytes(":scheme"), encodedBytes("https"),
		encodedBytes(":authority"), encodedBytes("example.com"),
		encodedBytes(":path"), encodedBytes("/index.html?query"),
		cbor.Encoded(cbor.TypeArray, 6),
		encodedBytes(":status"), encodedBytes("200"),
		encodedBytes("content-type"), encodedBytes("text/html"),
		encodedBytes("expires"), encodedBytes("Mon, 1 Jan 2018 01:00:00 GMT"),
		// Body:
		encodedBytes("I am example.com's index.html\n"),
	)

	assert.EqualValues(2, d.decodeType(cbor.TypeMap))
	assert.EqualValues("sha256", d.read(d.decodeType(cbor.TypeText)))
	assert.EqualValues(1, d.decodeType(cbor.TypeArray))
	sha256IndexHtml := sha256.Sum256(hashedIndexHtml)
	assert.EqualValues(sha256IndexHtml[:], d.read(d.decodeType(cbor.TypeBytes)))

	assert.EqualValues("sha512", d.read(d.decodeType(cbor.TypeText)))
	assert.EqualValues(1, d.decodeType(cbor.TypeArray))
	sha512IndexHtml := sha512.Sum512(hashedIndexHtml)
	assert.EqualValues(sha512IndexHtml[:], d.read(d.decodeType(cbor.TypeBytes)))

	manifestEnd := d.pos()
	manifestBytes := cborPack.Bytes()[manifestStart:manifestEnd]

	assert.EqualValues("signatures", d.read(d.decodeType(cbor.TypeText)))
	assert.EqualValues(1, d.decodeType(cbor.TypeArray))
	assert.EqualValues(2, d.decodeType(cbor.TypeMap))
	assert.EqualValues("keyIndex", d.read(d.decodeType(cbor.TypeText)))
	keyIndex := d.decodeType(cbor.TypePosInt)

	assert.EqualValues("signature", d.read(d.decodeType(cbor.TypeText)))
	signatureValue := d.read(d.decodeType(cbor.TypeBytes))

	assert.EqualValues("certificates", d.read(d.decodeType(cbor.TypeText)))
	numCertificates := d.decodeType(cbor.TypeArray)
	certs := make([]*x509.Certificate, numCertificates)
	for i := range certs {
		certBytes := d.read(d.decodeType(cbor.TypeBytes))
		certs[i], err = x509.ParseCertificate(certBytes)
		require.NoError(err)
	}
	signingCert := certs[keyIndex]
	signingCert.Verify(x509.VerifyOptions{
		DNSName:       "example.com",
		Roots:         poolOf(root1),
		Intermediates: poolOf(certs...),
		CurrentTime:   time.Date(2017, time.May, 17, 0, 0, 0, 0, time.UTC),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	assert.NoError(Verify(signingCert.PublicKey, manifestBytes, signatureValue))

	// indexed-content section:
	assert.Equal(sectionsStart+indexedContentOffset, d.pos(),
		"section-offsets should encode the position of the indexed-content section relative to the start of the sections item.")
	assert.EqualValues("indexed-content", d.read(d.decodeType(cbor.TypeText)))
	assert.EqualValues(2, d.decodeType(cbor.TypeArray))
	// index.
	assert.EqualValues(1, d.decodeType(cbor.TypeArray))
	assert.EqualValues(2, d.decodeType(cbor.TypeArray))
	assert.EqualValues(HTTPHeaders{
		httpHeader(":method", "GET"),
		httpHeader(":scheme", "https"),
		httpHeader(":authority", "example.com"),
		httpHeader(":path", "/index.html?query"),
	}, hpackDecode(t, d.read(d.decodeType(cbor.TypeBytes))))

	indexHtmlOffset := int(d.decodeType(cbor.TypePosInt))

	// responses.
	responsesStart := d.pos()
	assert.EqualValues(1, d.decodeType(cbor.TypeArray))
	assert.Equal(responsesStart+indexHtmlOffset, d.pos(),
		"the index should encode the position of the index.html response relative to the start of the responses item.")
	assert.EqualValues(2, d.decodeType(cbor.TypeArray))
	assert.EqualValues(HTTPHeaders{
		httpHeader(":status", "200"),
		httpHeader("content-type", "text/html"),
		httpHeader("expires", "Mon, 1 Jan 2018 01:00:00 GMT"),
	}, hpackDecode(t, d.read(d.decodeType(cbor.TypeBytes))))
	assert.EqualValues("I am example.com's index.html\n",
		d.read(d.decodeType(cbor.TypeBytes)))

	// length.
	assert.EqualValues(len(cborPack.Bytes()), d.decodeType(cbor.TypePosInt))
	// magic2.
	assert.EqualValues("🌐📦", d.read(d.decodeType(cbor.TypeBytes)))

}
