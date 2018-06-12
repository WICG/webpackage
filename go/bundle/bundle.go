package bundle

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/WICG/webpackage/go/signedexchange"
	"github.com/WICG/webpackage/go/signedexchange/cbor"
)

var HeaderMagicBytes = []byte{0x84, 0x48, 0xf0, 0x9f, 0x8c, 0x90, 0xf0, 0x9f, 0x93, 0xa6}

const FooterLength = 9

type Input struct {
	Exchanges []*signedexchange.Exchange
}

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
	Offset int
	Length int
}

type sectionOffsets []sectionOffset

func (so *sectionOffsets) AddSection(name string, length int) {
	offset := 0
	if len(*so) > 0 {
		last := (*so)[len(*so)-1]
		offset = last.Offset + last.Length
	}
	*so = append(*so, sectionOffset{name, offset, length})
}

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#load-metadata
// Steps 3-7.
func writeSectionOffsets(w io.Writer, so sectionOffsets) error {
	mes := []*cbor.MapEntryEncoder{}
	for _, e := range so {
		me := cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			// TODO(kouhei): error plumbing
			keyE.EncodeTextString(e.Name)
			valueE.EncodeArrayHeader(2)
			valueE.EncodeUInt(uint64(e.Offset))
			valueE.EncodeUInt(uint64(e.Length))
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
	bundleSize := uint64(offset) + FooterLength

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

func WriteBundle(w io.Writer, i *Input) error {
	cw := NewCountingWriter(w)

	is := &indexSection{}
	rs := newResponsesSection(len(i.Exchanges))

	for _, e := range i.Exchanges {
		if err := addExchange(is, rs, e); err != nil {
			return err
		}
	}
	if err := is.Finalize(); err != nil {
		return err
	}

	var so sectionOffsets
	so.AddSection("index", is.Len())
	so.AddSection("responses", rs.Len())

	if _, err := cw.Write(HeaderMagicBytes); err != nil {
		return err
	}
	if err := writeSectionOffsets(cw, so); err != nil {
		return err
	}
	if err := writeSectionHeader(cw, len(so)); err != nil {
		return err
	}
	if _, err := cw.Write(is.Bytes()); err != nil {
		return err
	}
	if _, err := cw.Write(rs.Bytes()); err != nil {
		return err
	}
	if err := writeFooter(cw, int(cw.Written)); err != nil {
		return err
	}

	return nil
}
