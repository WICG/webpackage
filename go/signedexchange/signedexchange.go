package signedexchange

import (
	"bytes"
	"encoding/binary"
	"errors"
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

const HeaderMagicBytesLen = 8

func HeaderMagicBytes(v version.Version) []byte {
	switch v {
	case version.Version1b1:
		return []byte("sxg1-b1\x00")
	case version.Version1b2:
		return []byte("sxg1-b2\x00")
	default:
		panic("not reached")
	}
}

type Exchange struct {
	Version version.Version

	// Request
	RequestURI     *url.URL
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

	valueGet = []byte("GET")
)

func NewExchange(ver version.Version, uri *url.URL, requestHeaders http.Header, status int, responseHeaders http.Header, payload []byte) (*Exchange, error) {
	if uri.Scheme != "https" {
		return nil, fmt.Errorf("signedexchange: The request with non-https scheme %q URI can't be captured inside signed exchange.", uri.Scheme)
	}
	for name := range requestHeaders {
		if IsStatefulRequestHeader(name) {
			return nil, fmt.Errorf("signedexchange: stateful request header %q can't be captured inside signed exchange", name)
		}
	}
	for name := range responseHeaders {
		if IsStatefulResponseHeader(name) {
			return nil, fmt.Errorf("signedexchange: stateful response header %q can't be captured inside signed exchange", name)
		}
	}

	return &Exchange{
		Version:         ver,
		RequestURI:      uri,
		ResponseStatus:  status,
		RequestHeaders:  requestHeaders,
		ResponseHeaders: responseHeaders,
		Payload:         payload,
	}, nil
}

func (e *Exchange) MiEncodePayload(recordSize int) error {
	switch e.Version {
	case version.Version1b1:
		if e.ResponseHeaders.Get("MI-Draft2") != "" {
			return errors.New("signedexchange: payload already MI encoded")
		}
		var buf bytes.Buffer
		mi, err := mice.Encode(&buf, e.Payload, recordSize, e.Version)
		if err != nil {
			return err
		}
		e.Payload = buf.Bytes()
		e.ResponseHeaders.Add("Content-Encoding", "mi-sha256-draft2")
		e.ResponseHeaders.Add("MI-Draft2", mi)

	case version.Version1b2:
		if e.ResponseHeaders.Get("Digest") != "" {
			return errors.New("signedexchange: response already has a Digest header")
		}
		var buf bytes.Buffer
		digest, err := mice.Encode(&buf, e.Payload, recordSize, e.Version)
		if err != nil {
			return err
		}
		e.Payload = buf.Bytes()
		e.ResponseHeaders.Add("Content-Encoding", "mi-sha256-03")
		e.ResponseHeaders.Add("Digest", digest)

	default:
		panic("not reached")
	}

	return nil
}

func (e *Exchange) AddSignatureHeader(s *Signer) error {
	h, err := s.signatureHeaderValue(e, e.Version)
	if err != nil {
		return err
	}
	e.SignatureHeaderValue = h
	return nil
}

func (e *Exchange) encodeRequestCommon(enc *cbor.Encoder) []*cbor.MapEntryEncoder {
	encoders := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString(keyMethod)
			valueE.EncodeByteString(valueGet)
		}),
	}
	if e.Version == version.Version1b1 {
		encoders = append(encoders,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeByteString(keyURL)
				valueE.EncodeByteString([]byte(e.RequestURI.String()))
			}))
	}
	return encoders
}

func (e *Exchange) encodeRequest(enc *cbor.Encoder) error {
	mes := e.encodeRequestCommon(enc)
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

func (e *Exchange) decodeRequest(dec *cbor.Decoder) error {
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
			if !bytes.Equal(value, valueGet) {
				return fmt.Errorf("singedexchange: method must be %q but %q", string(valueGet), value)
			}
		}
		if bytes.Equal(key, keyURL) {
			if e.Version == version.Version1b1 {
				e.RequestURI, err = url.Parse(string(value))
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

func (e *Exchange) encodeResponseHeaders(enc *cbor.Encoder) error {
	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString(keyStatus)
			valueE.EncodeByteString([]byte(strconv.Itoa(e.ResponseStatus)))
		}),
	}
	for name, value := range e.ResponseHeaders {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeByteString([]byte(strings.ToLower(name)))
				valueE.EncodeByteString([]byte(normalizeHeaderValues(value)))
			}))
	}
	return enc.EncodeMap(mes)
}

func (e *Exchange) decodeResponseHeaders(dec *cbor.Decoder) error {
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
	if err := enc.EncodeArrayHeader(2); err != nil {
		return fmt.Errorf("signedexchange: failed to encode top-level array header: %v", err)
	}
	if err := e.encodeRequest(enc); err != nil {
		return err
	}
	if err := e.encodeResponseHeaders(enc); err != nil {
		return err
	}
	return nil
}

func (e *Exchange) DumpExchangeHeaders(w io.Writer) error {
	enc := cbor.NewEncoder(w)
	return e.encodeExchangeHeaders(enc)
}

