package signingalgorithm

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/youmark/pkcs8"
	"golang.org/x/crypto/ssh/terminal"
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

		if block.Type == "ENCRYPTED PRIVATE KEY" {
			if privkey, err := parseEncryptedPrivateKeyBlock(block.Bytes); err == nil {
				return privkey, nil
			}
		} else {
			if privkey, err := parsePrivateKeyBlock(block.Bytes); err == nil {
				return privkey, nil
			}
		}
	}

	return nil, errors.New("signingalgorithm: could not find private key.")
}

func typeSupportedPKCS8key(keyInterface any) (crypto.PrivateKey, error) {
	switch typedKey := keyInterface.(type) {
	case *ecdsa.PrivateKey:
		return typedKey, nil
	case ed25519.PrivateKey:
		return typedKey, nil
	default:
		return nil, fmt.Errorf("signingalgorithm: unknown private key type in PKCS#8: %T", typedKey)
	}
}

func parsePrivateKeyBlock(derKey []byte) (crypto.PrivateKey, error) {
	// Try each of 2 key formats and take the first one that successfully parses.
	if key, err := x509.ParseECPrivateKey(derKey); err == nil {
		return key, nil
	}

	if keyInterface, err := x509.ParsePKCS8PrivateKey(derKey); err == nil {
		return typeSupportedPKCS8key(keyInterface)
	}

	return nil, errors.New("signingalgorithm: couldn't parse private key.")
}

// parseEncryptedPrivateKeyBlock reads the passphrase to decrypt an encrypted private key from either
// WEB_BUNDLE_SIGNING_PASSPHRASE environment variable or if not set, it prompts a passphrase from the user.
func parseEncryptedPrivateKeyBlock(derKey []byte) (crypto.PrivateKey, error) {
	passphrase := []byte(os.Getenv("WEB_BUNDLE_SIGNING_PASSPHRASE"))

	if len(passphrase) == 0 {
		fmt.Println("The key is passphrase-encrypted. Please provide the passphrase and then press ENTER. ")
		passphrase, _ = terminal.ReadPassword(0)
		if len(passphrase) == 0 {
			return nil, errors.New("signingalgorithm: invalid passphrase to decrypt the private key.")
		}
	} else {
		fmt.Println("The key is passphrase-encrypted. Passphrase was successfully read from WEB_BUNDLE_SIGNING_PASSPHRASE environment variable.")
	}

	if keyInterface, err := pkcs8.ParsePKCS8PrivateKey(derKey, passphrase); err == nil {
		return typeSupportedPKCS8key(keyInterface)
	}

	return nil, errors.New("signingalgorithm: couldn't parse encrypted private key.")
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
