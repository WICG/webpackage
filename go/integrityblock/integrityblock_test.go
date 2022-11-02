package integrityblock

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/WICG/webpackage/go/internal/cbor"
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

func TestAddNewSignatureToIntegrityBlock(t *testing.T) {
	integrityBlock := generateEmptyIntegrityBlock()
	attributes := map[string][]byte{"ed25519PublicKey": []byte("publickey")}
	signature := []byte("signature")

	integrityBlock.addNewSignatureToIntegrityBlock(attributes, signature)

	integrityBlockBytes, err := integrityBlock.CborBytes()
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
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

	var integritySignatureBuf bytes.Buffer
	enc := cbor.NewEncoder(&integritySignatureBuf)
	if err := integritySignature.cborBytes(enc); err != nil {
		t.Errorf("integritySignature.cborBytes. err: %v", err)
	}

	want := `[map["ed25519PublicKey":"publickey"] "signature"]`

	got, err := testhelper.CborBinaryToReadableString(integritySignatureBuf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, want)
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

func TestGenerateDataToBeSigned(t *testing.T) {
	signatureAttributes := make(map[string][]byte, 1)
	signatureAttributes["key"] = []byte("value")

	var attributesBytesBuf bytes.Buffer
	enc := cbor.NewEncoder(&attributesBytesBuf)
	if err := cborEncodeSignatureAttributesMap(signatureAttributes, enc); err != nil {
		t.Fatal(err)
	}

	h := []byte{0xf0, 0x9f, 0x96, 0x8b}
	ib := []byte{0xf0, 0x9f, 0x96, 0x8b, 0xf0, 0x9f, 0x93, 0xa6}

	got, err := generateDataToBeSigned(h, ib, signatureAttributes)
	if err != nil {
		t.Fatal(err)
	}

	want, _ := hex.DecodeString("0000000000000004" + hex.EncodeToString(h) + "0000000000000008" + hex.EncodeToString(ib) + "000000000000000b" + hex.EncodeToString(attributesBytesBuf.Bytes()))

	if !bytes.Equal(got, want) {
		t.Errorf("integrityblock: got: %s\nwant: %s", hex.EncodeToString(got), hex.EncodeToString(want))
	}
}

func TestCborBytesForSignatureAttributesMap(t *testing.T) {
	signatureAttributes := make(map[string][]byte, 1)
	signatureAttributes["key"] = []byte("value")

	var attributesBytesBuf bytes.Buffer
	enc := cbor.NewEncoder(&attributesBytesBuf)
	if err := cborEncodeSignatureAttributesMap(signatureAttributes, enc); err != nil {
		t.Fatal(err)
	}

	got, err := testhelper.CborBinaryToReadableString(attributesBytesBuf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	want := `map["key":"value"]`

	if got != want {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, want)
	}
}

func TestIntegrityBlockGeneratedWithTheToolIsDeterministic(t *testing.T) {
	integrityBlock := generateEmptyIntegrityBlock()
	integrityBlockBytes, err := integrityBlock.CborBytes()
	if err := cbor.Deterministic(integrityBlockBytes); err != nil {
		t.Error("Empty integrity block generated using our tool should be deterministic.")
	}

	attributes := map[string][]byte{"ed25519PublicKey": []byte("publickey")}
	signature := []byte("signature")

	integrityBlock.addNewSignatureToIntegrityBlock(attributes, signature)

	integrityBlockBytes, err = integrityBlock.CborBytes()
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
	}

	if err := cbor.Deterministic(integrityBlockBytes); err != nil {
		t.Error("Integrity block with one signature generated using our tool should be deterministic.")
	}
}

func TestUnsignedWebBundleDoesntHaveIntegrityBlock(t *testing.T) {
	bundleFile, err := os.Open("./testfile.wbn")
	if err != nil {
		t.Error("Error opening testfile.wbn")
	}
	defer bundleFile.Close()

	hasIntegrityBlock, err := WebBundleHasIntegrityBlock(bundleFile)
	if err != nil {
		t.Error(err)
	} else if hasIntegrityBlock {
		t.Error("Unsigned web bundle should not have an integrity block.")
	}
}

