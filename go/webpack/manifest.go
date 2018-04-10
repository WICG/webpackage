package webpack

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

type Manifest struct {
	Metadata     Metadata
	Signatures   []SignWith
	Certificates []*x509.Certificate
	HashTypes    []crypto.Hash
	Subpackages  []string
}

type SignWith struct {
	CertFilename string
	Certificate  *x509.Certificate
	KeyFilename  string
	pemKey       *pem.Block
	Key          crypto.Signer
}

// Loads a certificate and its key from two PEM files. The key can be PKCS#1,
// unencrypted PKCS#8, or openssl's EC format, and its public key must match the
// certificate.
//
// This operation is similar to tls.LoadX509KeyPair() except that if the key is
// encrypted, it can be decrypted with result.GivePassword().
func LoadSignWith(certFilename, keyFilename string) (result SignWith, err error) {
	result.CertFilename = certFilename
	var certs []*x509.Certificate
	if err = LoadCertificatesFromFile(certFilename, &certs); err != nil {
		return result, err
	}
	if len(certs) == 0 {
		return result, fmt.Errorf("no certificates found in %q.", certFilename)
	}
	result.Certificate = certs[0]
	if keyFilename != "" {
		if err := result.LoadKey(keyFilename); err != nil {
			return result, err
		}
	}
	return result, nil
}

// Loads the key for s's certificate from keyFilename.
func (s *SignWith) LoadKey(keyFilename string) error {
	s.KeyFilename = keyFilename
	var err error
	if s.pemKey, err = ReadPEMFile(keyFilename); err != nil {
		return err
	}
	if !x509.IsEncryptedPEMBlock(s.pemKey) {
		if s.pemKey.Type == "ENCRYPTED PRIVATE KEY" {
			return errors.New("Go cannot decrypt PKCS#8-format encrypted keys.")
		}
		if s.Key, err = ParsePrivateKey(s.Certificate, s.pemKey.Bytes); err != nil {
			return err
		}
	}
	return nil
}

// If the key loaded into s was encrypted, the caller needs to supply a password
// for it.
func (s *SignWith) GivePassword(password []byte) error {
	if s.Key != nil {
		return errors.New("Key isn't encrypted.")
	}
	if !x509.IsEncryptedPEMBlock(s.pemKey) {
		panic(fmt.Sprintf("%v holds an unencrypted PEM key, but somehow it wasn't decoded into s.key.", *s))
	}
	keyDer, err := x509.DecryptPEMBlock(s.pemKey, password)
	if err != nil {
		return err
	}
	key, err := ParsePrivateKey(s.Certificate, keyDer)
	if err != nil {
		return err
	}
	s.Key = key
	return nil
}
