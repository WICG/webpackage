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

func NewExchange(uri *url.URL, requestHeaders http.Header, status int, responseHeaders http.Header, payload []byte, miRecordSize int) (*Exchange, error) {
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

func (e *Exchange) parseSignedHeadersHeader() []string {
	unparsed := e.responseHeaders.Get("signed-headers")

	rawks := strings.Split(unparsed, ",")
	ks := make([]string, 0, len(rawks))
	for _, k := range rawks {
		ks = append(ks, strings.Trim(k, "\" "))
	}
	return ks
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

func (e *Exchange) encodeRequestWithHeaders(enc *cbor.Encoder) error {
	mes := e.encodeRequestCommon(enc)
	for name, value := range e.requestHeaders {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeByteString([]byte(strings.ToLower(name)))
				valueE.EncodeByteString([]byte(value[0]))
			}))
	}
	return enc.EncodeMap(mes)
}

func (e *Exchange) encodeResponseHeaders(enc *cbor.Encoder, onlySignedHeaders bool) error {
	// Only encode response headers which are specified in "signed-headers" header.
	var m map[string]struct{}
	if onlySignedHeaders {
		m = map[string]struct{}{}
		ks := e.parseSignedHeadersHeader()
		for _, k := range ks {
			m[k] = struct{}{}
		}
	}

	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString([]byte(":status"))
			valueE.EncodeByteString([]byte(strconv.Itoa(e.responseStatus)))
		}),
	}
	for name, value := range e.responseHeaders {
		if onlySignedHeaders {
			if _, ok := m[strings.ToLower(name)]; !ok {
				continue
			}
		}
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeByteString([]byte(strings.ToLower(name)))
				valueE.EncodeByteString([]byte(value[0]))
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
	if err := e.encodeResponseHeaders(enc, true); err != nil {
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

	if err := e.encodeResponseHeaders(enc, false); err != nil {
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
