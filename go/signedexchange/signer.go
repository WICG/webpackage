package signedexchange

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

type Signer struct {
	Date        time.Time
	Expires     time.Time
	Certs       []*x509.Certificate
	CertUrl     *url.URL
	ValidityUrl *url.URL
	PrivKey     crypto.PrivateKey
	Rand        io.Reader
}

func certSha256(certs []*x509.Certificate) []byte {
	// Binary content (Section 4.5 of [I-D.ietf-httpbis-header-structure])
	// holding the SHA-256 hash of the first certificate found at "certUrl".
	if len(certs) == 0 {
		return nil
	}
	sum := sha256.Sum256(certs[0].Raw)
	return sum[:]
}

func (s *Signer) serializeSignedMessage(e *Exchange) ([]byte, error) {
	// "Let message be the concatenation of the following byte strings.
	// This matches the [I-D.ietf-tls-tls13] format to avoid cross-protocol
	// attacks when TLS certificates are used to sign manifests." [spec text]
	var buf bytes.Buffer

	// "1. A context string: the ASCII encoding of "HTTP Exchange"." [spec text]
	buf.WriteString("HTTP Exchange")

	// "2. A single 0 byte which serves as a separator." [spec text]
	buf.WriteByte(0)

	// "3. The bytes of the canonical CBOR serialization (Section 3.5) of a CBOR map
	// mapping:" [spec text]
	mes := []*cbor.MapEntryEncoder{}

	// "3.1. If certSha256 is set: The text string "certSha256" to the byte string
	// certSha256." [spec text]
	if b := certSha256(s.Certs); len(b) > 0 {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("certSha256")
				valueE.EncodeByteString(b)
			}))
	}

	mes = append(mes,
		// "3.2. The text string "validityUrl" to the byte string value of validityUrl."
		// [spec text]
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("validityUrl")
			valueE.EncodeByteString([]byte(s.ValidityUrl.String()))
		}),
		// "3.3. The text string "date" to the integer value of date."
		// [spec text]
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("date")
			valueE.EncodeInt(s.Date.Unix())
		}),
		// "3.4. The text string "expires" to the integer value of expires."
		// [spec text]
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("expires")
			valueE.EncodeInt(s.Expires.Unix())
		}),
		// "3.5. The text string "headers" to the CBOR representation (Section
		// 3.4) of exchange's headers."
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("headers")
			e.encodeExchangeHeaders(valueE)
		}),
	)

	enc := cbor.NewEncoder(&buf)
	if err := enc.EncodeMap(mes); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Signer) sign(e *Exchange) ([]byte, error) {
	r := s.Rand
	if r == nil {
		r = rand.Reader
	}
	alg, err := SigningAlgorithmForPrivateKey(s.PrivKey, r)
	if err != nil {
		return nil, err
	}

	msg, err := s.serializeSignedMessage(e)
	if err != nil {
		return nil, err
	}

	return alg.Sign(msg)
}

func (s *Signer) signatureHeaderValue(e *Exchange) (string, error) {
	sig, err := s.sign(e)
	if err != nil {
		return "", err
	}

	label := "label"
	sigb64 := base64.RawStdEncoding.EncodeToString(sig)
	integrityStr := "mi"
	certUrl := s.CertUrl.String()
	validityUrl := s.ValidityUrl.String()
	certSha256b64 := base64.RawStdEncoding.EncodeToString(certSha256(s.Certs))
	dateUnix := s.Date.Unix()
	expiresUnix := s.Expires.Unix()

	return fmt.Sprintf(
		"%s; sig=*%s; validityUrl=%q; integrity=%q; certUrl=%q; certSha256=*%s; date=%d; expires=%d",
		label, sigb64, validityUrl, integrityStr, certUrl, certSha256b64, dateUnix, expiresUnix), nil
}
