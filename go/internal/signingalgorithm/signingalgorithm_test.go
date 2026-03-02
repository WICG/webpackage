package signingalgorithm_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"testing"

	. "github.com/WICG/webpackage/go/internal/signingalgorithm"
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

func TestSignVerify_ED25519(t *testing.T) {
	privateKeyString := "-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEIB8nP5PpWU7HiILHSfh5PYzb5GAcIfHZ+bw6tcd/LZXh\n-----END PRIVATE KEY-----"

	pk, err := ParsePrivateKey([]byte(privateKeyString))
	if err != nil {
		t.Errorf("integrityblock: Failed to parse the test private key. err: %v", err)
	}

	ed25519pri := pk.(ed25519.PrivateKey)
	ed25519pub := ed25519pri.Public().(ed25519.PublicKey)

	msg := []byte("foobar")
	signature := ed25519.Sign(ed25519pri, msg)

	if !ed25519.Verify(ed25519pub, msg, signature) {
		t.Error("Signature verification failed with unencrypted Ed25519 key")
	}
}

func TestSignVerify_ED25519_Encrypted(t *testing.T) {
	encryptedPrivateKeyString := "-----BEGIN ENCRYPTED PRIVATE KEY-----\nMIGbMFcGCSqGSIb3DQEFDTBKMCkGCSqGSIb3DQEFDDAcBAhOw2E7LxOkzQICCAAw\nDAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEEJZqH2axMFEvdFmJLZlnch4EQMfJ\nAa/4uAmWqu2N5aOn2yIz3Ri+vQ/rzBPrvIaoDxYUUxwujJFujSbr3lnagHlOPptU\n7XhjbPbeOqidLqyv5rA=\n-----END ENCRYPTED PRIVATE KEY-----"

	os.Setenv("WEB_BUNDLE_SIGNING_PASSPHRASE", "helloworld" /*=passphrase*/)

	pk, err := ParsePrivateKey([]byte(encryptedPrivateKeyString))
	if err != nil {
		t.Errorf("integrityblock: Failed to parse the test private key. err: %v", err)
	}

	ed25519pri := pk.(ed25519.PrivateKey)
	ed25519pub := ed25519pri.Public().(ed25519.PublicKey)

	msg := []byte("foobar")
	signature := ed25519.Sign(ed25519pri, msg)

	if !ed25519.Verify(ed25519pub, msg, signature) {
		t.Error("Signature verification failed with encrypted Ed25519 key")
	}
}
