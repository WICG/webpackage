package main

import (
	"crypto"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"

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

	// TODO(sonkkeli): Remove debug prints.
	integrityBlockBytes, err := integrityBlock.CborBytes()
	fmt.Println(hex.EncodeToString(integrityBlockBytes))
	fmt.Println(hex.EncodeToString(webBundleHash))

	// TODO(sonkkeli): Rest of the signing process.

	return nil
}
