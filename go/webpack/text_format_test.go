package webpack

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseText(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	unsignedSingleFile, err := ParseText("testdata/unsigned_single_file.manifest")
	require.NoError(err)
	assert.Len(unsignedSingleFile.Parts, 1, "Wrong number of parts.")

	index := unsignedSingleFile.Parts[0]
	assert.Equal(HTTPHeaders{
		httpHeader(":method", "GET"),
		httpHeader(":scheme", "https"),
		httpHeader(":authority", "example.com"),
		httpHeader(":path", "/index.html"),
	}, index.RequestHeaders)

	if url, err := index.URL(); assert.NoError(err) {
		assert.Equal(*staticUrl("https://example.com/index.html"), *url)
	}

	assert.Equal(HTTPHeaders{
		httpHeader(":status", "200"),
		httpHeader("content-type", "text/html"),
		httpHeader("expires", "Mon, 1 Jan 2018 01:00:00 GMT"),
	}, index.ResponseHeaders)

	content, err := index.Content()
	require.NoError(err)
	bytes, err := ioutil.ReadAll(content)
	require.NoError(err)
	assert.Equal(string(bytes), "I am example.com's index.html\n")
}

func TestParseTextVaryHeader(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	varyValid, err := ParseTextContent("testdata/", strings.NewReader(`[Content]
https://example.com/index.html
Allowed: value

200
Content-Type: text/html
Expires: Mon, 1 Jan 2018 01:00:00 GMT
Vary: allowed

content/example.com/index.html
`))
	require.NoError(err)
	require.Len(varyValid.Parts, 1)

	index := varyValid.Parts[0]
	assert.Equal(HTTPHeaders{
		httpHeader("allowed", "value"),
	}, index.NonPseudoRequestHeaders())
}

func TestParseTextRequestHeaderNotInVary(t *testing.T) {
	_, err := ParseTextContent("testdata/", strings.NewReader(`[Content]
https://example.com/index.html
DisAllowed: value

200
Content-Type: text/html
Expires: Mon, 1 Jan 2018 01:00:00 GMT
Vary: allowed

content/example.com/index.html
`))
	assert.Error(t, err)
}

func mustLoadCertificate(filename string) *x509.Certificate {
	var certs []*x509.Certificate
	err := LoadCertificatesFromFile(filename, &certs)
	if err != nil {
		panic(err)
	}
	return certs[0]
}

var root1 *x509.Certificate = mustLoadCertificate("testdata/pki/root1.cert")

func poolOf(certs ...*x509.Certificate) *x509.CertPool {
	result := x509.NewCertPool()
	for _, cert := range certs {
		result.AddCert(cert)
	}
	return result
}

func TestParseTextManifest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	manifestPackage, err := ParseTextContent("testdata/", strings.NewReader(`[Manifest]
hash-algorithms: sha384, sha256
sign-with: pki/example.com.cert; pki/example.com.key
certificate-chain: pki/intermediate1.cert
date: Fri, 12 May 2017 10:00:00 GMT
origin: https://example.com
unknown: "value"

[Content]
https://example.com/index.html

200
Content-Type: text/html
Expires: Mon, 1 Jan 2018 01:00:00 GMT

content/example.com/index.html
`))
	require.NoError(err)
	manifest := manifestPackage.Manifest

	assert.Equal(time.Date(2017, time.May, 12, 10, 0, 0, 0, time.UTC),
		manifest.Metadata.Date)
	assert.Equal(staticUrl("https://example.com"), manifest.Metadata.Origin)
	assert.Equal(map[string]interface{}{"unknown": "value"},
		manifest.Metadata.OtherFields)

	if assert.Len(manifest.Signatures, 1) {
		signature := manifest.Signatures[0]
		assert.NoError(signature.Certificate.VerifyHostname("example.com"))
		_, err = signature.Certificate.Verify(x509.VerifyOptions{
			DNSName:       "example.com",
			Roots:         poolOf(root1),
			Intermediates: poolOf(manifest.Certificates...),
			CurrentTime:   time.Date(2017, time.May, 17, 0, 0, 0, 0, time.UTC),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		})
		assert.NoError(err)

		assert.Nil(signature.Key)
		password, err := ioutil.ReadFile("testdata/pki/example.com.password")
		if assert.NoError(err) &&
			assert.NoError(signature.GivePassword(bytes.TrimSpace(password))) {
			assert.IsType(&ecdsa.PrivateKey{}, signature.Key)
		}
	}

	assert.Equal([]crypto.Hash{crypto.SHA256, crypto.SHA384}, manifest.HashTypes)
	assert.Len(manifest.Subpackages, 0)

	// Quickly check that the manifest didn't prevent the [Content] section
	// from parsing.
	assert.Len(manifestPackage.Parts, 1, "Wrong number of parts.")
}

func staticUrl(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestWriteText(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	pack := Package{
		Manifest: Manifest{},
		Parts: []*PackPart{
			&PackPart{
				RequestHeaders: HTTPHeaders{
					httpHeader(":method", "GET"),
					httpHeader(":scheme", "https"),
					httpHeader(":authority", "example.com"),
					httpHeader(":path", "/index.html"),
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

	dir, err := ioutil.TempDir("", "package")
	require.NoError(err)
	defer os.RemoveAll(dir)

	base := filepath.Join(dir, "unsigned_single_file")
	require.NoError(WriteTextTo(base, &pack))

	manifestContents, err := ioutil.ReadFile(filepath.Join(dir, "unsigned_single_file.manifest"))
	require.NoError(err)
	expectedManifestContents := strings.Replace(`[Content]
https://example.com/index.html

content-type: text/html
expires: Mon, 1 Jan 2018 01:00:00 GMT

https/example.com/index.html
`, "\n", "\r\n", -1)
	assert.Equal(expectedManifestContents, string(manifestContents))

	// Check that exactly the contained files were written out, to subdirectories of the manifest's basename.
	filenames := []string{}
	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			filenames = append(filenames, path[len(base)+1:])
		}
		return err
	})
	require.NoError(err)
	assert.Equal([]string{"https/example.com/index.html"}, filenames)
}
