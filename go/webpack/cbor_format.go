package webpack

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/WICG/webpackage/go/webpack/cbor"
)

func ParseCBOR(packageFilename string) (Package, error) {
	panic("Unimplemented")
	pack, err := ioutil.ReadFile(packageFilename)
	if err != nil {
		return Package{}, err
	}
	reader := bytes.NewReader(pack)

	parts := make([]*PackPart, 0)
	if err := parseIndexedContent(reader, parts); err != nil {
		return Package{}, err
	}

	return Package{Manifest{}, parts}, nil
}

func parseIndexedContent(reader *bytes.Reader, parts []*PackPart) error {
	panic("Not implemented")
}

func WriteCBOR(p *Package, to io.Writer) error {
	// Write the indexed-content/responses array first in order to compute
	// the offsets of each response within it.
	tempResponsesFile, err := ioutil.TempFile("", "webpack-responses")
	if err != nil {
		return err
	}
	defer os.Remove(tempResponsesFile.Name())
	defer tempResponsesFile.Close()

	partOffsets, err := writeCBORResourceBodies(p, tempResponsesFile)
	if err != nil {
		return err
	}

	cborPackage := cbor.New(to)

	arr := cborPackage.AppendArray(5)

	// "🌐📦" in UTF-8.
	var magicNumber = []byte{0xF0, 0x9F, 0x8C, 0x90, 0xF0, 0x9F, 0x93, 0xA6}

	arr.AppendBytes(magicNumber)

	// section-offsets:
	sectionOffsets := arr.AppendMap(1)
	sectionOffsets.AppendUTF8S("indexed-content")
	// "indexed-content" will appear at the start of the 'sections' map.
	const indexedContentOffset = 1
	sectionOffsets.AppendUint64(indexedContentOffset)
	sectionOffsets.Finish()

	sections := arr.AppendMap(1)

	// indexed-content major section:
	if sections.ByteLenSoFar() != indexedContentOffset {
		panic(fmt.Sprintf("Wrote incorrect offset (%v) for indexed-content section actually at offset %v",
			indexedContentOffset, sections.ByteLenSoFar()))
	}
	sections.AppendUTF8S("indexed-content")
	indexedContent := sections.AppendArray(2)

	// Write the requests and the byte offsets to their responses into the
	// index.
	index := indexedContent.AppendArray(uint64(len(p.parts)))
	for _, part := range p.parts {
		arr := index.AppendArray(2)
		arr.AppendBytes(part.requestHeaders.EncodeHPACK())
		partOffset, ok := partOffsets[part]
		if !ok {
			panic(fmt.Sprintf("%p missing from %v", part, partOffsets))
		}
		arr.AppendUint64(partOffset)
		arr.Finish()
	}
	index.Finish()

	// Append the whole responses array to indexed-content.
	offset, err := tempResponsesFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	if offset != 0 {
		panic(fmt.Sprintf("Seek to start seeked to %v instead.", offset))
	}
	indexedContent.AppendSerializedItem(tempResponsesFile)

	indexedContent.Finish()
	sections.Finish()

	// The whole size of the package is the size to here, plus two 8-byte
	// items and their 1-byte headers.
	arr.AppendFixedSizeUint64(uint64(arr.ByteLenSoFar() + 18))
	arr.AppendBytes(magicNumber)
	arr.Finish()
	return cborPackage.Finish()
}

// writeCBORResourceBodies returns a map from parts to their byte offsets within
// this item.
func writeCBORResourceBodies(p *Package, to io.Writer) (map[*PackPart]uint64, error) {
	partOffsets := make(map[*PackPart]uint64)
	cbor := cbor.New(to)
	responses := cbor.AppendArray(uint64(len(p.parts)))
	for _, part := range p.parts {
		partOffsets[part] = uint64(responses.ByteLenSoFar())

		arr := responses.AppendArray(2)
		arr.AppendBytes(part.responseHeaders.EncodeHPACK())
		content, err := part.Content()
		if err != nil {
			return nil, err
		}
		mainContent := arr.AppendBytesWriter(content.Size())
		if _, err := io.Copy(mainContent, content); err != nil {
			return nil, err
		}
		mainContent.Finish()
		arr.Finish()
	}
	responses.Finish()
	cbor.Finish()
	return partOffsets, nil
}
