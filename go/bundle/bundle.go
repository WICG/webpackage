package bundle

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

var HeaderMagicBytes = []byte{0x84, 0x48, 0xf0, 0x9f, 0x8c, 0x90, 0xf0, 0x9f, 0x93, 0xa6}

type Request struct {
	*url.URL
	http.Header
}

type Response struct {
	Status int
	http.Header
	Body []byte
}

func (r Response) String() string {
	return fmt.Sprintf("{Status: %q, Header: %v, body: %v}", r.Status, r.Header, string(r.Body))
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

func (r Response) EncodeHeader() ([]byte, error) {
	var b bytes.Buffer
	enc := cbor.NewEncoder(&b)

	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeByteString([]byte(":status"))
			valueE.EncodeByteString([]byte(strconv.Itoa(r.Status)))
		}),
	}
	for name, value := range r.Header {
		mes = append(mes,
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeByteString([]byte(strings.ToLower(name)))
				valueE.EncodeByteString([]byte(normalizeHeaderValues(value)))
			}))
	}
	if err := enc.EncodeMap(mes); err != nil {
		return nil, fmt.Errorf("bundle: Failed to encode response header: %v", err)
	}
	return b.Bytes(), nil
}

type Exchange struct {
	Request
	Response
}

func (e *Exchange) Dump(w io.Writer, dumpContentText bool) error {
	if _, err := fmt.Fprintf(w, "> :url: %v\n", e.Request.URL); err != nil {
		return err
	}
	for k, v := range e.Request.Header {
		if _, err := fmt.Fprintf(w, "> %v: %v\n", k, v); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "< :status: %v\n", e.Response.Status); err != nil {
		return err
	}
	for k, v := range e.Response.Header {
		if _, err := fmt.Fprintf(w, "< %v: %v\n", k, v); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "< [len(Body)]: %d\n", len(e.Response.Body)); err != nil {
		return err
	}
	if dumpContentText {
		ctype := e.Response.Header.Get("content-type")
		if strings.Contains(ctype, "text") {
			if _, err := fmt.Fprint(w, string(e.Response.Body)); err != nil {
				return err
			}
			if _, err := fmt.Fprint(w, "\n"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprint(w, "[non-text body]\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

type Bundle struct {
	Exchanges   []*Exchange
	ManifestURL *url.URL
}

var _ = io.WriterTo(&Bundle{})

type requestEntry struct {
	Request
	Length uint64
}

func (r requestEntry) String() string {
	return fmt.Sprintf("{URL: %v, Header: %v, Length: %d}", r.URL, r.Header, r.Length)
}

type requestEntryWithOffset struct {
	Request
	Length uint64
	Offset uint64
}

type section interface {
	Name() string
	Len() int
	io.WriterTo
}

// staging area for writing index section
type indexSection struct {
	es    []requestEntry
	bytes []byte
}

func (is *indexSection) addRequest(r Request, length int) error {
	ent := requestEntry{
		Request: r,
		Length:  uint64(length),
	}
	is.es = append(is.es, ent)
	return nil
}

func (is *indexSection) Finalize() error {
	if is.bytes != nil {
		panic("indexSection must be Finalize()-d only once.")
	}

	var b bytes.Buffer
	enc := cbor.NewEncoder(&b)
	if err := enc.EncodeArrayHeader(len(is.es) * 2); err != nil {
		return err
	}

	for _, e := range is.es {
		mes := []*cbor.MapEntryEncoder{
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				if err := keyE.EncodeByteString([]byte(":method")); err != nil {
					panic(err)
				}
				if err := valueE.EncodeByteString([]byte("GET")); err != nil {
					panic(err)
				}
			}),
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				if err := keyE.EncodeByteString([]byte(":url")); err != nil {
					panic(err)
				}
				if err := valueE.EncodeByteString([]byte(e.URL.String())); err != nil {
					panic(err)
				}
			}),
		}
		h := e.Header
		for name, _ := range h {
			lname := strings.ToLower(name)
			value := h.Get(name)
			me := cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				if err := keyE.EncodeByteString([]byte(lname)); err != nil {
					panic(err)
				}
				if err := valueE.EncodeByteString([]byte(value)); err != nil {
					panic(err)
				}
			})
			mes = append(mes, me)
		}

		if err := enc.EncodeMap(mes); err != nil {
			return err
		}
		if err := enc.EncodeUint(uint64(e.Length)); err != nil {
			return err
		}
	}

	is.bytes = b.Bytes()
	return nil
}

