package webbundleid

import (
	"crypto/ed25519"
	"encoding/base32"
	"strings"
)

var webBundleIdSuffix = []byte{0x00, 0x01, 0x02}

// GetWebBundleId returns a base32-encoded (without padding) ed25519 public key
// combined with a 3-byte long suffix and transformed to lowercase. More information:
// https://github.com/WICG/isolated-web-apps/blob/main/Scheme.md#signed-web-bundle-ids
func GetWebBundleId(ed25519publicKey ed25519.PublicKey) string {
	keyWithSuffix := append([]byte(ed25519publicKey), webBundleIdSuffix...)

	// StdEncoding is the standard base32 encoding, as defined in RFC 4648.
	return strings.ToLower(base32.StdEncoding.EncodeToString(keyWithSuffix))
}
