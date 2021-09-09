package bundle

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/WICG/webpackage/go/bundle/version"
	"github.com/WICG/webpackage/go/internal/testhelper"
)

func urlMustParse(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}

func TestVariants(t *testing.T) {
	variants, err := parseVariants("Accept-Encoding;gzip;br, Accept-Language;en;fr;ja")
	if err != nil {
		t.Errorf("parseListOfStringLists unexpectedly failed: %v", err)
	}
	if nk, err := variants.numberOfPossibleKeys(); nk != 6 || err != nil {
		t.Errorf("numberOfPossibleKeys: got: (%v, %v) want: (%v, %v)", nk, err, 6, nil)
	}

	cases := []struct {
		index      int
		variantKey []string
	}{
		{0, []string{"gzip", "en"}},
		{1, []string{"gzip", "fr"}},
		{2, []string{"gzip", "ja"}},
		{3, []string{"br", "en"}},
		{4, []string{"br", "fr"}},
		{5, []string{"br", "ja"}},
		{-1, []string{"gzip", "es"}},
		{-1, []string{}},
		{-1, []string{"gzip"}},
		{-1, []string{"gzip", "en", "foo"}},
	}
	for _, c := range cases {
		if i := variants.indexInPossibleKeys(c.variantKey); i != c.index {
			t.Errorf("indexInPossibleKeys: got: %v want: %v", i, c.index)
		}

		if c.index != -1 {
			key := variants.possibleKeyAt(c.index)
			if !reflect.DeepEqual(key, c.variantKey) {
				t.Errorf("possibleKeyAt(%d): got: %v\nwant: %v", c.index, key, c.variantKey)
			}
		}
	}
}

func TestIndexSectionWithVariants(t *testing.T) {
	url := urlMustParse("https://example.com/")
	variants := []string{"Accept-Encoding;gzip;br, Accept-Language;en;fr"}
	is := &indexSection{}
	is.addExchange(
		&Exchange{
			Request{URL: url},
			Response{Header: http.Header{
				"Variants":    variants,
				"Variant-Key": []string{"gzip;fr, br;en"},
			}},
		}, 20, 2)
	is.addExchange(
		&Exchange{
			Request{URL: url},
			Response{Header: http.Header{
				"Variants":    variants,
				"Variant-Key": []string{"gzip;en"},
			}},
		}, 10, 1)
	is.addExchange(
		&Exchange{
			Request{URL: url},
			Response{Header: http.Header{
				"Variants":    variants,
				"Variant-Key": []string{"br;fr"},
			}},
		}, 30, 3)
	if err := is.Finalize(version.VersionB1); err != nil {
		t.Fatal(err)
	}

	want := `map["https://example.com/":["Accept-Encoding;gzip;br, Accept-Language;en;fr" 10 1 20 2 20 2 30 3]]`

	got, err := testhelper.CborBinaryToReadableString(is.bytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got: %s\nwant: %s", got, want)
	}
}

func TestIndexSectionMultipleResourcesPerURL(t *testing.T) {
	url := urlMustParse("https://example.com/")
	is := &indexSection{}
	is.addExchange(
		&Exchange{
			Request{URL: url},
			Response{},
		}, 10, 1)
	is.addExchange(
		&Exchange{
			Request{URL: url},
			Response{},
		}, 20, 1)
	err := is.Finalize(version.VersionB2)
	if err.Error() != "This WebBundle version 'b2' does not support variants, so we cannot have multiple resources per URL." {
		t.Fatal(err)
	}
}