func (is *indexSection) Name() string {
	return "index"
}

func (is *indexSection) Len() int {
	if is.bytes == nil {
		panic("indexSection must be Finalize()-d before calling Len()")
	}
	return len(is.bytes)
}

func (is *indexSection) WriteTo(w io.Writer) (int64, error) {
	if is.bytes == nil {
		panic("indexSection must be Finalize()-d before calling Bytes()")
	}
	n, err := w.Write(is.bytes)
	return int64(n), err
}

// staging area for writing responses section
type responsesSection struct {
	buf bytes.Buffer
}

func newResponsesSection(n int) *responsesSection {
	ret := &responsesSection{}

	enc := cbor.NewEncoder(&ret.buf)
	if err := enc.EncodeArrayHeader(n); err != nil {
		panic(err)
	}

	return ret
}

func (rs *responsesSection) addResponse(r Response) (int, error) {
	offset := rs.buf.Len()

	headerCbor, err := r.EncodeHeader()
	if err != nil {
		return 0, err
	}

	enc := cbor.NewEncoder(&rs.buf)
	if err := enc.EncodeArrayHeader(2); err != nil {
		return 0, fmt.Errorf("bundle: failed to encode response array header: %v", err)
	}
	if err := enc.EncodeByteString(headerCbor); err != nil {
		return 0, fmt.Errorf("bundle: failed to encode response header cbor bytestring: %v", err)
	}
	if err := enc.EncodeByteString(r.Body); err != nil {
		return 0, fmt.Errorf("bundle: failed to encode response payload bytestring: %v", err)
	}

	length := rs.buf.Len() - offset
	return length, nil
}

func (rs *responsesSection) Name() string { return "responses" }
func (rs *responsesSection) Len() int     { return rs.buf.Len() }
func (rs *responsesSection) WriteTo(w io.Writer) (int64, error) {
	return rs.buf.WriteTo(w)
}

type manifestSection struct {
	bytes.Buffer
}

func (ms *manifestSection) Name() string { return "manifest" }

func newManifestSection(url *url.URL) (*manifestSection, error) {
	var ms manifestSection
	enc := cbor.NewEncoder(&ms)
	if err := enc.EncodeTextString(url.String()); err != nil {
		return nil, err
	}
	return &ms, nil
}

func addExchange(is *indexSection, rs *responsesSection, e *Exchange) error {
	length, err := rs.addResponse(e.Response)
	if err != nil {
		return err
	}

	if err := is.addRequest(e.Request, length); err != nil {
		return err
	}
	return nil
}

type sectionOffset struct {
	Name   string
	Length uint64
}

func FindSection(sos []sectionOffset, name string) (sectionOffset, uint64, bool) {
	offset := uint64(0)
	for _, e := range sos {
		if name == e.Name {
			return e, offset, true
		}
		offset += e.Length
	}
	return sectionOffset{}, 0, false
}

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#load-metadata
// Steps 3-7.
func writeSectionOffsets(w io.Writer, sections []section) error {
	var b bytes.Buffer
	nenc := cbor.NewEncoder(&b)
	if err := nenc.EncodeArrayHeader(len(sections) * 2); err != nil {
		return err
	}
	for _, s := range sections {
		if err := nenc.EncodeTextString(s.Name()); err != nil {
			return err
		}
		if err := nenc.EncodeUint(uint64(s.Len())); err != nil {
			return err
		}
	}

	enc := cbor.NewEncoder(w)
	if err := enc.EncodeByteString(b.Bytes()); err != nil {
		return err
	}
	return nil
}

