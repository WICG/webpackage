package webpack

import (
	"crypto"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"fmt"
	"hash"
	"io"
	"sort"

	"github.com/WICG/webpackage/go/webpack/cbor"
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

// Returns the CSP name of a given hash.
func HashName(h crypto.Hash) string {
	switch h {
	case crypto.SHA256:
		return "sha256"
	case crypto.SHA384:
		return "sha384"
	case crypto.SHA512:
		return "sha512"
	default:
		panic(fmt.Sprintf("Unknown Hash function: %v", h))
	}
}

// Sorts hashes by their names in the order needed for Canonical CBOR map keys
// (https://tools.ietf.org/html/rfc7049#section-3.9).
func SortHashByCBOR(hs []crypto.Hash) {
	sort.Slice(hs, func(i, j int) bool {
		return cbor.CanonicalLessStrings(HashName(hs[i]), HashName(hs[j]))
	})
}

// MultiHasher returns a Writer that hashes the bytes written to it using
// several hash types. It also returns a map that can be used to get the
// resulting hash values.
func MultiHasher(types []crypto.Hash) (io.Writer, map[crypto.Hash]hash.Hash) {
	hashers := make(map[crypto.Hash]hash.Hash, len(types))
	hasherArray := make([]io.Writer, len(types))
	for i, hashType := range types {
		hasher := hashType.New()
		hashers[hashType] = hasher
		hasherArray[i] = hasher
	}
	return io.MultiWriter(hasherArray...), hashers
}
