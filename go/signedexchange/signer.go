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
	"github.com/WICG/webpackage/go/signedexchange/internal/bigendian"
	"github.com/WICG/webpackage/go/signedexchange/internal/signingalgorithm"
	"github.com/WICG/webpackage/go/signedexchange/version"
)

func contextString(v version.Version) string {
	switch v {
	case version.Version1b1:
		// contextString is the "context string" in Step 7.2 of
		// https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#signature-validity
		return "HTTP Exchange 1 b1"
	case version.Version1b2:
		return "HTTP Exchange 1 b2"
	case version.Version1b3:
		return "HTTP Exchange 1 b3"
	default:
		panic("not reached")
	}
}

type Signer struct {
	Date        time.Time
	Expires     time.Time
	Certs       []*x509.Certificate
	CertUrl     *url.URL
	ValidityUrl *url.URL
	PrivKey     crypto.PrivateKey
	Rand        io.Reader
}

func calculateCertSha256(certs []*x509.Certificate) []byte {
	// Binary content (Section 4.5 of [I-D.ietf-httpbis-header-structure])
	// holding the SHA-256 hash of the first certificate found at "certUrl".
	if len(certs) == 0 {
		return nil
	}
	sum := sha256.Sum256(certs[0].Raw)
	return sum[:]
}

func serializeSignedMessage(e *Exchange, certSha256 []byte, validityUrl string, date, expires int64) ([]byte, error) {
	switch e.Version {
	case version.Version1b1:
		// "Let message be the concatenation of the following byte strings.
		// This matches the [I-D.ietf-tls-tls13] format to avoid cross-protocol
		// attacks when TLS certificates are used to sign manifests." [spec text]
		var buf bytes.Buffer

		// "1. A string that consists of octet 32 (0x20) repeated 64 times." [spec text]
		for i := 0; i < 64; i++ {
			buf.WriteByte(0x20)
		}

		// "2. A context string: the ASCII encoding of "HTTP Exchange"." [spec text]
		buf.WriteString(contextString(e.Version))

		// "3. A single 0 byte which serves as a separator." [spec text]
		buf.WriteByte(0)

		// "4. The bytes of the canonical CBOR serialization (Section 3.5) of a CBOR map
		// mapping:" [spec text]
		mes := []*cbor.MapEntryEncoder{}

		// "4.1. If cert-sha256 is set: The text string "cert-sha256" to the byte string
		// cert-sha256." [spec text]
		if certSha256 != nil {
			mes = append(mes,
				cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
					keyE.EncodeTextString("cert-sha256")
					valueE.EncodeByteString(certSha256)
				}))
		}

		mes = append(mes,
			// "4.2. The text string "validity-url" to the byte string value of validity-url."
			// [spec text]
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("validity-url")
				valueE.EncodeByteString([]byte(validityUrl))
			}),
			// "4.3. The text string "date" to the integer value of date."
			// [spec text]
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("date")
				valueE.EncodeInt(date)
			}),
			// "4.4. The text string "expires" to the integer value of expires."
			// [spec text]
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("expires")
				valueE.EncodeInt(expires)
			}),
			// "4.5. The text string "headers" to the CBOR representation (Section
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

	case version.Version1b2, version.Version1b3:
		// draft-yasskin-http-origin-signed-responses.html#signature-validity

		// "Let message be the concatenation of the following byte strings. This matches the [I-D.ietf-tls-tls13] format to avoid cross-protocol attacks if anyone uses the same key in a TLS certificate and an exchange-signing certificate." [spec text]
		var buf bytes.Buffer

		// "1. A string that consists of octet 32 (0x20) repeated 64 times." [spec text]
		for i := 0; i < 64; i++ {
			buf.WriteByte(0x20)
		}

		// "2. A context string: the ASCII encoding of “HTTP Exchange 1”." [spec text]
		buf.WriteString(contextString(e.Version))

		// "3. A single 0 byte which serves as a separator." [spec text]
		buf.WriteByte(0)

		// "4. If cert-sha256 is set, a byte holding the value 32 followed by the 32 bytes of the value of cert-sha256. Otherwise a 0 byte." [spec text]
		if certSha256 != nil {
			buf.WriteByte(32)
			buf.Write(certSha256)
		}

		// "5. The 8-byte big-endian encoding of the length in bytes of validity-url, followed by the bytes of validity-url." [spec text]
		//bigendian
		vurl := []byte(validityUrl)
		vurlLenBytes, _ := bigendian.EncodeBytesUint(int64(len(vurl)), 8)
		buf.Write(vurlLenBytes)
		buf.Write(vurl)

		// "6. The 8-byte big-endian encoding of date." [spec text]
		dateBytes, _ := bigendian.EncodeBytesUint(date, 8)
		buf.Write(dateBytes)

		// "7. The 8-byte big-endian encoding of expires." [spec text]
		expiresBytes, _ := bigendian.EncodeBytesUint(expires, 8)
		buf.Write(expiresBytes)

		// "8. The 8-byte big-endian encoding of the length in bytes of requestUrl, followed by the bytes of requestUrl." [spec text]
		rurl := []byte(e.RequestURI)
		rurlLenBytes, _ := bigendian.EncodeBytesUint(int64(len(rurl)), 8)
		buf.Write(rurlLenBytes)
		buf.Write(rurl)

		// "9. The 8-byte big-endian encoding of the length in bytes of headers, followed by the bytes of headers." [spec text]
		headerBuf := &bytes.Buffer{}
		if err := e.encodeExchangeHeaders(cbor.NewEncoder(headerBuf)); err != nil {
			return nil, err
		}
		headerLenBytes, _ := bigendian.EncodeBytesUint(int64(headerBuf.Len()), 8)
		buf.Write(headerLenBytes)
		headerBuf.WriteTo(&buf)

		return buf.Bytes(), nil
	default:
		panic("not reached")
	}
}