func writeSectionHeader(w io.Writer, numSections int) error {
	enc := cbor.NewEncoder(w)
	return enc.EncodeArrayHeader(numSections)
}

func writeFooter(w io.Writer, offset int) error {
	const footerLength = 9

	bundleSize := uint64(offset) + footerLength

	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, bundleSize); err != nil {
		return err
	}
	if b.Len() != 8 {
		panic("assert")
	}

	enc := cbor.NewEncoder(w)
	if err := enc.EncodeByteString(b.Bytes()); err != nil {
		return err
	}
	return nil
}

type meta struct {
	sectionOffsets []sectionOffset
	sectionsStart  uint64
	manifestURL    *url.URL
	requests       []requestEntryWithOffset
}

func decodeSectionLengthsCBOR(bs []byte) ([]sectionOffset, error) {
	// section-lengths = [* (section-name: tstr, length: uint) ],

	sos := []sectionOffset{}
	dec := cbor.NewDecoder(bytes.NewBuffer(bs))

	n, err := dec.DecodeArrayHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle: Failed to decode sectionOffset array header: %v", err)
	}

	for i := uint64(0); i < n; i += 2 {
		name, err := dec.DecodeTextString()
		if err != nil {
			return nil, fmt.Errorf("bundle.sectionLengths[%d]: Failed to decode sectionOffset name: %v", i, err)
		}

		// Step 14.2 "If sectionOffsets["name"] exists, return an error. That is, duplicate sections are forbidden" [spec text]
		if _, _, exists := FindSection(sos, name); exists {
			return nil, fmt.Errorf("bundle.sectionLengths[%d]: Duplicate section in sectionOffset array: %q", i, name)
		}

		length, err := dec.DecodeUint()
		if err != nil {
			return nil, fmt.Errorf("bundle.sectionLengths[%d]: Failed to decode sectionOffset[%q].length: %v", i, name, err)
		}

		sos = append(sos, sectionOffset{Name: name, Length: length})
	}

	return sos, nil
}

var reIsAscii = regexp.MustCompile("^[[:ascii:]]*$")

