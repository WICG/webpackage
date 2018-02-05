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

type Input struct {
	// Request
	requestUri *url.URL

	// Response
	responseStatus int
	responseHeader http.Header

	// Payload
	payload []byte
}

func NewInput(uri *url.URL, status int, headers http.Header, payload []byte, miRecordSize int) (*Input, error) {
	i := &Input{
		requestUri:     uri,
		responseStatus: status,
		responseHeader: headers,
	}
	if err := i.miEncode(payload, miRecordSize); err != nil {
		return nil, err
	}
	return i, nil
}

func (i *Input) miEncode(payload []byte, recordSize int) error {
	var buf bytes.Buffer
	mi, err := mice.Encode(&buf, payload, recordSize)
	if err != nil {
		return err
	}
	i.payload = buf.Bytes()
	i.responseHeader.Add("Content-Encoding", "mi-sha256")
	i.responseHeader.Add("MI", mi)
	return nil
}

// AddSignedHeadersHeader adds 'signed-headers' header to the response.
//
// Signed-Headers is a Structured Header as defined by
// [I-D.ietf-httpbis-header-structure]. Its value MUST be a list (Section 4.8
// of [I-D.ietf-httpbis-header-structure]) of lowercase strings (Section 4.2 of
// [I-D.ietf-httpbis-header-structure]) naming HTTP response header fields.
// Pseudo-header field names (Section 8.1.2.1 of [RFC7540]) MUST NOT appear in
// this list.
func (i *Input) AddSignedHeadersHeader(ks ...string) {
	strs := []string{}
	for _, k := range ks {
		strs = append(strs, fmt.Sprintf(`"%s"`, strings.ToLower(k)))
	}
	s := strings.Join(strs, ", ")
	i.responseHeader.Add("signed-headers", s)
}

func (i *Input) AddSignatureHeader(s *Signer) error {
	h, err := s.signatureHeaderValue(i)
	if err != nil {
		return err
	}
	i.responseHeader.Add("Signature", h)
	return nil
}

func (i *Input) parseSignedHeadersHeader() []string {
	unparsed := i.responseHeader.Get("signed-headers")

	rawks := strings.Split(unparsed, ",")
	ks := make([]string, 0, len(rawks))
	for _, k := range rawks {
		ks = append(ks, strings.Trim(k, "\" "))
	}
	return ks
}

func (i *Input) encodeCanonicalRequest(e *cbor.Encoder) error {
	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString([]byte(":method"))
			valueE.EncodeByteString([]byte("GET"))
		}),
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString([]byte(":url"))
			valueE.EncodeByteString([]byte(i.requestUri.String()))
		}),
	}
	return e.EncodeMap(mes)
}

func (i *Input) encodeResponseHeader(e *cbor.Encoder, onlySignedHeaders bool) error {
	// Only encode response headers which are specified in "signed-headers" header.
	var m map[string]struct{}
	if onlySignedHeaders {
		m = map[string]struct{}{}
		ks := i.parseSignedHeadersHeader()
		for _, k := range ks {
			m[k] = struct{}{}
		}
	}

	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString([]byte(":status"))
			valueE.EncodeByteString([]byte(strconv.Itoa(i.responseStatus)))
		}),
	}
	for name, value := range i.responseHeader {
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
	return e.EncodeMap(mes)
}

// draft-yasskin-http-origin-signed-responses.html#rfc.section.3.4
func (i *Input) encodeCanonicalExchangeHeaders(e *cbor.Encoder) error {
	if err := e.EncodeArrayHeader(2); err != nil {
		return fmt.Errorf("signedexchange: failed to encode top-level array header: %v", err)
	}
	if err := i.encodeCanonicalRequest(e); err != nil {
		return err
	}
	if err := i.encodeResponseHeader(e, true); err != nil {
		return err
	}
	return nil
}

// draft-yasskin-http-origin-signed-responses.html#application-http-exchange
func WriteExchangeFile(w io.Writer, i *Input) error {
	e := cbor.NewEncoder(w)
	if err := e.EncodeArrayHeader(7); err != nil {
		return err
	}
	if err := e.EncodeTextString("htxg"); err != nil {
		return err
	}

	if err := e.EncodeTextString("request"); err != nil {
		return err
	}
	// FIXME: This may diverge in future.
	if err := i.encodeCanonicalRequest(e); err != nil {
		return err
	}

	// FIXME: Support "request payload"

	if err := e.EncodeTextString("response"); err != nil {
		return err
	}

	if err := i.encodeResponseHeader(e, false); err != nil {
		return err
	}

	if err := e.EncodeTextString("payload"); err != nil {
		return err
	}
	if err := e.EncodeByteString(i.payload); err != nil {
		return err
	}

	// FIXME: Support "trailer"

	return nil
}