func TestSignedWebBundleHasIntegrityBlockestAnother(t *testing.T) {
	ib, err := generateEmptyIntegrityBlock().CborBytes()
	r := bytes.NewReader(ib)

	hasIntegrityBlock, err := WebBundleHasIntegrityBlock(r)
	if err != nil {
		t.Error(err)
	} else if !hasIntegrityBlock {
		t.Error("Signed web bundle should have an integrity block.")
	}
}

// Test SignAndAddNewSignature when there isn't any previous signatures in the signature stack.
func TestSignAndAddNewSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error("Failed to generate test keys")
	}

	bundleFile, err := os.Open("./testfile.wbn")
	if err != nil {
		t.Error("Failed to open the test file")
	}
	defer bundleFile.Close()

	webBundleHash, err := ComputeWebBundleSha512(bundleFile, 0)
	if err != nil {
		t.Fatal(err)
	}

	integrityBlock := generateEmptyIntegrityBlock()
	err = integrityBlock.SignAndAddNewSignature(priv, webBundleHash, map[string][]byte{})

	integrityBlockBytes, err := integrityBlock.CborBytes()
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
	}

	publicKeyAsReadableCborString, err := bytesToCborAndToReadableStringHelper(pub)
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
	}

	signatureAsReadableCborString, err := bytesToCborAndToReadableStringHelper(integrityBlock.SignatureStack[0].Signature)
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
	}

	want := `["🖋📦" "1b\x00\x00" [[map["ed25519PublicKey":` + publicKeyAsReadableCborString + `] ` + signatureAsReadableCborString + `]]]`

	got, err := testhelper.CborBinaryToReadableString(integrityBlockBytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, want)
	}
}

// Test SignAndAddNewSignature when there is an existing signature already in the stack.
// Signature attributes should always be "freshly" made.
func TestSignAndAddNewSignatureWithExistingSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error("Failed to generate test keys")
	}

	bundleFile, err := os.Open("./testfile.wbn")
	if err != nil {
		t.Error("Failed to open the test file")
	}
	defer bundleFile.Close()

	webBundleHash, err := ComputeWebBundleSha512(bundleFile, 0)
	if err != nil {
		t.Fatal(err)
	}

	integrityBlock := generateEmptyIntegrityBlock()
	err = integrityBlock.SignAndAddNewSignature(priv, webBundleHash, map[string][]byte{"hello": []byte("world")})
	err = integrityBlock.SignAndAddNewSignature(priv, webBundleHash, map[string][]byte{})

	integrityBlockBytes, err := integrityBlock.CborBytes()
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
	}

	publicKeyAsReadableCborString, err := bytesToCborAndToReadableStringHelper(pub)
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
	}

	signatureAsReadableCborString1, err := bytesToCborAndToReadableStringHelper(integrityBlock.SignatureStack[0].Signature)
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
	}
	signatureAsReadableCborString2, err := bytesToCborAndToReadableStringHelper(integrityBlock.SignatureStack[1].Signature)
	if err != nil {
		t.Errorf("integrityBlock.CborBytes. err: %v", err)
	}

	integritySignatureObject1 := `[map["ed25519PublicKey":` + publicKeyAsReadableCborString + `] ` + signatureAsReadableCborString1 + `]`
	integritySignatureObject2 := `[map["ed25519PublicKey":` + publicKeyAsReadableCborString + ` "hello":"world"] ` + signatureAsReadableCborString2 + `]`
	want := `["🖋📦" "1b\x00\x00" [` + integritySignatureObject1 + ` ` + integritySignatureObject2 + `]]`

	got, err := testhelper.CborBinaryToReadableString(integrityBlockBytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("integrityblock: got: %s\nwant: %s", got, want)
	}
}

func bytesToCborAndToReadableStringHelper(bts []byte) (string, error) {
	var buf bytes.Buffer
	enc := cbor.NewEncoder(&buf)

	err := enc.EncodeByteString(bts)
	if err != nil {
		return "", err
	}

	cborAsString, err := testhelper.CborBinaryToReadableString(buf.Bytes())
	if err != nil {
		return "", err
	}
	return cborAsString, nil
}
