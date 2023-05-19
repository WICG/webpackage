package main

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/WICG/webpackage/go/integrityblock"
	"github.com/WICG/webpackage/go/integrityblock/webbundleid"
	"github.com/WICG/webpackage/go/internal/cbor"
	"github.com/WICG/webpackage/go/internal/signingalgorithm"
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

func readPublicEd25519KeyFromFile(path string) (ed25519.PublicKey, error) {
	pubkeytext, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.New("SignIntegrityBlock: Unable to read the public key.")
	}
	pubKey, err := signingalgorithm.ParsePublicKey(pubkeytext)

	ed25519pubKey, ok := pubKey.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("SignIntegrityBlock: Public key is not Ed25519 type.")
	}
	return ed25519pubKey, nil
}

func readAndParseEd25519PrivateKey(path string) (ed25519.PrivateKey, error) {
	privKey, err := readPrivateKeyFromFile(path)
	if err != nil {
		return nil, errors.New("SignIntegrityBlock: Unable to read the private key.")
	}

	ed25519privKey, ok := privKey.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("SignIntegrityBlock: Private key is not Ed25519 type.")
	}
	return ed25519privKey, nil
}

func DumpWebBundleIdFromPrivateKey() error {
	ed25519privKey, err := readAndParseEd25519PrivateKey(*dumpIdFlagPrivateKey)
	if err != nil {
		return err
	}

	webBundleId := webbundleid.GetWebBundleId(ed25519privKey.Public().(ed25519.PublicKey))
	fmt.Printf("Web Bundle ID: %s\n", webBundleId)
	return nil
}

func DumpWebBundleIdFromPublicKey() error {
	ed25519pubKey, err := readPublicEd25519KeyFromFile(*dumpIdFlagPublicKey)
	if err != nil {
		return err
	}

	webBundleId := webbundleid.GetWebBundleId(ed25519pubKey)
	fmt.Printf("Web Bundle ID: %s\n", webBundleId)
	return nil
}

func DumpWebBundleId() error {
	if isFlagPassed(dumpWebBundleIdCmd, flagNamePublicKey) {
		return DumpWebBundleIdFromPublicKey()
	} else {
		return DumpWebBundleIdFromPrivateKey()
	}
}

// SignWithIntegrityBlockWithCmdFlags is just a wrapper class for `SignWithIntegrityBlock`
// function containing the actual logic so that it can be easily exported without having
// to rely on reading and writing to files specified to be read from the CMD tool flags.
func SignWithIntegrityBlockWithCmdFlags(signingStrategy integrityblock.ISigningStrategy) error {
	if *ibFlagInput == *ibFlagOutput {
		return errors.New("SignIntegrityBlock: Input and output file cannot be the same.")
	}

	bundleFile, err := os.Open(*ibFlagInput)
	if err != nil {
		return err
	}
	defer bundleFile.Close()

	signedBundleFile, err := os.Create(*ibFlagOutput)
	if err != nil {
		return err
	}
	defer signedBundleFile.Close()

	err = SignWithIntegrityBlock(bundleFile, signedBundleFile, signingStrategy)
	if err != nil {
		return err
	}

	return nil
}

// SignWithIntegrityBlock creates a CBOR integrity block containing a signature
// matching the hash of the web bundle read from `bundleFileIn`. Finally it
// writes the new signed web bundle into `bundleFileOut`. More details can be
// found in [Integrity Block Explainer](https://github.com/WICG/webpackage/blob/main/explainers/integrity-signature.md).
func SignWithIntegrityBlock(bundleFileIn, bundleFileOut *os.File, signingStrategy integrityblock.ISigningStrategy) error {
	integrityBlock, offset, err := integrityblock.ObtainIntegrityBlock(bundleFileIn)
	if err != nil {
		return err
	}

	webBundleHash, err := integrityblock.ComputeWebBundleSha512(bundleFileIn, offset)
	if err != nil {
		return err
	}

	ibs := integrityblock.IntegrityBlockSigner{
		SigningStrategy: signingStrategy,
		WebBundleHash:   webBundleHash,
		IntegrityBlock:  integrityBlock,
	}

	ed25519publicKey, err := ibs.SigningStrategy.GetPublicKey()
	if err != nil {
		return err
	}

	signatureAttributes := integrityblock.GenerateSignatureAttributesWithPublicKey(ed25519publicKey)

	err = ibs.SignAndAddNewSignature(ed25519publicKey, signatureAttributes)
	if err != nil {
		return err
	}

	// Update the integrity block bytes with the new integrity block.
	integrityBlockBytes, err := integrityBlock.CborBytes()
	if err != nil {
		return err
	}

	err = cbor.Deterministic(integrityBlockBytes)
	if err != nil {
		return err
	}

	webBundleId := webbundleid.GetWebBundleId(ed25519publicKey)
	fmt.Println("Web Bundle ID: " + webBundleId)

	if err := writeOutput(bundleFileIn, integrityBlockBytes, offset, bundleFileOut); err != nil {
		return err
	}

	return nil
}
