package signedexchange_test

import (
	"crypto/rand"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/asn1"
	"math/big"
	"testing"

	"github.com/WICG/webpackage/go/signedexchange/internal/signingalgorithm"
)

func TestSignVerify_ECDSA_P256_SHA256(t *testing.T) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Errorf("Failed to generate ecdsa private key: %v", err)
		return
	}

	alg, err := signingalgorithm.SigningAlgorithmForPrivateKey(pk, rand.Reader)
	if err != nil {
		t.Errorf("Failed to pick signing algorithm for ecdsa private key: %v", err)
		return
	}

	msg := []byte("foobar")
	sig, err := alg.Sign(msg)
	if err != nil {
		t.Errorf("Failed to sign: %v", err)
		return
	}

	var v struct {R, S *big.Int}
	if _, err := asn1.Unmarshal(sig, &v); err != nil {
		t.Errorf("asn1.Unmarshal failed: %v", err)
	}

	hashed := sha256.Sum256(msg)
	if !ecdsa.Verify(&pk.PublicKey, hashed[:], v.R, v.S) {
		t.Errorf("Failed to verify")
		return
	}
}
