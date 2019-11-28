package testhelper

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ugorji/go/codec"
)

// readableString converts an arbitrary value v to a string.
//
// readableString does a basically same thing as fmt.Sprintf("%q", v), but
// the difference is that map keys are ordered in alphabetical order so that
// the results are deterministic.
func readableString(v interface{}) string {
	switch v := v.(type) {
	case []interface{}:
		vals := []string{}
		for _, val := range v {
			vals = append(vals, readableString(val))
		}
		return "[" + strings.Join(vals, " ") + "]"
	case map[interface{}]interface{}:
		keys := []string{}
		// Assume that keys are strings.
		for k := range v {
			keys = append(keys, k.(string))
		}
		sort.Strings(keys)
		vals := []string{}
		for _, k := range keys {
			val := v[k]
			vals = append(vals, fmt.Sprintf("%q:", k)+readableString(val))
		}
		return "map[" + strings.Join(vals, " ") + "]"
	case string, []byte:
		return fmt.Sprintf("%q", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	default:
		panic(fmt.Sprintf("not supported type: %T", v))
	}
}

// CborBinaryToReadableString converts a CBOR binary to a readable string.
func CborBinaryToReadableString(b []byte) (string, error) {
	r := bytes.NewReader(b)

	var decoded interface{}
	handle := &codec.CborHandle{}
	if err := codec.NewDecoder(r, handle).Decode(&decoded); err != nil {
		return "", err
	}
	return readableString(decoded), nil
}
