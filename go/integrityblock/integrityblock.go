package integrityblock

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/WICG/webpackage/go/internal/cbor"
)

type SignatureAttributesMap map[string][]byte

type IntegritySignature struct {
	SignatureAttributes SignatureAttributesMap
	Signature           []byte
}

type IntegrityBlock struct {
	Magic          []byte
	Version        []byte
	SignatureStack []*IntegritySignature
}

const (
	Ed25519publicKeyAttributeName = "ed25519PublicKey"
)

var IntegrityBlockMagic = []byte{0xf0, 0x9f, 0x96, 0x8b, 0xf0, 0x9f, 0x93, 0xa6}

// "b1" as bytes and 2 empty bytes
var VersionB1 = []byte{0x31, 0x62, 0x00, 0x00}

// cborBytes writes the signature attributes map as CBOR using the given encoder so that the map's key is text string and value byte string.
func (sa SignatureAttributesMap) cborBytes(enc *cbor.Encoder) error {
	mes := []*cbor.MapEntryEncoder{}
	for key, value := range sa {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString(key)
				valueE.EncodeByteString(value)
			}))
	}
	if err := enc.EncodeMap(mes); err != nil {
		return fmt.Errorf("integrityblock: Failed to encode signature attributes: %v", err)
	}
	return nil
}

// cborBytes writes the integrity signature as CBOR using the given encoder containing the signature attributes and the signature.
func (is *IntegritySignature) cborBytes(enc *cbor.Encoder) error {
	enc.EncodeArrayHeader(2)

	if err := is.SignatureAttributes.cborBytes(enc); err != nil {
		return fmt.Errorf("integrityblock: Failed to encode signature attributes: %v", err)
	}

	if err := enc.EncodeByteString(is.Signature); err != nil {
		return fmt.Errorf("integrityblock: Failed to encode signature: %v", err)
	}
	return nil
}

// CborBytes returns the CBOR encoded bytes of the integrity block.
func (ib *IntegrityBlock) CborBytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := cbor.NewEncoder(&buf)

	err := enc.EncodeArrayHeader(3)
	if err != nil {
		return nil, err
	}

	err = enc.EncodeByteString(ib.Magic)
	if err != nil {
		return nil, err
	}

	err = enc.EncodeByteString(ib.Version)
	if err != nil {
		return nil, err
	}

	err = enc.EncodeArrayHeader(len(ib.SignatureStack))
	for _, integritySignature := range ib.SignatureStack {
		if err := integritySignature.cborBytes(enc); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// generateEmptyIntegrityBlock creates an empty integrity block which does not have any integrity signatures in the signature stack yet.
func generateEmptyIntegrityBlock() *IntegrityBlock {
	var integritySignatures []*IntegritySignature

	integrityBlock := &IntegrityBlock{
		Magic:          IntegrityBlockMagic,
		Version:        VersionB1,
		SignatureStack: integritySignatures,
	}
	return integrityBlock
}

// readWebBundlePayloadLength returns the length of the web bundle parsed from the last 8 bytes of the web bundle file.
// [Web Bundle's Trailing Length]: https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html#name-trailing-length
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
func ObtainIntegrityBlock(bundleFile *os.File) (*IntegrityBlock, int64, error) {
	webBundleLen, err := readWebBundlePayloadLength(bundleFile)
	if err != nil {
		return nil, 0, err
	}
	fileStats, err := bundleFile.Stat()
	if err != nil {
		return nil, 0, err
	}

	integrityBlockLen := fileStats.Size() - webBundleLen
	if integrityBlockLen < 0 {
		return nil, -1, errors.New("Integrity block length should never be negative. Web bundle length big endian seems to be bigger than the size of the file.")
	}

	if integrityBlockLen != 0 {
		// Read existing integrity block. Not supported in v1.
		return nil, integrityBlockLen, errors.New("Web bundle already contains an integrity block. Please provide an unsigned web bundle.")
	}

	integrityBlock := generateEmptyIntegrityBlock()
	return integrityBlock, integrityBlockLen, nil
}

func (integrityBlock *IntegrityBlock) addNewSignatureToIntegrityBlock(signatureAttributes SignatureAttributesMap, signature []byte) {
	is := []*IntegritySignature{{
		SignatureAttributes: signatureAttributes,
		Signature:           signature,
	}}

	integrityBlock.SignatureStack = append(is, integrityBlock.SignatureStack...)
}

// ComputeWebBundleSha512 computes the SHA-512 hash over the given web bundle file.
func ComputeWebBundleSha512(bundleFile io.ReadSeeker, offset int64) ([]byte, error) {
	h := sha512.New()

	// Move the file pointer to the start of the web bundle bytes.
	bundleFile.Seek(offset, io.SeekStart)

	// io.Copy() will do chunked read/write under the hood
	_, err := io.Copy(h, bundleFile)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// GenerateDataToBeSigned creates a bytes array containing the payload of which the signature of the web bundle will be calculated.
// The order must be the following, where the lengths are represented as 64 bit big-endian integers:
// (1) length of the web bundle hash, (2) web bundle hash, (3) length of the serialized integrity-block
// (4) serialized integrity-block, (5) length of the attributes, (6) serialized attributes
func GenerateDataToBeSigned(webBundleHash, integrityBlockBytes []byte, signatureAttributes SignatureAttributesMap) ([]byte, error) {
	var attributesBytesBuf bytes.Buffer
	enc := cbor.NewEncoder(&attributesBytesBuf)
	if err := signatureAttributes.cborBytes(enc); err != nil {
		return nil, fmt.Errorf("integrityblock: Failed to encode signature attributes: %v", err)
	}
	attributesBytes := attributesBytesBuf.Bytes()

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint64(len(webBundleHash)))
	buf.Write(webBundleHash)
	binary.Write(&buf, binary.BigEndian, uint64(len(integrityBlockBytes)))
	buf.Write(integrityBlockBytes)
	binary.Write(&buf, binary.BigEndian, uint64(len(attributesBytes)))
	buf.Write(attributesBytes)
	return buf.Bytes(), nil
}

// GenerateSignatureAttributesWithPublicKey generates the basis for the map for signature attributes containing the public key.
func GenerateSignatureAttributesWithPublicKey(ed25519publicKey ed25519.PublicKey) SignatureAttributesMap {
	return SignatureAttributesMap{Ed25519publicKeyAttributeName: []byte(ed25519publicKey)}
}

// WebBundleHasIntegrityBlock is a helper function that can be called with any file path to check if it has
// an integrtiy block. Basically this checks if the bytes fileBytes[2:10] match with the magic bytes.
func WebBundleHasIntegrityBlock(bundleFile io.ReadSeeker) (bool, error) {
	bundleFile.Seek(2, io.SeekStart)

	possibleMagic := make([]byte, len(IntegrityBlockMagic))
	numBytesRead, err := io.ReadFull(bundleFile, possibleMagic)
	if err != nil {
		return false, err
	}
	if numBytesRead != len(IntegrityBlockMagic) {
		return false, nil
	}

	// Return to the start of the file.
	bundleFile.Seek(0, io.SeekStart)

	return bytes.Compare(IntegrityBlockMagic, possibleMagic) == 0, nil
}
