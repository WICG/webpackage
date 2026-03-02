package webbundleid

import (
	"crypto/ed25519"
	"testing"

	"github.com/WICG/webpackage/go/internal/signingalgorithm"
)

func TestGetWebBundleId(t *testing.T) {
	privateKeyString := "-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEIB8nP5PpWU7HiILHSfh5PYzb5GAcIfHZ+bw6tcd/LZXh\n-----END PRIVATE KEY-----"
	privateKey, err := signingalgorithm.ParsePrivateKey([]byte(privateKeyString))
	if err != nil {
		t.Errorf("integrityblock: Failed to parse the test private key. err: %v", err)
	}

	got := GetWebBundleId(privateKey.(ed25519.PrivateKey).Public().(ed25519.PublicKey))
	want := "4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic"

	if got != want {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, want)
	}
}
