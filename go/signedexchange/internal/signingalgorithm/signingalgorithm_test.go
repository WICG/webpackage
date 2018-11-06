package signingalgorithm_test

import (
	"crypto/rand"
	"crypto/ecdsa"
	"crypto/elliptic"
	"testing"

	. "github.com/WICG/webpackage/go/signedexchange/internal/signingalgorithm"
)

func TestSignVerify_ECDSA_P256_SHA256(t *testing.T) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ecdsa private key: %v", err)
		return
	}

	alg, err := SigningAlgorithmForPrivateKey(pk, rand.Reader)
	if err != nil {
		t.Fatalf("Failed to pick signing algorithm for ecdsa private key: %v", err)
		return
	}

	msg := []byte("foobar")
	sig, err := alg.Sign(msg)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
		return
	}

	verifier, err := VerifierForPublicKey(pk.Public())
	if err != nil {
		t.Fatalf("Failed to pick verifier for ecdsa public key: %v", err)
		return
	}

	ok, err := verifier.Verify(msg, sig)
	if err != nil {
		t.Errorf("Verification failed: %v", err)
	}
	if !ok {
		t.Error("Unexpected verification failure")
	}

	msg[0] = 'q'
	ok, err = verifier.Verify(msg, sig)
	if err != nil {
		t.Errorf("Verification failed: %v", err)
	}
	if ok {
		t.Error("Unexpected verification success")
	}
}
