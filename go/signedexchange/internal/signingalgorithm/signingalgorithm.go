package signingalgorithm

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/asn1"
	"fmt"
	"io"
	"math/big"
)

type SigningAlgorithm interface {
	Sign(m []byte) ([]byte, error)
}

type ecdsaSigningAlgorithm struct {
	privKey *ecdsa.PrivateKey
	hash    crypto.Hash
	rand    io.Reader
}

func (e *ecdsaSigningAlgorithm) Sign(m []byte) ([]byte, error) {
	type ecdsaSigValue struct {
		R, S *big.Int
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
