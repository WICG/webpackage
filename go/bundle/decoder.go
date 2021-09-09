package bundle

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/WICG/webpackage/go/bundle/version"
	"github.com/WICG/webpackage/go/signedexchange/cbor"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

type requestEntryWithOffset struct {
	Request
	Length uint64
	Offset uint64 // Offset within the bundle stream
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

type meta struct {
	version        version.Version
	primaryURL     *url.URL
	sectionOffsets []sectionOffset
	sectionsStart  uint64
	manifestURL    *url.URL
	signatures     *Signatures
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
	n, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to decode request headers map header: %v", err)
	}

	headers := make(http.Header)

	pseudos := make(map[string]string)

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

		if strings.ToLower(name) != name {
			return nil, nil, fmt.Errorf("Failed to decode request headers map key: %q contains upper-case.", name)
		}

		if strings.HasPrefix(name, ":") {
			if _, exists := pseudos[name]; exists {
				return nil, nil, fmt.Errorf("Failed to decode request headers map entry. Pseudo %q appeared twice.", name)
			}

			pseudos[name] = value

			continue
		}

		if _, exists := headers[name]; exists {
			return nil, nil, fmt.Errorf("Failed to decode request headers map entry. Header %q appeared twice.", name)
		}

		headers.Set(name, value)
	}

	return headers, pseudos, nil
}

// https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html#name-the-index-section
func parseIndexSection(sectionContents []byte, sectionsStart uint64, sos []sectionOffset) ([]requestEntryWithOffset, error) {
	dec := cbor.NewDecoder(bytes.NewBuffer(sectionContents))
	numUrls, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle.index: failed to decode index section map header: %v", err)
	}
	requests := []requestEntryWithOffset{}

	respso, respSectionRelOffset, found := FindSection(sos, "responses")
	if !found {
		return nil, fmt.Errorf("bundle.index: \"responses\" section not found")
	}
	respSectionOffset := sectionsStart + respSectionRelOffset
	makeRelativeToStream := func(offset, length uint64) (uint64, uint64, error) {
		if offset+length > respso.Length {
			return 0, 0, errors.New("bundle.index: response length out-of-range")
		}
		return respSectionOffset + offset, length, nil
	}

	for i := uint64(0); i < numUrls; i++ {
		rawUrl, err := dec.DecodeTextString()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode map key: %v", i, err)
		}
		parsedUrl, err := url.Parse(rawUrl)
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to parse URL: %v", i, err)
		}
		if parsedUrl.Fragment != "" {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains fragment: %q", i, rawUrl)
		}
		if parsedUrl.User != nil {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains credentials: %q", i, rawUrl)
		}

		numItems, err := dec.DecodeArrayHeader()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode value array header: %v", i, err)
		}
		if numItems != 2 {
			return nil, fmt.Errorf("bundle.index[%d]: value array must be exactly 2 elements: offset and length.", i)
		}
		offset, err := dec.DecodeUint()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode offset: %v", i, err)
		}
		length, err := dec.DecodeUint()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode length: %v", i, err)
		}
		offset, length, err = makeRelativeToStream(offset, length)
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: %v", i, err)
		}
		requests = append(requests, requestEntryWithOffset{Request: Request{URL: parsedUrl}, Offset: offset, Length: length})
	}
	return requests, nil
}

