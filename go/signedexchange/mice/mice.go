// Package mice implements Merkle Integrity Content Encoding
// (https://martinthomson.github.io/http-mice/draft-thomson-http-mice.html).
package mice

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"io"
)

// Encoding identifies which draft version of http-mice to use.
type Encoding string
const (
	// https://tools.ietf.org/html/draft-thomson-http-mice-02
	Draft02Encoding Encoding = "mi-sha256-draft2"
	// https://tools.ietf.org/html/draft-thomson-http-mice-03
	Draft03Encoding Encoding = "mi-sha256-03"
)

// ContentEncoding returns content encoding name of the Encoding.
func (enc Encoding) ContentEncoding() string {
	return string(enc)
}

// DigestHeaderName returns the name of HTTP header that carries integrity proofs.
func (enc Encoding) DigestHeaderName() string {
	if enc == Draft02Encoding {
		return "MI-Draft2"
	}
	return "Digest"
}

func (enc Encoding) digestHeaderValue(topLevelProof []byte) string {
	return enc.ContentEncoding() + "=" + enc.base64Encoding().EncodeToString(topLevelProof)
}

func (enc Encoding) base64Encoding() *base64.Encoding {
	switch enc {
	case Draft02Encoding:
		return base64.RawURLEncoding
	case Draft03Encoding:
		return base64.StdEncoding
	default:
		panic("not reached")
	}
}

// Encode encodes content of buf and writes to w. Encode returns Digest header
// value (or MI header value in draft 02), and error if one exists.
func (enc Encoding) Encode(w io.Writer, buf []byte, recordSize int) (string, error) {

	numRecords := (len(buf) + recordSize - 1) / recordSize

	switch enc {
	case Draft02Encoding:
		if len(buf) == 0 {
			numRecords = 1
		}

	case Draft03Encoding:
		if len(buf) == 0 {
			// As a special case, the encoding of an empty payload is itself an
			// empty message (i.e. it omits the initial record size), and its
			// integrity proof is SHA-256("\0"). [spec text]
			h := sha256.New()
			h.Write([]byte{0})
			proof := h.Sum(nil)
			return enc.digestHeaderValue(proof), nil
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
	return enc.digestHeaderValue(proofs[0]), nil
}
