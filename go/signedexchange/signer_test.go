package signedexchange_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"testing"

	"github.com/WICG/webpackage/go/signedexchange"
)

func TestSignVerify_RSA_PSS_SHA256(t *testing.T) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Errorf("Failed to generate rsa private key: %v", err)
		return
	}

	alg, err := signedexchange.SigningAlgorithmForPrivateKey(pk, rand.Reader)
	if err != nil {
		t.Errorf("Failed to pick signing algorithm for rsa private key: %v", err)
		return
	}

	msg := []byte("foobar")
	sig, err := alg.Sign(msg)
	if err != nil {
		t.Errorf("Failed to sign: %v", err)
		return
	}

	hashed := sha256.Sum256(msg)
	if err := rsa.VerifyPSS(
		pk.Public().(*rsa.PublicKey), crypto.SHA256, hashed[:], sig,
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash},
	); err != nil {
		t.Errorf("Failed to verify: %v", err)
		return
	}
}