func isAscii(s string) bool {
	return reIsAscii.MatchString(s)
}

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#cbor-headers
func decodeCborHeaders(dec *cbor.Decoder) (http.Header, map[string]string, error) {
	// Step 1. "If item doesn’t match the headers rule in the above CDDL, return an error." [spec text]
	n, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to decode request headers map header: %v", err)
	}

	// Step 2. "Let headers be a new header list ([FETCH])." [spec text]
	headers := make(http.Header)

	// Step 3. "Let pseudos be an empty map ([INFRA])." [spec text]
	pseudos := make(map[string]string)

	// Step 4. "For each pair name/value in item:" [spec text]
	for j := uint64(0); j < n; j++ {
		namebs, err := dec.DecodeByteString()
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to decode request headers map key: %v", err)
		}
		valuebs, err := dec.DecodeByteString()
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to decode request headers map value: %v", err)
		}

		name := string(namebs)
		value := string(valuebs)
		if !isAscii(name) {
			return nil, nil, fmt.Errorf("Failed to decode request headers map key: non-ascii %q", name)
		}
		if !isAscii(value) {
			return nil, nil, fmt.Errorf("Failed to decode request headers map value: non-ascii %q", value)
		}

		// Step 4.1. "If name contains any upper-case or non-ASCII characters, return an error. This matches the requirement in Section 8.1.2 of [RFC7540]." [spec text]
		if strings.ToLower(name) != name {
			return nil, nil, fmt.Errorf("Failed to decode request headers map key: %q contains upper-case.", name)
		}

		// Step 4.2. "If name starts with a ':':" [spec text]
		if strings.HasPrefix(name, ":") {
			// Step 4.2.1. "Assert: pseudos[name] does not exist, because CBOR maps cannot contain duplicate keys." [spec text]
			if _, exists := pseudos[name]; exists {
				return nil, nil, fmt.Errorf("Failed to decode request headers map entry. Pseudo %q appeared twice.", name)
			}

			// Step 4.2.2. "Set pseudos[name] to value." [spec text]
			pseudos[name] = value

			// Step 4.2.3. "Continue." [spec text]
			continue
		}

		// Step 4.3. "If name or value doesn't satisfy the requirements for a header in [FETCH], return an error."
		// TODO: Implement this

		// Step 4.4. "Assert: headers does not contain ([FETCH]) name, because CBOR maps cannot contain duplicate keys and an earlier step rejected upper-case bytes." [spec text]
		if _, exists := headers[name]; exists {
			return nil, nil, fmt.Errorf("Failed to decode request headers map entry. Header %q appeared twice.", name)
		}

		// Step 4.5. "Append name/value to headers." [spec text]
		headers.Set(name, value)
	}

	// Step 5. "Return headers/pseudos." [spec text]
	return headers, pseudos, nil
}

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#index-section
// "To parse the index section, given its sectionContents, the sectionsStart offset, the sectionOffsets CBOR item, and the metadata map to fill in, the parser MUST do the following:" [spec text]
func parseIndexSection(sectionContents []byte, sectionsStart uint64, sos []sectionOffset, bs []byte) ([]requestEntryWithOffset, error) {
	// Step 1. "Let index be the result of parsing sectionContents as a CBOR item matching the index rule in the above CDDL (Section 3.4). If index is an error, return nil, an error." [spec text]
	idxdec := cbor.NewDecoder(bytes.NewBuffer(sectionContents))
	nidx, err := idxdec.DecodeArrayHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle.index: Failed to decode \"index\" array header: %v", err)
	}

	// Step 2. "Check that the responses array has the right number of items:" [spec text]
	// Step 2.1. "Seek to offset sectionOffsets["responses"].offset in stream. If this fails, return an error." [spec text]
	respso, respSectionRelOffset, found := FindSection(sos, "responses")
	if !found {
		return nil, fmt.Errorf("bundle.index: \"responses\" section not found")
	}
	respSectionOffset := sectionsStart + respSectionRelOffset

	// Step 2.2. "Let (responsesType, numResponses) be the result of parsing the type and argument of a CBOR item from the stream. If this returns an error, return that error." [spec text]
	respb := bytes.NewBuffer(bs[respSectionOffset : respSectionOffset+respso.Length])
	respdec := cbor.NewDecoder(respb)
	// Step 2.3. "If responsesType is not 4 (a CBOR array) or ..." [spec text]
	nresp, err := respdec.DecodeArrayHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle.index: Failed to decode \"response\" array header: %v", err)
	}
	// "numResponses is not half of the length of index, return an error." [spec text
	if nresp*2 != nidx {
		return nil, fmt.Errorf("bundle.index: numResponses=%d is not half of the length of index=%d", nresp, nidx)
	}

	// Step 2.4. "Let currentOffset be the current offset within stream minus sectionOffsets["responses"].offset" [spec text]
	currentOffset := respso.Length - uint64(respb.Len())

	// Step 3. "Let requests be an initially-empty map from HTTP requests to structs with items named "offset" and "length"." [spec text]
	requests := []requestEntryWithOffset{}

	// Step 4. "For each (cbor-http-request, length) pair of adjacent elements in index:" [spec text]
	for i := uint64(0); i < nresp; i++ {
		// Step 4.1. "Let headers/pseudos be the result of converting cbor-http-request to a header list and pseudoheaders using the algorithm in Section 3.5. If this returns an error, return nil, that error."
		headers, pseudos, err := decodeCborHeaders(idxdec)
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: %v", i, err)
		}

		// parse length.
		length, err := idxdec.DecodeUint()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode encoded response length: %v", i, err)
		}

		// Step 4.2. "If pseudos does not have keys named ':method' and ':url', or its size isn't 2, return nil, an error." [spec text]
		method, exists := pseudos[":method"]
		if !exists {
			return nil, fmt.Errorf("bundle.index[%d]: The pseudo map must have key named \":method\"", i)
		}
		rawurl, exists := pseudos[":url"]
		if !exists {
			return nil, fmt.Errorf("bundle.index[%d]: The pseudo map must have key named \":url\"", i)
		}
		if len(pseudos) != 2 {
			return nil, fmt.Errorf("bundle.index[%d]: The size of pseudo map must be 2", i)
		}

		// Step 4.3. "If pseudos[':method'] is not 'GET', return nil, an error." [spec text]
		if method != "GET" {
			return nil, fmt.Errorf("bundle.index[%d]: pseudo[\":method\"] must be \"GET\"", i)
		}

		// Step 4.4. "Let parsedUrl be the result of parsing ([URL]) pseudos[':url'] with no base URL." [spec text]
		parsedUrl, err := url.Parse(rawurl)

		// Step 4.5. "If parsedUrl is a failure, its fragment is not null, or it includes credentials, return nil, an error."
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to parse URL: %v", i, err)
		}
		if parsedUrl.Fragment != "" {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains fragment: %q", i, rawurl)
		}
		if parsedUrl.User != nil {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains credentials: %q", i, rawurl)
		}

		// Step 4.6 appears later.

		// Step 4.7. "Let responseOffset be sectionOffsets["responses"].offset + currentOffset." [spec text]
		responseOffset := respSectionOffset + currentOffset

		// Step 4.8. "If currentOffset + length is greater than sectionOffsets["responses"].length, return nil, an error." [spec text]
		if currentOffset+length > respso.Length {
			return nil, fmt.Errorf("bundle.index[%d]: responses length out-of-range", i)
		}

		// Step 4.6. "Let http-request be a new request ([FETCH]) whose:..." [spec text]
		// Step 4.9. "Set requests[http-request] to a struct ..." [spec text]
		e := requestEntryWithOffset{
			// Step 4.6. cont "... method is pseudos[':method'], ..." => omitted, since this must be always "GET"
			Request: Request{
				// "... url is parsedUrl, ... "
				URL: parsedUrl,
				// "... header list is headers, and ..."
				Header: headers,
			},

			// "... client is null." => not impl

			// Step 4.9. cont "... whose "offset" item is responseOffset and ..."
			Offset: responseOffset,
			// "... whose "length" item is length." [spec text]
			Length: length,
		}
		requests = append(requests, e)

		// Step 4.10. "Set currentOffset to currentOffset + length".
		currentOffset += length
	}

	// Step 5. "Set metadata["requests"] to requests." [spec text]
	return requests, nil
}

