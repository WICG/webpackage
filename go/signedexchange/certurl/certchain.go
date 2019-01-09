package certurl

import (
	"bytes"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"

	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

type CertChainItem struct {
	Cert         *x509.Certificate // A parsed X.509 certificate.
	OCSPResponse []byte            // DER-encoded OCSP response for Cert.
	SCTList      []byte            // SignedCertificateTimestampList (Section 3.3 of RFC6962) for Cert.
}

type CertChain []*CertChainItem

const magicString = "\U0001F4DC\u26D3" // "📜⛓"

// NewCertChain creates a new CertChain from a list of X.509 certificates,
// an OCSP response, and a SCT.
func NewCertChain(certs []*x509.Certificate, ocsp, sct []byte) (CertChain, error) {
	if len(certs) == 0 {
		return nil, errors.New("cert-chain: cert chain must not be empty")
	}

	certChain := CertChain{}
	for _, cert := range certs {
		certChain = append(certChain, &CertChainItem{Cert: cert})
	}
	certChain[0].OCSPResponse = ocsp
	certChain[0].SCTList = sct
	return certChain, nil
}

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

// ReadCertChain parses the application/cert-chain+cbor format.
// See https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cert-chain-format for the spec.
func ReadCertChain(r io.Reader) (CertChain, error) {
	dec := cbor.NewDecoder(r)
	n, err := dec.DecodeArrayHeader()
	if err != nil {
		return nil, fmt.Errorf("cert-chain: failed to decode top-level array header: %v", err)
	}
	if n < 2 {
		return nil, fmt.Errorf("cert-chain: length of top-level array must be at least 2 but %d", n)
	}
	magic, err := dec.DecodeTextString()
	if err != nil {
		return nil, fmt.Errorf("cert-chain: failed to decode magic string: %v", err)
	}
	if magic != magicString {
		return nil, fmt.Errorf("cert-chain: wrong magic string: %v", magic)
	}
	certChain := CertChain{}
	for i := uint64(1); i < n; i++ {
		m, err := dec.DecodeMapHeader()
		if err != nil {
			return nil, fmt.Errorf("cert-chain: failed to decode certificate map header: %v", err)
		}
		item := &CertChainItem{}
		for j := uint64(0); j < m; j++ {
			key, err := dec.DecodeTextString()
			if err != nil {
				return nil, fmt.Errorf("cert-chain: failed to decode map key: %v", err)
			}
			value, err := dec.DecodeByteString()
			if err != nil {
				return nil, fmt.Errorf("cert-chain: failed to decode map value: %v", err)
			}
			switch key {
			case "cert":
				item.Cert, err = x509.ParseCertificate(value)
				if err != nil {
					return nil, fmt.Errorf("cert-chain: cannot parse X.509 certificate at position %d: %v", i, err)
				}
			case "ocsp":
				item.OCSPResponse = value
			case "sct":
				item.SCTList = value
			}
		}
		if item.Cert == nil {
			return nil, fmt.Errorf("cert-chain: certificate map at position %d has no \"cert\" key.", i)
		}
		if i == 1 && item.OCSPResponse == nil {
			return nil, fmt.Errorf("cert-chain: the first certificate must have \"ocsp\" key.")
		}
		if i != 1 && item.OCSPResponse != nil {
			return nil, fmt.Errorf("cert-chain: certificate map at position %d must not have \"ocsp\" key.", i)
		}
		certChain = append(certChain, item)
	}
	return certChain, nil
}

func (chain CertChain) PrettyPrint(w io.Writer) {
	for i, item := range chain {
		fmt.Fprintf(w, "Certificate #%d:\n", i)
		fmt.Fprintln(w, "  Subject:", item.Cert.Subject.CommonName)
		fmt.Fprintln(w, "  Valid from:", item.Cert.NotBefore)
		fmt.Fprintln(w, "  Valid until:", item.Cert.NotAfter)
		fmt.Fprintln(w, "  Issuer:", item.Cert.Issuer.CommonName)
		prettyPrintSCTFromCert(w, item.Cert)

		if i == 0 {
			// Check that the main certificate has canSignHttpExchangesDraft extension.
			// https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cross-origin-cert-req
			oidCanSignHttpExchangesDraft := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 1, 22}
			ext := findExtensionWithOID(item.Cert.Extensions, oidCanSignHttpExchangesDraft)
			if ext == nil {
				fmt.Fprintln(w, "Error: The main certificate does not have canSignHttpExchangesDraft extension")
			} else if !bytes.Equal(ext.Value, asn1.NullBytes) {
				fmt.Fprintln(w, "Error: Value of canSignHttpExchangesDraft extension must be ASN1:NULL. got:", ext.Value)
			} else {
				fmt.Fprintln(w, "  Has canSignHttpExchangesDraft extension")
			}
		}

		if item.OCSPResponse != nil {
			fmt.Fprintln(w, "OCSP response:")
			chain.prettyPrintOCSP(w, item.OCSPResponse)
		}
		if item.SCTList != nil {
			fmt.Fprintln(w, "SCT:")
			prettyPrintSCT(w, item.SCTList)
		}
	}
}
