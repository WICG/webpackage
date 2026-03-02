package bundle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/WICG/webpackage/go/bundle/version"
	"github.com/WICG/webpackage/go/internal/cbor"
	"github.com/WICG/webpackage/go/signedexchange/structuredheader"
)

const maxNumVariantsForSingleURL = 10000

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

func (r Response) HeaderSha256() ([]byte, error) {
	headerBytes, err := r.EncodeHeader()
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(headerBytes)
	return sum[:], nil
}

var _ = io.WriterTo(&Bundle{})

type indexEntry struct {
	Request
	Variants   string
	VariantKey string
	Offset     uint64 // Offset within the responses section
	Length     uint64
}

func (r indexEntry) String() string {
	return fmt.Sprintf("{URL: %v, Header: %v, Offset: %d, Length: %d}", r.URL, r.Header, r.Offset, r.Length)
}

type section interface {
	Name() string
	Len() int
	io.WriterTo
}

// staging area for writing index section
type indexSection struct {
	es    []*indexEntry
	bytes []byte
}

func (is *indexSection) addExchange(e *Exchange, offset, length int) error {
	variants := normalizeHeaderValues(e.Response.Header[http.CanonicalHeaderKey("variants")])
	variantKey := normalizeHeaderValues(e.Response.Header[http.CanonicalHeaderKey("variant-key")])
	ent := &indexEntry{
		Request:    e.Request,
		Variants:   variants,
		VariantKey: variantKey,
		Offset:     uint64(offset),
		Length:     uint64(length),
	}
	is.es = append(is.es, ent)
	return nil
}

func (is *indexSection) Finalize(ver version.Version) error {
	if is.bytes != nil {
		panic("indexSection must be Finalize()-d only once.")
	}

	var b bytes.Buffer
	enc := cbor.NewEncoder(&b)

	if ver.SupportsVariants() {
		// CDDL:
		//   index = {* whatwg-url => [ variants-value, +location-in-responses ] }
		//   variants-value = bstr
		//   location-in-responses = (offset: uint, length: uint)
		m := make(map[string][]*indexEntry)
		for _, e := range is.es {
			url := e.URL.String()
			m[url] = append(m[url], e)
		}

		mes := []*cbor.MapEntryEncoder{}
		for url, es := range m {
			var variantsValue []byte
			if len(es) > 1 {
				variantsValue = []byte(es[0].Variants)
				var err error
				es, err = entriesInPossibleKeyOrder(es)
				if err != nil {
					return fmt.Errorf("bundle: cannot construct index entry for %s: %v", url, err)
				}
			}

			me := cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				if err := keyE.EncodeTextString(url); err != nil {
					panic(err)
				}

				if err := valueE.EncodeArrayHeader(1 + len(es)*2); err != nil {
					panic(err)
				}
				if err := valueE.EncodeByteString(variantsValue); err != nil {
					panic(err)
				}
				for _, e := range es {
					if err := valueE.EncodeUint(e.Offset); err != nil {
						panic(err)
					}
					if err := valueE.EncodeUint(e.Length); err != nil {
						panic(err)
					}
				}
			})
			mes = append(mes, me)
		}
		if err := enc.EncodeMap(mes); err != nil {
			return err
		}
	} else {
		// CDDL:
		// index = {* whatwg-url => [ location-in-responses ] }
		// whatwg-url = tstr
		// location-in-responses = (offset: uint, length: uint)
		m := make(map[string][]*indexEntry)
		for _, e := range is.es {
			url := e.URL.String()
			m[url] = append(m[url], e)
		}

		mes := []*cbor.MapEntryEncoder{}
		for url, es := range m {
			if len(es) > 1 {
				return errors.New("This WebBundle version '" + string(ver) + "' does not support variants, so we cannot have multiple resources per URL.")
			}
			me := cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				if err := keyE.EncodeTextString(url); err != nil {
					panic(err)
				}

				if err := valueE.EncodeArrayHeader(2); err != nil {
					panic(err)
				}
				if err := valueE.EncodeUint(es[0].Offset); err != nil {
					panic(err)
				}
				if err := valueE.EncodeUint(es[0].Length); err != nil {
					panic(err)
				}
			})
			mes = append(mes, me)
		}
		if err := enc.EncodeMap(mes); err != nil {
			return err
		}
	}

	is.bytes = b.Bytes()
	return nil
}

