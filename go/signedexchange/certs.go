package signedexchange

import (
	"crypto"
	"crypto/ecdsa"
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
			return nil, fmt.Errorf("signedexchange: found a block that contains %q.", block.Type)
		}
		if len(block.Headers) > 0 {
			return nil, fmt.Errorf("signedexchange: unexpected certificate headers: %v", block.Headers)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

func ParsePrivateKey(derKey []byte) (crypto.PrivateKey, error) {
	// Try each of 3 key formats and take the first one that successfully parses.
	if key, err := x509.ParsePKCS1PrivateKey(derKey); err == nil {
		return key, nil
	}
	if keyInterface, err := x509.ParsePKCS8PrivateKey(derKey); err == nil {
		switch typedKey := keyInterface.(type) {
		case *ecdsa.PrivateKey:
			return typedKey, nil
		default:
			return nil, fmt.Errorf("signedexchange: unknown private key type in PKCS#8: %T", typedKey)
		}
	}
	if key, err := x509.ParseECPrivateKey(derKey); err == nil {
		return key, nil
	}
	return nil, errors.New("signedexchange: couldn't parse private key.")
}
