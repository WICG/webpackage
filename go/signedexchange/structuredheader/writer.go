package structuredheader

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func (ll ListOfLists) String() (string, error) {
	var b strings.Builder
	if err := ll.serialize(&b); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (pl ParameterisedList) String() (string, error) {
	var b strings.Builder
	if err := pl.serialize(&b); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (pi *ParameterisedIdentifier) String() (string, error) {
	var b strings.Builder
	if err := pi.serialize(&b); err != nil {
		return "", err
	}
	return b.String(), nil
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.1.3
func (ll ListOfLists) serialize(out *strings.Builder) error {
	if len(ll) == 0 {
		return errors.New("structuredheader: empty List of Lists")
	}

	outerSep := ""
	for _, list := range ll {
		if len(list) == 0 {
			return errors.New("structuredheader: empty inner list in List of Lists")
		}

		out.WriteString(outerSep)
		outerSep = ", "

		innerSep := ""
		for _, item := range list {
			out.WriteString(innerSep)
			innerSep = "; "

			if err := serializeItem(item, out); err != nil {
				return err
			}
		}
	}
	return nil
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.1.4
func (pl ParameterisedList) serialize(out *strings.Builder) error {
	if len(pl) == 0 {
		return errors.New("structuredheader: empty Parameterised List")
	}

	sep := ""
	for _, pi := range pl {
		out.WriteString(sep)
		sep = ", "
		if err := pi.serialize(out); err != nil {
			return err
		}
	}
	return nil
}

// Step 2.1-2.3 of
// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.1.4
func (pi *ParameterisedIdentifier) serialize(out *strings.Builder) error {
	if !isValidToken(string(pi.Label)) {
		return fmt.Errorf("structuredheader: label %q is not a valid token", pi.Label)
	}
	out.WriteString(string(pi.Label))

	// Format in sorted order for reproducibility.
	var keys []string
	for k := range pi.Params {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	for _, k := range keys {
		out.WriteByte(';')
		if !isValidKey(k) {
			return fmt.Errorf("structuredheader: invalid key %q", k)
		}
		out.WriteString(k)
		val := pi.Params[Key(k)]
		if val != nil {
			out.WriteByte('=')
			if err := serializeItem(val, out); err != nil {
				return err
			}
		}
	}
	return nil
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.1.5
func serializeItem(i Item, out *strings.Builder) error {
	switch v := i.(type) {
	case int64:
		// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.1.6
		out.WriteString(strconv.FormatInt(v, 10))
		return nil

	case string:
		// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.1.8
		for _, c := range v {
			if c < ' ' || c > '~' {
				return fmt.Errorf("structuredheader: couldn't serialize %q as string", v)
			}
		}
		out.WriteString(strconv.Quote(v))
		return nil

	case Token:
		// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.1.9
		if !isValidToken(string(v)) {
			return fmt.Errorf("structuredheader: couldn't serialize %q as token", v)
		}
		out.WriteString(string(v))
		return nil

	case []byte:
		// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.1.10
		out.WriteByte('*')
		out.WriteString(base64.StdEncoding.EncodeToString(v))
		out.WriteByte('*')
		return nil

	default:
		return fmt.Errorf("structuredheader: couldn't serialize %v as item", i)
	}
}

func isValidKey(s string) bool {
	if len(s) == 0 || !isLCAlpha(s[0]) {
		return false
	}
	for _, c := range s {
		if c > unicode.MaxASCII || !isKeyChar(byte(c)) {
			return false
		}
	}
	return true
}

func isValidToken(s string) bool {
	if len(s) == 0 || !isAlpha(s[0]) {
		return false
	}
	for _, c := range s {
		if c > unicode.MaxASCII || !isTokenChar(byte(c)) {
			return false
		}
	}
	return true
}
