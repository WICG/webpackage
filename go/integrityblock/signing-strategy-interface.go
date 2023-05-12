package integrityblock

import (
	"crypto/ed25519"
)

type ISigningStrategy interface {
	Sign(data []byte) ([]byte, error)
	GetPublicKey() (ed25519.PublicKey, error)

	// TODO(sonkkeli): Implement once we have security approval.
	// IsDevSigning() bool
}