// https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#manifest-section
// "To parse the manifest section, given its sectionContents and the metadata map to fill in, the parser MUST do the following:" [spec text]
func parseManifestSection(sectionContents []byte) (*url.URL, error) {
	// Step 1. "Let urlString be the result of parsing sectionContents as a CBOR item matching the above manifest rule (Section 3.5). If urlString is an error, return that error." [spec text]
	dec := cbor.NewDecoder(bytes.NewBuffer(sectionContents))
	urlString, err := dec.DecodeTextString()
	if err != nil {
		return nil, fmt.Errorf("bundle: failed to parse manifest section: %v", err)
	}
	// Step 2. "Let url be the result of parsing ([URL]) urlString with no base URL." [spec text]
	manifestURL, err := url.Parse(urlString)
	// Step 3. "If url is a failure, its fragment is not null, or it includes credentials, return an error." [spec text]
	if err != nil {
		return nil, fmt.Errorf("bundle: failed to parse manifest URL (%s): %v", urlString, err)
	}
	if !manifestURL.IsAbs() || manifestURL.Fragment != "" || manifestURL.User != nil {
		return nil, fmt.Errorf("bundle: manifest URL (%s) must be an absolute url without fragment or credentials.", urlString)
	}
	// Step 4. "Set metadata["manifest"] to url." [spec text]
	return manifestURL, nil
}

