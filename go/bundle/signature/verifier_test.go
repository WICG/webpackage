package signature_test

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/WICG/webpackage/go/bundle"
	. "github.com/WICG/webpackage/go/bundle/signature"
)

func createTestSignedBundle(t *testing.T) *bundle.Bundle {
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

	return &bundle.Bundle{
		Version:    signer.Version,
		Exchanges:  []*bundle.Exchange{e},
		Signatures: signatures,
	}
}

func TestVerification(t *testing.T) {
	b := createTestSignedBundle(t)

	verifier, err := NewVerifier(b.Signatures, signatureDate, b.Version)
	if err != nil {
		t.Fatalf("NewVerifier failed: %v", err)
	}
	result, err := verifier.VerifyExchange(b.Exchanges[0])
	if err != nil {
		t.Fatalf("VerifyExchange failed: %v", err)
	}
	if result.Authority != b.Signatures.Authorities[0] {
		t.Fatalf("VerifyExchange: unexpected result.Authority %v", result.Authority)
	}
	if !bytes.Equal(result.VerifiedPayload, []byte("hello, world!")) {
		t.Fatalf("VerifyExchange: unexpected result.VerifiedPayload %v", result.VerifiedPayload)
	}
}

func TestSignatureNotYetValid(t *testing.T) {
	b := createTestSignedBundle(t)

	if _, err := NewVerifier(b.Signatures, signatureDate.Add(-1*time.Second), b.Version); err == nil {
		t.Error("NewVerifier should fail")
	}
}

func TestSignatureExpired(t *testing.T) {
	b := createTestSignedBundle(t)

	if _, err := NewVerifier(b.Signatures, signatureDate.Add(signatureDuration+time.Second), b.Version); err == nil {
		t.Error("NewVerifier should fail")
	}
}

func TestSignatureVerificationFailure(t *testing.T) {
	b := createTestSignedBundle(t)

	// Mutate the signature.
	b.Signatures.VouchedSubsets[0].Sig[3] ^= 1

	if _, err := NewVerifier(b.Signatures, signatureDate, b.Version); err == nil {
		t.Error("NewVerifier should fail")
	}
}

func TestExchangeNotCoveredBySignature(t *testing.T) {
	b := createTestSignedBundle(t)

	// This URL is not covered by the signature.
	b.Exchanges[0].Request.URL = urlMustParse("https://example.org/unsigned.html")

	verifier, err := NewVerifier(b.Signatures, signatureDate, b.Version)
	if err != nil {
		t.Fatalf("NewVerifier failed: %v", err)
	}
	result, err := verifier.VerifyExchange(b.Exchanges[0])
	if err != nil {
		t.Errorf("VerifyExchange failed: %v", err)
	}
	if result != nil {
		t.Errorf("VerifyExchange unexpectedly returned a result: %v", result)
	}
}

func TestResponseHeaderVerificationFailure(t *testing.T) {
	b := createTestSignedBundle(t)

	b.Exchanges[0].Response.Status = 201

	verifier, err := NewVerifier(b.Signatures, signatureDate, b.Version)
	if err != nil {
		t.Fatalf("NewVerifier failed: %v", err)
	}
	if _, err := verifier.VerifyExchange(b.Exchanges[0]); err == nil {
		t.Error("VerifyExchange should fail")
	}
}

func TestResponsePayloadVerificationFailure(t *testing.T) {
	b := createTestSignedBundle(t)

	// Mutate the last byte of the response body.
	b.Exchanges[0].Response.Body[len(b.Exchanges[0].Response.Body)-1] ^= 1

	verifier, err := NewVerifier(b.Signatures, signatureDate, b.Version)
	if err != nil {
		t.Fatalf("NewVerifier failed: %v", err)
	}
	if _, err := verifier.VerifyExchange(b.Exchanges[0]); err == nil {
		t.Error("VerifyExchange should fail")
	}
}
