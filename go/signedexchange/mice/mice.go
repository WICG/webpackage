// Package mice implements Merkle Integrity Content Encoding
// (https://martinthomson.github.io/http-mice/draft-thomson-http-mice.html).
package mice

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Encoding identifies which draft version of http-mice to use.
type Encoding string

const (
	// https://tools.ietf.org/html/draft-thomson-http-mice-02
	Draft02Encoding Encoding = "mi-sha256-draft2"
	// https://tools.ietf.org/html/draft-thomson-http-mice-03
	Draft03Encoding Encoding = "mi-sha256-03"
)

// ErrValidationFailure is returned when integrity check have failed.
var ErrValidationFailure = errors.New("mice: failed to validate record")

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

func (enc Encoding) FormatDigestHeader(topLevelProof []byte) string {
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
			return enc.FormatDigestHeader(proof), nil
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
	return enc.FormatDigestHeader(proofs[0]), nil
}

func (enc Encoding) parseDigestHeader(digestHeaderValue string) ([]byte, error) {
	// TODO: Support multiple digest values (Section 4.3.2 of RFC3230).
	chunks := strings.SplitN(digestHeaderValue, "=", 2)
	if len(chunks) != 2 {
		return nil, fmt.Errorf("mice: cannot parse digest value %q", digestHeaderValue)
	}
	algorithm, digest := chunks[0], chunks[1]

	if algorithm != enc.ContentEncoding() {
		return nil, fmt.Errorf("mice: unsupported digest algorithm %q", algorithm)
	}
	proof, err := enc.base64Encoding().DecodeString(digest)
	if err != nil {
		return nil, fmt.Errorf("mice: failed to decode digest value %q: %v", digest, err)
	}
	if len(proof) != sha256.Size {
		return nil, fmt.Errorf("mice: wrong digest length %q", digest)
	}
	return proof, nil
}

type decoder struct {
	encoding   Encoding
	recordSize uint64
	r          io.Reader
	nextProof  []byte
	recordBuf  []byte
	out        []byte // leftover decoded output
}

// NewDecoder creates a new http-mice stream decoder. It reads first few bytes
// from r to determine the record size, and fails if the record size exceeds
// maxRecordSize.
func (enc Encoding) NewDecoder(r io.Reader, digestHeaderValue string, maxRecordSize uint64) (io.Reader, error) {
	toplevelProof, err := enc.parseDigestHeader(digestHeaderValue)
	if err != nil {
		return nil, err
	}

	var recordSize uint64
	err = binary.Read(r, binary.BigEndian, &recordSize)
	if err == io.EOF && enc != Draft02Encoding {
		// As a special case, the encoding of an empty payload is itself an
		// empty message (i.e. it omits the initial record size), and its
		// integrity proof is SHA-256("\0"). [spec text]
		if !validateRecord(nil, toplevelProof, true) {
			return nil, ErrValidationFailure
		}
		// Return an empty reader.
		return &decoder{encoding: enc}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("mice: cannot read record size: %v", err)
	}
	if recordSize == 0 || recordSize > maxRecordSize {
		return nil, fmt.Errorf("mice: invalid record size %v", recordSize)
	}
	return &decoder{
		encoding:   enc,
		recordSize: recordSize,
		r:          r,
		nextProof:  toplevelProof,
		recordBuf:  make([]byte, recordSize+sha256.Size),
	}, nil
}

func (d *decoder) Read(dst []byte) (int, error) {
	if len(d.out) == 0 {
		if d.nextProof == nil {
			// Already processed all records.
			return 0, io.EOF
		}
		if err := d.readNextRecord(); err != nil {
			return 0, err
		}
	}
	n := copy(dst, d.out)
	d.out = d.out[n:]
	return n, nil
}

func (d *decoder) readNextRecord() error {
	readBytes, err := io.ReadFull(d.r, d.recordBuf)
	if err == io.ErrUnexpectedEOF {
		if uint64(readBytes) > d.recordSize {
			return errors.New("mice: end of input reached in the middle of hash")
		}
		if !validateRecord(d.recordBuf[:readBytes], d.nextProof, true) {
			return ErrValidationFailure
		}
		d.out = d.recordBuf[:readBytes]
		d.nextProof = nil
		return nil
	}
	if err == io.EOF {
		// Draft02 allows empty final record.
		if d.encoding == Draft02Encoding {
			if !validateRecord(nil, d.nextProof, true) {
				return ErrValidationFailure
			}
			d.out = nil
			d.nextProof = nil
			return io.EOF
		}
		return errors.New("mice: unexpected end of input")
	}
	if err != nil {
		return err
	}
	if !validateRecord(d.recordBuf, d.nextProof, false) {
		return ErrValidationFailure
	}
	d.out = d.recordBuf[:d.recordSize]
	copy(d.nextProof, d.recordBuf[d.recordSize:])
	return nil
}

func validateRecord(record, proof []byte, isLastRecord bool) bool {
	h := sha256.New()
	h.Write(record)
	if isLastRecord {
		h.Write([]byte{0})
	} else {
		h.Write([]byte{1})
	}
	return bytes.Equal(h.Sum(nil), proof)
}
