package bundle

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/WICG/webpackage/go/signedexchange"
	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

var HeaderMagicBytes = []byte{0x84, 0x48, 0xf0, 0x9f, 0x8c, 0x90, 0xf0, 0x9f, 0x93, 0xa6}

type Bundle struct {
	Exchanges []*signedexchange.Exchange
}

var _ = io.WriterTo(&Bundle{})

// staging area for writing index section
type indexSection struct {
	mes   []*cbor.MapEntryEncoder
	bytes []byte
}

func (is *indexSection) addExchange(e *signedexchange.Exchange, offset, length int) error {
	me := cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
		if err := e.EncodeRequestWithHeaders(keyE); err != nil {
			panic(err) // fixme
		}
		if err := valueE.EncodeArrayHeader(2); err != nil {
			panic(err)
		}
		if err := valueE.EncodeUInt(uint64(offset)); err != nil {
			panic(err)
		}
		if err := valueE.EncodeUInt(uint64(length)); err != nil {
			panic(err)
		}
	})
	is.mes = append(is.mes, me)
	return nil
}

func (is *indexSection) Finalize() error {
	if is.bytes != nil {
		panic("indexSection must be Finalize()-d only once.")
	}

	var b bytes.Buffer
	enc := cbor.NewEncoder(&b)
	if err := enc.EncodeMap(is.mes); err != nil {
		return err
	}

	is.bytes = b.Bytes()
	return nil
}

func (is *indexSection) Len() int {
	if is.bytes == nil {
		panic("indexSection must be Finalize()-d before calling Len()")
	}
	return len(is.bytes)
}

