package integrityblock

import (
	"crypto/ed25519"
)

// ParsedEd25519KeySigningStrategy implementing `ISigningStrategy` is the simplest way to
// sign a web bundle just by passing a parsed private key.
type ParsedEd25519KeySigningStrategy struct {
	ed25519privKey ed25519.PrivateKey
}

func NewParsedEd25519KeySigningStrategy(ed25519privKey ed25519.PrivateKey) *ParsedEd25519KeySigningStrategy {
	return &ParsedEd25519KeySigningStrategy{
		ed25519privKey: ed25519privKey,
	}
}

func (bss ParsedEd25519KeySigningStrategy) Sign(data []byte) ([]byte, error) {
	return ed25519.Sign(bss.ed25519privKey, data), nil
}

func (bss ParsedEd25519KeySigningStrategy) GetPublicKey() (ed25519.PublicKey, error) {
	return bss.ed25519privKey.Public().(ed25519.PublicKey), nil
}
