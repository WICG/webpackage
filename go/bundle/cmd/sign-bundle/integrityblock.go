package main

import (
	"crypto"
	"crypto/ed25519"
	"errors"

	"os"

	"github.com/WICG/webpackage/go/integrityblock"
)

func SignIntegrityBlock(privKey crypto.PrivateKey) error {
	ed25519privKey, ok := privKey.(ed25519.PrivateKey)
	if !ok {
		return errors.New("Private key is not Ed25519 type.")
	}
	ed25519publicKey := ed25519privKey.Public().(ed25519.PublicKey)

	bundleFile, err := os.Open(*flagInput)
	if err != nil {
		return err
	}
	defer bundleFile.Close()

	integrityBlock, offset, err := integrityblock.ObtainIntegrityBlock(bundleFile)
	if err != nil {
		return err
	}

	webBundleHash, err := integrityblock.ComputeWebBundleSha512(bundleFile, offset)
	if err != nil {
		return err
	}

	signatureAttributes := integrityblock.GetLastSignatureAttributes(integrityBlock)
	signatureAttributes[integrityblock.Ed25519publicKeyAttributeName] = []byte(ed25519publicKey)

	integrityBlockBytes, err := integrityBlock.CborBytes()
	if err != nil {
		return err
	}

	// TODO(sonkkeli): Check deterministicy of integrityBlockBytes.

	dataToBeSigned, err := integrityblock.GenerateDataToBeSigned(webBundleHash, integrityBlockBytes, signatureAttributes)
	if err != nil {
		return err
	}

	signature, err := integrityblock.ComputeEd25519Signature(ed25519privKey, dataToBeSigned)
	if err != nil {
		return err
	}

	// TODO(sonkkeli): Remove debug prints.
	integrityBlock.AddNewSignatureToIntegrityBlock(signatureAttributes, signature)

	// TODO(sonkkeli): Check deterministicy of integrityBlockBytes.

	// TODO(sonkkeli): Write to file new integrity block + web bundle bytes.

	return nil
}
