package mice

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"io"

	"github.com/WICG/webpackage/go/signedexchange/version"
)

// Encode encodes the given content buf to MICE (Merkle Integrity Content Encoding)
// format.
//
// Encode returns MI header field parameter string and error if one exists for version 1b1, and
// returns Digest header value parameter string and error if one exists for version 1b2.
//
// Spec: https://tools.ietf.org/html/draft-thomson-http-mice-02 for version 1b1, and
// https://tools.ietf.org/html/draft-thomson-http-mice-03 for version 1b2
func Encode(w io.Writer, buf []byte, recordSize int, ver version.Version) (string, error) {

	numRecords := (len(buf) + recordSize - 1) / recordSize

	switch ver {
	case version.Version1b1:
		if len(buf) == 0 {
			numRecords = 1
		}

	case version.Version1b2:
		if len(buf) == 0 {
			// As a special case, the encoding of an empty payload is itself an
			// empty message (i.e. it omits the initial record size), and its
			// integrity proof is SHA-256("\0"). [spec text]
			h := sha256.New()
			h.Write([]byte{0})
			proof := h.Sum(nil)
			mi := "mi-sha256-03=" + base64.StdEncoding.EncodeToString(proof)
			return mi, nil
		}

	default:
		panic("not reached")
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

	mi := ""
	switch ver {
	case version.Version1b1:
		mi = "mi-sha256-draft2=" + base64.RawURLEncoding.EncodeToString(proofs[0])
	case version.Version1b2:
		mi = "mi-sha256-03=" + base64.StdEncoding.EncodeToString(proofs[0])
	default:
		panic("not reached")
	}
	return mi, nil
}
