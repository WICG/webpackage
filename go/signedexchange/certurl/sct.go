package certurl

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/ocsp"
)

const maxSerializedSCTLength = 0xffff

// Serializes a list of SignedCertificateTimestamps into a
// SignedCertificateTimestampList (RFC6962 Section 3.3).
func SerializeSCTList(scts [][]byte) ([]byte, error) {
	total_length := 0
	for _, sct := range scts {
		if len(sct) > maxSerializedSCTLength {
			return nil, fmt.Errorf("SCT too large")
		}
		total_length += len(sct) + 2 // +2 for length
	}
	if total_length > maxSerializedSCTLength {
		return nil, fmt.Errorf("SCT list too large")
	}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, uint16(total_length)); err != nil {
		return nil, err
	}
	for _, sct := range scts {
		if err := binary.Write(&buf, binary.BigEndian, uint16(len(sct))); err != nil {
			return nil, err
		}
		if _, err := buf.Write(sct); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// Returns true if the certificate or the OCSP response have embedded SCT list.
func HasEmbeddedSCT(cert *x509.Certificate, ocsp_resp *ocsp.Response) bool {
	// OIDs for embedded SCTs (Section 3.3 of RFC6962).
	oidCertExtension := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 2}
	oidOCSPExtension := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 5}

	return (cert != nil && hasExtensionWithOID(cert.Extensions, oidCertExtension)) ||
		(ocsp_resp != nil && hasExtensionWithOID(ocsp_resp.Extensions, oidOCSPExtension))
}

func hasExtensionWithOID(extensions []pkix.Extension, oid asn1.ObjectIdentifier) bool {
	for _, ext := range extensions {
		if ext.Id.Equal(oid) {
			return true
		}
	}
	return false
}
