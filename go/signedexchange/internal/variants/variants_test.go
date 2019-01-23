package variants_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/WICG/webpackage/go/signedexchange/internal/variants"
)

func TestParseListOfList(t *testing.T) {
	cases := []struct {
		Name string
		In   string
		Out  [][]string
	}{
		{
			Name: "basic list of lists",
			In:   "1;2, 42;43",
			Out:  [][]string{{"1", "2"}, {"42", "43"}},
		},
		{
			Name: "empty list of lists",
			In:   "",
			Out:  nil,
		},
		{
			Name: "single item list of lists",
			In:   "42",
			Out:  [][]string{{"42"}},
		},
		{
			Name: "no whitespace list of lists",
			In:   "1,42",
			Out:  [][]string{{"1"}, {"42"}},
		},
		{
			Name: "no inner whitespace list of lists",
			In:   "1;2, 42;43",
			Out:  [][]string{{"1", "2"}, {"42", "43"}},
		},
		{
			Name: "extra whitespace list of lists",
			In:   "1 , 42",
			Out:  [][]string{{"1"}, {"42"}},
		},
		{
			Name: "extra inner whitespace list of lists",
			In:   "1 ; 2,42 ; 43",
			Out:  [][]string{{"1", "2"}, {"42", "43"}},
		},
		/*{
			Name: "trailing comma list of lists",
			In:   "1;2, 42,",
			Out:  nil,
		},*/
		{
			Name: "trailing semicolon list of lists",
			In:   "1;2, 42;43;",
			Out:  nil,
		},
		/*{
			Name: "empty item list of lists",
			In:   "1,,42",
			Out:  nil,
		},*/
		{
			Name: "empty inner item list of lists",
			In:   "1;;2,42",
			Out:  nil,
		},
	}
	for _, c := range cases {
		header := http.Header{}
		header.Add("foo", c.In)
		got := variants.ParseListOfLists(header, "foo")
		want := c.Out
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got: %v, want: %v", c.Name, got, want)
		}
	}
}

func TestCacheBehavior(t *testing.T) {
	cases := []struct {
		Variants [][]string
		Request  map[string]string
		Want     [][]string
	}{
		{
			Variants: [][]string{{"Accept-Language", "en", "ja"}},
			Request: map[string]string{
				"Accept-Language": "en",
			},
			Want: [][]string{{"en"}},
		},
	}
	for _, c := range cases {
		req := http.Header{}
		for k, v := range c.Request {
			req.Add(k, v)
		}
		got := variants.CacheBehavior(c.Variants, req)
		want := c.Want
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}
}

func TestMatchRequest(t *testing.T) {
	cases := []struct {
		Name     string
		Request  map[string]string
		Response map[string]string
		Want     bool
	}{
		{
			Name: "no variants",
			Request: map[string]string{
				"accept": "text/html",
			},
			Want: true,
		},
		{
			Name: "has variants but no varient key",
			Request: map[string]string{
				"accept": "text/html",
			},
			Response: map[string]string{
				"variants": "Accept; text/html",
			},
			Want: false,
		},
		{
			Name: "has variant key but no varients",
			Request: map[string]string{
				"accept": "text/html",
			},
			Response: map[string]string{
				"variant-key": "text/html",
			},
			Want: false,
		},
		{
			Name:    "bad variant key item length",
			Request: nil,
			Response: map[string]string{
				"variants":    "Accept;text/html, Accept-Language;en;fr",
				"variant-key": "text/html;en, text/html;fr;oops",
			},
			Want: false,
		},
		{
			Name: "content type",
			Request: map[string]string{
				"accept": "text/html",
			},
			Response: map[string]string{
				"variants":    "Accept; text/html",
				"variant-key": "text/html",
			},
			Want: true,
		},
		{
			Name: "client supports two content types",
			Request: map[string]string{
				"accept": "text/html,image/jpeg",
			},
			Response: map[string]string{
				"variants":    "Accept; text/html",
				"variant-key": "text/html",
			},
			Want: true,
		},
		{
			Name: "variant key miss",
			Request: map[string]string{
				"accept": "image/jpeg",
			},
			Response: map[string]string{
				"variants":    "Accept; text/html; image/jpeg",
				"variant-key": "text/html",
			},
			Want: false,
		},
		{
			Name: "format miss",
			Request: map[string]string{
				"accept": "image/jpeg",
			},
			Response: map[string]string{
				"variants":    "Accept; text/html",
				"variant-key": "text/html",
			},
			Want: true,
		},
		{
			Name:    "no format preference",
			Request: nil,
			Response: map[string]string{
				"variants":    "Accept; text/html",
				"variant-key": "text/html",
			},
			Want: true,
		},
		{
			Name: "language",
			Request: map[string]string{
				"accept-language": "en",
			},
			Response: map[string]string{
				"variants":    "Accept-Language; en",
				"variant-key": "en",
			},
			Want: true,
		},
		{
			Name: "language no match",
			Request: map[string]string{
				"accept-language": "ja",
			},
			Response: map[string]string{
				"variants":    "Accept-Language; en;ja",
				"variant-key": "en",
			},
			Want: false,
		},
		{
			Name: "language multiple",
			Request: map[string]string{
				"accept-language": "en,ja",
			},
			Response: map[string]string{
				"variants":    "Accept-Language;en;fr",
				"variant-key": "en",
			},
			Want: true,
		},
		{
			Name: "language multiple no match",
			Request: map[string]string{
				"accept-language": "en,ja",
			},
			Response: map[string]string{
				"variants":    "Accept-Language;en;fr",
				"variant-key": "fr",
			},
			Want: false,
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			req := http.Header{}
			for k, v := range c.Request {
				req.Add(k, v)
			}
			res := http.Header{}
			for k, v := range c.Response {
				res.Add(k, v)
			}

			got := variants.MatchRequest(req, res)
			want := c.Want
			if got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}
