package integrityblock

import (
	"bytes"
	"fmt"

	"github.com/WICG/webpackage/go/internal/cbor"
)

type IntegritySignature struct {
	SignatureAttributes map[string][]byte
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

// CborBytes returns the CBOR encoded bytes of an integrity signature.
func (is *IntegritySignature) CborBytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := cbor.NewEncoder(&buf)
	enc.EncodeArrayHeader(2)

	mes := []*cbor.MapEntryEncoder{}
	for key, value := range is.SignatureAttributes {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString(key)
				valueE.EncodeByteString(value)
			}))
	}
	if err := enc.EncodeMap(mes); err != nil {
		return nil, fmt.Errorf("integrityblock: Failed to encode signature attribute: %v", err)
	}

	if err := enc.EncodeByteString(is.Signature); err != nil {
		return nil, fmt.Errorf("integrityblock: Failed to encode signature: %v", err)
	}
	return buf.Bytes(), nil
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
		isb, err := integritySignature.CborBytes()
		if err != nil {
			return nil, err
		}
		buf.Write(isb)
	}

	return buf.Bytes(), nil
}

// GenerateEmptyIntegrityBlock creates an empty integrity block which does not have any integrity signatures in the signature stack yet.
func GenerateEmptyIntegrityBlock() *IntegrityBlock {
	var integritySignatures []*IntegritySignature

	integrityBlock := &IntegrityBlock{
		Magic:          IntegrityBlockMagic,
		Version:        VersionB1,
		SignatureStack: integritySignatures,
	}
	return integrityBlock
}

// getLastSignatureAttributes returns the signature attributes from the newest (the first)
// signature stack or a new empty map if the signature stack is empty.
func GetLastSignatureAttributes(integrityBlock *IntegrityBlock) map[string][]byte {
	var signatureAttributes map[string][]byte
	if len(integrityBlock.SignatureStack) == 0 {
		signatureAttributes = make(map[string][]byte, 1)
	} else {
		signatureAttributes = (*integrityBlock.SignatureStack[0]).SignatureAttributes
	}
	return signatureAttributes
}