func (is *indexSection) Bytes() []byte {
	if is.bytes == nil {
		panic("indexSection must be Finalize()-d before calling Bytes()")
	}
	return is.bytes
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

func (rs *responsesSection) addExchange(e *signedexchange.Exchange) (int, int, error) {
	offset := rs.buf.Len()

	var resHdrBuf bytes.Buffer
	if err := signedexchange.WriteResponseHeaders(&resHdrBuf, e); err != nil {
		return 0, 0, err
	}

	enc := cbor.NewEncoder(&rs.buf)
	if err := enc.EncodeArrayHeader(2); err != nil {
		return 0, 0, fmt.Errorf("bundle: failed to encode response array header: %v", err)
	}
	if err := enc.EncodeByteString(resHdrBuf.Bytes()); err != nil {
		return 0, 0, fmt.Errorf("bundle: failed to encode response header cbor bytestring: %v", err)
	}
	if err := enc.EncodeByteString(e.Payload()); err != nil {
		return 0, 0, fmt.Errorf("bundle: failed to encode response payload bytestring: %v", err)
	}

	length := rs.buf.Len() - offset
	return offset, length, nil
}

func (rs *responsesSection) Len() int      { return rs.buf.Len() }
func (rs *responsesSection) Bytes() []byte { return rs.buf.Bytes() }

func addExchange(is *indexSection, rs *responsesSection, e *signedexchange.Exchange) error {
	offset, length, err := rs.addExchange(e)
	if err != nil {
		return err
	}

	if err := is.addExchange(e, offset, length); err != nil {
		return err
	}
	return nil
}

type sectionOffset struct {
	Name   string
	Offset uint64
	Length uint64
}

type sectionOffsets []sectionOffset

func (so *sectionOffsets) AddSectionOrdered(name string, length uint64) {
	offset := uint64(0)
	if len(*so) > 0 {
		last := (*so)[len(*so)-1]
		offset = last.Offset + last.Length
	}
	*so = append(*so, sectionOffset{name, offset, length})
}

func (so *sectionOffsets) FindSection(name string) (sectionOffset, bool) {
	for _, e := range *so {
		if name == e.Name {
			return e, true
		}
	}
	return sectionOffset{}, false
}

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#load-metadata
// Steps 3-7.
func writeSectionOffsets(w io.Writer, so sectionOffsets) error {
	sectionHeaderSize := 1 //fixme

	mes := []*cbor.MapEntryEncoder{}
	for _, e := range so {
		me := cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			// TODO(kouhei): error plumbing
			keyE.EncodeTextString(e.Name)
			valueE.EncodeArrayHeader(2)
			valueE.EncodeUInt(e.Offset + uint64(sectionHeaderSize))
			valueE.EncodeUInt(e.Length)
		})

		mes = append(mes, me)
	}

	var b bytes.Buffer
	nestedEnc := cbor.NewEncoder(&b)
	if err := nestedEnc.EncodeMap(mes); err != nil {
		return err
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

type requestEntry struct {
	*url.URL
	http.Header
	Offset uint64
	Length uint64
}

func (r requestEntry) String() string {
	return fmt.Sprintf("{URL: %v, Header: %v, Offset: %d, Length: %d}", r.URL, r.Header, r.Offset, r.Length)
}

type meta struct {
	sectionOffsets
	sectionsStart uint64
	requests      []requestEntry
}

func decodeSectionOffsetsCBOR(bs []byte) (sectionOffsets, error) {
	// section-offsets = {* tstr => [ offset: uint, length: uint] },

	so := make(sectionOffsets, 0)
	dec := cbor.NewDecoder(bytes.NewBuffer(bs))

	n, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle: Failed to decode sectionOffset map header: %v", err)
	}

	for i := uint64(0); i < n; i++ {
		name, err := dec.DecodeTextString()
		if err != nil {
			return nil, fmt.Errorf("bundle: Failed to decode sectionOffset map key: %v", err)
		}
		if _, exists := so.FindSection(name); exists {
			return nil, fmt.Errorf("bundle: Duplicated section in sectionOffset map: %q", name)
		}

		m, err := dec.DecodeArrayHeader()
		if err != nil {
			return nil, fmt.Errorf("bundle: Failed to decode sectionOffset map value: %v", err)
		}
		if m != 2 {
			return nil, fmt.Errorf("bundle: Failed to decode sectionOffset map value. Array of invalid length %d", m)
		}

		offset, err := dec.DecodeUInt()
		if err != nil {
			return nil, fmt.Errorf("bundle: Failed to decode sectionOffset[%q].offset: %v", name, err)
		}
		length, err := dec.DecodeUInt()
		if err != nil {
			return nil, fmt.Errorf("bundle: Failed to decode sectionOffset[%q].length: %v", name, err)
		}

		so = append(so, sectionOffset{Name: name, Offset: offset, Length: length})
	}

	return so, nil
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
		if !utf8.Valid(namebs) { // FIXME: should be isAscii
			return nil, nil, fmt.Errorf("Failed to decode request headers map key: Invalid UTF8")
		}
		if !utf8.Valid(valuebs) { // FIXME: should be isAscii
			return nil, nil, fmt.Errorf("Failed to decode request headers map value: Invalid UTF8")
		}

		// Step 4.1. "If name contains any upper-case or non-ASCII characters, return an error. This matches the requirement in Section 8.1.2 of [RFC7540]." [spec text]
		if strings.ToLower(name) != name {
			return nil, nil, fmt.Errorf("Failed to decode request headers map key: Invalid UTF8")
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
func parseIndexSection(sectionContents []byte, sectionsStart uint64, sectionOffsets sectionOffsets) ([]requestEntry, error) {
	respso, found := sectionOffsets.FindSection("responses")
	if !found {
		return nil, fmt.Errorf("bundle.index: \"responses\" section not found")
	}

	// Step 1. "Let index be the result of parsing sectionContents as a CBOR item matching the index rule in the above CDDL (Section 3.4). If index is an error, return nil, an error." [spec text]
	dec := cbor.NewDecoder(bytes.NewBuffer(sectionContents))
	n, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle.index: Failed to decode map header: %v", err)
	}

	// Step 2. "Let requests be an initially-empty map from HTTP requests to structs with items named "offset" and "length"." [spec text]

	// Step 3. "For each cbor-http-request/[offset, length] triple in index:" [spec text]
	requests := []requestEntry{}
	for i := uint64(0); i < n; i++ {
		// Step 3.1. "Let headers/pseudos be the result of converting cbor-http-request to a header list and pseudoheaders using the algorithm in Section 3.5. If this returns an error, return nil, that error."
		headers, pseudos, err := decodeCborHeaders(dec)
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: %v", i, err)
		}

		// parse [offset,length]
		m, err := dec.DecodeArrayHeader()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode response byte-range array", i, err)
		}
		if m != 2 {
			return nil, fmt.Errorf("bundle.index[%d]: The response byte-range array must be composed of 2 elements.", i, err)
		}

		offset, err := dec.DecodeUInt()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode response byte-range offset", i, err)
		}
		length, err := dec.DecodeUInt()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode response byte-range length", i, err)
		}

		// Step 3.2. "If pseudos does not have keys named ':method' and ':url', or its size isn't 2, return nil, an error." [spec text]
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

		// Step 3.3. "If pseudos[':method'] is not 'GET', return nil, an error." [spec text]
		if method != "GET" {
			return nil, fmt.Errorf("bundle.index[%d]: pseudo[\":method\"] must be \"GET\"", i)
		}

		// Step 3.4. "Let parsedUrl be the result of parsing ([URL]) pseudos[':url'] with no base URL." [spec text]
		parsedUrl, err := url.Parse(rawurl)

		// Step 3.5. "If parsedUrl is a failure, its fragment is not null, or it includes credentials, return nil, an error."
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to parse URL: %v", i, err)
		}
		if parsedUrl.Fragment != "" {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains fragment: %q", i, rawurl)
		}
		if parsedUrl.User != nil {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains credentials: %q", i, rawurl)
		}

		// Step 3.6 appears later.

		// Step 3.7. "Let streamOffset be sectionsStart + section-offsets["responses"].offset + offset. That is, offsets in the index are relative to the start of the "responses" section." [spec text]
		streamOffset := sectionsStart + respso.Offset + offset

		// Step 3.8. "If offset + length is greater than sectionOffsets["responses"].length, return nil, an error." [spec text]
		if offset+length > respso.Length {
			return nil, fmt.Errorf("bundle.index[%d]: responses length out-of-range")
		}

		// Step 3.6. "Let http-request be a new request ([FETCH]) whose:..." [spec text]
		// Step 3.9. "Set requests[http-request] to a struct ..." [spec text]
		e := requestEntry{
			// "... method is pseudos[':method'], ..." => omitted, since this must be always "GET"
			// "... url is parsedUrl, ... "
			URL: parsedUrl,
			// "... header list is headers, and ..."
			Header: headers,
			// "... client is null." => not impl

			// "whose "offset" item is streamOffset and whose "length" item is length." [spec text]
			Offset: streamOffset,
			Length: length,
		}
		requests = append(requests, e)
	}

	// Step 4. "Set metadata["requests"] to requests." [spec text]
	return requests, nil
}