// entriesInPossibleKeyOrder reorders es by VariantKey, in the order they should
// appear in the index section; the row-major order of possible keys for
// Variants. All entries in es must have the same Variants value.
//
// For example, if the Variants value is
// "Accept-Language;en;fr, Accept-Encoding:gzip;br", the result would satisfy
// this:
//   result[0].VariantKey == "en;gzip"
//   result[1].VariantKey == "en;br"
//   result[2].VariantKey == "fr;gzip"
//   result[3].VariantKey == "fr;br"
//
// Note that VariantKey can have multiple keys (e.g. "en;gzip, fr;gzip"). Such
// entrys will appear multiple times in the result. e.g.:
//   result[0].VariantKey == "en;gzip, fr;gzip"
//   result[1].VariantKey == "en;br"
//   result[2] == result[0]
//   result[3].VariantKey == "fr;br"
//
// If entries in es do not cover all combination of possible keys or two entries
// have overwrapping possible keys, this returns an error.
func entriesInPossibleKeyOrder(es []*indexEntry) ([]*indexEntry, error) {
	if es[0].Variants == "" {
		return nil, errors.New("no Variants header")
	}
	variants, err := parseVariants(es[0].Variants)
	if err != nil {
		return nil, fmt.Errorf("cannot parse Variants header value %q: %v", es[0].Variants, err)
	}
	numPossibleKeys, err := variants.numberOfPossibleKeys()
	if err != nil {
		return nil, fmt.Errorf("invalid Variants header value %q: %v", es[0].Variants, err)
	}

	result := make([]*indexEntry, numPossibleKeys)
	for _, e := range es {
		// TODO: Compare Variants values as lists
		// (e.g. "Accept;foo;bar" == "Accept; foo; bar").
		if e.Variants != es[0].Variants {
			return nil, fmt.Errorf("inconsistent Variants value. %q != %q", e.Variants, es[0].Variants)
		}
		vks, err := parseListOfStringLists(e.VariantKey)
		if err != nil {
			return nil, fmt.Errorf("cannot parse Variant-Key header %q: %v", e.VariantKey, err)
		}
		for _, vk := range vks {
			i := variants.indexInPossibleKeys(vk)
			if i == -1 {
				return nil, fmt.Errorf("Variant-Key %q is not covered by variants %q", e.VariantKey, e.Variants)
			}
			if result[i] != nil {
				return nil, fmt.Errorf("duplicated entries with Variant-Key %q", vk)
			}
			result[i] = e
		}
	}
	for i, e := range result {
		if e == nil {
			return nil, fmt.Errorf("no entry for Variant-Key %v", variants.possibleKeyAt(i))
		}
	}
	return result, nil
}

func parseListOfStringLists(s string) ([][]string, error) {
	ll, err := structuredheader.ParseListOfLists(s)
	if err != nil {
		return nil, err
	}
	// Convert [][]structuredheader.Item to [][]string.
	var result [][]string
	for _, l := range ll {
		var sl []string
		for _, item := range l {
			switch v := item.(type) {
			case string:
				sl = append(sl, v)
			case structuredheader.Token:
				sl = append(sl, string(v))
			default:
				return nil, fmt.Errorf("unexpected value of type %T", v)
			}
		}
		result = append(result, sl)
	}
	return result, nil
}

// Variants represents a Variants: header value.
type Variants [][]string

func parseVariants(s string) (Variants, error) {
	vs, err := parseListOfStringLists(s)
	return Variants(vs), err
}

func (v Variants) numberOfPossibleKeys() (int, error) {
	n := 1
	for _, vals := range v {
		// vals is [header-name, possible-value1, possible-value2, ...]
		if len(vals) <= 1 {
			return 0, errors.New("no possible key")
		}
		n *= len(vals) - 1
		if n > maxNumVariantsForSingleURL {
			return 0, errors.New("too many possible keys")
		}
	}
	return n, nil
}

// indexInPossibleKeys returns the index of variantKey within the all possible
// key combinations for v. If variantKey is not a possible key for v, this
// returns -1.
func (v Variants) indexInPossibleKeys(variantKey []string) int {
	if len(v) != len(variantKey) {
		return -1
	}

	index := 0
OuterLoop:
	for i, vals := range v {
		vals = vals[1:] // Drop header-name
		for indexInAxis, val := range vals {
			if val == variantKey[i] {
				index = index*len(vals) + indexInAxis
				continue OuterLoop
			}
		}
		return -1
	}
	return index
}

