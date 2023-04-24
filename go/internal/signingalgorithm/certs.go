package signingalgorithm

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

func ParseCertificates(text []byte) ([]*x509.Certificate, error) {
	certs := []*x509.Certificate{}
	for len(text) > 0 {
		var block *pem.Block
		block, text = pem.Decode(text)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("signingalgorithm: found a block that contains %q.", block.Type)
		}
		if len(block.Headers) > 0 {
			return nil, fmt.Errorf("signingalgorithm: unexpected certificate headers: %v", block.Headers)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

func ParsePrivateKey(text []byte) (crypto.PrivateKey, error) {
	for len(text) > 0 {
		var block *pem.Block
		block, text = pem.Decode(text)
		if block == nil {
			return nil, errors.New("signingalgorithm: invalid PEM block in private key.")
		}

		privkey, err := parsePrivateKeyBlock(block.Bytes)
		if err == nil {
			return privkey, nil
		}
	}
	return nil, errors.New("signingalgorithm: could not find private key.")
}

func parsePrivateKeyBlock(derKey []byte) (crypto.PrivateKey, error) {
	// Try each of 2 key formats and take the first one that successfully parses.
	if keyInterface, err := x509.ParsePKCS8PrivateKey(derKey); err == nil {
		switch typedKey := keyInterface.(type) {
		case *ecdsa.PrivateKey:
			return typedKey, nil
		case ed25519.PrivateKey:
			return typedKey, nil
		default:
			return nil, fmt.Errorf("signingalgorithm: unknown private key type in PKCS#8: %T", typedKey)
		}
	}
	if key, err := x509.ParseECPrivateKey(derKey); err == nil {
		return key, nil
	}
	return nil, errors.New("signingalgorithm: couldn't parse private key.")
}

func ParsePublicKey(text []byte) (crypto.PublicKey, error) {
	for len(text) > 0 {
		var block *pem.Block
		block, text = pem.Decode(text)
		if block == nil {
			return nil, errors.New("signingalgorithm: invalid PEM block in public key.")
		}

		pubkey, err := parsePublicKeyBlock(block.Bytes)
		if err == nil {
			return pubkey, nil
		}
	}
	return nil, errors.New("signingalgorithm: could not find public key.")
}

// parsePublicKeyBlock parses any allowed type public key. Currently only allows parsing ed25519 type public keys.
func parsePublicKeyBlock(derKey []byte) (crypto.PublicKey, error) {
	if keyInterface, err := x509.ParsePKIXPublicKey(derKey); err == nil {
		switch typedKey := keyInterface.(type) {
		case ed25519.PublicKey:
			return typedKey, nil
		default:
			return nil, fmt.Errorf("signingalgorithm: unknown public key type: %T", typedKey)
		}
	}
	return nil, errors.New("signingalgorithm: couldn't parse public key.")
}