var knownSections = map[string]struct{}{
	"index":     struct{}{},
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

	// Step 3. "Let sectionOffsetsLength be the result of getting the length of the CBOR bytestring header from stream (Section 3.4.2). If this is an error, return that error." [spec text]
	// Step 4. "If sectionOffsetsLength is TBD or greater, return an error." [spec text]
	// TODO(kouhei): Not Implemented
	// Step 5. "Let sectionOffsetsBytes be the result of reading sectionOffsetsLength bytes from stream. If sectionOffsetsBytes is an error, return that error." [spec text]
	dec := cbor.NewDecoder(r)
	sobytes, err := dec.DecodeByteString()
	if err != nil {
		return nil, fmt.Errorf("bundle: Failed to read sectionOffset byte string: %v", err)
	}

	// Step 6. "Let sectionOffsets be the result of parsing one CBOR item (Section 3.4) from sectionOffsetsBytes, matching the section-offsets rule in the CDDL ([I-D.ietf-cbor-cddl]) above. If sectionOffsets is an error, return an error." [spec text]
	so, err := decodeSectionOffsetsCBOR(sobytes)
	if err != nil {
		return nil, err
	}

	// Step 7. "Let sectionsStart be the current offset within stream. For example, if sectionOffsetsLength were 52, sectionsStart would be 64." [spec text]
	sectionsStart := 12 + uint64(len(sobytes))

	// Step 8. "Let knownSections be the subset of the Section 6.2 that this client has implemented." [spec text]
	// Step 9. "Let ignoredSections be an empty set." [spec text]
	// Step 10. "For each "name" key in sectionOffsets, if "name"'s specification in knownSections says not to process other sections, add those sections' names to ignoredSections." [spec text]
	// Note: Per discussion in #218, the steps 9-10 are not implemented since they are no-ops as of now.

	// Step 11. Let metadata be an empty map
	// Note: We use a struct rather than a map here.
	meta := &meta{
		sectionOffsets: so,
		sectionsStart:  sectionsStart,
	}

	// Step 12. For each "name"/[offset, length] triple in sectionOffsets:
	for _, e := range so {
		// Step 12.1. If "name" isn't in knownSections, continue to the next triple.
		if _, exists := knownSections[e.Name]; !exists {
			continue
		}
		// Step 12.2. If "name"’s Metadata field is "No", continue to the next triple.
		// Note: the "responses" section is currently the only section with its Metadata field "No".
		if e.Name == "responses" {
			continue
		}
		// Step 12.3. If "name" is in ignoredSections, continue to the next triple.
		// Note: Per discussion in #218, the step 12.3 is not implemented since it is no-op as of now.

		// Step 12.4. Seek to offset sectionsStart + offset in stream. If this fails, return an error.
		offset := sectionsStart + e.Offset
		if uint64(len(bs)) <= offset {
			return nil, fmt.Errorf("bundle: section %q's computed offset %q out-of-range.", e.Name, offset)
		}
		end := offset + e.Length
		if uint64(len(bs)) <= end {
			return nil, fmt.Errorf("bundle: section %q's end %q out-of-range.", e.Name, end)
		}

		// Step 12.5. Let sectionContents be the result of reading length bytes from stream. If sectionContents is an error, return that error.
		sectionContents := bs[offset:end]
		//log.Printf("Section[%q] stream offset %x end %x", e.Name, offset, end)

		// Step 12.6. Follow "name"'s specification from knownSections to process the section, passing sectionContents, stream, sectionOffsets, sectionsStart, and metadata. If this returns an error, return it.
		switch e.Name {
		case "index":
			requests, err := parseIndexSection(sectionContents, sectionsStart, so)
			if err != nil {
				return nil, err
			}
			meta.requests = requests
		case "responses":
			// FIXME
		default:
			panic("aaa")
		}
	}

	// Step 13. If metadata doesn't have entries with keys "requests" and "manifest", return an error.

	// Step 14. Return metadata.
	return meta, nil
}