// possibleKeyAt returns a variant key at given index within the all possible
// key combinations for v, or nil if index is out of range.
func (v Variants) possibleKeyAt(index int) []string {
	keys := make([]string, len(v))
	for i := len(v) - 1; i >= 0; i-- {
		vals := v[i][1:] // Drop header-name
		keys[i] = vals[index%len(vals)]
		index /= len(vals)
	}
	if index != 0 {
		return nil // index out of range
	}
	return keys
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

func (rs *responsesSection) addResponse(r Response) (int, int, error) {
	offset := rs.buf.Len()

	headerCbor, err := r.EncodeHeader()
	if err != nil {
		return 0, 0, err
	}

	enc := cbor.NewEncoder(&rs.buf)
	if err := enc.EncodeArrayHeader(2); err != nil {
		return 0, 0, fmt.Errorf("bundle: failed to encode response array header: %v", err)
	}
	if err := enc.EncodeByteString(headerCbor); err != nil {
		return 0, 0, fmt.Errorf("bundle: failed to encode response header cbor bytestring: %v", err)
	}
	if err := enc.EncodeByteString(r.Body); err != nil {
		return 0, 0, fmt.Errorf("bundle: failed to encode response payload bytestring: %v", err)
	}

	length := rs.buf.Len() - offset
	return offset, length, nil
}

func (rs *responsesSection) Name() string { return "responses" }
func (rs *responsesSection) Len() int     { return rs.buf.Len() }
func (rs *responsesSection) WriteTo(w io.Writer) (int64, error) {
	return rs.buf.WriteTo(w)
}

type primarySection struct {
	bytes.Buffer
}

func (ps *primarySection) Name() string { return "primary" }

func newPrimarySection(url *url.URL) (*primarySection, error) {
	var ps primarySection
	enc := cbor.NewEncoder(&ps)
	if err := enc.EncodeTextString(url.String()); err != nil {
		return nil, err
	}
	return &ps, nil
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

type signaturesSection struct {
	bytes.Buffer
}

func (rs *signaturesSection) Name() string { return "signatures" }

func newSignaturesSection(sigs *Signatures) (*signaturesSection, error) {
	var ss signaturesSection
	enc := cbor.NewEncoder(&ss)

	enc.EncodeArrayHeader(2)

	enc.EncodeArrayHeader(len(sigs.Authorities))
	for _, auth := range sigs.Authorities {
		if err := auth.EncodeTo(enc); err != nil {
			return nil, err
		}
	}

	enc.EncodeArrayHeader(len(sigs.VouchedSubsets))
	for _, vs := range sigs.VouchedSubsets {
		mes := []*cbor.MapEntryEncoder{
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("authority")
				valueE.EncodeUint(vs.Authority)
			}),
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("sig")
				valueE.EncodeByteString(vs.Sig)
			}),
			cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				keyE.EncodeTextString("signed")
				valueE.EncodeByteString(vs.Signed)
			}),
		}
		if err := enc.EncodeMap(mes); err != nil {
			return nil, err
		}
	}

	return &ss, nil
}

func addExchange(is *indexSection, rs *responsesSection, e *Exchange) error {
	offset, length, err := rs.addResponse(e.Response)
	if err != nil {
		return err
	}

	if err := is.addExchange(e, offset, length); err != nil {
		return err
	}
	return nil
}

func writePrimaryURL(w io.Writer, url *url.URL) error {
	enc := cbor.NewEncoder(w)
	return enc.EncodeTextString(url.String())
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

func (b *Bundle) WriteTo(w io.Writer) (int64, error) {
	cw := NewCountingWriter(w)

	is := &indexSection{}
	rs := newResponsesSection(len(b.Exchanges))

	for _, e := range b.Exchanges {
		if err := addExchange(is, rs, e); err != nil {
			return cw.Written, err
		}
	}
	if err := is.Finalize(b.Version); err != nil {
		return cw.Written, err
	}

	sections := []section{}
	sections = append(sections, is)
	if !b.Version.HasPrimaryURLFieldInHeader() && b.PrimaryURL != nil {
		ps, err := newPrimarySection(b.PrimaryURL)
		if err != nil {
			return cw.Written, err
		}
		sections = append(sections, ps)
	}
	if b.ManifestURL != nil {
		if !b.Version.SupportsManifestSection() {
			return cw.Written, errors.New("This version of the WebBundle does not support storing manifest URL.")
		}
		ms, err := newManifestSection(b.ManifestURL)
		if err != nil {
			return cw.Written, err
		}
		sections = append(sections, ms)
	}
	if b.Signatures != nil && b.Version.SupportsSignatures() {
		ss, err := newSignaturesSection(b.Signatures)
		if err != nil {
			return cw.Written, err
		}
		sections = append(sections, ss)
	}
	sections = append(sections, rs) // resources section must be the last.

	if _, err := cw.Write(b.Version.HeaderMagicBytes()); err != nil {
		return cw.Written, err
	}
	if b.Version.HasPrimaryURLFieldInHeader() {
		if err := writePrimaryURL(cw, b.PrimaryURL); err != nil {
			return cw.Written, err
		}
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
