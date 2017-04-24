package webpack

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/net/http2/hpack"

	"github.com/dimich-g/webpackage/go/webpack/cbor"
)

func ParseCbor(packageFilename string) (Package, error) {
	pack, err := ioutil.ReadFile(packageFilename)
	if err != nil {
		return Package{}, err
	}
	reader := bytes.NewReader(pack)

	parts := make([]*PackPart, 0)
	if err := parseIndexedContent(reader, parts); err != nil {
		return Package{}, err
	}

	return Package{Manifest{}, parts, nil, nil}, nil
}

func parseIndexedContent(reader *bytes.Reader, parts []*PackPart) error {
	panic("Not implemented")
}

func WriteCbor(p *Package) (result []byte, err error) {
	cborPackage := cbor.New()

	arr := cborPackage.AppendArray(5)
	arr.AppendBytes([]byte{0xF0, 0x9F, 0x8C, 0x90, 0xF0, 0x9F, 0x93, 0xA6})
	sectionOffsets := arr.AppendMap(1)
	sectionOffsets.AppendUtf8S("indexed-content")
	// "indexed-content" will appear at the start of the 'sections' map.
	indexedContentOffset := sectionOffsets.AppendPendingUint()
	sectionOffsets.Finish()

	sections := arr.AppendMap(1)
	indexedContentOffset.Complete(uint64(sections.ByteLenSoFar()))
	sections.AppendUtf8S("indexed-content")
	indexedContent := sections.AppendArray(2)
	index := indexedContent.AppendArray(len(p.parts))
	pendingOffsets := make(map[*PackPart]*cbor.PendingInt, len(p.parts))
	for _, part := range p.parts {
		arr := index.AppendArray(2)
		arr.AppendBytes(encodeResourceKey(part))
		pendingOffsets[part] = arr.AppendPendingUint()
		arr.Finish()
	}
	index.Finish()
	responses := indexedContent.AppendArray(len(p.parts))
	for _, part := range p.parts {
		pendingOffset, ok := pendingOffsets[part]
		if !ok {
			panic(fmt.Sprintf("%p missing from %v", part, pendingOffsets))
		}
		pendingOffset.Complete(uint64(responses.ByteLenSoFar()))
		delete(pendingOffsets, part)

		arr := responses.AppendArray(2)
		arr.AppendBytes(encodeResponseHeaders(part))
		content, err := part.Content()
		if err != nil {
			return nil, err
		}
		contentBytes, err := ioutil.ReadAll(content)
		if err != nil {
			return nil, err
		}
		arr.AppendBytes(contentBytes)
		arr.Finish()
	}
	responses.Finish()
	indexedContent.Finish()
	sections.Finish()
	arr.AppendFixedSizeUint64(uint64(arr.ByteLenSoFar() + 18))
	arr.AppendBytes([]byte{0xF0, 0x9F, 0x8C, 0x90, 0xF0, 0x9F, 0x93, 0xA6})
	arr.Finish()
	return cborPackage.Finish(), err
}

func encodeResourceKey(part *PackPart) []byte {
	var buf bytes.Buffer
	encoder := hpack.NewEncoder(&buf)
	encoder.WriteField(httpHeader(":method", "GET"))
	encoder.WriteField(httpHeader(":scheme", part.url.Scheme))
	encoder.WriteField(httpHeader(":authority", part.url.Host))
	encoder.WriteField(httpHeader(":path", part.url.RequestURI()))
	for _, field := range part.requestHeaders {
		encoder.WriteField(field)
	}
	return buf.Bytes()
}

func encodeResponseHeaders(part *PackPart) []byte {
	var buf bytes.Buffer
	encoder := hpack.NewEncoder(&buf)
	encoder.WriteField(httpHeader(":status", strconv.FormatInt(int64(part.status), 10)))
	for _, field := range part.responseHeaders {
		encoder.WriteField(field)
	}
	return buf.Bytes()
}

func writeCborPart(w *bufio.Writer, base string, part *PackPart) (err error) {
	if _, err = io.WriteString(w, part.url.String()); err != nil {
		return
	}
	if err = part.requestHeaders.WriteHttp1(w); err != nil {
		return
	}
	if _, err = io.WriteString(w, "\r\n"); err != nil {
		return
	}
	if err = part.responseHeaders.WriteHttp1(w); err != nil {
		return
	}

	// Write the content to a file under base/.
	relativeOutContentFilename := filepath.Join(part.url.Scheme, part.url.Host,
		part.url.Path+part.url.RawQuery)
	outContentFilename := filepath.Join(base, relativeOutContentFilename)
	if err = os.MkdirAll(filepath.Dir(outContentFilename), 0755); err != nil {
		return
	}
	outContentFile, err := os.Create(outContentFilename)
	if err != nil {
		return
	}
	defer outContentFile.Close()
	inContent, err := part.Content()
	if err != nil {
		return
	}
	defer inContent.Close()
	io.Copy(outContentFile, inContent)

	if _, err = io.WriteString(w, relativeOutContentFilename); err != nil {
		return
	}
	if _, err = io.WriteString(w, "\r\n"); err != nil {
		return
	}
	return
}
