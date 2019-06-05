package signedexchange

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/WICG/webpackage/go/signedexchange/cbor"
	"github.com/WICG/webpackage/go/signedexchange/internal/bigendian"
	"github.com/WICG/webpackage/go/signedexchange/mice"
	"github.com/WICG/webpackage/go/signedexchange/version"
)

type Exchange struct {
	Version version.Version

	// Request
	RequestURI     string
	RequestMethod  string
	RequestHeaders http.Header

	// Response
	ResponseStatus       int
	ResponseHeaders      http.Header
	SignatureHeaderValue string

	// Payload
	Payload []byte
}

var (
	keyMethod = []byte(":method")
	keyURL    = []byte(":url")
	keyStatus = []byte(":status")
)

func NewExchange(ver version.Version, uri string, method string, requestHeaders http.Header, status int, responseHeaders http.Header, payload []byte) *Exchange {
	return &Exchange{
		Version:         ver,
		RequestURI:      uri,
		RequestMethod:   method,
		ResponseStatus:  status,
		RequestHeaders:  requestHeaders,
		ResponseHeaders: responseHeaders,
		Payload:         payload,
	}
}

func (e *Exchange) MiEncodePayload(recordSize int) error {
	var enc mice.Encoding
	switch e.Version {
	case version.Version1b1:
		enc = mice.Draft02Encoding
	case version.Version1b2, version.Version1b3:
		enc = mice.Draft03Encoding
	default:
		panic("not reached")
	}

	if e.ResponseHeaders.Get(enc.DigestHeaderName()) != "" {
		return fmt.Errorf("signedexchange: response already has %q header", enc.DigestHeaderName())
	}
	var buf bytes.Buffer
	digest, err := enc.Encode(&buf, e.Payload, recordSize)
	if err != nil {
		return err
	}
	e.Payload = buf.Bytes()
	e.ResponseHeaders.Add("Content-Encoding", enc.ContentEncoding())
	e.ResponseHeaders.Add(enc.DigestHeaderName(), digest)
	return nil
}

func (e *Exchange) AddSignatureHeader(s *Signer) error {
	h, err := s.signatureHeaderValue(e)
	if err != nil {
		return err
	}
	e.SignatureHeaderValue = h
	return nil
}

func (e *Exchange) encodeRequestMap(enc *cbor.Encoder) error {
	if e.Version != version.Version1b1 && e.Version != version.Version1b2 {
		panic("signedexchange: b3 and beyond don't have request map.")
	}

	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString(keyMethod)
			valueE.EncodeByteString([]byte(e.RequestMethod))
		}),
	}
	if e.Version == version.Version1b1 {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeByteString(keyURL)
				valueE.EncodeByteString([]byte(e.RequestURI))
			}))
	}
	mes = encodeHeaders(mes, e.RequestHeaders)

	return enc.EncodeMap(mes)
}

func normalizeHeaderValues(values []string) string {
	// RFC 2616 - Hypertext Transfer Protocol -- HTTP/1.1
	// 4.2 Message Headers
	// https://tools.ietf.org/html/rfc2616#section-4.2
	//
	// Multiple message-header fields with the same field-name MAY be
	// present in a message if and only if the entire field-value for that
	// header field is defined as a comma-separated list [i.e., #(values)].
	// It MUST be possible to combine the multiple header fields into one
	// "field-name: field-value" pair, without changing the semantics of the
	// message, by appending each subsequent field-value to the first, each
	// separated by a comma. The order in which header fields with the same
	// field-name are received is therefore significant to the
	// interpretation of the combined field value, and thus a proxy MUST NOT
	// change the order of these field values when a message is forwarded.
	return strings.Join(values, ",")
}