// https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html#name-the-index-section
func parseIndexSectionWithVariants(sectionContents []byte, sectionsStart uint64, sos []sectionOffset) ([]requestEntryWithOffset, error) {
	dec := cbor.NewDecoder(bytes.NewBuffer(sectionContents))
	numUrls, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle.index: failed to decode index section map header: %v", err)
	}
	requests := []requestEntryWithOffset{}

	respso, respSectionRelOffset, found := FindSection(sos, "responses")
	if !found {
		return nil, fmt.Errorf("bundle.index: \"responses\" section not found")
	}
	respSectionOffset := sectionsStart + respSectionRelOffset
	makeRelativeToStream := func(offset, length uint64) (uint64, uint64, error) {
		if offset+length > respso.Length {
			return 0, 0, errors.New("bundle.index: response length out-of-range")
		}
		return respSectionOffset + offset, length, nil
	}

	for i := uint64(0); i < numUrls; i++ {
		rawUrl, err := dec.DecodeTextString()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode map key: %v", i, err)
		}
		parsedUrl, err := url.Parse(rawUrl)
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to parse URL: %v", i, err)
		}
		if parsedUrl.Fragment != "" {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains fragment: %q", i, rawUrl)
		}
		if parsedUrl.User != nil {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains credentials: %q", i, rawUrl)
		}

		numItems, err := dec.DecodeArrayHeader()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode value array header: %v", i, err)
		}
		if numItems == 0 {
			return nil, fmt.Errorf("bundle.index[%d]: value array must not be empty.", i)
		}
		variants_value, err := dec.DecodeByteString()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode variants-value: %v", i, err)
		}
		if len(variants_value) == 0 {
			if numItems != 3 {
				return nil, fmt.Errorf("bundle.index[%d]: The size of value array must be 3", i)
			}
			offset, err := dec.DecodeUint()
			if err != nil {
				return nil, fmt.Errorf("bundle.index[%d]: Failed to decode offset: %v", i, err)
			}
			length, err := dec.DecodeUint()
			if err != nil {
				return nil, fmt.Errorf("bundle.index[%d]: Failed to decode length: %v", i, err)
			}
			offset, length, err = makeRelativeToStream(offset, length)
			if err != nil {
				return nil, fmt.Errorf("bundle.index[%d]: %v", i, err)
			}
			requests = append(requests, requestEntryWithOffset{Request: Request{URL: parsedUrl}, Offset: offset, Length: length})
		} else {
			variants, err := parseVariants(string(variants_value))
			if err != nil {
				return nil, fmt.Errorf("bundle.index[%d]: Cannot parse variants value %q: %v", i, string(variants_value), err)
			}
			numVariantKeys, err := variants.numberOfPossibleKeys()
			if err != nil {
				return nil, fmt.Errorf("bundle.index[%d]: Invalid variants value %q: %v", i, string(variants_value), err)
			}
			if numItems != 2*uint64(numVariantKeys)+1 {
				return nil, fmt.Errorf("bundle.index[%d]: Unexpected size of value array: %d", i, numItems)
			}
			// Currently this implementation just appends all entries to `requests`.
			// TODO: Preserve the map structure from variant-key to location-in-stream.
			for j := 0; j < numVariantKeys; j++ {
				offset, err := dec.DecodeUint()
				if err != nil {
					return nil, fmt.Errorf("bundle.index[%d][%d]: Failed to decode offset: %v", i, j, err)
				}
				length, err := dec.DecodeUint()
				if err != nil {
					return nil, fmt.Errorf("bundle.index[%d][%d]: Failed to decode length: %v", i, j, err)
				}
				offset, length, err = makeRelativeToStream(offset, length)
				if err != nil {
					return nil, fmt.Errorf("bundle.index[%d][%d]: %v", i, j, err)
				}
				requests = append(requests, requestEntryWithOffset{Request: Request{URL: parsedUrl}, Offset: offset, Length: length})
			}
		}
	}
	return requests, nil
}

// The "primary" section records a single URL identifying the primary URL of the bundle. The URL MUST refer to a resource with representations contained in the bundle itself.
func parsePrimarySection(sectionContents []byte) (*url.URL, error) {
	dec := cbor.NewDecoder(bytes.NewBuffer(sectionContents))
	urlString, err := dec.DecodeTextString()
	if err != nil {
		return nil, fmt.Errorf("bundle: failed to parse primary section: %v", err)
	}
	primaryURL, err := url.Parse(urlString)
	// If url is a failure, its fragment is not null, or it includes credentials, return an error.
	if err != nil {
		return nil, fmt.Errorf("bundle: failed to parse primary URL (%s): %v", urlString, err)
	}
	if !primaryURL.IsAbs() || primaryURL.Fragment != "" || primaryURL.User != nil {
		return nil, fmt.Errorf("bundle: primary URL (%s) must be an absolute url without fragment or credentials.", urlString)
	}
	return primaryURL, nil
}

// https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#manifest-section
func parseManifestSection(sectionContents []byte) (*url.URL, error) {
	dec := cbor.NewDecoder(bytes.NewBuffer(sectionContents))
	urlString, err := dec.DecodeTextString()
	if err != nil {
		return nil, fmt.Errorf("bundle: failed to parse manifest section: %v", err)
	}
	manifestURL, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("bundle: failed to parse manifest URL (%s): %v", urlString, err)
	}
	if !manifestURL.IsAbs() || manifestURL.Fragment != "" || manifestURL.User != nil {
		return nil, fmt.Errorf("bundle: manifest URL (%s) must be an absolute url without fragment or credentials.", urlString)
	}
	return manifestURL, nil
}