var knownSections = map[string]struct{}{
	"index":     struct{}{},
	"manifest":  struct{}{},
	"responses": struct{}{},
}

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#load-metadata
func loadMetadata(bs []byte) (*meta, error) {
	// Step 1. "Seek to offset 0 in stream. Assert: this operation doesn't fail." [spec text]

	r := bytes.NewBuffer(bs)

	// Step 2. "If reading 10 bytes from stream returns an error or doesn't return the bytes with hex encoding "84 48 F0 9F 8C 90 F0 9F 93 A6" (the CBOR encoding of the 4-item array initial byte and 8-byte bytestring initial byte, followed by 🌐📦 in UTF-8), return an error." [spec text]
	magic := make([]byte, len(HeaderMagicBytes))
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, err
	}
	if bytes.Compare(magic, HeaderMagicBytes) != 0 {
		return nil, errors.New("bundle: Header magic mismatch.")
	}

	// Step 3. "Let sectionLengthsLength be the result of getting the length of the CBOR bytestring header from stream (Section 3.4.2). If this is an error, return that error." [spec text]
	// Step 4. "If sectionLengthsLength is TBD or greater, return an error." [spec text]
	// TODO(kouhei): Not Implemented
	// Step 5. "Let sectionLengthsBytes be the result of reading sectionLengthsLength bytes from stream. If sectionLengthsBytes is an error, return that error." [spec text]
	dec := cbor.NewDecoder(r)
	slbytes, err := dec.DecodeByteString()
	if err != nil {
		return nil, fmt.Errorf("bundle: Failed to read sectionLengths byte string: %v", err)
	}

	// Step 6. "Let sectionLengths be the result of parsing one CBOR item (Section 3.4) from sectionLengthsBytes, matching the section-lengths rule in the CDDL ([I-D.ietf-cbor-cddl]) above. If sectionLengths is an error, return an error." [spec text]
	sos, err := decodeSectionLengthsCBOR(slbytes)
	if err != nil {
		return nil, err
	}

	// Step 7. "Let (sectionsType, numSections) be the result of parsing the type and argument of a CBOR item from stream." [spec text]
	numSections, err := dec.DecodeArrayHeader()
	// Step 8. "If sectionsType is not 4 (a CBOR array) or..." [spec text]
	if err != nil {
		return nil, fmt.Errorf("bundle: Failed to read section header.")
	}
	// "numSections is not half of the length of sectionLengths, return an error." [spec text]
	if numSections != uint64(len(sos)) {
		return nil, fmt.Errorf("bundle: Expected %d sections, got %d sections", len(sos), numSections)
	}

	// Step 9. "Let sectionsStart be the current offset within stream" [spec text]
	sectionsStart := uint64(len(bs) - r.Len())

	// Step 10. "Let knownSections be the subset of the Section 6.2 that this client has implemented." [spec text]
	// Step 11. "Let ignoredSections be an empty set." [spec text]

	// Step 12. "Let sectionOffsets be an empty map from section names to (offset, length) pairs. These offsets are relative to the start of stream." [spec text]
	// Note: We store this on "sos"

	// Step 13. "Let currentOffset be sectionsStart"
	// currentOffset := sectionsStart

	// Step 14. "For each ("name", length) pair of adjacent elements in sectionLengths:" [spec text]
	// for _, so := range sos {
	// Step 14.1 "If "name"'s specification in knownSections says not to process other sections, add those sections' names to ignoredSections." [spec text]
	// Not implemented

	// Step 14.2-14.4 implemented inside decodeSectionLengthsCBOR()
	// }

	// Step 15. "If responses section is not last in sectionLengths, return an error." [spec text]
	if len(sos) == 0 || sos[len(sos)-1].Name != "responses" {
		return nil, fmt.Errorf("bundle: Last section is not \"responses\"")
	}

	// Step 16. "Let metadata be an empty map" [spec text]
	// Note: We use a struct rather than a map here.
	meta := &meta{
		sectionOffsets: sos,
		sectionsStart:  sectionsStart,
	}

	offset := sectionsStart

	// Step 17. "For each "name" -> (offset, length) triple in sectionOffsets:" [spec text]
	for _, so := range sos {
		// Step 17.1. "If "name" isn't in knownSections, continue to the next triple." [spec text]
		if _, exists := knownSections[so.Name]; !exists {
			continue
		}
		// Step 17.2. "If "name"'s Metadata field is "No", continue to the next triple." [spec text]
		// Note: the "responses" section is currently the only section with its Metadata field "No".
		if so.Name == "responses" {
			continue
		}
		// Step 17.3. "If "name" is in ignoredSections, continue to the next triple." [spec text]
		// Note: Per discussion in #218, the step 12.3 is not implemented since it is no-op as of now.

		// Step 17.4. "Seek to offset offset in stream. If this fails, return an error." [spec text]
		if uint64(len(bs)) <= offset {
			return nil, fmt.Errorf("bundle: section %q's computed offset %q out-of-range.", so.Name, offset)
		}
		end := offset + so.Length
		if uint64(len(bs)) <= end {
			return nil, fmt.Errorf("bundle: section %q's end %q out-of-range.", so.Name, end)
		}

		// Step 17.5. "Let sectionContents be the result of reading length bytes from stream. If sectionContents is an error, return that error."
		sectionContents := bs[offset:end]
		//log.Printf("Section[%q] stream offset %x end %x", so.Name, offset, end)

		// Step 17.6. "Follow "name"'s specification from knownSections to process the section, passing sectionContents, stream, sectionOffsets, and metadata. If this returns an error, return it." [spec text]
		switch so.Name {
		case "index":
			requests, err := parseIndexSection(sectionContents, sectionsStart, sos, bs)
			if err != nil {
				return nil, err
			}
			meta.requests = requests
		case "manifest":
			manifestURL, err := parseManifestSection(sectionContents)
			if err != nil {
				return nil, err
			}
			meta.manifestURL = manifestURL
		case "responses":
			continue
		default:
			return nil, fmt.Errorf("bundle: unknown section: %q", so.Name)
		}

		offset = end
	}

	// Step 18. If metadata doesn't have entries with keys "requests" and "manifest", return an error.
	// FIXME

	// Step 19. Return metadata.
	return meta, nil
}