func (e *Exchange) decodeRequestMap(dec *cbor.Decoder) error {
	n, err := dec.DecodeMapHeader()
	if err != nil {
		return fmt.Errorf("signedexchange: failed to decode response map header: %v", err)
	}
	for i := uint64(0); i < n; i++ {
		key, err := dec.DecodeByteString()
		if err != nil {
			return err
		}
		value, err := dec.DecodeByteString()
		if err != nil {
			return err
		}
		if bytes.Equal(key, keyMethod) {
			e.RequestMethod = string(value)
			continue
		}
		if bytes.Equal(key, keyURL) {
			if e.Version == version.Version1b1 {
				e.RequestURI, err = validateFallbackURL(value)
				if err != nil {
					return err
				}
				continue
			}
			return fmt.Errorf("signedexchange: found a deprecated request key %q", keyURL)
		}
		e.RequestHeaders.Add(string(key), string(value))
	}
	return nil
}

func (e *Exchange) encodeResponseMap(enc *cbor.Encoder) error {
	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString(keyStatus)
			valueE.EncodeByteString([]byte(strconv.Itoa(e.ResponseStatus)))
		}),
	}
	mes = encodeHeaders(mes, e.ResponseHeaders)
	return enc.EncodeMap(mes)
}

func encodeHeaders(encs []*cbor.MapEntryEncoder, headers http.Header) []*cbor.MapEntryEncoder {
	for name, value := range headers {
		encs = append(encs,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeByteString([]byte(strings.ToLower(name)))
				valueE.EncodeByteString([]byte(normalizeHeaderValues(value)))
			}))
	}
	return encs
}

func (e *Exchange) decodeResponseMap(dec *cbor.Decoder) error {
	n, err := dec.DecodeMapHeader()
	if err != nil {
		return fmt.Errorf("signedexchange: failed to decode response map header: %v", err)
	}
	for i := uint64(0); i < n; i++ {
		key, err := dec.DecodeByteString()
		if err != nil {
			return err
		}
		value, err := dec.DecodeByteString()
		if err != nil {
			return err
		}
		if bytes.Equal(key, keyStatus) {
			e.ResponseStatus, err = strconv.Atoi(string(value))
			if err != nil {
				return err
			}
			continue
		}
		e.ResponseHeaders.Add(string(key), string(value))
	}
	return nil
}

// draft-yasskin-http-origin-signed-responses.html#rfc.section.3.4
func (e *Exchange) encodeExchangeHeaders(enc *cbor.Encoder) error {
	if e.Version == version.Version1b1 || e.Version == version.Version1b2 {
		if err := enc.EncodeArrayHeader(2); err != nil {
			return fmt.Errorf("signedexchange: failed to encode top-level array header: %v", err)
		}
		if err := e.encodeRequestMap(enc); err != nil {
			return err
		}
	}
	if err := e.encodeResponseMap(enc); err != nil {
		return err
	}
	return nil
}

func (e *Exchange) DumpExchangeHeaders(w io.Writer) error {
	enc := cbor.NewEncoder(w)
	return e.encodeExchangeHeaders(enc)
}

func (e *Exchange) decodeExchangeHeaders(dec *cbor.Decoder) error {
	if e.Version == version.Version1b1 || e.Version == version.Version1b2 {
		n, err := dec.DecodeArrayHeader()
		if err != nil {
			return fmt.Errorf("signedexchange: failed to decode top-level array header: %v", err)
		}
		if n != 2 {
			return fmt.Errorf("singedexchange: length of header array must be 2 but %d", n)
		}
		if err := e.decodeRequestMap(dec); err != nil {
			return err
		}
	} else {
		e.RequestMethod = http.MethodGet
	}
	if err := e.decodeResponseMap(dec); err != nil {
		return err
	}
	return nil
}