// https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#signatures-section
func parseSignaturesSection(sectionContents []byte) (*Signatures, error) {
	// signatures = [
	//   authorities: [*authority],
	//   vouched-subsets: [*{
	//     authority: index-in-authorities,
	//     sig: bstr,
	//     signed: bstr  ; Expected to hold a signed-subset item.
	//   }],
	// ]
	dec := cbor.NewDecoder(bytes.NewBuffer(sectionContents))
	signaturesLength, err := dec.DecodeArrayHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle.signatures: failed to decode array header: %v", err)
	}
	if signaturesLength != 2 {
		return nil, fmt.Errorf("bundle.signatures: unexpected array length: %d", signaturesLength)
	}

	authoritiesLength, err := dec.DecodeArrayHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle.signatures: failed to decode array header: %v", err)
	}
	var authorities []*certurl.AugmentedCertificate
	for i := uint64(0); i < authoritiesLength; i++ {
		a, err := certurl.DecodeAugmentedCertificateFrom(dec)
		if err != nil {
			return nil, fmt.Errorf("bundle.signatures: cannot parse certificate: %v", err)
		}
		authorities = append(authorities, a)
	}

	vouchedSubsetsLength, err := dec.DecodeArrayHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle.signatures: failed to decode array header: %v", err)
	}
	var vouchedSubsets []*VouchedSubset
	for i := uint64(0); i < vouchedSubsetsLength; i++ {
		n, err := dec.DecodeMapHeader()
		if err != nil {
			return nil, fmt.Errorf("bundle.signatures: cannot decode map header: %v", err)
		}
		if n != 3 {
			return nil, fmt.Errorf("bundle.signatures: unexpected map size: %d", n)
		}

		vs := &VouchedSubset{}
		for i := uint64(0); i < n; i++ {
			label, err := dec.DecodeTextString()
			if err != nil {
				return nil, fmt.Errorf("bundle.signatures: cannot decode map key: %v", err)
			}
			switch label {
			case "authority":
				vs.Authority, err = dec.DecodeUint()
				if err != nil {
					return nil, fmt.Errorf("bundle.signatures: cannot decode authority: %v", err)
				}
			case "sig":
				vs.Sig, err = dec.DecodeByteString()
				if err != nil {
					return nil, fmt.Errorf("bundle.signatures: cannot decode sig: %v", err)
				}
			case "signed":
				vs.Signed, err = dec.DecodeByteString()
				if err != nil {
					return nil, fmt.Errorf("bundle.signatures: cannot decode signed: %v", err)
				}
			default:
				return nil, fmt.Errorf("bundle.signatures: unexpected map key %q", label)
			}
		}
		vouchedSubsets = append(vouchedSubsets, vs)
	}
	return &Signatures{
		Authorities:    authorities,
		VouchedSubsets: vouchedSubsets,
	}, nil
}

var knownSections = map[string]struct{}{
	"index":      {},
	"manifest":   {},
	"primary":    {},
	"signatures": {},
	"responses":  {},
}

type MetadataErrorType int

const (
	FormatError MetadataErrorType = iota
	VersionError
)

