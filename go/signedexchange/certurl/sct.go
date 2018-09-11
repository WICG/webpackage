package certurl

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/ocsp"
	"io"
)

const maxSerializedSCTLength = 0xffff

var (
	// OIDs for embedded SCTs (Section 3.3 of RFC6962).
	oidCertExtension = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 2}
	oidOCSPExtension = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 5}
)

// SerializeSCTList serializes a list of SignedCertificateTimestamps into a
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

// HasEmbeddedSCT returns true if the certificate or the OCSP response have
// embedded SCT list.
func HasEmbeddedSCT(cert *x509.Certificate, ocsp_resp *ocsp.Response) bool {
	return (cert != nil && findExtensionWithOID(cert.Extensions, oidCertExtension) != nil) ||
		(ocsp_resp != nil && findExtensionWithOID(ocsp_resp.Extensions, oidOCSPExtension) != nil)
}

func findExtensionWithOID(extensions []pkix.Extension, oid asn1.ObjectIdentifier) *pkix.Extension {
	for _, ext := range extensions {
		if ext.Id.Equal(oid) {
			return &ext
		}
	}
	return nil
}

func prettyPrintSCTFromCert(w io.Writer, cert *x509.Certificate) {
	prettyPrintSCTExtension(w, cert.Extensions, oidCertExtension)
}

func prettyPrintSCTFromOCSP(w io.Writer, ocspResp *ocsp.Response) {
	prettyPrintSCTExtension(w, ocspResp.Extensions, oidOCSPExtension)
}

func prettyPrintSCTExtension(w io.Writer, extensions []pkix.Extension, oid asn1.ObjectIdentifier) {
	ext := findExtensionWithOID(extensions, oid)
	if ext == nil {
		return
	}
	var sct []byte
	if _, err := asn1.Unmarshal(ext.Value, &sct); err != nil {
		fmt.Fprintln(w, "Error: Cannot parse SCT extension as ASN.1 OCTET STRING:", err)
		return
	}
	fmt.Fprintln(w, "  Embedded SCT:")
	prettyPrintSCT(w, sct)
}

func prettyPrintSCT(w io.Writer, SCTList []byte) {
	buf := bytes.NewBuffer(SCTList)

	var total_length uint16
	if err := binary.Read(buf, binary.BigEndian, &total_length); err != nil {
		fmt.Fprintln(w, "Error: Cannot parse length of SignedCertificateTimestampList:", err)
		return
	}
	if int(total_length) != buf.Len() {
		fmt.Fprintf(w, "Error: Unexpected length of SignedCertificateTimestampList. expected: %d, actual: %d\n", total_length, buf.Len())
		return
	}

	for buf.Len() > 0 {
		var length uint16
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			fmt.Fprintln(w, "Error: Cannot parse length of SerializedSCT:", err)
			return
		}
		sct := buf.Next(int(length))
		if int(length) != len(sct) {
			fmt.Fprintf(w, "Error: Unexpected length of SerializedSCT. expected: %d, actual: %d\n", length, len(sct))
			return
		}

		// sct[0] is the Version and sct[1:33] is the LogID of the SCT (Section 3.2 of RFC6962).
		if len(sct) < 33 {
			fmt.Fprintf(w, "Error: SCT too short (%d bytes)\n", len(sct))
			return
		}
		if sct[0] != 0 {
			fmt.Fprintf(w, "Error: Unknown version of SCT (%d)\n", sct[0])
			return
		}
		fmt.Fprintln(w, "    LogID:", base64.StdEncoding.EncodeToString(sct[1:33]))
	}
}