var reStatus = regexp.MustCompile("^\\d\\d\\d$")

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#load-response
func loadResponse(req requestEntryWithOffset, bs []byte) (Response, error) {
	// Step 1. "Seek to offset requestMetadata.offset in stream. If this fails, return an error." [spec text]
	r := bytes.NewBuffer(bs[req.Offset : req.Offset+req.Length])

	// Step 2. "Read 1 byte from stream. If this is an error or isn't 0x82, return an error." [spec text]
	b, err := r.ReadByte()
	if err != nil {
		return Response{}, fmt.Errorf("bundle: Failed to read first byte of the encoded response: %v", err)
	}
	if b != 0x82 {
		return Response{}, fmt.Errorf("bundle: The first byte of the encoded response is %x, expected 0x82", b)
	}

	// Step 3. "Let headerLength be the result of getting the length of a CBOR bytestring header from stream (Section 3.4.2). If headerLength is an error, return that error." [spec text]
	// Step 4. "If headerLength is TBD or greater, return an error." [spec text]
	dec := cbor.NewDecoder(r)
	headerCborBytes, err := dec.DecodeByteString()
	if err != nil {
		return Response{}, fmt.Errorf("bundle: Failed to decode response header cbor bytestring: %v", err)
	}

	// Step 5. "Let headerCbor be the result of reading headerLength bytes from stream and parsing a CBOR item from them matching the headers CDDL rule. If either the read or parse returns an error, return that error." [spec text]
	rhdr := bytes.NewBuffer(headerCborBytes)
	dechdr := cbor.NewDecoder(rhdr)
	// Step 6. "Let headers/pseudos be the result of converting cbor-http-request to a header list and pseudoheaders using the algorithm in Section 3.5. If this returns an error, return that error." [spec text]
	headers, pseudos, err := decodeCborHeaders(dechdr)
	if err != nil {
		return Response{}, fmt.Errorf("bundle.response headerCbor: %v", err)
	}

	// Step 7. "If pseudos does not have a key named ':status' or its size isn't 1, return an error." [spec text]
	status, exists := pseudos[":status"]
	if !exists {
		return Response{}, fmt.Errorf("bundle.response headerCbor: pseudos don't have a key named \":status\"")
	}
	if len(pseudos) != 1 {
		return Response{}, fmt.Errorf("bundle.response headerCbor: len(pseudos) is %d, expected to be 1", len(pseudos))
	}

	// Step 8. "If pseudos[':status'] isn't exactly 3 ASCII decimal digits, return an error." [spec text]
	if !reStatus.MatchString(status) {
		return Response{}, fmt.Errorf("bundle.response headerCbor: pseudos['status'] %q invalid", status)
	}

	// Step 9. "Let payloadLength be the result of getting the length of a CBOR bytestring header from stream (Section 3.4.2). If payloadLength is an error, return that error." [spec text]
	// Step 11. "Let body be a new body ([FETCH]) whose stream is a tee’d copy of stream starting at the current offset and ending after payloadLength bytes. [spec text]
	body, err := dec.DecodeByteString()
	if err != nil {
		return Response{}, fmt.Errorf("bundle.response.body: %v", err)
	}

	// Step 10. "If stream.currentOffset + payloadLength != requestMetadata.offset + requestMetadata.length, return an error." [spec text]
	if r.Len() != 0 {
		return Response{}, fmt.Errorf("bundle.response: invalid request stream end")
	}

	nstatus, err := strconv.Atoi(status)
	if err != nil {
		panic(err)
	}

	// Step 12. "Let response be a new response ([FETCH]) whose:" [spec text]
	res := Response{
		// "... Url list is request’s url list, ..." [spec text]
		// URL: req.URL,
		// "... status is pseudos[':status'], ..." [spec text]
		Status: nstatus,
		// "... header list is headers, and ..." [spec text]
		Header: headers,
		// "... body is body." [spec text]
		Body: body,
	}

	// Step 13. "Return response." [spec text]
	return res, nil
}

