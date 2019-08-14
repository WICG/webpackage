package signature_test

import (
	"bytes"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/WICG/webpackage/go/bundle"
	. "github.com/WICG/webpackage/go/bundle/signature"
	"github.com/WICG/webpackage/go/bundle/version"
	"github.com/WICG/webpackage/go/signedexchange"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

const (
	// A certificate for "example.org"
	pemCerts = `-----BEGIN CERTIFICATE-----
MIIBhjCCAS2gAwIBAgIJAOhR3xtYd5QsMAoGCCqGSM49BAMCMDIxFDASBgNVBAMM
C2V4YW1wbGUub3JnMQ0wCwYDVQQKDARUZXN0MQswCQYDVQQGEwJVUzAeFw0xODEx
MDUwOTA5MjJaFw0xOTEwMzEwOTA5MjJaMDIxFDASBgNVBAMMC2V4YW1wbGUub3Jn
MQ0wCwYDVQQKDARUZXN0MQswCQYDVQQGEwJVUzBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABH1E6odXRm3+r7dMYmkJRmftx5IYHAsqgA7zjsFfCvPqL/fM4Uvi8EFu
JVQM/oKEZw3foCZ1KBjo/6Tenkoj/wCjLDAqMBAGCisGAQQB1nkCARYEAgUAMBYG
A1UdEQQPMA2CC2V4YW1wbGUub3JnMAoGCCqGSM49BAMCA0cAMEQCIEbxRKhlQYlw
Ja+O9h7misjLil82Q82nhOtl4j96awZgAiB6xrvRZIlMtWYKdi41BTb5fX22gL9M
L/twWg8eWpYeJA==
-----END CERTIFICATE-----
`
	pemPrivateKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIEMac81NMjwO4pQ2IGKZ3UdymYtnFAXEjKdvAdEx4DQwoAoGCCqGSM49
AwEHoUQDQgAEfUTqh1dGbf6vt0xiaQlGZ+3HkhgcCyqADvOOwV8K8+ov98zhS+Lw
QW4lVAz+goRnDd+gJnUoGOj/pN6eSiP/AA==
-----END EC PRIVATE KEY-----`

	miRecordSize = 4096
	validityURL  = "https://example.org/resource.validity"
)

var signatureDate = time.Date(2018, 1, 31, 17, 13, 20, 0, time.UTC)
var signatureDuration = 1 * time.Hour

var expectedSig = []byte{
	0x30, 0x44, 0x02, 0x20, 0x17, 0x92, 0xb5, 0x06, 0xb6, 0xda, 0xff, 0x8d,
	0xc9, 0x92, 0xa5, 0x7f, 0x9d, 0x5c, 0xb2, 0xd0, 0x24, 0xb5, 0xda, 0xd2,
	0x6d, 0xa8, 0x53, 0x7d, 0x80, 0x4d, 0xf5, 0x12, 0xec, 0x5b, 0x94, 0x27,
	0x02, 0x20, 0x03, 0xfc, 0xb0, 0x1b, 0x65, 0xa7, 0xd1, 0xaa, 0x19, 0xaa,
	0x02, 0x7f, 0xa3, 0x87, 0x81, 0x41, 0x5c, 0x5a, 0x56, 0xb8, 0xef, 0x03,
	0x2e, 0xd8, 0xf4, 0x6e, 0xa1, 0x65, 0x30, 0x27, 0x37, 0x3c,
}

type zeroReader struct{}

func (zeroReader) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = 0
	}
	return len(b), nil
}

func urlMustParse(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}

func createTestCertChain(t *testing.T) certurl.CertChain {
	certs, err := signedexchange.ParseCertificates([]byte(pemCerts))
	if err != nil {
		t.Fatal(err)
	}
	chain, err := certurl.NewCertChain(certs, []byte("dummy ocsp"), nil)
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

func createTestSigner(t *testing.T) *Signer {
	certChain := createTestCertChain(t)

	privKey, err := signedexchange.ParsePrivateKey([]byte(pemPrivateKey))
	if err != nil {
		t.Fatal(err)
	}

	validityUrl := urlMustParse(validityURL)

	signer, err := NewSigner(version.VersionB1, certChain, privKey, validityUrl, signatureDate, signatureDuration)
	if err != nil {
		t.Fatalf("Failed to create Signer: %v", err)
	}
	signer.Rand = zeroReader{}
	return signer
}

func TestCanSignForURL(t *testing.T) {
	signer := createTestSigner(t)

	if !signer.CanSignForURL(urlMustParse("https://example.org/index.html")) {
		t.Error("CanSignFor unexpectedly returned false for https://example.org/index.html")
	}
	if signer.CanSignForURL(urlMustParse("https://example.com/index.html")) {
		t.Error("CanSignFor unexpectedly returned true for https://example.com/index.html")
	}
}

func TestSignatureGeneration(t *testing.T) {
	signer := createTestSigner(t)

	e := &bundle.Exchange{
		bundle.Request{
			URL: urlMustParse("https://example.org/index.html"),
		},
		bundle.Response{
			Status: 200,
			Header: http.Header{"Content-Type": []string{"text/html"}},
			Body:   []byte("hello, world!"),
		},
	}
	integrity, err := e.AddPayloadIntegrity(signer.Version, miRecordSize)
	if err != nil {
		t.Fatalf("AddPayloadIntegrity failed: %v", err)
	}

	if err := signer.AddExchange(e, integrity); err != nil {
		t.Fatalf("signer.AddExchange failed: %v", err)
	}

	signatures, err := signer.UpdateSignatures(nil)
	if err != nil {
		t.Fatalf("signer.UpdateSignatures failed: %v", err)
	}

	if len(signatures.Authorities) != 1 {
		t.Fatalf("Unexpected size of signatures.Authorities: %d", len(signatures.Authorities))
	}
	expectedCerts := createTestCertChain(t)
	if !reflect.DeepEqual(signatures.Authorities[0], expectedCerts[0]) {
		t.Errorf("signatures.Authorities[0]:\n got: %v\n want: %v", signatures.Authorities[0], expectedCerts[0])
	}

	if len(signatures.VouchedSubsets) != 1 {
		t.Fatalf("Unexpected size of signatures.VouchedSubsets: %d", len(signatures.VouchedSubsets))
	}
	vh := signatures.VouchedSubsets[0]
	if vh.Authority != 0 {
		t.Errorf("Authority: got %d, want %d", vh.Authority, 0)
	}
	if !bytes.Equal(vh.Sig, expectedSig) {
		t.Errorf("Sig:\n got: %v\n want: %v", vh.Sig, expectedSig)
	}

	headerSha256, err := e.Response.HeaderSha256()
	if err != nil {
		t.Fatalf("HeaderSha256 failed: %v", err)
	}
	expectedSigned, err := (&SignedSubset{
		ValidityUrl: urlMustParse(validityURL),
		AuthSha256: expectedCerts[0].CertSha256(),
		Date: signatureDate,
		Expires: signatureDate.Add(signatureDuration),
		SubsetHashes: map[string]*ResponseHashes{
			e.Request.URL.String(): &ResponseHashes{
				VariantsValue: "",
				Hashes: []*ResourceIntegrity{
					&ResourceIntegrity{headerSha256, "digest/mi-sha256-03"},
				},
			},
		},
	}).Encode()

	if !bytes.Equal(vh.Signed, expectedSigned) {
		t.Errorf("Signed:\n got: %v\n want: %v", vh.Signed, expectedSigned)
	}
}
