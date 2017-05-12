package webpack

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
)

// Parses a sequence of x509 certificates from filename.
func LoadCertificatesInto(filename string, certs *[]*x509.Certificate) error {
	pemData, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	for len(pemData) > 0 {
		var block *pem.Block
		block, pemData = pem.Decode(pemData)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			return fmt.Errorf("expected %q to contain CERTIFICATEs, but contains %q.", filename, block.Type)
		}
		if len(block.Headers) != 0 {
			return fmt.Errorf("unexpected certificate headers in %q: %v", filename, block.Headers)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return err
		}
		*certs = append(*certs, cert)
	}
	return nil
}

// Reads the first PEM block from filename, failing if it's not a PEM file.
func ReadPEMFile(filename string) (*pem.Block, error) {
	pemData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("No PEM data found in %q.", filename)
	}
	return block, nil
}

// Parses the private key for cert from derKey, failing if the public keys don't match.
func ParsePrivateKey(cert *x509.Certificate, derKey []byte) (key crypto.Signer, err error) {
	// Try each of 3 key formats and take the first one that successfully parses.
	if key, err = x509.ParsePKCS1PrivateKey(derKey); err == nil {
	} else if keyInterface, err := x509.ParsePKCS8PrivateKey(derKey); err == nil {
		switch typedKey := keyInterface.(type) {
		case *rsa.PrivateKey:
			key = typedKey
		case *ecdsa.PrivateKey:
			key = typedKey
		default:
			return nil, fmt.Errorf("Unknown private key type in PKCS#8: %T", typedKey)
		}
	} else if key, err = x509.ParseECPrivateKey(derKey); err == nil {
	} else {
		return nil, errors.New("Couldn't parse private key.")
	}

	if err := checkSamePublicKey(cert, key); err != nil {
		return nil, err
	}
	return key, nil
}

// Returns an error if cert and privKey don't have the same public key.
func checkSamePublicKey(cert *x509.Certificate, privKey crypto.Signer) error {
	if reflect.TypeOf(cert.PublicKey) != reflect.TypeOf(privKey.Public()) {
		return fmt.Errorf("Private key type %T doesn't match certificate key type %T.", privKey.Public(), cert.PublicKey)
	}
	switch certKey := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		privPubKey := privKey.Public().(*rsa.PublicKey)
		if certKey.N.Cmp(privPubKey.N) != 0 {
			return errors.New("Private key doesn't match certificate key.")
		}
		return nil
	case *ecdsa.PublicKey:
		privPubKey := privKey.Public().(*ecdsa.PublicKey)
		if certKey.X.Cmp(privPubKey.X) != 0 || certKey.Y.Cmp(privPubKey.Y) != 0 {
			return errors.New("Private key doesn't match certificate key.")
		}
		return nil
	default:
		return fmt.Errorf("Unexpected public key type, %T", cert.PublicKey)
	}
}
