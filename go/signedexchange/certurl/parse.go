package certurl

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

// CertificateMessageFromPEM parses a PEM formatted content to a certUrl content.
// See https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html for the spec.
func CertificateMessageFromPEM(pemFileContent, ocspFileContent, sctFileContent []byte) ([]byte, error) {
	b := pemFileContent

	entries := []*x509.Certificate{}
	for {
		block, rest := pem.Decode(b)
		if block == nil && len(rest) > 0 {
			return nil, fmt.Errorf("failed to parse PEM file")
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		entries = append(entries, c)

		if len(rest) == 0 {
			break
		}
		b = rest
	}

	buf := &bytes.Buffer{}
	enc := cbor.NewEncoder(buf)

	if err := enc.EncodeArrayHeader(len(entries) + 1); err != nil {
		return nil, err
	}
	if err := enc.EncodeTextString("\U0001F4DC\u26D3"); err != nil {
		return nil, err
	}
	for i, entry := range entries {
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
