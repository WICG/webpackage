package structuredheader

import (
	"bytes"
	"reflect"
	"testing"
)

func TestParseNumber(t *testing.T) {
	cases := []struct {
		input         string
		shouldSucceed bool
		expected      int64
		rest          string
	}{
		{"", false, 0, ""},
		{"42", true, 42, ""},
		{"-42", true, -42, ""},
		{"42rest", true, 42, "rest"},
		{"--2", false, 0, ""},
		{"-", false, 0, ""},
		{"4-2", true, 4, "-2"},
		{" 42", false, 0, ""},
		{" 42", false, 0, ""},
		{"0", true, 0, ""},
		{"-0", true, 0, ""},
		{"9223372036854775807", true, 9223372036854775807, ""},   // int64 max
		{"-9223372036854775808", true, -9223372036854775808, ""}, // int64 min
		{"9223372036854775808", false, 0, ""},                    // int64 max + 1
		{"-9223372036854775809", false, 0, ""},                   // int64 min - 1
	}
	for _, c := range cases {
		p := &parser{c.input}
		n, err := p.parseNumber()
		if c.shouldSucceed {
			if err != nil {
				t.Errorf("parseNumber(%q) unexpectedly failed: %v", c.input, err)
			}
			if n != c.expected {
				t.Errorf("parseNumber(%q): got %v, want %v", c.input, n, c.expected)
			}
			if p.input != c.rest {
				t.Errorf("parseNumber(%q): remaining input was %q, should be %q", c.input, p.input, c.rest)
			}
		} else if err == nil {
			t.Errorf("parseNumber(%q) did not fail", c.input)
		}
	}
}

