package signedexchange

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/WICG/webpackage/go/signedexchange/cbor"
	"github.com/WICG/webpackage/go/signedexchange/mice"
)

type Exchange struct {
	// Request
	requestUri     *url.URL
	requestHeaders http.Header

	// Response
	responseStatus       int
	responseHeaders      http.Header
	signatureHeaderValue string

	// Payload
	payload []byte
}

var HeaderMagicBytes = []byte("sxg1-b1\x00")

// https://jyasskin.github.io/webpackage/implementation-draft/draft-yasskin-httpbis-origin-signed-exchanges-impl.html#stateful-headers.
var (
	statefulRequestHeaders = map[string]struct{}{
		"Authorization":       struct{}{},
		"Cookie":              struct{}{},
		"Cookie2":             struct{}{},
		"Proxy-Authorization": struct{}{},
		"Sec-WebSocket-Key":   struct{}{},
	}
	statefulResponseHeaders = map[string]struct{}{
		"Authentication-Control":    struct{}{},
		"Authentication-Info":       struct{}{},
		"Optional-WWW-Authenticate": struct{}{},
		"Proxy-Authenticate":        struct{}{},
		"Proxy-Authentication-Info": struct{}{},
		"Sec-WebSocket-Accept":      struct{}{},
		"Set-Cookie":                struct{}{},
		"Set-Cookie2":               struct{}{},
		"SetProfile":                struct{}{},
		"WWW-Authenticate":          struct{}{},
	}
)

var (
	keyMethod = []byte(":method")
	keyURL    = []byte(":url")
	keyStatus = []byte(":status")

	valueGet = []byte("GET")
)

func NewExchange(uri *url.URL, requestHeaders http.Header, status int, responseHeaders http.Header, payload []byte, miRecordSize int) (*Exchange, error) {
	for h, _ := range statefulRequestHeaders {
		if _, ok := requestHeaders[h]; ok {
			return nil, fmt.Errorf("signedexchange: stateful request header %q can't be captured inside signed exchange", h)
		}
	}
	for h, _ := range statefulResponseHeaders {
		if _, ok := responseHeaders[h]; ok {
			return nil, fmt.Errorf("signedexchange: stateful response header %q can't be captured inside signed exchange", h)
		}
	}

	e := &Exchange{
		requestUri:      uri,
		responseStatus:  status,
		requestHeaders:  requestHeaders,
		responseHeaders: responseHeaders,
	}
	if err := e.miEncode(payload, miRecordSize); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Exchange) miEncode(payload []byte, recordSize int) error {
	var buf bytes.Buffer
	mi, err := mice.Encode(&buf, payload, recordSize)
	if err != nil {
		return err
	}
	e.payload = buf.Bytes()
	e.responseHeaders.Add("Content-Encoding", "mi-sha256")
	e.responseHeaders.Add("MI", mi)
	return nil
}

func (e *Exchange) AddSignatureHeader(s *Signer) error {
	h, err := s.signatureHeaderValue(e)
	if err != nil {
		return err
	}
	e.signatureHeaderValue = h
	return nil
}

func (e *Exchange) encodeRequestCommon(enc *cbor.Encoder) []*cbor.MapEntryEncoder {
	return []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString(keyMethod)
			valueE.EncodeByteString(valueGet)
		}),
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString(keyURL)
			valueE.EncodeByteString([]byte(e.requestUri.String()))
		}),
	}
}

func (e *Exchange) encodeRequest(enc *cbor.Encoder) error {
	mes := e.encodeRequestCommon(enc)
	return enc.EncodeMap(mes)
}

