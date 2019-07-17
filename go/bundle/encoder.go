package bundle

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/WICG/webpackage/go/bundle/version"
	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

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

var _ = io.WriterTo(&Bundle{})

type requestEntry struct {
	Request
	Offset uint64 // Offset within the responses section
	Length uint64
}

func (r requestEntry) String() string {
	return fmt.Sprintf("{URL: %v, Header: %v, Offset: %d, Length: %d}", r.URL, r.Header, r.Offset, r.Length)
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

func (is *indexSection) addRequest(r Request, offset, length int) error {
	ent := requestEntry{
		Request: r,
		Offset:  uint64(offset),
		Length:  uint64(length),
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

	if ver.HasVariantsSupport() {
		// CDDL:
		//   index = {* whatwg-url => [ variants-value, +location-in-responses ] }
		//   variants-value = bstr
		//   location-in-responses = (offset: uint, length: uint)
		mes := []*cbor.MapEntryEncoder{}
		for _, e := range is.es {
			me := cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
				if err := keyE.EncodeTextString(e.URL.String()); err != nil {
					panic(err)
				}
				// Currently, this encoder does not support variants. So, the
				// map value is always a three-element array ['', offset, length].
				if err := valueE.EncodeArrayHeader(3); err != nil {
					panic(err)
				}
				if err := valueE.EncodeByteString(nil); err != nil {
					panic(err)
				}
				if err := valueE.EncodeUint(e.Offset); err != nil {
					panic(err)
				}
				if err := valueE.EncodeUint(e.Length); err != nil {
					panic(err)
				}
			})
			mes = append(mes, me)
		}
		if err := enc.EncodeMap(mes); err != nil {
			return err
		}
	} else {
		// CDDL: index = [* (headers, length: uint) ]
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
	offset, length, err := rs.addResponse(e.Response)
	if err != nil {
		return err
	}

	if err := is.addRequest(e.Request, offset, length); err != nil {
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
	if b.ManifestURL != nil {
		ms, err := newManifestSection(b.ManifestURL)
		if err != nil {
			return cw.Written, err
		}
		sections = append(sections, ms)
	}
	sections = append(sections, rs)

	if _, err := cw.Write(b.Version.HeaderMagicBytes()); err != nil {
		return cw.Written, err
	}
	if b.Version.HasPrimaryURLField() {
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
