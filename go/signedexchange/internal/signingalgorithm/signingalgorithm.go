package signingalgorithm

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"encoding/asn1"
	"errors"
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

// Ecdsa-Sig-Value structure in Section 2.2.3 of RFC 3279.
type ecdsaSigValue struct {
	R, S *big.Int
}

func (e *ecdsaSigningAlgorithm) Sign(m []byte) ([]byte, error) {
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
			return nil, fmt.Errorf("signingalgorithm: unknown ECDSA curve: %s", name)
		}
	}
	return nil, fmt.Errorf("signingalgorithm: unknown private key type: %T", pk)
}

type Verifier interface {
	Verify(msg, sig []byte) (bool, error)
}

type ecdsaVerifier struct {
	pubKey *ecdsa.PublicKey
	hash   crypto.Hash
}

func (e *ecdsaVerifier) Verify(msg, sig []byte) (bool, error) {
	var v ecdsaSigValue
	rest, err := asn1.Unmarshal(sig, &v)
	if err != nil {
		return false, fmt.Errorf("signingalgorithm: failed to ASN.1 decode the signature: %v", err)
	}
	if len(rest) > 0 {
		return false, errors.New("signingalgorithm: extra data at the signature end")
	}

	hash := e.hash.New()
	hash.Write(msg)
	return ecdsa.Verify(e.pubKey, hash.Sum(nil), v.R, v.S), nil
}

func VerifierForPublicKey(k crypto.PublicKey) (Verifier, error) {
	switch k := k.(type) {
	case *ecdsa.PublicKey:
		switch name := k.Params().Name; name {
		case elliptic.P256().Params().Name:
			return &ecdsaVerifier{k, crypto.SHA256}, nil
		case elliptic.P384().Params().Name:
			return &ecdsaVerifier{k, crypto.SHA384}, nil
		default:
			return nil, fmt.Errorf("signingalgorithm: unknown ECDSA curve: %s", name)
		}
	}
	return nil, fmt.Errorf("signingalgorithm: unknown public key type: %T", k)
}
