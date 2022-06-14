package integrityblock

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"github.com/WICG/webpackage/go/internal/testhelper"
)

func TestEmptyIntegrityBlock(t *testing.T) {
	integrityBlock := generateEmptyIntegrityBlock()

	integrityBlockBytes, err := integrityBlock.CborBytes()
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
	}

	want := `["🖋📦" "1b\x00\x00" []]`

	got, err := testhelper.CborBinaryToReadableString(integrityBlockBytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, want)
	}
}

func TestIntegrityBlockWithOneSignature(t *testing.T) {
	attributes := map[string][]byte{"ed25519PublicKey": []byte("publickey")}

	integritySignatures := []*IntegritySignature{{
		SignatureAttributes: attributes,
		Signature:           []byte("signature"),
	}}

	integrityBlock := &IntegrityBlock{
		Magic:          IntegrityBlockMagic,
		Version:        VersionB1,
		SignatureStack: integritySignatures,
	}

	integrityBlockBytes, err := integrityBlock.CborBytes()
	if err != nil {
		t.Errorf("integritySignature.CborBytes. err: %v", err)
	}

	want := `["🖋📦" "1b\x00\x00" [[map["ed25519PublicKey":"publickey"] "signature"]]]`

	got, err := testhelper.CborBinaryToReadableString(integrityBlockBytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, want)
	}
}

func TestIntegritySignature(t *testing.T) {
	var integritySignature *IntegritySignature
	attributes := map[string][]byte{"ed25519PublicKey": []byte("publickey")}

	integritySignature = &IntegritySignature{
		SignatureAttributes: attributes,
		Signature:           []byte("signature"),
	}

	integritySignatureBytes, err := integritySignature.CborBytes()
	if err != nil {
		t.Errorf("integritySignature.CborBytes. err: %v", err)
	}

	want := `[map["ed25519PublicKey":"publickey"] "signature"]`

	got, err := testhelper.CborBinaryToReadableString(integritySignatureBytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, want)
	}
}

func TestGetLastSignatureAttributesWithEmptySingatureStack(t *testing.T) {
	got := GetLastSignatureAttributes(generateEmptyIntegrityBlock())
	if len(got) != 0 {
		t.Error("integrityblock: GetLastSignatureAttributes is not empty.")
	}
}

func TestGetLastSignatureAttributesWithOneSingatureInTheStack(t *testing.T) {
	pubKey := []byte("publickey")
	attributes := map[string][]byte{Ed25519publicKeyAttributeName: pubKey}

	integritySignatures := []*IntegritySignature{{
		SignatureAttributes: attributes,
		Signature:           []byte("signature"),
	}}

	integrityBlock := &IntegrityBlock{
		Magic:          IntegrityBlockMagic,
		Version:        VersionB1,
		SignatureStack: integritySignatures,
	}

	got := GetLastSignatureAttributes(integrityBlock)
	if len(got) != 1 {
		t.Error("integrityblock: GetLastSignatureAttributes is either empty or contains other attributes.")
	}
	if !bytes.Equal(got[Ed25519publicKeyAttributeName], pubKey) {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, pubKey)
	}
}

func TestComputeWebBundleSha512(t *testing.T) {
	bundleFile, err := os.Open("./testfile.wbn")
	if err != nil {
		t.Error("Failed to open the test file")
	}
	defer bundleFile.Close()

	want, err := hex.DecodeString("95f8713d382ffefb8f1e4f464e39a2bf18280c8b26434d2fcfc08d7d710c8919ace5a652e25e66f9292cda424f20e4b53bf613bf9488140272f56a455393f7e6")
	if err != nil {
		t.Fatal(err)
	}

	got, err := ComputeWebBundleSha512(bundleFile, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, want)
	}
}
