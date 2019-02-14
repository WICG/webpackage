package variants

import (
	"net/http"
	"net/textproto"
	"strings"
	"sort"

	"github.com/golang/gddo/httputil/header"
)

func MatchesMimeType(mimeTypePattern, mimeType string) bool {
	// TODO(hajimehoshi): Implement this. See Chromium's MatchesMimeType.
	return mimeTypePattern == mimeType
}

func ParseListOfLists(h http.Header, key string) [][]string {
	lists := header.ParseList(h, key)

	var ll [][]string
	for _, list := range lists {
		var l []string
		for _, item := range strings.Split(list, ";") {
			item = strings.TrimSpace(item)
			if item == "" {
				return nil
			}
			l = append(l, item)
		}
		if len(l) == 0 {
			return nil
		}
		ll = append(ll, l)
	}
	return ll
}

func ParseRequestHeaderValue(h http.Header, key string) []string {
	var specs []header.AcceptSpec
	for _, spec := range header.ParseAccept(h, key) {
		if spec.Q == 0 {
			continue
		}
		specs = append(specs, spec)
	}

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Q > specs[j].Q
	})

	var items []string
	for _, spec := range specs {
		items = append(items, spec.Value)
	}
	return items
}

// https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html#variant-key
func ParseVariantKey(h http.Header, numVariantAxes int) [][]string {
	parsed := ParseListOfLists(h, "Variant-Key")
	if len(parsed) == 0 {
		return nil
	}

	// Each inner-list MUST have the same number of list-members as there are
	// variant-axes in the representation’s Variants header field. If not, the
	// client MUST treat the representation as having no Variant-Key header field.
	// [spec text]
	for _, list := range parsed {
		if len(list) != numVariantAxes {
			return nil
		}
	}
	return parsed
}

func matchedLanguages(availableValues []string, preferredLang string) []string {
	if preferredLang == "*" {
		return availableValues
	}

	var out []string
	prefix := strings.ToLower(preferredLang) + "-"
	for _, available := range availableValues {
		if strings.ToLower(preferredLang) == strings.ToLower(available) ||
			strings.HasPrefix(strings.ToLower(available), prefix) {
			out = append(out, available)
		}
	}
	return out
}

// https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html#content-type
func contentTypeNegotiation(availableValues []string, request http.Header, fieldName string) []string {
	// Step 1. Let preferred-available be an empty list. [spec text]
	var preferredAvailable []string

	// Step 2. Let preferred-types be a list of the types in the request-value (or
	// the empty list if request-value is null), ordered by their weight, highest
	// to lowest, as per Section 5.3.2 of [RFC7231] (omitting any coding with a
	// weight of 0). If a type lacks an explicit weight, an implementation MAY
	// assign one.
	var preferredTypes []string
	if request.Get(fieldName) != "" {
		preferredTypes = ParseRequestHeaderValue(request, fieldName)
	}

	// Step 3. For each preferred-type in preferred-types: [spec text]
	for _, preferredType := range preferredTypes {
		// 3.1. If any member of available-values matches preferred-type, using
		// the media-range matching mechanism specified in Section 5.3.2 of
		// [RFC7231] (which is case-insensitive), append those members of
		// available-values to preferred-available (preserving the precedence order
		// implied by the media ranges’ specificity).
		for _, available := range availableValues {
			if MatchesMimeType(preferredType, available) {
				preferredAvailable = append(preferredAvailable, available)
			}
		}
	}

	// Step 4. If preferred-available is empty, append the first member of
	// available-values to preferred-available. This makes the first
	// available-value the default when none of the client’s preferences are
	// available. [spec text]
	if len(preferredAvailable) == 0 && len(availableValues) > 0 {
		preferredAvailable = append(preferredAvailable, availableValues[0])
	}

	// Step 5. Return preferred-available. [spec text]
	return preferredAvailable
}

// https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html#content-language
func acceptLanguageNegotiation(availableValues []string, request http.Header, fieldName string) []string {
	// Step 1. Let preferred-available be an empty list. [spec text]
	var preferredAvailable []string

	// Step 2. Let preferred-langs be a list of the language-ranges in the
	// request-value (or the empty list if request-value is null), ordered by
	// their weight, highest to lowest, as per Section 5.3.1 of [RFC7231]
	// (omitting any language-range with a weight of 0). If a language-range lacks
	// a weight, an implementation MAY assign one. [spec text]
	var preferredLangs []string
	if request.Get(fieldName) != "" {
		preferredLangs = ParseRequestHeaderValue(request, fieldName)
	}

	// Step 3. For each preferred-lang in preferred-langs: [spec text]
	for _, preferredLang := range preferredLangs {
		// 3.1. If any member of available-values matches preferred-lang, using
		// either the Basic or Extended Filtering scheme defined in Section 3.3 of
		// [RFC4647], append those members of available-values to
		// preferred-available (preserving their order). [spec text]
		// TODO(ksakamoto): Use Basic or Extended Filtering scheme defined in
		// Section 3.3 of [RFC4647].
		preferredAvailable = append(preferredAvailable, matchedLanguages(availableValues, preferredLang)...)
	}

	// Step 4. If preferred-available is empty, append the first member of
	// available-values to preferred-available. This makes the first
	// available-value the default when none of the client’s preferences are
	// available. [spec text]
	if len(preferredAvailable) == 0 && len(availableValues) > 0 {
		preferredAvailable = append(preferredAvailable, availableValues[0])
	}

	// Step 5. Return preferred-available. [spec text]
	return preferredAvailable
}