func (e *Exchange) decodeRequest(dec *cbor.Decoder) error {
	nelem, err := dec.DecodeMapHeader()
	if err != nil {
		return err
	}

	for i := uint64(0); i < nelem; i++ {
		key, err := dec.DecodeByteString()
		if err != nil {
			return fmt.Errorf("signedexchange: Failed to decode key bytestring: %s", err)
		}
		value, err := dec.DecodeByteString()
		if err != nil {
			return fmt.Errorf("signedexchange: Failed to decode value bytestring: %s", err)
		}
		// TODO: add key/value str validation?

		if bytes.Equal(key, keyMethod) {
			if !bytes.Equal(value, valueGet) {
				// TODO: Consider alternative to log.Printf to communicate ill-formed signed-exchange
				log.Printf("Request map key %q: Expected %q, got %q", keyMethod, valueGet, value)
			}
		} else if bytes.Equal(key, keyURL) {
			e.requestUri, err = url.Parse(string(value))
			if err != nil {
				// TODO: Consider alternative to log.Printf to communicate ill-formed signed-exchange
				log.Printf("Failed to parse URI: %q", value)
			}
		} else {
			// TODO: dup chk
			e.requestHeaders.Add(string(key), string(value))
		}
	}
	return nil
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

func (e *Exchange) encodeRequestWithHeaders(enc *cbor.Encoder) error {
	mes := e.encodeRequestCommon(enc)
	for name, value := range e.requestHeaders {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeByteString([]byte(strings.ToLower(name)))
				valueE.EncodeByteString([]byte(normalizeHeaderValues(value)))
			}))
	}
	return enc.EncodeMap(mes)
}

func (e *Exchange) encodeResponseHeaders(enc *cbor.Encoder) error {
	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString(keyStatus)
			valueE.EncodeByteString([]byte(strconv.Itoa(e.responseStatus)))
		}),
	}
	for name, value := range e.responseHeaders {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeByteString([]byte(strings.ToLower(name)))
				valueE.EncodeByteString([]byte(normalizeHeaderValues(value)))
			}))
	}
	return enc.EncodeMap(mes)
}

