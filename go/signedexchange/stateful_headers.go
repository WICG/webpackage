package signedexchange

import (
	"strings"
)

// https://jyasskin.github.io/webpackage/implementation-draft/draft-yasskin-httpbis-origin-signed-exchanges-impl.html#stateful-headers.
var (
	statefulRequestHeaders = map[string]struct{}{
		"authorization":       struct{}{},
		"cookie":              struct{}{},
		"cookie2":             struct{}{},
		"proxy-authorization": struct{}{},
		"sec-websocket-key":   struct{}{},
	}
	statefulResponseHeaders = map[string]struct{}{
		"authentication-control":    struct{}{},
		"authentication-info":       struct{}{},
		"optional-www-authenticate": struct{}{},
		"proxy-authenticate":        struct{}{},
		"proxy-authentication-info": struct{}{},
		"sec-webSocket-accept":      struct{}{},
		"set-cookie":                struct{}{},
		"set-cookie2":               struct{}{},
		"setprofile":                struct{}{},
		"www-authenticate":          struct{}{},
	}
)

func IsStatefulRequestHeader(name string) bool {
	cname := strings.ToLower(name)
	_, exists := statefulRequestHeaders[cname]
	return exists
}

func IsStatefulResponseHeader(name string) bool {
	cname := strings.ToLower(name)
	_, exists := statefulResponseHeaders[cname]
	return exists
}
