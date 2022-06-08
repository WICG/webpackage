package integrityblock

import (
	"testing"

	"github.com/WICG/webpackage/go/internal/testhelper"
)

func TestEmptyIntegrityBlock(t *testing.T) {
	integrityBlock := GenerateEmptyIntegrityBlock()

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
		t.Errorf("got: %s\nwant: %s", got, want)
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
		t.Errorf("got: %s\nwant: %s", got, want)
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
		t.Errorf("got: %s\nwant: %s", got, want)
	}
}
