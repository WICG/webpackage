package certurl

import (
	"bytes"
	"crypto/x509"

	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

// CreateCertChainCBOR generates a certificate chain of application/cert-chain+cbor format.
// See https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cert-chain-format for the spec.
func CreateCertChainCBOR(certs []*x509.Certificate, ocspFileContent, sctFileContent []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := cbor.NewEncoder(buf)

	if err := enc.EncodeArrayHeader(len(certs) + 1); err != nil {
		return nil, err
	}
	if err := enc.EncodeTextString("\U0001F4DC\u26D3"); err != nil {
		return nil, err
	}
	for i, entry := range certs {
		mes := []*cbor.MapEntryEncoder{
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("cert")
				valueE.EncodeByteString(entry.Raw)
			}),
		}
		if i == 0 {
			if ocspFileContent != nil {
				mes = append(mes,
					cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
						keyE.EncodeTextString("ocsp")
						valueE.EncodeByteString(ocspFileContent)
					}))
			}
			if sctFileContent != nil {
				mes = append(mes,
					cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
						keyE.EncodeTextString("sct")
						valueE.EncodeByteString(sctFileContent)
					}))
			}
		}
		if err := enc.EncodeMap(mes); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
