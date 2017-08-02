package webpack

import (
	"bytes"
	"crypto"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"

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
	// the offsets and hashes of each response within it.
	tempResponsesFile, err := ioutil.TempFile("", "webpack-responses")
	if err != nil {
		return err
	}
	defer os.Remove(tempResponsesFile.Name())
	defer tempResponsesFile.Close()

	partInfo, err := writeCBORResourceBodies(p, tempResponsesFile)
	if err != nil {
		return err
	}

	haveManifest := len(p.manifest.signatures) > 0
	manifestLength := uint64(0)
	var tempManifestFile *os.File
	if haveManifest {
		tempManifestFile, err = ioutil.TempFile("", "webpack-manifest")
		if err != nil {
			return err
		}
		defer os.Remove(tempResponsesFile.Name())
		defer tempResponsesFile.Close()

		manifestLength, err = writeCBORSignedManifest(p, partInfo, tempManifestFile)
		if err != nil {
			return err
		}
	}

	cborPackage := cbor.New(to)

	arr := cborPackage.AppendArray(5)

	// "🌐📦" in UTF-8.
	var magicNumber = []byte{0xF0, 0x9F, 0x8C, 0x90, 0xF0, 0x9F, 0x93, 0xA6}

	arr.AppendBytes(magicNumber)

	// section-offsets:
	numSections := uint64(1)
	sectionOffsetValues := make(map[string]uint64)
	sectionOffsetValues["indexed-content"] = 1
	if haveManifest {
		numSections++
		sectionOffsetValues["manifest"] = 1
		sectionOffsetValues["indexed-content"] += uint64(len(cbor.Encoded(cbor.TypeText, len("manifest")))) + uint64(len("manifest")) + manifestLength
	}
	sectionOffsets := arr.AppendMap(numSections)
	if haveManifest {
		sectionOffsets.AppendUTF8S("manifest")
		sectionOffsets.AppendUint64(sectionOffsetValues["manifest"])
	}
	sectionOffsets.AppendUTF8S("indexed-content")
	// "indexed-content" will appear at the start of the 'sections' map.
	sectionOffsets.AppendUint64(sectionOffsetValues["indexed-content"])
	sectionOffsets.Finish()

	sections := arr.AppendMap(numSections)

	// manifest major section:
	if haveManifest {
		if sections.ByteLenSoFar() != sectionOffsetValues["manifest"] {
			panic(fmt.Sprintf("Wrote incorrect offset (%v) for manifest section actually at offset %v",
				sectionOffsetValues["manifest"], sections.ByteLenSoFar()))
		}
		sections.AppendUTF8S("manifest")
		// Append the whole manifest.
		offset, err := tempManifestFile.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
		if offset != 0 {
			panic(fmt.Sprintf("Seek to start seeked to %v instead.", offset))
		}
		sections.AppendSerializedItem(tempManifestFile)
	}

	// indexed-content major section:
	if sections.ByteLenSoFar() != sectionOffsetValues["indexed-content"] {
		panic(fmt.Sprintf("Wrote incorrect offset (%v) for indexed-content section actually at offset %v",
			sectionOffsetValues["indexed-content"], sections.ByteLenSoFar()))
	}
	sections.AppendUTF8S("indexed-content")
	indexedContent := sections.AppendArray(2)

	// Write the requests and the byte offsets to their responses into the
	// index.
	index := indexedContent.AppendArray(uint64(len(p.parts)))
	for _, part := range p.parts {
		arr := index.AppendArray(2)
		arr.AppendBytes(part.requestHeaders.EncodeHPACK())
		partInf, ok := partInfo[part]
		if !ok {
			panic(fmt.Sprintf("%p missing from %v", part, partInfo))
		}
		arr.AppendUint64(partInf.offset)
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

type PartInfo struct {
	offset uint64
	hashes map[crypto.Hash][]byte
}

// writeCBORResourceBodies returns a map from parts to their byte offsets within
// this item.
func writeCBORResourceBodies(p *Package, to io.Writer) (map[*PackPart]PartInfo, error) {
	partInfo := make(map[*PackPart]PartInfo)
	cbor := cbor.New(to)
	responses := cbor.AppendArray(uint64(len(p.parts)))
	for _, part := range p.parts {
		info, err := writeCborResourceBody(part, responses, &p.manifest)
		if err != nil {
			return nil, err
		}
		partInfo[part] = info
	}
	responses.Finish()
	cbor.Finish()
	return partInfo, nil
}

// Writes part to the next item in responses, returning its PartInfo
func writeCborResourceBody(part *PackPart, responses *cbor.Array, manifest *Manifest) (PartInfo, error) {
	info := PartInfo{
		offset: uint64(responses.ByteLenSoFar()),
		hashes: make(map[crypto.Hash][]byte),
	}
	// Hashes will include all request headers, response headers, and the bodies.
	hashWriter, hashers := MultiHasher(manifest.hashTypes)
	hashCbor := cbor.New(hashWriter)
	hashCborArray := hashCbor.AppendArray(3)
	part.requestHeaders.EncodeToCBOR(hashCborArray)
	part.responseHeaders.EncodeToCBOR(hashCborArray)

	arr := responses.AppendArray(2)
	arr.AppendBytes(part.responseHeaders.EncodeHPACK())
	content, err := part.Content()
	if err != nil {
		return info, err
	}
	defer content.Close()

	mainContent := arr.AppendBytesWriter(content.Size())
	hashContent := hashCborArray.AppendBytesWriter(content.Size())
	if _, err := io.Copy(io.MultiWriter(mainContent, hashContent), content); err != nil {
		return info, err
	}
	mainContent.Finish()
	arr.Finish()

	hashContent.Finish()
	hashCborArray.Finish()
	hashCbor.Finish()
	for hashType, hasher := range hashers {
		info.hashes[hashType] = hasher.Sum(nil)
	}
	return info, nil
}

type appendArrayer interface {
	AppendArray(size uint64) *cbor.Array
}

// This uses an array instead of a map because HTTP headers can be repeated, and
// because the order is significant.
func (h HTTPHeaders) EncodeToCBOR(c appendArrayer) {
	arr := c.AppendArray(uint64(len(h)) * 2)
	for _, header := range h {
		arr.AppendBytes([]byte(header.Name))
		arr.AppendBytes([]byte(header.Value))
	}
	arr.Finish()
}

// https://github.com/WICG/webpackage#manifest
func writeCBORSignedManifest(p *Package, partInfo map[*PackPart]PartInfo, to io.Writer) (byteLen uint64, err error) {
	// Write the manifest to an in-memory buffer, which we can then sign.
	var manifestCbor bytes.Buffer
	manifestTop := cbor.New(&manifestCbor)
	manifest := manifestTop.AppendMap(2)

	manifest.AppendUTF8S("metadata")
	metadataKeys := []string{"origin", "date"}
	for key, _ := range p.manifest.metadata.otherFields {
		metadataKeys = append(metadataKeys, key)
	}
	sort.Slice(metadataKeys, func(i, j int) bool {
		return cbor.CanonicalLessStrings(metadataKeys[i], metadataKeys[j])
	})
	metadata := manifest.AppendMap(uint64(len(metadataKeys)))
	for _, key := range metadataKeys {
		metadata.AppendUTF8S(key)
		switch key {
		case "date":
			time := metadata.AppendTag(cbor.TagTime)
			time.AppendInt64(p.manifest.metadata.date.Unix())
			time.Finish()
		case "origin":
			uri := metadata.AppendTag(cbor.TagURI)
			uri.AppendUTF8S(p.manifest.metadata.origin.String())
			uri.Finish()
		default:
			metadata.AppendGeneric(p.manifest.metadata.otherFields[key])
		}
	}

	metadata.Finish()

	manifest.AppendUTF8S("resource-hashes")
	resourceHashes := manifest.AppendMap(uint64(len(p.manifest.hashTypes)))
	for _, hashType := range p.manifest.hashTypes {
		resourceHashes.AppendUTF8S(HashName(hashType))
		hashes := make([][]byte, 0, len(partInfo))
		for _, part := range partInfo {
			hashes = append(hashes, part.hashes[hashType])
		}
		sort.Slice(hashes, func(i, j int) bool {
			return cbor.CanonicalLessBytes(hashes[i], hashes[j])
		})

		hashArray := resourceHashes.AppendArray(uint64(len(hashes)))
		for _, hash := range hashes {
			hashArray.AppendBytes(hash)
		}
		hashArray.Finish()
	}
	resourceHashes.Finish()
	manifest.Finish()
	manifestTop.Finish()

	top := cbor.New(to)
	signedManifest := top.AppendMap(3)
	signedManifest.AppendUTF8S("manifest")
	signedManifest.AppendSerializedItem(bytes.NewReader(manifestCbor.Bytes()))

	signedManifest.AppendUTF8S("signatures")
	signatureArray := signedManifest.AppendArray(uint64(len(p.manifest.signatures)))
	haveSignatureForOrigin := false
	signatureForOriginErrs := []error{}
	for i, signWith := range p.manifest.signatures {
		if err := signWith.certificate.VerifyHostname(p.manifest.metadata.origin.Hostname()); err != nil {
			signatureForOriginErrs = append(signatureForOriginErrs, err)
		} else {
			haveSignatureForOrigin = true
		}
		signature := signatureArray.AppendMap(2)
		signature.AppendUTF8S("keyIndex")
		signature.AppendUint64(uint64(i))
		signature.AppendUTF8S("signature")
		sigBytes, err := Sign(signWith.key, manifestCbor.Bytes())
		if err != nil {
			return 0, err
		}
		signature.AppendBytes(sigBytes)
		signature.Finish()
	}
	if !haveSignatureForOrigin {
		return 0, fmt.Errorf("No signing certificate was valid for origin %q: %v",
			p.manifest.metadata.origin.Hostname(),
			signatureForOriginErrs)
	}
	signatureArray.Finish()

	signedManifest.AppendUTF8S("certificates")
	// The certificate array contains both the signing certificates and their chains.
	certificateArray := signedManifest.AppendArray(uint64(len(p.manifest.signatures) + len(p.manifest.certificates)))
	for _, signWith := range p.manifest.signatures {
		certificateArray.AppendBytes(signWith.certificate.Raw)
	}
	for _, cert := range p.manifest.certificates {
		certificateArray.AppendBytes(cert.Raw)
	}
	certificateArray.Finish()
	signedManifest.Finish()

	byteLen = top.ByteLenSoFar()
	top.Finish()
	return byteLen, nil
}
