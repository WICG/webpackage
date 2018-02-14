package signedexchange

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/asn1"
	"fmt"
	"io"
	"math/big"
)

type SigningAlgorithm interface {
	Sign(m []byte) ([]byte, error)
}

type rsaPSSSigningAlgorithm struct {
	privKey *rsa.PrivateKey
	hash    crypto.Hash
	rand    io.Reader
}

func (s *rsaPSSSigningAlgorithm) Sign(m []byte) ([]byte, error) {
	hash := s.hash.New()
	hash.Write(m)
	return rsa.SignPSS(
		s.rand, s.privKey, s.hash, hash.Sum(nil),
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
}

type ecdsaSigningAlgorithm struct {
	privKey *ecdsa.PrivateKey
	hash    crypto.Hash
	rand    io.Reader
}

func (e *ecdsaSigningAlgorithm) Sign(m []byte) ([]byte, error) {
	type ecdsaSigValue struct {
		r, s *big.Int
	}

	hash := e.hash.New()
	hash.Write(m)
	r, s, err := ecdsa.Sign(e.rand, e.privKey, hash.Sum(nil))
	if err != nil {
		return nil, err
	}
	return asn1.Marshal(ecdsaSigValue{r, s})
}

func SigningAlgorithmForPrivateKey(pk crypto.PrivateKey, rand io.Reader) (SigningAlgorithm, error) {
	switch pk := pk.(type) {
	case *rsa.PrivateKey:
		bits := pk.N.BitLen()
		if bits == 2048 {
			return &rsaPSSSigningAlgorithm{pk, crypto.SHA256, rand}, nil
		}
		return nil, fmt.Errorf("signedexchange: unsupported RSA key size: %d bits", bits)
	case *ecdsa.PrivateKey:
		switch name := pk.Curve.Params().Name; name {
		case elliptic.P256().Params().Name:
			return &ecdsaSigningAlgorithm{pk, crypto.SHA256, rand}, nil
		case elliptic.P384().Params().Name:
			return &ecdsaSigningAlgorithm{pk, crypto.SHA384, rand}, nil
		default:
			return nil, fmt.Errorf("signedexchange: unknown ECDSA curve: %s", name)
		}
	}
	return nil, fmt.Errorf("signedexchange: unknown public key type: %T", pk)
}
