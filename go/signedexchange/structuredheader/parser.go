// Package structuredheader parses Structured Headers for HTTP
// (https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09).
package structuredheader

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-3.1
type Key string

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-3.3
type ListOfLists [][]Item

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-3.4
type ParameterisedList []ParameterisedIdentifier
type ParameterisedIdentifier struct {
	Label  Token
	Params Parameters
}
type Parameters map[Key]Item

// Item is one of these:
//
//	int64  for Numbers
//	string for Strings
//	Token  for Tokens
//	[]byte for Byte Sequences
//
// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-3.5
type Item interface{}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-3.9
type Token string


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

// ParseListOfLists parses input as a List of Lists.
// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2
func ParseListOfLists(input string) (ListOfLists, error) {
	p := &parser{input}
	p.discardLeadingOWS()
	ll, err := p.parseListOfLists()
	if err != nil {
		return nil, err
	}
	p.discardLeadingOWS()
	if !p.isEmpty() {
		return nil, errors.New("structuredheader: extraneous data at the end")
	}
	return ll, nil
}

// ParseParameterisedList parses input as a Parameterised List.
// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2
func ParseParameterisedList(input string) (ParameterisedList, error) {
	p := &parser{input}
	p.discardLeadingOWS()
	pl, err := p.parseParameterisedList()
	if err != nil {
		return nil, err
	}
	p.discardLeadingOWS()
	if !p.isEmpty() {
		return nil, errors.New("structuredheader: extraneous data at the end")
	}
	return pl, nil
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2.2
func (p *parser) parseKey() (Key, error) {
	if p.isEmpty() {
		return "", errors.New("structuredheader: token expected, got EOS")
	}
	if !isLCAlpha(p.input[0]) {
		return "", fmt.Errorf("structuredheader: token expected, got '%c'", p.input[0])
	}
	i := 0
	for i < len(p.input) && isKeyChar(p.input[i]) {
		i++
	}
	return Key(p.getString(i)), nil
}
// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2.4
func (p *parser) parseListOfLists() (ListOfLists, error) {
	var topList ListOfLists
	var innerList []Item
	for !p.isEmpty() {
		item, err := p.parseItem()
		if err != nil {
			return nil, err
		}
		innerList = append(innerList, item)
		p.discardLeadingOWS()
		if p.isEmpty() {
			topList = append(topList, innerList)
			return topList, nil
		}
		if p.consumeChar(',') {
			topList = append(topList, innerList)
			innerList = nil
		} else if !p.consumeChar(';') {
			return nil, fmt.Errorf("structuredheader: ',' or ';' expected, got '%c'", p.input[0])
		}
		p.discardLeadingOWS()
	}
	return nil, errors.New("structuredheader: unexpected end of input; List of Lists expected")
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2.5
func (p *parser) parseParameterisedList() (ParameterisedList, error) {
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
			return nil, fmt.Errorf("structuredheader: ',' expacted, got '%c'", p.input[0])
		}
		p.discardLeadingOWS()
	}
	return nil, errors.New("structuredheader: unexpected end of input; Parameterised Identifier expected")
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2.6
func (p *parser) parseParameterisedIdentifier() (ParameterisedIdentifier, error) {
	primaryIdentifier, err := p.parseToken()
	if err != nil {
		return ParameterisedIdentifier{}, err
	}
	parameters := make(Parameters)
	for {
		p.discardLeadingOWS()
		if !p.consumeChar(';') {
			break
		}
		p.discardLeadingOWS()
		paramName, err := p.parseKey()
		if err != nil {
			return ParameterisedIdentifier{}, err
		}
		if _, ok := parameters[paramName]; ok {
			return ParameterisedIdentifier{}, fmt.Errorf("structuredheader: duplicated parameter '%s'", paramName)
		}
		var paramValue Item
		if p.consumeChar('=') {
			paramValue, err = p.parseItem()
			if err != nil {
				return ParameterisedIdentifier{}, err
			}
		}
		parameters[paramName] = paramValue
	}
	return ParameterisedIdentifier{primaryIdentifier, parameters}, nil
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2.7
func (p *parser) parseItem() (Item, error) {
	if p.isEmpty() {
		return nil, errors.New("structuredheader: item expected, got EOS")
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
	if isAlpha(c) {
		return p.parseToken()
	}
	return nil, fmt.Errorf("structuredheader: item expected, got '%c'", c)
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2.8
func (p *parser) parseNumber() (int64, error) {
	if p.isEmpty() {
		return 0, errors.New("structuredheader: number expected, got EOS")
	}
	if p.input[0] != '-' && !isDigit(p.input[0]) {
		return 0, fmt.Errorf("structuredheader: number expected, got '%c'", p.input[0])
	}
	// TODO: Support Floats.
	i := 1
	for i < len(p.input) && isDigit(p.input[i]) {
		i++
	}
	s := p.getString(i)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("structuredheader: couldn't parse %q as number: %v", s, err)
	}
	return n, nil
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2.9
func (p *parser) parseString() (string, error) {
	if p.isEmpty() {
		return "", errors.New("structuredheader: string expected, got EOS")
	}
	if !p.consumeChar('"') {
		return "", fmt.Errorf("structuredheader: '\"' expected, got '%c'", p.input[0])
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
				return "", fmt.Errorf("structuredheader: invalid escape \\%c", c)
			}
			b.WriteByte(c)
		} else if c == '"' {
			return b.String(), nil
		} else if c < ' ' || c > '~' {
			return "", fmt.Errorf("structuredheader: invalid character \\x%02x", c)
		} else {
			b.WriteByte(c)
		}
	}
	return "", errors.New("structuredheader: missing closing '\"'")
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2.10
func (p *parser) parseToken() (Token, error) {
	if p.isEmpty() {
		return "", errors.New("structuredheader: token expected, got EOS")
	}
	if !isAlpha(p.input[0]) {
		return "", fmt.Errorf("structuredheader: token expected, got '%c'", p.input[0])
	}
	i := 0
	for i < len(p.input) && isTokenChar(p.input[i]) {
		i++
	}
	return Token(p.getString(i)), nil
}

// https://tools.ietf.org/html/draft-ietf-httpbis-header-structure-09#section-4.2.11
func (p *parser) parseByteSequence() ([]byte, error) {
	if p.isEmpty() {
		return nil, errors.New("structuredheader: byte sequence expected, got EOS")
	}
	if !p.consumeChar('*') {
		return nil, fmt.Errorf("structuredheader: '*' expected, got '%c'", p.input[0])
	}
	len := strings.IndexByte(p.input, '*')
	if len < 0 {
		return nil, errors.New("structuredheader: missing closing '*'")
	}
	s := p.getString(len)
	enc := base64.StdEncoding
	if len%4 != 0 {
		// Allow unpadded encoding.
		enc = base64.RawStdEncoding
	}
	data, err := enc.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("structuredheader: couldn't decode base64 %q: %v", s, err)
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

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isKeyChar returns true if c is allowed in subsequent characters of Keys.
func isKeyChar(c byte) bool {
	return isLCAlpha(c) || isDigit(c) || c == '_' || c == '-'
}

// isTokenChar returns true if c is allowed in subsequent characters of Tokens.
func isTokenChar(c byte) bool {
	return isAlpha(c) || isDigit(c) || c == '_' || c == '-' || c == '.' || c == ':' || c == '%' || c == '*' || c == '/'
}
