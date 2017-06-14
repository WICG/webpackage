package webpack

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
)

// Sign() signs message using a TLS1.3 algorithm chosen depending on pk:
//   RSA <= 3072 bits:    rsa_pss_sha256
//   RSA, 3073-7680 bits: rsa_pss_sha384
//   secp256r1:           ecdsa_secp256r1_sha256
//   secp384r1:           ecdsa_secp384r1_sha384
//   secp521r1:           ecdsa_secp521r1_sha512
func Sign(pk crypto.PrivateKey, message []byte) ([]byte, error) {
	signer, err := signerForPrivateKey(pk)
	if err != nil {
		return nil, err
	}
	return signer.sign(message)
}

// Verify() verifies that signature signs message using pk, using the algorithm
// identified by Sign(). Returns nil for when signature is valid.
func Verify(pk crypto.PublicKey, message []byte, signature []byte) error {
	verifier, err := verifierForPublicKey(pk)
	if err != nil {
		return err
	}
	return verifier.verify(message, signature)
}

type messageSigner interface {
	sign(message []byte) ([]byte, error)
}

func signerForPrivateKey(pk crypto.PrivateKey) (messageSigner, error) {
	switch pk := pk.(type) {
	case *rsa.PrivateKey:
		switch bits := pk.N.BitLen(); {
		case bits <= 3072:
			return rsaPSSSigner{pk, crypto.SHA256}, nil
		case bits <= 7680:
			return rsaPSSSigner{pk, crypto.SHA384}, nil
		default:
			return nil, fmt.Errorf("RSA key too big: %v bits", bits)
		}
	case *ecdsa.PrivateKey:
		switch name := pk.Curve.Params().Name; name {
		case "P-256":
			return ecdsaSigner{pk, crypto.SHA256}, nil
		case "P-384":
			return ecdsaSigner{pk, crypto.SHA384}, nil
		case "P-521":
			return ecdsaSigner{pk, crypto.SHA512}, nil
		default:
			return nil, fmt.Errorf("unknown ECDSA curve: %v", name)
		}
	}
	return nil, fmt.Errorf("unknown public key type: %T", pk)
}

type rsaPSSSigner struct {
	privKey *rsa.PrivateKey
	hash    crypto.Hash
}

func (s rsaPSSSigner) sign(message []byte) ([]byte, error) {
	hash := s.hash.New()
	hash.Write(message)
	return rsa.SignPSS(rand.Reader, s.privKey, s.hash, hash.Sum(nil), nil)
}

type ecdsaSigner struct {
	privKey *ecdsa.PrivateKey
	hash    crypto.Hash
}

// From RFC5480:
type ecdsaSigValue struct {
	R, S *big.Int
}

func (es ecdsaSigner) sign(message []byte) ([]byte, error) {
	hash := es.hash.New()
	hash.Write(message)
	r, s, err := ecdsa.Sign(rand.Reader, es.privKey, hash.Sum(nil))
	if err != nil {
		return nil, err
	}
	return asn1.Marshal(ecdsaSigValue{r, s})
}

type messageVerifier interface {
	verify(message []byte, signature []byte) error
}

func verifierForPublicKey(pk crypto.PublicKey) (messageVerifier, error) {
	switch pk := pk.(type) {
	case *rsa.PublicKey:
		switch bits := pk.N.BitLen(); {
		case bits <= 3072:
			return rsaPSSVerifier{pk, crypto.SHA256}, nil
		case bits <= 7680:
			return rsaPSSVerifier{pk, crypto.SHA384}, nil
		default:
			return nil, fmt.Errorf("RSA key too big: %v bits", bits)
		}
	case *ecdsa.PublicKey:
		switch name := pk.Curve.Params().Name; name {
		case "P-256":
			return ecdsaVerifier{pk, crypto.SHA256}, nil
		case "P-384":
			return ecdsaVerifier{pk, crypto.SHA384}, nil
		case "P-521":
			return ecdsaVerifier{pk, crypto.SHA512}, nil
		default:
			return nil, fmt.Errorf("unknown ECDSA curve: %v", name)
		}
	}
	return nil, fmt.Errorf("unknown public key type: %T", pk)
}

type rsaPSSVerifier struct {
	pubKey *rsa.PublicKey
	hash   crypto.Hash
}

func (v rsaPSSVerifier) verify(message []byte, sig []byte) error {
	hash := v.hash.New()
	hash.Write(message)
	return rsa.VerifyPSS(v.pubKey, v.hash, hash.Sum(nil), sig, nil)
}

type ecdsaVerifier struct {
	pubKey *ecdsa.PublicKey
	hash   crypto.Hash
}

func (ev ecdsaVerifier) verify(message []byte, sig []byte) error {
	var parsedSig ecdsaSigValue
	rest, err := asn1.Unmarshal(sig, &parsedSig)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return fmt.Errorf("%d extra bytes in ECDSA signature", len(rest))
	}
	hash := ev.hash.New()
	hash.Write(message)
	if ecdsa.Verify(ev.pubKey, hash.Sum(nil), parsedSig.R, parsedSig.S) {
		return nil
	}
	return errors.New("signature verification failed")
}