type LoadMetadataError struct {
	error
	Type        MetadataErrorType
	FallbackURL *url.URL
}

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#load-metadata
func loadMetadata(bs []byte) (*meta, error) {

	r := bytes.NewBuffer(bs)

	ver, err := version.ParseMagicBytes(r)
	// TODO(ksakamoto): Continue and return VersionError after parsing fallbackUrl.
	if err != nil {
		return nil, &LoadMetadataError{err, FormatError, nil}
	}

	var fallbackURL *url.URL
	dec := cbor.NewDecoder(r)
	if ver.HasPrimaryURLFieldInHeader() {
		fallbackURLBytes, err := dec.DecodeTextString()
		if err != nil {
			return nil, &LoadMetadataError{fmt.Errorf("bundle: Failed to read fallbackURL string: %v", err), FormatError, nil}
		}
		fallbackURL, err = url.Parse(fallbackURLBytes)
		if err != nil {
			return nil, &LoadMetadataError{fmt.Errorf("bundle: Failed to parse fallbackURL: %v", err), FormatError, nil}
		}
	}

	slbytes, err := dec.DecodeByteString()
	if err != nil {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: Failed to read sectionLengths byte string: %v", err), FormatError, fallbackURL}
	}
	if len(slbytes) >= 8192 {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: sectionLengthsLength is too long (%d bytes)", slbytes), FormatError, fallbackURL}
	}

	sos, err := decodeSectionLengthsCBOR(slbytes)
	if err != nil {
		return nil, &LoadMetadataError{err, FormatError, fallbackURL}
	}

	numSections, err := dec.DecodeArrayHeader()
	if err != nil {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: Failed to read section header."), FormatError, fallbackURL}
	}
	if numSections != uint64(len(sos)) {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: Expected %d sections, got %d sections", len(sos), numSections), FormatError, fallbackURL}
	}

	sectionsStart := uint64(len(bs) - r.Len())

	if len(sos) == 0 || sos[len(sos)-1].Name != "responses" {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: Last section is not \"responses\""), FormatError, fallbackURL}
	}

	meta := &meta{
		version:        ver,
		primaryURL:     fallbackURL,
		sectionOffsets: sos,
		sectionsStart:  sectionsStart,
	}

	offset := sectionsStart

	for _, so := range sos {
		if _, exists := knownSections[so.Name]; !exists {
			continue
		}
		if so.Name == "responses" {
			continue
		}
		if uint64(len(bs)) <= offset {
			return nil, &LoadMetadataError{fmt.Errorf("bundle: section %q's computed offset %q out-of-range.", so.Name, offset), FormatError, fallbackURL}
		}
		end := offset + so.Length
		if uint64(len(bs)) <= end {
			return nil, &LoadMetadataError{fmt.Errorf("bundle: section %q's end %q out-of-range.", so.Name, end), FormatError, fallbackURL}
		}

		sectionContents := bs[offset:end]

		switch so.Name {
		case "index":
			if ver.SupportsVariants() {
				requests, err := parseIndexSectionWithVariants(sectionContents, sectionsStart, sos)
				if err != nil {
					return nil, &LoadMetadataError{err, FormatError, fallbackURL}
				}
				meta.requests = requests
			} else {
				requests, err := parseIndexSection(sectionContents, sectionsStart, sos)
				if err != nil {
					return nil, &LoadMetadataError{err, FormatError, fallbackURL}
				}
				meta.requests = requests
			}
		case "primary":
			primaryURL, err := parsePrimarySection(sectionContents)
			if err != nil {
				return nil, &LoadMetadataError{err, FormatError, fallbackURL}
			}
			fallbackURL = primaryURL
			meta.primaryURL = primaryURL
		case "manifest":
			manifestURL, err := parseManifestSection(sectionContents)
			if err != nil {
				return nil, &LoadMetadataError{err, FormatError, fallbackURL}
			}
			meta.manifestURL = manifestURL
		case "signatures":
			if ver.SupportsSignatures() {
				signatures, err := parseSignaturesSection(sectionContents)
				if err != nil {
					return nil, &LoadMetadataError{err, FormatError, fallbackURL}
				}
				meta.signatures = signatures
			} else {
				return nil, &LoadMetadataError{errors.New("bundle: signatures section not allowed in this version of bundle"), FormatError, fallbackURL}
			}
		case "responses":
			continue
		default:
			return nil, &LoadMetadataError{fmt.Errorf("bundle: unknown section: %q", so.Name), FormatError, fallbackURL}
		}

		offset = end
	}

	return meta, nil
}

var reStatus = regexp.MustCompile("^\\d\\d\\d$")

// https://wicg.github.io/webpackage/draft-yasskin-dispatch-bundled-exchanges.html#load-response
func loadResponse(req requestEntryWithOffset, bs []byte) (Response, error) {
	r := bytes.NewBuffer(bs[req.Offset : req.Offset+req.Length])

	b, err := r.ReadByte()
	if err != nil {
		return Response{}, fmt.Errorf("bundle: Failed to read first byte of the encoded response: %v", err)
	}
	if b != 0x82 {
		return Response{}, fmt.Errorf("bundle: The first byte of the encoded response is %x, expected 0x82", b)
	}

	dec := cbor.NewDecoder(r)
	headerCborBytes, err := dec.DecodeByteString()
	if err != nil {
		return Response{}, fmt.Errorf("bundle: Failed to decode response header cbor bytestring: %v", err)
	}

	rhdr := bytes.NewBuffer(headerCborBytes)
	dechdr := cbor.NewDecoder(rhdr)
	headers, pseudos, err := decodeCborHeaders(dechdr)
	if err != nil {
		return Response{}, fmt.Errorf("bundle.response headerCbor: %v", err)
	}

	status, exists := pseudos[":status"]
	if !exists {
		return Response{}, fmt.Errorf("bundle.response headerCbor: pseudos don't have a key named \":status\"")
	}
	if len(pseudos) != 1 {
		return Response{}, fmt.Errorf("bundle.response headerCbor: len(pseudos) is %d, expected to be 1", len(pseudos))
	}

	if !reStatus.MatchString(status) {
		return Response{}, fmt.Errorf("bundle.response headerCbor: pseudos['status'] %q invalid", status)
	}

	body, err := dec.DecodeByteString()
	if err != nil {
		return Response{}, fmt.Errorf("bundle.response.body: %v", err)
	}

	if r.Len() != 0 {
		return Response{}, fmt.Errorf("bundle.response: invalid request stream end")
	}

	nstatus, err := strconv.Atoi(status)
	if err != nil {
		panic(err)
	}

	res := Response{
		Status: nstatus,
		Header: headers,
		Body: body,
	}

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

	b := &Bundle{Version: m.version, PrimaryURL: m.primaryURL, Exchanges: es, ManifestURL: m.manifestURL, Signatures: m.signatures}
	return b, nil
}
