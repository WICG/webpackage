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

// https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#index-section
func parseIndexSection(sectionContents []byte, sectionsStart uint64, sos []sectionOffset) ([]requestEntryWithOffset, error) {
	// Step 1. "Let index be the result of parsing sectionContents as a CBOR item matching the index rule in the above CDDL (Section 3.5). If index is an error, return an error." [spec text]
	dec := cbor.NewDecoder(bytes.NewBuffer(sectionContents))
	numUrls, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, fmt.Errorf("bundle.index: failed to decode index section map header: %v", err)
	}
	// Step 2. "Let requests be an initially-empty map ([INFRA]) from URLs to response descriptions, each of which is either a single location-in-stream value or a pair of a Variants header field value ([I-D.ietf-httpbis-variants]) and a map from that value's possible Variant-Keys to location-in-stream values, as described in Section 2.2." [spec text]
	requests := []requestEntryWithOffset{}

	// Step 3. "Let MakeRelativeToStream be a function that takes a location-in-responses value (offset, length) and returns a ResponseMetadata struct or error by running the following sub-steps:" [spec text]
	respso, respSectionRelOffset, found := FindSection(sos, "responses")
	if !found {
		return nil, fmt.Errorf("bundle.index: \"responses\" section not found")
	}
	respSectionOffset := sectionsStart + respSectionRelOffset
	makeRelativeToStream := func(offset, length uint64) (uint64, uint64, error) {
		// Step 3.1. "If offset + length is larger than sectionOffsets["responses"].length, return an error." [spec text]
		if offset+length > respso.Length {
			return 0, 0, errors.New("bundle.index: response length out-of-range")
		}
		// Step 3.2. "Otherwise, return a ResponseMetadata struct whose offset is sectionOffsets["responses"].offset + offset and whose length is length." [spec text]
		return respSectionOffset + offset, length, nil
	}

	// Step 4. "For each (url, responses) entry in the index map:" [spec text]
	for i := uint64(0); i < numUrls; i++ {
		// Step 4.1. "Let parsedUrl be the result of parsing ([URL]) url with no base URL." [spec text]
		rawUrl, err := dec.DecodeTextString()
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to decode map key: %v", i, err)
		}
		parsedUrl, err := url.Parse(rawUrl)
		// Step 4.2. "If parsedUrl is a failure, its fragment is not null, or it includes credentials, return an error." [spec text]
		if err != nil {
			return nil, fmt.Errorf("bundle.index[%d]: Failed to parse URL: %v", i, err)
		}
		if parsedUrl.Fragment != "" {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains fragment: %q", i, rawUrl)
		}
		if parsedUrl.User != nil {
			return nil, fmt.Errorf("bundle.index[%d]: URL contains credentials: %q", i, rawUrl)
		}

		// Step 4.3. "If the first element of responses is the empty string:" [spec text]
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
			// Step 4.3.1. "If the length of responses is not 3 (i.e. there is more than one location-in-responses in responses), return an error." [spec text]
			if numItems != 3 {
				return nil, fmt.Errorf("bundle.index[%d]: The size of value array must be 3", i)
			}
			// Step 4.3.2. "Otherwise, assert that requests[parsedUrl] does not exist, and set requests[parsedUrl] to MakeRelativeToStream(location-in-responses), where location-in-responses is the second and third elements of responses. If that returns an error, return an error." [spec text]
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
			// Step 4.4. "Otherwise:" [spec text]
			// Step 4.4.1. "Let variants be the result of parsing the first element of responses as the value of the Variants HTTP header field (Section 2 of [I-D.ietf-httpbis-variants]). If this fails, return an error." [spec text]
			variants, err := parseVariants(string(variants_value))
			if err != nil {
				return nil, fmt.Errorf("bundle.index[%d]: Cannot parse variants value %q: %v", i, string(variants_value), err)
			}
			// Step 4.4.2. "Let variantKeys be the Cartesian product of the lists of available-values for each variant-axis in lexicographic (row-major) order. See the examples above." [spec text]
			numVariantKeys, err := variants.numberOfPossibleKeys()
			if err != nil {
				return nil, fmt.Errorf("bundle.index[%d]: Invalid variants value %q: %v", i, string(variants_value), err)
			}
			// Step 4.4.3. "If the length of responses is not 2 * len(variantKeys) + 1, return an error." [spec text]
			if numItems != 2*uint64(numVariantKeys)+1 {
				return nil, fmt.Errorf("bundle.index[%d]: Unexpected size of value array: %d", i, numItems)
			}
			// Step 4.4.4. "Set requests[parsedUrl] to a map from variantKeys[i] to the result of calling MakeRelativeToStream on the location-in-responses at responses[2*i+1] and responses[2*i+2], for i in [0, len(variantKeys)). If any MakeRelativeToStream call returns an error, return an error." [spec text]
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
	// Step 1. "Seek to offset 0 in stream. Assert: this operation doesn't fail." [spec text]

	r := bytes.NewBuffer(bs)

	// Step 2. "If reading 10 bytes from stream returns an error or doesn't return the bytes with hex encoding "86 48 F0 9F 8C 90 F0 9F 93 A6" (the CBOR encoding of the 6-item array initial byte and 8-byte bytestring initial byte, followed by 🌐📦 in UTF-8) or "85 48 F0 9F 8C 90 F0 9F 93 A6" (same as before, but a 5-item CBOR-array), return a "format error"." [spec text]
	// Step 3. "Let version be the result of reading 5 bytes from stream. If this is an error, return a "format error"." [spec text]
	ver, err := version.ParseMagicBytes(r)
	// TODO(ksakamoto): Continue and return VersionError after parsing fallbackUrl.
	if err != nil {
		return nil, &LoadMetadataError{err, FormatError, nil}
	}

	var fallbackURL *url.URL
	dec := cbor.NewDecoder(r)
	if ver.HasPrimaryURLFieldInHeader() {
		// Step 4. "Let urlType and urlLength be the result of reading the type and argument of a CBOR item from stream (Section 3.5.3). If this is an error or urlType is not 3 (a CBOR text string), return a "format error"." [spec text]
		// Step 5. "Let fallbackUrlBytes be the result of reading urlLength bytes from stream. If this is an error, return a "format error"." [spec text]
		fallbackURLBytes, err := dec.DecodeTextString()
		if err != nil {
			return nil, &LoadMetadataError{fmt.Errorf("bundle: Failed to read fallbackURL string: %v", err), FormatError, nil}
		}
		// Step 6. "Let fallbackUrl be the result of parsing ([URL]) the UTF-8 decoding of fallbackUrlBytes with no base URL. If either the UTF-8 decoding or parsing fails, return a "format error". " [spec text]
		fallbackURL, err = url.Parse(fallbackURLBytes)
		if err != nil {
			return nil, &LoadMetadataError{fmt.Errorf("bundle: Failed to parse fallbackURL: %v", err), FormatError, nil}
		}
	}

	// Step 7. "If version does not have the hex encoding "44 62 31 00 00"/"44 62 32 00 00" (the CBOR encoding of a 4-byte byte string holding an ASCII "b1"/"b2" followed by two 0 bytes), return a "version error" with fallbackUrl. " [spec text]
	// This is checked inside version.ParseMagicBytes(r) above.

	// Step 8. "Let sectionLengthsLength be the result of getting the length of the CBOR bytestring header from stream (Section 3.5.2). If this is an error, return a "format error" with fallbackUrl." [spec text]
	// Step 9. "If sectionLengthsLength is 8192 (8*1024) or greater, return a "format error" with fallbackUrl." [spec text]
	// Step 10. "Let sectionLengthsBytes be the result of reading sectionLengthsLength bytes from stream. If sectionLengthsBytes is an error, return a "format error" with fallbackUrl." [spec text]
	slbytes, err := dec.DecodeByteString()
	if err != nil {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: Failed to read sectionLengths byte string: %v", err), FormatError, fallbackURL}
	}
	if len(slbytes) >= 8192 {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: sectionLengthsLength is too long (%d bytes)", slbytes), FormatError, fallbackURL}
	}

	// Step 11. "Let sectionLengths be the result of parsing one CBOR item (Section 3.5) from sectionLengthsBytes, matching the section-lengths rule in the CDDL ([CDDL]) above. If sectionLengths is an error, return a "format error" with fallbackUrl." [spec text]
	sos, err := decodeSectionLengthsCBOR(slbytes)
	if err != nil {
		return nil, &LoadMetadataError{err, FormatError, fallbackURL}
	}

	// Step 12. "Let (sectionsType, numSections) be the result of parsing the type and argument of a CBOR item from stream." [spec text]
	numSections, err := dec.DecodeArrayHeader()
	// Step 13. "If sectionsType is not 4 (a CBOR array) or..." [spec text]
	if err != nil {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: Failed to read section header."), FormatError, fallbackURL}
	}
	// "numSections is not half of the length of sectionLengths, return a "format error" with fallbackUrl." [spec text]
	if numSections != uint64(len(sos)) {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: Expected %d sections, got %d sections", len(sos), numSections), FormatError, fallbackURL}
	}

	// Step 14. "Let sectionsStart be the current offset within stream" [spec text]
	sectionsStart := uint64(len(bs) - r.Len())

	// Step 15. "Let knownSections be the subset of the Section 6.2 that this client has implemented." [spec text]
	// Step 16. "Let ignoredSections be an empty set." [spec text]

	// Step 17. "Let sectionOffsets be an empty map from section names to (offset, length) pairs. These offsets are relative to the start of stream." [spec text]
	// Note: We store this on "sos"

	// Step 18. "Let currentOffset be sectionsStart"
	// currentOffset := sectionsStart

	// Step 19. "For each ("name", length) pair of adjacent elements in sectionLengths:" [spec text]
	// for _, so := range sos {
	// Step 19.1 "If "name"'s specification in knownSections says not to process other sections, add those sections' names to ignoredSections." [spec text]
	// Not implemented

	// Step 19.2-19.4 implemented inside decodeSectionLengthsCBOR()
	// }

	// Step 20. "If the "responses" section is not last in sectionLengths, return a "format error" with fallbackUrl." [spec text]
	if len(sos) == 0 || sos[len(sos)-1].Name != "responses" {
		return nil, &LoadMetadataError{fmt.Errorf("bundle: Last section is not \"responses\""), FormatError, fallbackURL}
	}

	// Step 21. "Let metadata be a map ([INFRA]) initially containing the single key/value pair "primaryUrl"/fallbackUrl" if the version matches b1. [spec text]
	// Note: We use a struct rather than a map here.
	meta := &meta{
		version:        ver,
		primaryURL:     fallbackURL,
		sectionOffsets: sos,
		sectionsStart:  sectionsStart,
	}

	offset := sectionsStart

	// Step 22. "For each "name" -> (offset, length) triple in sectionOffsets:" [spec text]
	for _, so := range sos {
		// Step 22.1. "If "name" isn't in knownSections, continue to the next triple." [spec text]
		if _, exists := knownSections[so.Name]; !exists {
			continue
		}
		// Step 22.2. "If "name"'s Metadata field is "No", continue to the next triple." [spec text]
		// Note: the "responses" section is currently the only section with its Metadata field "No".
		if so.Name == "responses" {
			continue
		}
		// Step 22.3. "If "name" is in ignoredSections, continue to the next triple." [spec text]
		// Note: Per discussion in #218, the step 12.3 is not implemented since it is no-op as of now.

		// Step 22.4. "Seek to offset offset in stream. If this fails, return a "format error" with fallbackUrl." [spec text]
		if uint64(len(bs)) <= offset {
			return nil, &LoadMetadataError{fmt.Errorf("bundle: section %q's computed offset %q out-of-range.", so.Name, offset), FormatError, fallbackURL}
		}
		end := offset + so.Length
		if uint64(len(bs)) <= end {
			return nil, &LoadMetadataError{fmt.Errorf("bundle: section %q's end %q out-of-range.", so.Name, end), FormatError, fallbackURL}
		}

		// Step 22.5. "Let sectionContents be the result of reading length bytes from stream. If sectionContents is an error, return a "format error" with fallbackUrl."
		sectionContents := bs[offset:end]
		//log.Printf("Section[%q] stream offset %x end %x", so.Name, offset, end)

		// Step 22.6. "Follow "name"'s specification from knownSections to process the section, passing sectionContents, stream, sectionOffsets, and metadata. If this returns an error, return a "format error" with fallbackUrl." [spec text]
		switch so.Name {
		case "index":
			if ver.SupportsVariants() {
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

	// Step 23. "Assert: metadata has an entry with the key "primaryUrl"." [spec text]
	// Step 24. "If metadata doesn't have entries with keys "requests" and "manifest", return a "format error" with fallbackUrl." [spec text]
	// FIXME

	// Step 25. Return metadata.
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

	b := &Bundle{Version: m.version, PrimaryURL: m.primaryURL, Exchanges: es, ManifestURL: m.manifestURL, Signatures: m.signatures}
	return b, nil
}
