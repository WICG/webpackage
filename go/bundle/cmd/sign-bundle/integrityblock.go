package main

import (
	"crypto"
	"crypto/ed25519"
	"errors"
	"io"
	"os"

	"github.com/WICG/webpackage/go/integrityblock"
	"github.com/WICG/webpackage/go/internal/cbor"
)

func writeOutput(bundleFile io.ReadSeeker, integrityBlockBytes []byte, originalIntegrityBlockOffset int64, signedBundleFile *os.File) error {
	signedBundleFile.Write(integrityBlockBytes)

	// Move the file pointer to the start of the web bundle bytes.
	bundleFile.Seek(originalIntegrityBlockOffset, io.SeekStart)

	// io.Copy() will do chunked read/write under the hood
	_, err := io.Copy(signedBundleFile, bundleFile)
	if err != nil {
		return err
	}
	return nil
}

// SignWithIntegrityBlock creates a CBOR integrity block and prepends that to the web bundle containing
// a signature of the hash of the web bundle. Finally it writes the new signed web bundle into file.
// More details can be found in the [explainer](https://github.com/WICG/webpackage/blob/main/explainers/integrity-signature.md).
func SignWithIntegrityBlock(privKey crypto.PrivateKey) error {
	if *flagInput == *flagOutput {
		return errors.New("SignIntegrityBlock: Input and output file cannot be the same.")
	}

	ed25519privKey, ok := privKey.(ed25519.PrivateKey)
	if !ok {
		return errors.New("SignIntegrityBlock: Private key is not Ed25519 type.")
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

	// Ensure the CBOR on the integrity block follows the deterministic principles.
	err = cbor.Deterministic(integrityBlockBytes)
	if err != nil {
		return err
	}

	dataToBeSigned, err := integrityblock.GenerateDataToBeSigned(webBundleHash, integrityBlockBytes, signatureAttributes)
	if err != nil {
		return err
	}

	signature, err := integrityblock.ComputeEd25519Signature(ed25519privKey, dataToBeSigned)
	if err != nil {
		return err
	}

	integrityBlock.AddNewSignatureToIntegrityBlock(signatureAttributes, signature)

	// Update the integrity block bytes after editing the integrity block.
	integrityBlockBytes, err = integrityBlock.CborBytes()
	if err != nil {
		return err
	}

	err = cbor.Deterministic(integrityBlockBytes)
	if err != nil {
		return err
	}

	signedBundleFile, err := os.Create(*flagOutput)
	if err != nil {
		return err
	}
	defer signedBundleFile.Close()
	if err := writeOutput(bundleFile, integrityBlockBytes, offset, signedBundleFile); err != nil {
		return err
	}

	return nil
}
