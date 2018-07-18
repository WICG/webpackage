package certurl

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