func (e *Exchange) Write(w io.Writer) error {
	var headerBuf bytes.Buffer
	if err := e.DumpExchangeHeaders(&headerBuf); err != nil {
		return err
	}
	headerLength := headerBuf.Len()

	switch e.Version {
	case version.Version1b1:
		// draft-yasskin-http-origin-signed-responses.html#application-http-exchange

		// Step 1. "The ASCII characters "sxg1" followed by a 0 byte, to serve as a file signature. This is redundant with the MIME type, and recipients that receive both MUST check that they match and stop parsing if they don't." [spec text]
		// "Note: RFC EDITOR PLEASE DELETE THIS NOTE; The implementation of the final RFC MUST use this file signature, but implementations of drafts MUST NOT use it and MUST use another implementation-specific string beginning with "sxg1-" and ending with a 0 byte instead." [spec text]
		if _, err := w.Write(e.Version.HeaderMagicBytes()); err != nil {
			return err
		}

		// Step 2. "3 bytes storing a big-endian integer sigLength. If this is larger than TBD, parsing MUST fail." [spec text]
		encodedSigLength, err := bigendian.EncodeBytesUint(int64(len(e.SignatureHeaderValue)), 3)
		if err != nil {
			return err
		}
		if _, err := w.Write(encodedSigLength); err != nil {
			return err
		}

		// Step 3. "3 bytes storing a big-endian integer headerLength. If this is larger than TBD, parsing MUST fail." [spec text]
		encodedHeaderLength, err := bigendian.EncodeBytesUint(int64(headerLength), 3)
		if err != nil {
			return err
		}
		if _, err := w.Write(encodedHeaderLength); err != nil {
			return err
		}

		// Step 4. "sigLength bytes holding the Signature header field's value (Section 3.1)." [spec text]
		if _, err := w.Write([]byte(e.SignatureHeaderValue)); err != nil {
			return err
		}

		// Step 5. "headerLength bytes holding the signed headers, the canonical serialization (Section 3.4) of the CBOR representation of the request and response headers of the exchange represented by the application/signed-exchange resource (Section 3.2), excluding the Signature header field." [spec text]
		if _, err := io.Copy(w, &headerBuf); err != nil {
			return err
		}

		// Step 6. "The payload body (Section 3.3 of [RFC7230]) of the exchange represented by the application/signed-exchange resource." [spec text]
		if _, err := w.Write(e.Payload); err != nil {
			return err
		}

	case version.Version1b2, version.Version1b3:
		// draft-yasskin-http-origin-signed-responses.html#rfc.section.5.3

		// "1. 8 bytes consisting of the ASCII characters “sxg1” followed by 4 0x00 bytes, to serve as a file signature. This is redundant with the MIME type, and recipients that receive both MUST check that they match and stop parsing if they don’t.
		// Note: RFC EDITOR PLEASE DELETE THIS NOTE; The implementation of the final RFC MUST use this file signature, but implementations of drafts MUST NOT use it and MUST use another implementation-specific 8-byte string beginning with “sxg1-“." [spec text]
		if _, err := w.Write(e.Version.HeaderMagicBytes()); err != nil {
			return err
		}

		// "2. 2 bytes storing a big-endian integer fallbackUrlLength." [spec text]
		urlLength, err := bigendian.EncodeBytesUint(int64(len(e.RequestURI)), 2)
		if err != nil {
			return err
		}
		if _, err := w.Write(urlLength); err != nil {
			return err
		}

		// "3. fallbackUrlLength bytes holding a fallbackUrl, which MUST be an absolute URL with a scheme of “https”.
		// Note: The byte location of the fallback URL is intended to remain invariant across versions of the application/signed-exchange format so that parsers encountering unknown versions can always find a URL to redirect to." [spec text]
		if _, err := w.Write([]byte(e.RequestURI)); err != nil {
			return err
		}

		const (
			maxSignatureHeaderValueLen = 16 * 1024
			maxHeaderLen               = 512 * 1024
		)
		// "4. 3 bytes storing a big-endian integer sigLength. If this is larger than 16384 (16*1024), parsing MUST fail." [spec text]
		if len(e.SignatureHeaderValue) > maxSignatureHeaderValueLen {
			return fmt.Errorf("signedexchange: sigLength must <= %d but %d", maxSignatureHeaderValueLen, len(e.SignatureHeaderValue))
		}

		encodedSigLength, err := bigendian.EncodeBytesUint(int64(len(e.SignatureHeaderValue)), 3)
		if err != nil {
			return err
		}
		if _, err := w.Write(encodedSigLength); err != nil {
			return err
		}

		// "5. 3 bytes storing a big-endian integer headerLength. If this is larger than 524288 (512*1024), parsing MUST fail." [spec text]
		if headerLength > maxHeaderLen {
			return fmt.Errorf("signedexchange: headerLength must <= %d but %d", maxHeaderLen, headerLength)
		}
		encodedHeaderLength, err := bigendian.EncodeBytesUint(int64(headerLength), 3)
		if err != nil {
			return err
		}
		if _, err := w.Write(encodedHeaderLength); err != nil {
			return err
		}

		// "6. sigLength bytes holding the Signature header field’s value (Section 3.1)." [spec text]
		if _, err := w.Write([]byte(e.SignatureHeaderValue)); err != nil {
			return err
		}

		// "7. headerLength bytes holding signedHeaders, the canonical serialization (Section 3.4) of the CBOR representation of the request and response headers of the exchange represented by the application/signed-exchange resource (Section 3.2), excluding the Signature header field." [spec text]
		if _, err := io.Copy(w, &headerBuf); err != nil {
			return err
		}

		// "8. The payload body (Section 3.3 of [RFC7230]) of the exchange represented by the application/signed-exchange resource.
		// Note that the use of the payload body here means that a Transfer-Encoding header field inside the application/signed-exchange header block has no effect. A Transfer-Encoding header field on the outer HTTP response that transfers this resource still has its normal effect." [spec text]
		if _, err := w.Write(e.Payload); err != nil {
			return err
		}

	default:
		panic("not reached")
	}

	return nil
}

