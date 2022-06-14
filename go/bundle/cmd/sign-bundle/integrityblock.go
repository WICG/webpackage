package main

import (
	"crypto"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"os"

	"github.com/WICG/webpackage/go/integrityblock"
)

// readWebBundlePayloadLength returns the length of the web bundle parsed from the last 8 bytes of the web bundle file.
// [Web Bundle's Trailing Length]: https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#name-trailing-length
func readWebBundlePayloadLength(bundleFile *os.File) (int64, error) {
	// Finds the offset, from which the 8 bytes containing the web bundle length start.
	_, err := bundleFile.Seek(-8, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	// Reads from the offset to the end of the file (those 8 bytes).
	webBundleLengthBytes, err := ioutil.ReadAll(bundleFile)
	if err != nil {
		return 0, err
	}

	return int64(binary.BigEndian.Uint64(webBundleLengthBytes)), nil
}

// obtainIntegrityBlock returns either the existing integrity block parsed (not supported in v1) or a newly
// created empty integrity block. Integrity block preceeds the actual web bundle bytes. The second return
// value marks the offset from which point onwards we need to copy the web bundle bytes from. It will be
// needed later in the signing process (TODO) because we cannot rely on the integrity block length, because
// we don't know if the integrity block already existed or not.
func obtainIntegrityBlock(bundleFile *os.File) (*integrityblock.IntegrityBlock, int, error) {
	webBundleLen, err := readWebBundlePayloadLength(bundleFile)
	if err != nil {
		return nil, 0, err
	}
	fileStats, err := bundleFile.Stat()
	if err != nil {
		return nil, 0, err
	}

	// Unlike web bundle length, integrity block length cannot be more than max int.
	integrityBlockLen := int(fileStats.Size() - webBundleLen)

	if integrityBlockLen != 0 {
		// Read existing integrity block. Not supported in v1.
		return nil, integrityBlockLen, errors.New("Web bundle already contains an integrity block. Please provide an unsigned web bundle.")
	}

	integrityBlock := integrityblock.GenerateEmptyIntegrityBlock()
	return integrityBlock, integrityBlockLen, nil
}

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

	integrityBlock, _, err := obtainIntegrityBlock(bundleFile)
	if err != nil {
		return err
	}

	signatureAttributes := integrityblock.GetLastSignatureAttributes(integrityBlock)
	signatureAttributes[integrityblock.Ed25519publicKeyAttributeName] = []byte(ed25519publicKey)

	// TODO(sonkkeli): Remove debug prints.
	integrityBlockBytes, err := integrityBlock.CborBytes()
	fmt.Println(hex.EncodeToString(integrityBlockBytes))

	// TODO(sonkkeli): Rest of the signing process.

	return nil
}