func (e *Exchange) decodeResponseHeaders(dec *cbor.Decoder) error {
	nelem, err := dec.DecodeMapHeader()
	if err != nil {
		return err
	}

	for i := uint64(0); i < nelem; i++ {
		key, err := dec.DecodeByteString()
		if err != nil {
			return fmt.Errorf("signedexchange: Failed to decode key bytestring: %s", err)
		}
		value, err := dec.DecodeByteString()
		if err != nil {
			return fmt.Errorf("signedexchange: Failed to decode value bytestring: %s", err)
		}
		// TODO: add key/value str validation?

		if bytes.Equal(key, keyStatus) {
			// TODO: add value str validation that it only contains [0-9]
			e.responseStatus, err = strconv.Atoi(string(value))
			if err != nil {
				log.Printf("Failed to parse responseStatus: %q", value)
			}
		} else {
			// TODO: dup chk
			e.responseHeaders.Add(string(key), string(value))
		}
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

// draft-yasskin-http-origin-signed-responses.html#application-http-exchange
func WriteExchangeFile(w io.Writer, e *Exchange) error {
	// Step 1. "The ASCII characters "sxg1" followed by a 0 byte, to serve as a file signature. This is redundant with the MIME type, and receipients that receive both MUST check that they match and stop parsing if they don't." [spec text]
	// "Note: RFC EDITOR PLEASE DELETE THIS NOTE; The implementation of the final RFC MUST use this file signature, but implementations of drafts MUST NOT use it and MUST use another implementation-specific string beginning with "sxg1-" and ending with a 0 byte instead." [spec text]
	if _, err := w.Write(HeaderMagicBytes); err != nil {
		return err
	}

	// Step 2. "3 bytes storing a big-endian integer sigLength. If this is larger than TBD, parsing MUST fail." [spec text]
	encodedSigLength, err := Encode3BytesBigEndianUint(len(e.signatureHeaderValue))
	if err != nil {
		return err
	}

	if _, err := w.Write(encodedSigLength[:]); err != nil {
		return err
	}

	// Step 3. "3 bytes storing a big-endian integer headerLength. If this is larger than TBD, parsing MUST fail." [spec text]
	headerBuf := &bytes.Buffer{}
	enc := cbor.NewEncoder(headerBuf)
	if err := e.encodeExchangeHeaders(enc); err != nil {
		return err
	}

	headerLength := headerBuf.Len()
	encodedHeaderLength, err := Encode3BytesBigEndianUint(headerLength)
	if err != nil {
		return err
	}

	if _, err := w.Write(encodedHeaderLength[:]); err != nil {
		return err
	}

	// Step 4. "sigLength bytes holding the Signature header field's value (Section 3.1)." [spec text]
	if _, err := w.Write([]byte(e.signatureHeaderValue)); err != nil {
		return err
	}

	// Step 5. "headerLength bytes holding the signed headers, the canonical serialization (Section 3.4) of the CBOR representation of the request and response headers of the exchange represented by the application/signed-exchange resource (Section 3.2), excluding the Signature header field." [spec text]
	if _, err := io.Copy(w, headerBuf); err != nil {
		return err
	}

	// Step 6. "The payload body (Section 3.3 of [RFC7230]) of the exchange represented by the application/signed-exchange resource." [spec text]
	if _, err := w.Write(e.payload); err != nil {
		return err
	}

	return nil
}

func ReadExchangeFile(r io.Reader) (*Exchange, error) {
	var encodedCborLength [3]byte
	if _, err := io.ReadFull(r, encodedCborLength[:]); err != nil {
		return nil, fmt.Errorf("signedexchange: Failed to read length header")
	}
	cborLength := int(encodedCborLength[0])<<16 |
		int(encodedCborLength[1])<<8 |
		int(encodedCborLength[2])

	cborBytes := make([]byte, cborLength)
	if _, err := io.ReadFull(r, cborBytes); err != nil {
		return nil, fmt.Errorf("signedexchange: Failed to read CBOR header binary")
	}

	buf := bytes.NewBuffer(cborBytes)
	dec := cbor.NewDecoder(buf)
	nelem, err := dec.DecodeArrayHeader()
	if err != nil {
		return nil, fmt.Errorf("signedexchange: Failed to read CBOR header array")
	}
	if nelem != 2 {
		// TODO: Consider alternative to log.Printf to communicate ill-formed signed-exchange
		log.Printf("Expected 2 elements in top-level array, but got %d elements", nelem)
	}

	e := &Exchange{
		requestHeaders:  http.Header{},
		responseHeaders: http.Header{},
	}
	if err := e.decodeRequest(dec); err != nil {
		return nil, fmt.Errorf("signedexchange: Failed to decode request map: %v", err)
	}
	if err := e.decodeResponseHeaders(dec); err != nil {
		return nil, fmt.Errorf("signedexchange: Failed to decode response headers map: %v", err)
	}

	miHeaderValue := e.responseHeaders.Get("mi")
	var payloadBuf bytes.Buffer
	if err := mice.Decode(&payloadBuf, r, miHeaderValue); err != nil {
		return nil, fmt.Errorf("signedexchange: Failed to mice decode payload: %v", err)
	}
	e.payload = payloadBuf.Bytes()

	return e, nil
}

func (e *Exchange) PrettyPrint(w io.Writer) {
	fmt.Fprintln(w, "request:")
	fmt.Fprintf(w, "  uri: %s\n", e.requestUri.String())
	fmt.Fprintln(w, "  headers:")
	for k, _ := range e.requestHeaders {
		fmt.Fprintf(w, "    %s: %s\n", k, e.responseHeaders.Get(k))
	}
	fmt.Fprintln(w, "response:")
	fmt.Fprintf(w, "  status: %d\n", e.responseStatus)
	fmt.Fprintln(w, "  headers:")
	for k, _ := range e.responseHeaders {
		fmt.Fprintf(w, "    %s: %s\n", k, e.responseHeaders.Get(k))
	}
	fmt.Fprintf(w, "payload [%d bytes]:\n", len(e.payload))
	w.Write(e.payload)
}