// draft-yasskin-http-origin-signed-responses.html#application-http-exchange
func ReadExchange(r io.Reader) (*Exchange, error) {
	// Step 1. "8 bytes consisting of the ASCII characters “sxg1” followed by 4 0x00 bytes, to serve as a file signature. This is redundant with the MIME type, and recipients that receive both MUST check that they match and stop parsing if they don’t." [spec text]
	// "Note: RFC EDITOR PLEASE DELETE THIS NOTE; The implementation of the final RFC MUST use this file signature, but implementations of drafts MUST NOT use it and MUST use another implementation-specific 8-byte string beginning with “sxg1-“." [spec text]
	magic := make([]byte, version.HeaderMagicBytesLen)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, err
	}
	ver, err := version.FromMagicBytes(magic)
	if err != nil {
		return nil, err
	}

	e := &Exchange{
		Version:         ver,
		RequestHeaders:  http.Header{},
		ResponseHeaders: http.Header{},
	}

	if ver != version.Version1b1 {
		var fallbackUrlLength uint16
		// Step 2. "2 bytes storing a big-endian integer fallbackUrlLength." [spec text]
		if err := binary.Read(r, binary.BigEndian, &fallbackUrlLength); err != nil {
			return nil, err
		}
		// Step 3. "fallbackUrlLength bytes holding a fallbackUrl, which MUST be an absolute URL with a scheme of “https”." [spec text]
		// "Note: The byte location of the fallback URL is intended to remain invariant across versions of the application/signed-exchange format so that parsers encountering unknown versions can always find a URL to redirect to." [spec text]
		fallbackUrl := make([]byte, fallbackUrlLength)
		if _, err := io.ReadFull(r, fallbackUrl); err != nil {
			return nil, err
		}
		var err error
		e.RequestURI, err = validateFallbackURL(fallbackUrl)
		if err != nil {
			return nil, err
		}
	}

	// Step 4. "3 bytes storing a big-endian integer sigLength. If this is larger than 16384 (16*1024), parsing MUST fail." [spec text]
	sigLengthBytes := [3]byte{}
	if _, err := io.ReadFull(r, sigLengthBytes[:]); err != nil {
		return nil, err
	}
	sigLength := bigendian.Decode3BytesUint(sigLengthBytes)

	// Step 5. "3 bytes storing a big-endian integer headerLength. If this is larger than 524288 (512*1024), parsing MUST fail." [spec text]
	headerLengthBytes := [3]byte{}
	if _, err := io.ReadFull(r, headerLengthBytes[:]); err != nil {
		return nil, err
	}
	headerLength := bigendian.Decode3BytesUint(headerLengthBytes)

	// Step 6. "sigLength bytes holding the Signature header field’s value (Section 3.1)." [spec text]
	sig := make([]byte, sigLength)
	if _, err := io.ReadFull(r, sig); err != nil {
		return nil, err
	}
	e.SignatureHeaderValue = string(sig)

	// Step 7. "headerLength bytes holding signedHeaders, the canonical serialization (Section 3.4) of the CBOR representation of the request and response headers of the exchange represented by the application/signed-exchange resource (Section 3.2), excluding the Signature header field." [spec text]
	encodedHeader := make([]byte, headerLength)
	if _, err := io.ReadFull(r, encodedHeader); err != nil {
		return nil, err
	}

	dec := cbor.NewDecoder(bytes.NewReader(encodedHeader))
	if err := e.decodeExchangeHeaders(dec); err != nil {
		return nil, err
	}

	// Step 8. "The payload body (Section 3.3 of [RFC7230]) of the exchange represented by the application/signed-exchange resource." [spec text]
	// "Note that the use of the payload body here means that a Transfer-Encoding header field inside the application/signed-exchange header block has no effect. A Transfer-Encoding header field on the outer HTTP response that transfers this resource still has its normal effect." [spec text]
	e.Payload, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func validateFallbackURL(urlBytes []byte) (string, error) {
	// draft-yasskin-http-origin-signed-responses.html#application-signed-exchange
	// Step 3. "fallbackUrlLength bytes holding a fallbackUrl, which MUST be an absolute URL with a scheme of “https”. " [spec text]
	urlStr := string(urlBytes)
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("signedexchange: cannot parse fallback URL %q: %v", urlStr, err)
	}
	if parsedUrl.Scheme != "https" { // This also ensures that parsedUrl is absolute.
		return "", fmt.Errorf("signedexchange: non-https fallback URL: %q", urlStr)
	}
	return urlStr, nil
}

