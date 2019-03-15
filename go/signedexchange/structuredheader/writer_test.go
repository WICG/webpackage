package structuredheader_test

import (
	. "github.com/WICG/webpackage/go/signedexchange/structuredheader"
	"testing"
)

func TestSerializeItem(t *testing.T) {
	cases := []struct {
		input    Item
		expected string // empty if serialization should fail
	}{
		{int64(42), "42"},
		{int64(9223372036854775807), "9223372036854775807"},   // int64 max
		{int64(-9223372036854775808), "-9223372036854775808"}, // int64 min
		{"string", `"string"`},
		{"", `""`},
		{"\x7f", ""},
		{"\x1f", ""},
		{`foo"bar`, `"foo\"bar"`},
		{`foo\bar`, `"foo\\bar"`},
		{`foo\"bar`, `"foo\\\"bar"`},
		{" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_abcdefghijklmnopqrstuvwxyz{|}~", `" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_abcdefghijklmnopqrstuvwxyz{|}~"`},
		{"\u65e5\u672c\u8a9e", ""},
		{Token("foo"), "foo"},
		{Token("BAR"), "BAR"},
		{Token("A123_-.:%*/"), "A123_-.:%*/"},
		{Token("123"), ""},
		{Token("_foo"), ""},
		{Token(""), ""},
		{[]byte{}, "**"},
		{[]byte("foo"), "*Zm9v*"},
		{[]byte("hoge"), "*aG9nZQ==*"},
	}
	for _, c := range cases {
		s, err := ListOfLists{{c.input}}.String()
		if c.expected != "" {
			if err != nil {
				t.Errorf("Unexpetedly failed to serialize %v: %v", c.input, err)
			}
			if s != c.expected {
				t.Errorf("%v should serialize to %q, but got %q", c.input, c.expected, s)
			}
		} else {
			if err == nil {
				t.Errorf("serialization of %v did not fail", c.input)
			}
		}
	}
}

func TestSerializeListOfLists(t *testing.T) {
	cases := []struct {
		input    ListOfLists
		expected string // empty if serialization should fail
	}{
		{ListOfLists{{}}, ""},
		{ListOfLists{{int64(42)}}, "42"},
		{ListOfLists{{int64(42)}, {}}, ""},
		{ListOfLists{{int64(42), Token("foo")}}, "42; foo"},
		{ListOfLists{{int64(42)}, {Token("foo")}}, "42, foo"},
		{ListOfLists{{int64(42), int64(-42)}, {Token("foo"), Token("bar")}}, "42; -42, foo; bar"},
	}
	for _, c := range cases {
		s, err := c.input.String()
		if c.expected != "" {
			if err != nil {
				t.Errorf("Unexpetedly failed to serialize %v: %v", c.input, err)
			}
			if s != c.expected {
				t.Errorf("%v should serialize to %q, but got %q", c.input, c.expected, s)
			}
		} else {
			if err == nil {
				t.Errorf("serialization of %v did not fail", c.input)
			}
		}
	}
}

func TestSerializeParameterisedList(t *testing.T) {
	cases := []struct {
		input    ParameterisedList
		expected string // empty if serialization should fail
	}{
		{ParameterisedList{}, ""},
		{ParameterisedList{{"label", Parameters{}}}, "label"},
		{ParameterisedList{{"_label", Parameters{}}}, ""},
		{ParameterisedList{{"item1", Parameters{"n": int64(123)}}}, "item1;n=123"},
		{ParameterisedList{
			{"item1", Parameters{"n": int64(123)}},
			{"item2", Parameters{}},
			{"item3", Parameters{"n": int64(456)}},
		}, "item1;n=123, item2, item3;n=456"},
		{ParameterisedList{{"item1", Parameters{"InvalidKey": int64(123)}}}, ""},
	}
	for _, c := range cases {
		s, err := c.input.String()
		if c.expected != "" {
			if err != nil {
				t.Errorf("Unexpetedly failed to serialize %v: %v", c.input, err)
			}
			if s != c.expected {
				t.Errorf("%v should serialize to %q, but got %q", c.input, c.expected, s)
			}
		} else {
			if err == nil {
				t.Errorf("serialization of %v did not fail", c.input)
			}
		}
	}
}
