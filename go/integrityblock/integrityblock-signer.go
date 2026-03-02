package integrityblock

import (
	"crypto/ed25519"
	"errors"

	"github.com/WICG/webpackage/go/internal/cbor"
)

type IntegrityBlockSigner struct {
	SigningStrategy ISigningStrategy
	WebBundleHash   []byte
	IntegrityBlock  *IntegrityBlock
}

// VerifyEd25519Signature verifies that the given signature can be verified with the given public key and matches the data signed.
func VerifyEd25519Signature(publicKey ed25519.PublicKey, signature, dataToBeSigned []byte) (bool, error) {
	signatureOk := ed25519.Verify(publicKey, dataToBeSigned, signature)
	if !signatureOk {
		return signatureOk, errors.New("integrityblock: Signature verification failed.")
	}
	return signatureOk, nil
}

// SignAndAddNewSignature contains the main logic for generating the new signature and
// prepending the integrity block's signature stack with a new integrity signature object.
func (ibs *IntegrityBlockSigner) SignAndAddNewSignature(ed25519publicKey ed25519.PublicKey, signatureAttributes SignatureAttributesMap) error {
	integrityBlockBytes, err := ibs.IntegrityBlock.CborBytes()
	if err != nil {
		return err
	}

	// Ensure the CBOR on the integrity block follows the deterministic principles.
	err = cbor.Deterministic(integrityBlockBytes)
	if err != nil {
		return err
	}

	dataToBeSigned, err := GenerateDataToBeSigned(ibs.WebBundleHash, integrityBlockBytes, signatureAttributes)
	if err != nil {
		return err
	}

	signature, err := ibs.SigningStrategy.Sign(dataToBeSigned)
	if err != nil {
		return err
	}

	// Verification is done after signing to ensure that the signing was successful and that the obtained public key
	// is not corrupted and corresponds to the private key used for signing.
	VerifyEd25519Signature(ed25519publicKey, signature, dataToBeSigned)

	ibs.IntegrityBlock.addNewSignatureToIntegrityBlock(signatureAttributes, signature)
	return nil
}