func Read(r io.Reader) (*Bundle, error) {
	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	m, err := loadMetadata(bs)
	if err != nil {
		return nil, err
	}

	// log.Printf("meta: %+v", m)

	es := []*Exchange{}
	for _, req := range m.requests {
		res, err := loadResponse(req, bs)
		if err != nil {
			return nil, err
		}

		e := &Exchange{
			Request:  req.Request,
			Response: res,
		}
		es = append(es, e)
	}

	b := &Bundle{Exchanges: es, ManifestURL: m.manifestURL}
	return b, nil
}

func (b *Bundle) WriteTo(w io.Writer) (int64, error) {
	cw := NewCountingWriter(w)

	is := &indexSection{}
	rs := newResponsesSection(len(b.Exchanges))

	for _, e := range b.Exchanges {
		if err := addExchange(is, rs, e); err != nil {
			return cw.Written, err
		}
	}
	if err := is.Finalize(); err != nil {
		return cw.Written, err
	}

	sections := []section{}
	sections = append(sections, is)
	if b.ManifestURL != nil {
		ms, err := newManifestSection(b.ManifestURL)
		if err != nil {
			return cw.Written, err
		}
		sections = append(sections, ms)
	}
	sections = append(sections, rs)

	if _, err := cw.Write(HeaderMagicBytes); err != nil {
		return cw.Written, err
	}
	if err := writeSectionOffsets(cw, sections); err != nil {
		return cw.Written, err
	}
	if err := writeSectionHeader(cw, len(sections)); err != nil {
		return cw.Written, err
	}
	for _, s := range sections {
		if _, err := s.WriteTo(cw); err != nil {
			return cw.Written, err
		}
	}
	if err := writeFooter(cw, int(cw.Written)); err != nil {
		return cw.Written, err
	}

	return cw.Written, nil
}
