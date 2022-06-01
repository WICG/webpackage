package main

import (
	"crypto"
	"crypto/ed25519"
	"errors"
)

func SignIntegrityBlock(privKey crypto.PrivateKey) error {
	if _, ok := privKey.(ed25519.PrivateKey); !ok {
		return errors.New("Private key is not Ed25519 type.")
	}

	// TODO(sonkkeli): Actual logic.

	return nil
}