// Implements "Cache Behaviour" [1] when "stored-responses" is a singleton list
// containing a response that has "Variants" header whose value is |variants|.
// [1] https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html#cache
func CacheBehavior(variants [][]string, request http.Header) [][]string {
	// 1. If stored-responses is empty, return an empty list. [spec text]
	// The size of stored-responses is always 1.

	// 2. Order stored-responses by the “Date” header field, most recent to
	// least recent. [spec text]
	// This is no-op because stored-responses is a single-element list.

	// 3. Let sorted-variants be an empty list. [spec text]
	var sortedVariants [][]string

	// 4. If the freshest member of stored-responses (as per [RFC7234], Section
	// 4.2) has one or more “Variants” header field(s) that successfully parse
	// according to Section 2: [spec text]

	// 4.1. Select one member of stored-responses with a “Variants” header
	// field-value(s) that successfully parses according to Section 2 and let
	// variants-header be this parsed value. This SHOULD be the most recent
	// response, but MAY be from an older one as long as it is still fresh.
	// [spec text]
	// |variants| is the parsed "Variants" header field value.

	// 4.2. For each variant-axis in variants-header: [spec text]
	for _, variantAxis := range variants {
		if len(variantAxis) == 0 {
			// TODO: Should this panic?
			return nil
		}

		// 4.2.1. If variant-axis’ field-name corresponds to the request header
		// field identified by a content negotiation mechanism that the
		// implementation supports: [spec text]
		fieldName := textproto.CanonicalMIMEHeaderKey(variantAxis[0])
		var algo func(availableValues []string, request http.Header, fieldName string) []string
		switch fieldName {
		case "Accept":
			algo = contentTypeNegotiation
		case "Accept-Language":
			algo = acceptLanguageNegotiation
		}

		if algo != nil {
			// 4.2.1.1. Let request-value be the field-value associated with
			// field-name in incoming-request (after being combined as allowed by
			// Section 3.2.2 of [RFC7230]), or null if field-name is not in
			// incoming-request. [spec text]
			if request.Get(fieldName) == "" {
				continue
			}

			// 4.2.1.2. Let sorted-values be the result of running the algorithm
			// defined by the content negotiation mechanism with request-value and
			// variant-axis’ available-values. [spec text]
			sortedValues := algo(variantAxis[1:], request, fieldName)

			// 4.2.1.3. Append sorted-values to sorted-variants. [spec text]
			sortedVariants = append(sortedVariants, sortedValues)
		}
	}

	// At this point, sorted-variants will be a list of lists, each member of the
	// top-level list corresponding to a variant-axis in the Variants header
	// field-value, containing zero or more items indicating available-values
	// that are acceptable to the client, in order of preference, greatest to
	// least. [spec text]

	// 5. Return result of running Compute Possible Keys (Section 4.1) on
	// sorted-variants, an empty list and an empty list. [spec text]
	// Instead of computing the cross product of sorted_variants, this
	// implementation just returns sorted_variants.
	return sortedVariants
}

// Implements step 3.- of
// https://wicg.github.io/webpackage/loading.html#request-matching
func MatchRequest(request, response http.Header) bool {
	v := response.Get("Variants")
	vk := response.Get("Variant-Key")

	// Step 3. If storedExchange’s response's header list contains:
	// - Neither a `Variants` nor a `Variant-Key` header
	//   Return "match".
	if v == "" && vk == "" {
		return true
	}

	// - A `Variant-Key` header but no `Variants` header
	//   Return "mismatch".
	if v == "" {
		return false
	}

	// - A `Variants` header but no `Variant-Key` header
	//   Return "mismatch".
	if vk == "" {
		return false
	}

	// - Both a `Variants` and a `Variant-Key` header
	//   Proceed to the following steps.

	// Step 4. If getting `Variants` from storedExchange’s response's header list
	// returns a value that fails to parse according to the instructions for the
	// Variants Header Field, return "mismatch".
	parsedVariants := ParseListOfLists(response, "Variants")
	if len(parsedVariants) == 0 {
		return false
	}

	// Step 5. Let acceptableVariantKeys be the result of running the Variants
	// Cache Behavior on an incoming-request of browserRequest and
	// stored-responses of a list containing storedExchange’s response.
	sortedVariants := CacheBehavior(parsedVariants, request)

	// Step 6. Let variantKeys be the result of getting `Variant-Key` from
	// storedExchange’s response's header list, and parsing it into a list of
	// lists as described in Variant-Key Header Field.
	parsedVariantKeys := ParseVariantKey(response, len(parsedVariants))

	// Step 7. If parsing variantKeys failed, return "mismatch".
	if len(parsedVariantKeys) == 0 {
		return false
	}

	// Step 8. If the intersection of acceptableVariantKeys and variantKeys is
	// empty, return "mismatch".
	for _, k := range parsedVariantKeys {
		i := 0
		for ; i < len(sortedVariants); i++ {
			if !includes(sortedVariants[i], k[i]) {
				break
			}
		}
		if i == len(sortedVariants) {
			// Step 9. Return "match".
			return true
		}
	}

	return false
}
