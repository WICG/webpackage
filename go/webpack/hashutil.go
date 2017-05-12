package webpack

import (
	"crypto"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"fmt"
)

// Parses a CSP hash name into a crypto.Hash.
func parseHashName(name string) (crypto.Hash, error) {
	switch name {
	case "sha256":
		return crypto.SHA256, nil
	case "sha384":
		return crypto.SHA384, nil
	case "sha512":
		return crypto.SHA512, nil
	default:
		return 0, fmt.Errorf("Unknown hash name %q; expected a value from https://w3c.github.io/webappsec-csp/#grammardef-hash-algorithm.", name)
	}
}