func (e *Exchange) DumpSignedMessage(w io.Writer, s *Signer) error {
	bs, err := serializeSignedMessage(e, calculateCertSha256(s.Certs), s.ValidityUrl.String(), s.Date.Unix(), s.Expires.Unix())
	if err != nil {
		return err
	}

	if _, err := w.Write(bs); err != nil {
		return err
	}
	return nil
}

func (e *Exchange) PrettyPrintHeaders(w io.Writer) {
	fmt.Fprintf(w, "format version: %s\n", e.Version)
	fmt.Fprintln(w, "request:")
	fmt.Fprintf(w, "  method: %s\n", e.RequestMethod)
	fmt.Fprintf(w, "  uri: %s\n", e.RequestURI)
	fmt.Fprintln(w, "  headers:")
	for k := range e.RequestHeaders {
		fmt.Fprintf(w, "    %s: %s\n", k, e.RequestHeaders.Get(k))
	}
	fmt.Fprintln(w, "response:")
	fmt.Fprintf(w, "  status: %d\n", e.ResponseStatus)
	fmt.Fprintln(w, "  headers:")
	for k := range e.ResponseHeaders {
		fmt.Fprintf(w, "    %s: %s\n", k, e.ResponseHeaders.Get(k))
	}
	fmt.Fprintf(w, "signature: %s\n", e.SignatureHeaderValue)
}

func (e *Exchange) PrettyPrintPayload(w io.Writer) {
	fmt.Fprintf(w, "payload [%d bytes]:\n", len(e.Payload))
	w.Write(e.Payload)
}