func TestParseString(t *testing.T) {
	cases := []struct {
		input         string
		shouldSucceed bool
		expected      string
		rest          string
	}{
		{``, false, "", ""},
		{`"`, false, "", ""},
		{`""`, true, "", ""},
		{`"abc"`, true, "abc", ""},
		{`"abc`, false, "", ""},
		{`abc"`, false, "", ""},
		{`"abc"def`, true, "abc", "def"},
		{`"abc\"def"`, true, "abc\"def", ""},
		{`"abc\\def"`, true, "abc\\def", ""},
		{`"abc\`, false, "", ""},
		{`"abc\"`, false, "", ""},
		{`"\n"`, false, "", ""},
		{"\"\u2318\"", false, "", ""},
	}
	for _, c := range cases {
		p := &parser{c.input}
		s, err := p.parseString()
		if c.shouldSucceed {
			if err != nil {
				t.Errorf("parseString(%q) unexpectedly failed: %v", c.input, err)
			}
			if s != c.expected {
				t.Errorf("parseString(%q): got %v, want %v", c.input, s, c.expected)
			}
			if p.input != c.rest {
				t.Errorf("parseString(%q): remaining input was %q, should be %q", c.input, p.input, c.rest)
			}
		} else if err == nil {
			t.Errorf("parseString(%q) did not fail", c.input)
		}
	}
}

func TestParseKey(t *testing.T) {
	cases := []struct {
		input         string
		shouldSucceed bool
		expected      Key
		rest          string
	}{
		{"", false, "", ""},
		{"a", true, "a", ""},
		{"foo", true, "foo", ""},
		{"Foo", false, "", ""},
		{"foo123", true, "foo123", ""},
		{"1foo", false, "", ""},
		{"foo_-*/", true, "foo_-", "*/"},
		{"foo=bar", true, "foo", "=bar"},
		{"_foo", false, "", ""},
	}
	for _, c := range cases {
		p := &parser{c.input}
		id, err := p.parseKey()
		if c.shouldSucceed {
			if err != nil {
				t.Errorf("parseKey(%q) unexpectedly failed: %v", c.input, err)
			}
			if id != c.expected {
				t.Errorf("parseKey(%q): got %v, want %v", c.input, id, c.expected)
			}
			if p.input != c.rest {
				t.Errorf("parseKey(%q): remaining input was %q, should be %q", c.input, p.input, c.rest)
			}
		} else if err == nil {
			t.Errorf("parseKey(%q) did not fail", c.input)
		}
	}
}

func TestParseToken(t *testing.T) {
	cases := []struct {
		input         string
		shouldSucceed bool
		expected      Token
		rest          string
	}{
		{"", false, "", ""},
		{"a", true, "a", ""},
		{"foo", true, "foo", ""},
		{"Foo", true, "Foo", ""},
		{"foo123", true, "foo123", ""},
		{"1foo", false, "", ""},
		{"foo_-*/", true, "foo_-*/", ""},
		{"foo=bar", true, "foo", "=bar"},
		{"_foo", false, "", ""},
	}
	for _, c := range cases {
		p := &parser{c.input}
		id, err := p.parseToken()
		if c.shouldSucceed {
			if err != nil {
				t.Errorf("parseToken(%q) unexpectedly failed: %v", c.input, err)
			}
			if id != c.expected {
				t.Errorf("parseToken(%q): got %v, want %v", c.input, id, c.expected)
			}
			if p.input != c.rest {
				t.Errorf("parseToken(%q): remaining input was %q, should be %q", c.input, p.input, c.rest)
			}
		} else if err == nil {
			t.Errorf("parseToken(%q) did not fail", c.input)
		}
	}
}

func TestParseByteSequence(t *testing.T) {
	cases := []struct {
		input         string
		shouldSucceed bool
		expected      []byte
		rest          string
	}{
		{"", false, nil, ""},
		{"*", false, nil, ""},
		{"**", true, []byte{}, ""},
		{"*Zm9v*", true, []byte("foo"), ""},
		{"*Zm9v", false, nil, ""},
		{"*Zm9v**", true, []byte("foo"), "*"},
		{"*Zm9_*", false, nil, ""},
		{"*aG9nZQ==*", true, []byte("hoge"), ""},
		{"*aG9nZQ*", true, []byte("hoge"), ""},
		{"*aG9nZQ=*", false, nil, ""},
	}
	for _, c := range cases {
		p := &parser{c.input}
		b, err := p.parseByteSequence()
		if c.shouldSucceed {
			if err != nil {
				t.Errorf("parseByteSequence(%q) unexpectedly failed: %v", c.input, err)
			}
			if !bytes.Equal(b, c.expected) {
				t.Errorf("parseByteSequence(%q): got %v, want %v", c.input, b, c.expected)
			}
			if p.input != c.rest {
				t.Errorf("parseByteSequence(%q): remaining input was %q, should be %q", c.input, p.input, c.rest)
			}
		} else if err == nil {
			t.Errorf("parseByteSequence(%q) did not fail", c.input)
		}
	}
}

func TestParseItem(t *testing.T) {
	cases := []struct {
		input    string
		expected Item
		rest     string
	}{
		{"", nil, ""},
		{"42", int64(42), ""},
		{"foo", Token("foo"), ""},
		{`" foo "`, " foo ", ""},
		{"*Zm9v*;", []byte("foo"), ";"},
	}
	for _, c := range cases {
		p := &parser{c.input}
		r, err := p.parseItem()
		if c.expected != nil {
			if err != nil {
				t.Errorf("parseItem(%q) unexpectedly failed: %v", c.input, err)
			}
			if !reflect.DeepEqual(r, c.expected) {
				t.Errorf("parseItem(%q): got %v, want %v", c.input, r, c.expected)
			}
			if p.input != c.rest {
				t.Errorf("parseItem(%q): remaining input was %q, should be %q", c.input, p.input, c.rest)
			}
		} else if err == nil {
			t.Errorf("parseItem(%q) did not fail", c.input)
		}
	}
}

func TestParseParameterisedIdentifier(t *testing.T) {
	cases := []struct {
		input    string
		expected *ParameterisedIdentifier
		rest     string
	}{
		{"", nil, ""},
		{"label", &ParameterisedIdentifier{"label", Parameters{}}, ""},
		{";foo", nil, ""},
		{"label;foo", &ParameterisedIdentifier{"label", Parameters{"foo": nil}}, ""},
		{"label;foo=bar;n=42", &ParameterisedIdentifier{"label", Parameters{"foo": Token("bar"), "n": int64(42)}}, ""},
		{"label;n=123;", nil, ""},
		{"label ; n=123 ; m=42", &ParameterisedIdentifier{"label", Parameters{"n": int64(123), "m": int64(42)}}, ""},
		{"label;n =123", &ParameterisedIdentifier{"label", Parameters{"n": nil}}, "=123"},
		{"label;n= 123", nil, ""},
	}
	for _, c := range cases {
		p := &parser{c.input}
		r, err := p.parseParameterisedIdentifier()
		if c.expected != nil {
			if err != nil {
				t.Errorf("parseParameterisedIdentifier(%q) unexpectedly failed: %v", c.input, err)
			}
			if !reflect.DeepEqual(r, *c.expected) {
				t.Errorf("parseParameterisedIdentifier(%q): got %v, want %v", c.input, r, c.expected)
			}
			if p.input != c.rest {
				t.Errorf("parseParameterisedIdentifier(%q): remaining input was %q, should be %q", c.input, p.input, c.rest)
			}
		} else if err == nil {
			t.Errorf("parseParameterisedIdentifier(%q) did not fail", c.input)
		}
	}
}

func TestParseListOfLists(t *testing.T) {
	cases := []struct {
		input    string
		expected ListOfLists
	}{
		{"", nil},
		{"1;2", ListOfLists{[]Item{int64(1), int64(2)}}},
		{"1;2,foo;bar", ListOfLists{[]Item{int64(1), int64(2)}, []Item{Token("foo"), Token("bar")}}},
		{" 1 ; 2 , foo ; bar ", ListOfLists{[]Item{int64(1), int64(2)}, []Item{Token("foo"), Token("bar")}}},
		{"42,", nil},
		{",42", nil},
		{"1;2;", nil},
		{";2", nil},
		{"1;;2", nil},
		{"1,,2", nil},
	}
	for _, c := range cases {
		r, err := ParseListOfLists(c.input)
		if c.expected != nil {
			if err != nil {
				t.Errorf("ParseListOfLists(%q) unexpectedly failed: %v", c.input, err)
			}
			if !reflect.DeepEqual(r, c.expected) {
				t.Errorf("ParseListOfLists(%q): got %v, want %v", c.input, r, c.expected)
			}
		} else if err == nil {
			t.Errorf("ParseListOfLists(%q) did not fail", c.input)
		}
	}
}

func TestParseParameterisedList(t *testing.T) {
	cases := []struct {
		input    string
		expected ParameterisedList
	}{
		{"", nil},
		{"item1;n=123", ParameterisedList{{"item1", Parameters{"n": int64(123)}}}},
		{"item1;n=123,", nil},
		{",item1;n=123", nil},
		{"item1;n=123,item2,item3;n=456", ParameterisedList{
			{"item1", Parameters{"n": int64(123)}},
			{"item2", Parameters{}},
			{"item3", Parameters{"n": int64(456)}}}},
		{" \t item1 , item2;n=123 ", ParameterisedList{
			{"item1", Parameters{}},
			{"item2", Parameters{"n": int64(123)}}}},
	}
	for _, c := range cases {
		r, err := ParseParameterisedList(c.input)
		if c.expected != nil {
			if err != nil {
				t.Errorf("ParseParameterisedList(%q) unexpectedly failed: %v", c.input, err)
			}
			if !reflect.DeepEqual(r, c.expected) {
				t.Errorf("ParseParameterisedList(%q): got %v, want %v", c.input, r, c.expected)
			}
		} else if err == nil {
			t.Errorf("ParseParameterisedList(%q) did not fail", c.input)
		}
	}
}
