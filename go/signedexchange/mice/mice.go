package mice

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"io"
)

// Encode encodes the given content buf to MICE (Merkle Integrity Content Encoding)
// format.
//
// Encode returns MI header field parameter string and error if exists.
//
// Spec: https://tools.ietf.org/html/draft-thomson-http-mice-02
func Encode(w io.Writer, buf []byte, recordSize int) (string, error) {
	numRecords := (len(buf) + recordSize - 1) / recordSize
	if len(buf) == 0 {
		numRecords = 1
	}

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