func (e *Exchange) decodeExchangeHeaders(dec *cbor.Decoder) error {
	n, err := dec.DecodeArrayHeader()
	if err != nil {
		return fmt.Errorf("signedexchange: failed to decode top-level array header: %v", err)
	}
	if n != 2 {
		return fmt.Errorf("singedexchange: length of header array must be 2 but %d", n)
	}
	if err := e.decodeRequest(dec); err != nil {
		return err
	}
	if err := e.decodeResponseHeaders(dec); err != nil {
		return err
	}
	return nil
}

func (e *Exchange) Write(w io.Writer) error {
	headerBuf := &bytes.Buffer{}
	if err := e.DumpExchangeHeaders(headerBuf); err != nil {
		return err
	}
	headerLength := headerBuf.Len()

	switch e.Version {
	case version.Version1b1:
		// draft-yasskin-http-origin-signed-responses.html#application-http-exchange

		// Step 1. "The ASCII characters "sxg1" followed by a 0 byte, to serve as a file signature. This is redundant with the MIME type, and recipients that receive both MUST check that they match and stop parsing if they don't." [spec text]
		// "Note: RFC EDITOR PLEASE DELETE THIS NOTE; The implementation of the final RFC MUST use this file signature, but implementations of drafts MUST NOT use it and MUST use another implementation-specific string beginning with "sxg1-" and ending with a 0 byte instead." [spec text]
		if _, err := w.Write(HeaderMagicBytes(e.Version)); err != nil {
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
		if _, err := io.Copy(w, headerBuf); err != nil {
			return err
		}

		// Step 6. "The payload body (Section 3.3 of [RFC7230]) of the exchange represented by the application/signed-exchange resource." [spec text]
		if _, err := w.Write(e.Payload); err != nil {
			return err
		}

	case version.Version1b2:
		// draft-yasskin-http-origin-signed-responses.html#rfc.section.5.3

		// "1. 8 bytes consisting of the ASCII characters “sxg1” followed by 4 0x00 bytes, to serve as a file signature. This is redundant with the MIME type, and recipients that receive both MUST check that they match and stop parsing if they don’t.
		// Note: RFC EDITOR PLEASE DELETE THIS NOTE; The implementation of the final RFC MUST use this file signature, but implementations of drafts MUST NOT use it and MUST use another implementation-specific 8-byte string beginning with “sxg1-“." [spec text]
		if _, err := w.Write(HeaderMagicBytes(e.Version)); err != nil {
			return err
		}

		// "2. 2 bytes storing a big-endian integer fallbackUrlLength." [spec text]
		url := e.RequestURI.String()
		urlLength, err := bigendian.EncodeBytesUint(int64(len(url)), 2)
		if err != nil {
			return err
		}
		if _, err := w.Write(urlLength); err != nil {
			return err
		}

		// "3. fallbackUrlLength bytes holding a fallbackUrl, which MUST be an absolute URL with a scheme of “https”.
		// Note: The byte location of the fallback URL is intended to remain invariant across versions of the application/signed-exchange format so that parsers encountering unknown versions can always find a URL to redirect to." [spec text]
		if _, err := w.Write([]byte(url)); err != nil {
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
		if _, err := io.Copy(w, headerBuf); err != nil {
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
	magic := make([]byte, HeaderMagicBytesLen)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, err
	}
	var ver version.Version
	if bytes.Equal(magic, HeaderMagicBytes(version.Version1b1)) {
		ver = version.Version1b1
	} else if bytes.Equal(magic, HeaderMagicBytes(version.Version1b2)) {
		ver = version.Version1b2
	} else {
		return nil, fmt.Errorf("singedexchange: wrong magic bytes: %v", magic)
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
		e.RequestURI, err = url.Parse(string(fallbackUrl))
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
	var err error
	e.Payload, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Exchange) DumpSignedMessage(w io.Writer, s *Signer) error {
	bs, err := s.serializeSignedMessage(e, e.Version)
	if err != nil {
		return err
	}

	if _, err := w.Write(bs); err != nil {
		return err
	}
	return nil
}

func (e *Exchange) PrettyPrint(w io.Writer) {
	fmt.Fprintf(w, "format version: %s\n", e.Version)
	fmt.Fprintln(w, "request:")
	fmt.Fprintf(w, "  uri: %s\n", e.RequestURI.String())
	fmt.Fprintln(w, "  headers:")
	for k := range e.RequestHeaders {
		fmt.Fprintf(w, "    %s: %s\n", k, e.ResponseHeaders.Get(k))
	}
	fmt.Fprintln(w, "response:")
	fmt.Fprintf(w, "  status: %d\n", e.ResponseStatus)
	fmt.Fprintln(w, "  headers:")
	for k := range e.ResponseHeaders {
		fmt.Fprintf(w, "    %s: %s\n", k, e.ResponseHeaders.Get(k))
	}
	fmt.Fprintf(w, "signature: %s\n", e.SignatureHeaderValue)
	fmt.Fprintf(w, "payload [%d bytes]:\n", len(e.Payload))
	w.Write(e.Payload)
}