type Response struct {
	*url.URL
	Status string
	http.Header
	Body []byte
}

func (r Response) String() string {
	return fmt.Sprintf("{URL: %v, Status: %q, Header: %v, body: %v}", r.URL, r.Status, r.Header, string(r.Body))
}

var reStatus = regexp.MustCompile("^\\d\\d\\d$")

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#load-response
func loadResponse(req requestEntry, bs []byte) (Response, error) {
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
		return Response{}, fmt.Errorf("bundle: Failed to decode response header cbor bytestring", err)
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

	// Step 12. "Let response be a new response ([FETCH]) whose:" [spec text]
	res := Response{
		// "... Url list is request’s url list, ..." [spec text]
		URL: req.URL,
		// "... status is pseudos[':status'], ..." [spec text]
		Status: status,
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

	log.Printf("meta: %+v", m)

	es := []*signedexchange.Exchange{}
	for _, req := range m.requests {
		res, err := loadResponse(req, bs)
		if err != nil {
			return nil, err
		}
		nstatus, err := strconv.Atoi(res.Status)
		if err != nil {
			panic("Atoi(status) must not fail here, since loadResponse ensures that it is 3digit int")
		}

		e, err := signedexchange.NewExchange(
			req.URL, req.Header,
			nstatus, res.Header, res.Body,
		)
		if err != nil {
			return nil, err
		}
		es = append(es, e)

		// log.Printf("req: %+v", req)
		// log.Printf("res: %+v", res)
		// log.Printf("-------------------------------------------------------")
	}

	b := &Bundle{Exchanges: es}
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

	var so sectionOffsets
	so.AddSectionOrdered("index", uint64(is.Len()))
	so.AddSectionOrdered("responses", uint64(rs.Len()))

	if _, err := cw.Write(HeaderMagicBytes); err != nil {
		return cw.Written, err
	}
	if err := writeSectionOffsets(cw, so); err != nil {
		return cw.Written, err
	}
	if err := writeSectionHeader(cw, len(so)); err != nil {
		return cw.Written, err
	}
	if _, err := cw.Write(is.Bytes()); err != nil {
		return cw.Written, err
	}
	if _, err := cw.Write(rs.Bytes()); err != nil {
		return cw.Written, err
	}
	if err := writeFooter(cw, int(cw.Written)); err != nil {
		return cw.Written, err
	}

	return cw.Written, nil
}
