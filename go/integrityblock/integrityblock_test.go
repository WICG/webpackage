package integrityblock

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/WICG/webpackage/go/internal/cbor"
	"github.com/WICG/webpackage/go/internal/signingalgorithm"
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
	attributes := SignatureAttributesMap{Ed25519publicKeyAttributeName: []byte("publickey")}
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
	attributes := SignatureAttributesMap{Ed25519publicKeyAttributeName: []byte("publickey")}

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
	signatureAttributes := SignatureAttributesMap{"key": []byte("value")}

	var attributesBytesBuf bytes.Buffer
	enc := cbor.NewEncoder(&attributesBytesBuf)
	if err := cborEncodeSignatureAttributesMap(signatureAttributes, enc); err != nil {
		t.Fatal(err)
	}

	hashBytes := []byte("hash")
	integrityBlockBytes := []byte("integrityblock")

	// The numbers to display as big endian numbers
	hashLen := []byte{0, 0, 0, 0, 0, 0, 0, 0x04}           // 4
	integrityBlockLen := []byte{0, 0, 0, 0, 0, 0, 0, 0x0e} // 14
	attributesLen := []byte{0, 0, 0, 0, 0, 0, 0, 0x0b}     // 11

	got, err := generateDataToBeSigned(hashBytes, integrityBlockBytes, signatureAttributes)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.Write(hashLen)
	buf.Write(hashBytes)
	buf.Write(integrityBlockLen)
	buf.Write(integrityBlockBytes)
	buf.Write(attributesLen)
	buf.Write(attributesBytesBuf.Bytes())
	want := buf.Bytes()

	if !bytes.Equal(got, want) {
		t.Errorf("integrityblock: got: %s\nwant: %s", hex.EncodeToString(got), hex.EncodeToString(want))
	}
}

func TestCborBytesForSignatureAttributesMap(t *testing.T) {
	signatureAttributes := SignatureAttributesMap{"key": []byte("value")}

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

	attributes := SignatureAttributesMap{Ed25519publicKeyAttributeName: []byte("publickey")}
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
	err = integrityBlock.SignAndAddNewSignature(priv, webBundleHash, GenerateSignatureAttributesWithPublicKey(pub))

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
	signatureAttributesWithAdditionalAttribute := GenerateSignatureAttributesWithPublicKey(pub)
	signatureAttributesWithAdditionalAttribute["hello"] = []byte("world")

	err = integrityBlock.SignAndAddNewSignature(priv, webBundleHash, signatureAttributesWithAdditionalAttribute)
	err = integrityBlock.SignAndAddNewSignature(priv, webBundleHash, GenerateSignatureAttributesWithPublicKey(pub))

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

func TestGetWebBundleId(t *testing.T) {
	privateKeyString := "-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEIB8nP5PpWU7HiILHSfh5PYzb5GAcIfHZ+bw6tcd/LZXh\n-----END PRIVATE KEY-----"
	privateKey, err := signingalgorithm.ParsePrivateKey([]byte(privateKeyString))
	if err != nil {
		t.Errorf("integrityblock: Failed to parse the test private key. err: %v", err)
	}

	got := GetWebBundleId(privateKey.(ed25519.PrivateKey).Public().(ed25519.PublicKey))
	want := "4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic"

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
