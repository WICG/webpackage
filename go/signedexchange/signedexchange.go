package signedexchange

import (
	"bytes"
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
)

type Exchange struct {
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

var HeaderMagicBytes = []byte("sxg1-b1\x00")

func NewExchange(uri *url.URL, requestHeaders http.Header, status int, responseHeaders http.Header, payload []byte) (*Exchange, error) {
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
		RequestURI:      uri,
		ResponseStatus:  status,
		RequestHeaders:  requestHeaders,
		ResponseHeaders: responseHeaders,
		Payload:         payload,
	}, nil
}

func (e *Exchange) MiEncodePayload(recordSize int) error {
	if e.ResponseHeaders.Get("MI-Draft2") != "" {
		return errors.New("Payload already MI encoded.")
	}

	var buf bytes.Buffer
	mi, err := mice.Encode(&buf, e.Payload, recordSize)
	if err != nil {
		return err
	}
	e.Payload = buf.Bytes()
	e.ResponseHeaders.Add("Content-Encoding", "mi-sha256-draft2")
	e.ResponseHeaders.Add("MI-Draft2", mi)
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

func (e *Exchange) encodeRequestCommon(enc *cbor.Encoder) []*cbor.MapEntryEncoder {
	return []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString(keyMethod)
			valueE.EncodeByteString(valueGet)
		}),
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString(keyURL)
			valueE.EncodeByteString([]byte(e.RequestURI.String()))
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
			e.RequestURI, err = url.Parse(string(value))
			if err != nil {
				return err
			}
			continue
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

// draft-yasskin-http-origin-signed-responses.html#application-http-exchange
func (e *Exchange) Write(w io.Writer) error {
	// Step 1. "The ASCII characters "sxg1" followed by a 0 byte, to serve as a file signature. This is redundant with the MIME type, and recipients that receive both MUST check that they match and stop parsing if they don't." [spec text]
	// "Note: RFC EDITOR PLEASE DELETE THIS NOTE; The implementation of the final RFC MUST use this file signature, but implementations of drafts MUST NOT use it and MUST use another implementation-specific string beginning with "sxg1-" and ending with a 0 byte instead." [spec text]
	if _, err := w.Write(HeaderMagicBytes); err != nil {
		return err
	}

	// Step 2. "3 bytes storing a big-endian integer sigLength. If this is larger than TBD, parsing MUST fail." [spec text]
	encodedSigLength, err := bigendian.Encode3BytesUint(len(e.SignatureHeaderValue))
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
	encodedHeaderLength, err := bigendian.Encode3BytesUint(headerLength)
	if err != nil {
		return err
	}

	if _, err := w.Write(encodedHeaderLength[:]); err != nil {
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

	return nil
}

// draft-yasskin-http-origin-signed-responses.html#application-http-exchange
func ReadExchange(r io.Reader) (*Exchange, error) {
	// Step 1. "The ASCII characters “sxg1” followed by a 0 byte, to serve as a file signature. This is redundant with the MIME type, and recipients that receive both MUST check that they match and stop parsing if they don’t." [spec text]
	// "Note: RFC EDITOR PLEASE DELETE THIS NOTE; The implementation of the final RFC MUST use this file signature, but implementations of drafts MUST NOT use it and MUST use another implementation-specific string beginning with “sxg1-“ and ending with a 0 byte instead." [spec text]
	magic := make([]byte, len(HeaderMagicBytes))
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, err
	}
	if !bytes.Equal(magic, HeaderMagicBytes) {
		return nil, fmt.Errorf("singedexchange: wrong magic bytes: %v", magic)
	}

	// Step 2. "3 bytes storing a big-endian integer sigLength. If this is larger than TBD, parsing MUST fail." [spec text]
	sigLengthBytes := [3]byte{}
	if _, err := io.ReadFull(r, sigLengthBytes[:]); err != nil {
		return nil, err
	}
	sigLength := bigendian.Decode3BytesUint(sigLengthBytes)

	// Step 3. "3 bytes storing a big-endian integer headerLength. If this is larger than TBD, parsing MUST fail." [spec text]
	headerLengthBytes := [3]byte{}
	if _, err := io.ReadFull(r, headerLengthBytes[:]); err != nil {
		return nil, err
	}
	headerLength := bigendian.Decode3BytesUint(headerLengthBytes)

	e := &Exchange{
		RequestHeaders:  http.Header{},
		ResponseHeaders: http.Header{},
	}

	// Step 4. "sigLength bytes holding the Signature header field’s value (Section 3.1)." [spec text]
	sig := make([]byte, sigLength)
	if _, err := io.ReadFull(r, sig); err != nil {
		return nil, err
	}
	e.SignatureHeaderValue = string(sig)

	// Step 5. "headerLength bytes holding the signed headers, the canonical serialization (Section 3.4) of the CBOR representation of the request and response headers of the exchange represented by the application/signed-exchange resource (Section 3.2), excluding the Signature header field." [spec text]
	// "Note that this is exactly the bytes used when checking signature validity in Section 3.5." [spec text]
	encodedHeader := make([]byte, headerLength)
	if _, err := io.ReadFull(r, encodedHeader); err != nil {
		return nil, err
	}

	dec := cbor.NewDecoder(bytes.NewReader(encodedHeader))
	if err := e.decodeExchangeHeaders(dec); err != nil {
		return nil, err
	}

	// Step 6. "The payload body (Section 3.3 of [RFC7230]) of the exchange represented by the application/signed-exchange resource." [spec text]
	// "Note that the use of the payload body here means that a Transfer-Encoding header field inside the application/signed-exchange header block has no effect. A Transfer-Encoding header field on the outer HTTP response that transfers this resource still has its normal effect." [spec text]
	var err error
	e.Payload, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Exchange) DumpSignedMessage(w io.Writer, s *Signer) error {
	bs, err := s.serializeSignedMessage(e)
	if err != nil {
		return err
	}

	if _, err := w.Write(bs); err != nil {
		return err
	}
	return nil
}

func (e *Exchange) PrettyPrint(w io.Writer) {
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
