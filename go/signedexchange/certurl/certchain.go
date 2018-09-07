package certurl

import (
	"crypto/x509"
	"fmt"
	"io"

	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

type CertChainItem struct {
	Cert         *x509.Certificate // A parsed X.509 certificate.
	OCSPResponse []byte            // DER-encoded OCSP response for Cert.
	SCTList      []byte            // SignedCertificateTimestampList (Section 3.3 of RFC6962) for Cert.
}

type CertChain []CertChainItem

const magicString = "\U0001F4DC\u26D3";  // "📜⛓"

// Write generates a certificate chain of application/cert-chain+cbor format and writes to w.
// See https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cert-chain-format for the spec.
func (certChain CertChain) Write(w io.Writer) error {
	enc := cbor.NewEncoder(w)

	if err := enc.EncodeArrayHeader(len(certChain) + 1); err != nil {
		return err
	}
	if err := enc.EncodeTextString(magicString); err != nil {
		return err
	}
	for i, item := range certChain {
		mes := []*cbor.MapEntryEncoder{
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("cert")
				valueE.EncodeByteString(item.Cert.Raw)
			}),
		}
		if i == 0 {
			if item.OCSPResponse == nil {
				return fmt.Errorf("The first certificate must have an OCSP response.")
			}
			mes = append(mes,
				cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
					keyE.EncodeTextString("ocsp")
					valueE.EncodeByteString(item.OCSPResponse)
				}))
		} else {
			if item.OCSPResponse != nil {
				return fmt.Errorf("Certificate at position %d must not have an OCSP response.", i)
			}
		}
		if item.SCTList != nil {
			mes = append(mes,
				cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
					keyE.EncodeTextString("sct")
					valueE.EncodeByteString(item.SCTList)
				}))
		}
		if err := enc.EncodeMap(mes); err != nil {
			return err
		}
	}

	return nil
}