func (s *Signer) sign(e *Exchange) ([]byte, error) {
	r := s.Rand
	if r == nil {
		r = rand.Reader
	}
	alg, err := signingalgorithm.SigningAlgorithmForPrivateKey(s.PrivKey, r)
	if err != nil {
		return nil, err
	}

	msg, err := serializeSignedMessage(e, calculateCertSha256(s.Certs), s.ValidityUrl.String(), s.Date.Unix(), s.Expires.Unix())
	if err != nil {
		return nil, err
	}

	return alg.Sign(msg)
}

func (s *Signer) signatureHeaderValue(e *Exchange) (string, error) {
	switch s.CertUrl.Scheme {
	case "https", "data":
		break
	default:
		return "", fmt.Errorf("signedexchange: cert-url with disallowed scheme %q. cert-url must have a scheme of \"https\" or \"data\".", s.CertUrl.Scheme)
	}

	sig, err := s.sign(e)
	if err != nil {
		return "", err
	}

	label := "label"
	sigb64 := base64.StdEncoding.EncodeToString(sig)
	integrityStr := ""
	switch e.Version {
	case version.Version1b1:
		integrityStr = "mi-draft2"
	case version.Version1b2, version.Version1b3:
		integrityStr = "digest/mi-sha256-03"
	default:
		panic("not reached")
	}
	certUrl := s.CertUrl.String()
	validityUrl := s.ValidityUrl.String()
	certSha256b64 := base64.StdEncoding.EncodeToString(calculateCertSha256(s.Certs))
	dateUnix := s.Date.Unix()
	expiresUnix := s.Expires.Unix()

	return fmt.Sprintf(
		"%s; sig=*%s*; validity-url=%q; integrity=%q; cert-url=%q; cert-sha256=*%s*; date=%d; expires=%d",
		label, sigb64, validityUrl, integrityStr, certUrl, certSha256b64, dateUnix, expiresUnix), nil
}
