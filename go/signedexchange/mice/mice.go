package mice

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
)

// Encode encodes the given content buf to MICE (Merkle Integrity Content Encoding)
// format.
//
// Encode returns MI header field parameter string and error if one exists.
//
// Spec: https://tools.ietf.org/html/draft-thomson-http-mice-02
func Encode(w io.Writer, buf []byte, recordSize int) (string, error) {
	numRecords := (len(buf) + recordSize - 1) / recordSize
	if len(buf) == 0 {
		numRecords = 1
	}

	// Calculate proofs. This loop iterates from the tail of the content and creates
	// the proof chain.
	proofs := make([][]byte, numRecords)
	for i := 0; i < numRecords; i++ {
		rec := numRecords - i - 1
		h := sha256.New()
		if i == 0 {
			h.Write(buf[rec*recordSize:])
			h.Write([]byte{0})
		} else {
			h.Write(buf[rec*recordSize : (rec+1)*recordSize])
			h.Write(proofs[rec+1])
			h.Write([]byte{1})
		}
		proofs[rec] = h.Sum(nil)
	}

	if err := binary.Write(w, binary.BigEndian, uint64(recordSize)); err != nil {
		return "", err
	}
	for i, proof := range proofs {
		if i != 0 {
			if _, err := w.Write(proof); err != nil {
				return "", err
			}
		}
		high := (i + 1) * recordSize
		if high > len(buf) {
			high = len(buf)
		}
		if _, err := w.Write(buf[i*recordSize : high]); err != nil {
			return "", err
		}
	}

	mi := "mi-sha256=" + base64.RawURLEncoding.EncodeToString(proofs[0])
	return mi, nil
}

func Decode(w io.Writer, r io.Reader, miHeaderValue string) error {
	// TODO(hajimehoshi, kouhei): Check the header with miHeaderValue.

	var recordSize uint64
	if err := binary.Read(r, binary.BigEndian, &recordSize); err != nil {
		return fmt.Errorf("Failed to read recordSize: %v", err)
	}

	proof := make([]byte, sha256.Size)
	record := make([]byte, recordSize)
	readFirstRecord := false
	for {
		if readFirstRecord {
			if _, err := io.ReadFull(r, proof); err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("mice: failed to read proof: %v", err)
			}
		}
		readFirstRecord = true
		n, err := io.ReadFull(r, record)
		if err != nil {
			return fmt.Errorf("mice: failed to read record: %v", err)
		}
		// TODO: verify integrity
		if _, err = w.Write(record[:n]); err != nil {
			return fmt.Errorf("mice: failed to write record: %v", err)
		}
	}
}
