// Package certurl implements a parser and a serializer for application/cert-chain+cbor format.
// See https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cert-chain-format for the spec.
package certurl

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

// AugmentedCertificate represents an augmented-certificate CBOR structure.
// https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cert-chain-format
type AugmentedCertificate struct {
	Cert         *x509.Certificate // A parsed X.509 certificate.
	OCSPResponse []byte            // DER-encoded OCSP response for Cert.
	SCTList      []byte            // SignedCertificateTimestampList (Section 3.3 of RFC6962) for Cert.
}

type CertChain []*AugmentedCertificate

const magicString = "\U0001F4DC\u26D3" // "📜⛓"

// NewCertChain creates a new CertChain from a list of X.509 certificates,
// an OCSP response, and a SCT.
func NewCertChain(certs []*x509.Certificate, ocsp, sct []byte) (CertChain, error) {
	if len(certs) == 0 {
		return nil, errors.New("cert-chain: cert chain must not be empty")
	}

	certChain := CertChain{}
	for _, cert := range certs {
		certChain = append(certChain, &AugmentedCertificate{Cert: cert})
	}
	certChain[0].OCSPResponse = ocsp
	certChain[0].SCTList = sct
	return certChain, nil
}

// Validate performs basic sanity checks on the cert chain.
// It returns nil if the chain is valid, or else an error describing a problem.
func (certChain CertChain) Validate() error {
	if len(certChain) == 0 {
		return errors.New("cert-chain: cert chain must not be empty")
	}
	for i, item := range certChain {
		if i == 0 && item.OCSPResponse == nil {
			return errors.New("cert-chain: the first certificate must have an OCSP response")
		}
		if i != 0 && item.OCSPResponse != nil {
			return fmt.Errorf("cert-chain: certificate at position %d must not have an OCSP response.", i)
		}
	}
	return nil
}

// Write generates a certificate chain of application/cert-chain+cbor format and writes to w.
func (certChain CertChain) Write(w io.Writer) error {
	if err := certChain.Validate(); err != nil {
		// Don't serialize invalid cert chain.
		return err
	}

	enc := cbor.NewEncoder(w)

	if err := enc.EncodeArrayHeader(len(certChain) + 1); err != nil {
		return err
	}
	if err := enc.EncodeTextString(magicString); err != nil {
		return err
	}
	for _, item := range certChain {
		if err := item.EncodeTo(enc); err != nil {
			return err
		}
	}

	return nil
}

// EncodeTo encodes ac as an augmented-certificate CBOR item.
func (ac *AugmentedCertificate) EncodeTo(enc *cbor.Encoder) error {
	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("cert")
			valueE.EncodeByteString(ac.Cert.Raw)
		}),
	}
	if ac.OCSPResponse != nil {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("ocsp")
				valueE.EncodeByteString(ac.OCSPResponse)
			}))
	}
	if ac.SCTList != nil {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("sct")
				valueE.EncodeByteString(ac.SCTList)
			}))
	}
	return enc.EncodeMap(mes)
}

// CertSha256 returns SHA-256 hash of the DER-encoded X.509v3 certificate.
func (ac *AugmentedCertificate) CertSha256() []byte {
	sum := sha256.Sum256(ac.Cert.Raw)
	return sum[:]
}

// ReadCertChain parses the application/cert-chain+cbor format.
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
		ac, err := DecodeAugmentedCertificateFrom(dec)
		if err != nil {
			return nil, err
		}
		certChain = append(certChain, ac)
	}
	if err := certChain.Validate(); err != nil {
		return nil, err
	}
	return certChain, nil
}

// DecodeAugmentedCertificateFrom decodes an augmented-certificate CBOR item
// from dec and returns it as an AugmentedCertificate.
func DecodeAugmentedCertificateFrom(dec *cbor.Decoder) (*AugmentedCertificate, error) {
	m, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, fmt.Errorf("cert-chain: failed to decode certificate map header: %v", err)
	}
	ac := &AugmentedCertificate{}
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
			ac.Cert, err = x509.ParseCertificate(value)
			if err != nil {
				return nil, fmt.Errorf("cert-chain: cannot parse X.509 certificate: %v", err)
			}
		case "ocsp":
			ac.OCSPResponse = value
		case "sct":
			ac.SCTList = value
		}
	}
	if ac.Cert == nil {
		return nil, fmt.Errorf("cert-chain: certificate map must have \"cert\" key.")
	}
	return ac, nil
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
			// Check if the main certificate meets the requirements:
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

			validityDuration := item.Cert.NotAfter.Sub(item.Cert.NotBefore)
			if validityDuration > 90*24*time.Hour {
				if item.Cert.NotBefore.After(time.Date(2019, 5, 1, 0, 0, 0, 0, time.UTC)) {
					// - Clients MUST reject certificates with this extension that were issued
					// after 2019-05-01 and have a Validity Period longer than 90 days.
					fmt.Fprintln(w, "Error: Signed Exchange's certificate issued after 2019-05-01 must not have a validity period longer than 90 days.")
				} else {
					// - After 2019-08-01, clients MUST reject all certificates with this
					// extension that have a Validity Period longer than 90 days.
					if time.Now().After(time.Date(2019, 8, 1, 0, 0, 0, 0, time.UTC)) {
						fmt.Fprintln(w, "Error: After 2019-08-01, Signed Exchange's certificate must not have a validity period longer than 90 days.")
					} else {
						fmt.Fprintln(w, "Warning: Signed Exchange's certificate must not have a validity period longer than 90 days. This certificate will be rejected after 2019-08-01.")
					}
				}
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
