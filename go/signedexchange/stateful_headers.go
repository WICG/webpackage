package signedexchange

import (
	"fmt"
	"net/http"
	"strings"
)

// https://jyasskin.github.io/webpackage/implementation-draft/draft-yasskin-httpbis-origin-signed-exchanges-impl.html#stateful-headers.
var statefulRequestHeadersSet map[string]struct{}
var uncachedHeadersSet map[string]struct{}

func init() {
	statefulRequestHeaders := []string{
		"authorization",
		"cookie",
		"cookie2",
		"proxy-authorization",
		"sec-websocket-key",
	}
	statefulRequestHeadersSet = make(map[string]struct{})
	for _, e := range statefulRequestHeaders {
		statefulRequestHeadersSet[e] = struct{}{}
	}

	uncachedHeaders := []string{
		// "Hop-by-hop header fields listed in the Connection header field
		// (Section 6.1 of {{!RFC7230}})." [spec text]
		// Note: The Connection header field itself is banned as uncached headers, so no-op.

		// "Header fields listed in the no-cache response directive in the
		// "Cache-Control header field (Section 5.2.2.2 of {{!RFC7234}})."
		// [spec text]
		// Note: This is to be handled specifically in VerifyUncachedHeader, but
		// is not currently implemented.

		// "Header fields defined as hop-by-hop" [spec text] and the entries from
		// the spec.
		"connection",
		"keep-alive",
		"proxy-connection",
		"trailer",
		"transfer-encoding",
		"upgrade",

		// "Stateful headers" [spec text]
		// draft-yasskin-http-origin-signed-responses.html#stateful-headers
		"authentication-control",
		"authentication-info",
		"clear-site-data",
		"optional-www-authenticate",
		"proxy-authenticate",
		"proxy-authentication-info",
		"public-key-pins",
		"sec-websocket-accept",
		"set-cookie",
		"set-cookie2",
		"setprofile",
		"strict-transport-security",
		"www-authenticate",
	}
	uncachedHeadersSet = make(map[string]struct{})
	for _, e := range uncachedHeaders {
		uncachedHeadersSet[e] = struct{}{}
	}
}

// IsStatefulRequestHeader returns true if the HTTP header n is considered stateful and is not allowed to be included in a signed exchange
// Note that this only applies to signed exchanges of versions 1b1 and 1b2.
func IsStatefulRequestHeader(n string) bool {
	cname := strings.ToLower(n)
	_, exists := statefulRequestHeadersSet[cname]
	return exists
}

func IsUncachedHeader(n string) bool {
	cname := strings.ToLower(n)
	_, exists := uncachedHeadersSet[cname]
	return exists
}

// VerifyUncachedHeader returns non-nil error if h has any uncached header fields as specified in
// draft-yasskin-http-origin-signed-responses.html#uncached-headers
func VerifyUncachedHeader(h http.Header) error {
	// TODO: Implement https://tools.ietf.org/html/rfc7234#section-5.2.2.2

	for n := range h {
		if IsUncachedHeader(n) {
			return fmt.Errorf("signedexchange: uncached header %q can't be captured inside a signed exchange.", n)
		}
	}
	return nil
}
