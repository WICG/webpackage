package certurl_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/WICG/webpackage/go/internal/signingalgorithm"
	"github.com/WICG/webpackage/go/internal/testhelper"
	. "github.com/WICG/webpackage/go/signedexchange/certurl"
)

func createCertChain(t *testing.T) CertChain {
	in, err := ioutil.ReadFile("test-cert.pem")
	if err != nil {
		t.Fatalf("Cannot read test-cert.pem: %v", err)
	}
	certs, err := signingalgorithm.ParseCertificates(in)
	if err != nil {
		t.Fatalf("Cannot parse test-cert.pem: %v", err)
	}
	chain, err := NewCertChain(certs, []byte("OCSP"), []byte("SCT"))
	if err != nil {
		t.Fatalf("NewCertChain failed: %v", err)
	}
	return chain
}

func TestParsePEM(t *testing.T) {
	expected, err := ioutil.ReadFile("certchain-expected.cbor")
	if err != nil {
		t.Fatalf("Cannot read certchain-expected.cbor: %v", err)
	}

	certChain := createCertChain(t)

	buf := &bytes.Buffer{}
	if err := certChain.Write(buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), expected) {
		got, err := testhelper.CborBinaryToReadableString(buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		want, err := testhelper.CborBinaryToReadableString(expected)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("CertificateMessageFromPEM:\ngot: %q,\nwant: %q", got, want)
	}
}

func TestRoundtrip(t *testing.T) {
	original := createCertChain(t)
	buf := &bytes.Buffer{}
	if err := original.Write(buf); err != nil {
		t.Fatal(err)
	}

	parsed, err := ReadCertChain(buf)
	if err != nil {
		t.Fatal(err)
	}

	if len(original) != len(parsed) {
		t.Fatalf("Cert chain length differs: want %d, got %d", len(original), len(parsed))
	}
	for i := 0; i < len(original); i++ {
		want := original[i]
		got := parsed[i]
		if !bytes.Equal(want.Cert.Raw, got.Cert.Raw) {
			t.Errorf("Cert at position %d differs:\n want: %v\n got: %v", i, want.Cert.Raw, got.Cert.Raw)
		}
		if !bytes.Equal(want.OCSPResponse, got.OCSPResponse) {
			t.Errorf("OCSP at position %d differs:\n want: %v\n got: %v", i, want.OCSPResponse, got.OCSPResponse)
		}
		if !bytes.Equal(want.SCTList, got.SCTList) {
			t.Errorf("SCT at position %d differs:\n want: %v\n got: %v", i, want.SCTList, got.SCTList)
		}
	}
}
