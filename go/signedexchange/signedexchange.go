package signedexchange

import (
	"bytes"
	"fmt"
	"io"
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
	responseStatus  int
	responseHeaders http.Header

	// Payload
	payload []byte
}

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
	e.responseHeaders.Add("Signature", h)
	return nil
}

func (e *Exchange) encodeRequestCommon(enc *cbor.Encoder) []*cbor.MapEntryEncoder {
	return []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString([]byte(":method"))
			valueE.EncodeByteString([]byte("GET"))
		}),
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString([]byte(":url"))
			valueE.EncodeByteString([]byte(e.requestUri.String()))
		}),
	}
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
			keyE.EncodeByteString([]byte(":status"))
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
	enc := cbor.NewEncoder(w)
	if err := enc.EncodeArrayHeader(7); err != nil {
		return err
	}
	if err := enc.EncodeTextString("htxg"); err != nil {
		return err
	}

	if err := enc.EncodeTextString("request"); err != nil {
		return err
	}
	// FIXME: This may diverge in future.
	if err := e.encodeRequestWithHeaders(enc); err != nil {
		return err
	}

	// FIXME: Support "request payload"

	if err := enc.EncodeTextString("response"); err != nil {
		return err
	}

	if err := e.encodeResponseHeaders(enc); err != nil {
		return err
	}

	if err := enc.EncodeTextString("payload"); err != nil {
		return err
	}
	if err := enc.EncodeByteString(e.payload); err != nil {
		return err
	}

	// FIXME: Support "trailer"

	return nil
}
