// Package structuredheader parses Structured Headers for HTTP
// (https://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html).
package structuredheader

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#identifier
type Identifier string

// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#param
type ParameterisedList []*ParameterisedIdentifier
type ParameterisedIdentifier struct {
	Label  Identifier
	Params Parameters
}

// Parameter represents a set of parameters in a Parameterised Identifier.
// The interface{} value is one of these:
//
//	int64      for Numbers
//	string     for Strings
//	Identifier for Identifiers
//	[]byte     for Byte Sequences
//	nil        for parameters with no value
type Parameters map[Identifier]interface{}

type parser struct {
	input string
}

func (p *parser) discardLeadingOWS() {
	p.input = strings.TrimLeft(p.input, " \t")
}

func (p *parser) isEmpty() bool {
	return len(p.input) == 0
}

func (p *parser) getChar() byte {
	c := p.input[0]
	p.input = p.input[1:]
	return c
}

func (p *parser) getString(n int) string {
	s := p.input[:n]
	p.input = p.input[n:]
	return s
}

func (p *parser) consumeChar(c byte) bool {
	if len(p.input) > 0 && p.input[0] == c {
		p.input = p.input[1:]
		return true
	}
	return false
}

// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#parse-param-list
func ParseParameterisedList(input string) (ParameterisedList, error) {
	p := &parser{input}
	// In the spec, this is done in the Step 1 of the top-level parsing algorithm.
	// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#text-parse
	p.discardLeadingOWS()

	var items ParameterisedList
	for !p.isEmpty() {
		item, err := p.parseParameterisedIdentifier()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		p.discardLeadingOWS()
		if p.isEmpty() {
			return items, nil
		}
		if !p.consumeChar(',') {
			return nil, fmt.Errorf("',' expacted, got '%c'", input[0])
		}
		p.discardLeadingOWS()
	}
	return nil, errors.New("unexpected end of input; Parameterised Identifier expected")
}

// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#parse-param-id
func (p *parser) parseParameterisedIdentifier() (*ParameterisedIdentifier, error) {
	primary_identifier, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	parameters := make(Parameters)
	for {
		// This is not in the spec algorithm but ABNF allows OWS here.
		// https://github.com/httpwg/http-extensions/issues/703
		p.discardLeadingOWS()

		if !p.consumeChar(';') {
			break
		}
		p.discardLeadingOWS()
		param_name, err := p.parseIdentifier()
		if err != nil {
			return nil, err
		}
		if _, ok := parameters[param_name]; ok {
			return nil, fmt.Errorf("duplicated parameter '%s'", param_name)
		}
		var param_value interface{}
		if p.consumeChar('=') {
			param_value, err = p.parseItem()
			if err != nil {
				return nil, err
			}
		}
		parameters[param_name] = param_value
	}
	return &ParameterisedIdentifier{primary_identifier, parameters}, nil
}

// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#parse-item
func (p *parser) parseItem() (interface{}, error) {
	// The spec algorithm has "Discard OWS" here but ABNF doesn't permit leading
	// OWS for items. https://github.com/httpwg/http-extensions/issues/703

	if p.isEmpty() {
		return nil, errors.New("Item expected, got EOS")
	}
	c := p.input[0]
	if c == '-' || isDigit(c) {
		return p.parseNumber()
	}
	if c == '"' {
		return p.parseString()
	}
	if c == '*' {
		return p.parseByteSequence()
	}
	// TODO: Support Booleans.
	if isLCAlpha(c) {
		return p.parseIdentifier()
	}
	return nil, fmt.Errorf("Item expected, got '%c'", c)
}

// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#parse-number
func (p *parser) parseNumber() (int64, error) {
	if p.isEmpty() {
		return 0, errors.New("Number expected, got EOS")
	}
	if p.input[0] != '-' && !isDigit(p.input[0]) {
		return 0, fmt.Errorf("Number expected, got '%c'", p.input[0])
	}
	// TODO: Support Floats.
	i := 1
	for i < len(p.input) && isDigit(p.input[i]) {
		i++
	}
	s := p.getString(i)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("couldn't parse %q as number: %v", s, err)
	}
	return n, nil
}

// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#parse-string
func (p *parser) parseString() (string, error) {
	if p.isEmpty() {
		return "", errors.New("String expected, got EOS")
	}
	if !p.consumeChar('"') {
		return "", fmt.Errorf("'\"' expected, got '%c'", p.input[0])
	}

	var b strings.Builder
	for !p.isEmpty() {
		c := p.getChar()
		if c == '\\' {
			if p.isEmpty() {
				break
			}
			c = p.getChar()
			if c != '"' && c != '\\' {
				return "", fmt.Errorf("invalid escape \\%c", c)
			}
			b.WriteByte(c)
		} else if c == '"' {
			return b.String(), nil
		} else if c < ' ' || c > '~' {
			return "", fmt.Errorf("invalid character \\x%02x", c)
		} else {
			b.WriteByte(c)
		}
	}
	return "", errors.New("missing closing '\"'")
}

// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#parse-identifier
func (p *parser) parseIdentifier() (Identifier, error) {
	if p.isEmpty() {
		return "", errors.New("Identifier expected, got EOS")
	}
	if !isLCAlpha(p.input[0]) {
		return "", fmt.Errorf("Identifier expected, got '%c'", p.input[0])
	}
	i := 0
	for i < len(p.input) && isIdent(p.input[i]) {
		i++
	}
	id := Identifier(p.getString(i))
	return id, nil
}

// http://httpwg.org/http-extensions/draft-ietf-httpbis-header-structure.html#parse-binary
func (p *parser) parseByteSequence() ([]byte, error) {
	if p.isEmpty() {
		return nil, errors.New("Byte Sequence expected, got EOS")
	}
	if !p.consumeChar('*') {
		return nil, fmt.Errorf("'*' expected, got '%c'", p.input[0])
	}
	len := strings.IndexByte(p.input, '*')
	if len < 0 {
		return nil, errors.New("missing closing '*'")
	}
	s := p.getString(len)
	enc := base64.StdEncoding
	if len%4 != 0 {
		// Allow unpadded encoding.
		enc = base64.RawStdEncoding
	}
	data, err := enc.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("couldn't decode base64 %q: %v", s, err)
	}
	if !p.consumeChar('*') {
		panic("cannot happen")
	}
	return data, nil
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isLCAlpha(c byte) bool {
	return c >= 'a' && c <= 'z'
}

// isIdent returns true if c is allowed in subsequent characters of Identifiers.
func isIdent(c byte) bool {
	return isLCAlpha(c) || isDigit(c) || c == '_' || c == '-' || c == '*' || c == '/'
}
