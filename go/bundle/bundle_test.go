package bundle_test

import (
	"bytes"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	. "github.com/WICG/webpackage/go/bundle"
)

func urlMustParse(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}

func createTestBundle() *Bundle {
	return &Bundle{
		Exchanges: []*Exchange{
			&Exchange{
				Request{
					URL:    urlMustParse("https://bundle.example.com/"),
					Header: make(http.Header),
				},
				Response{
					Status: 200,
					Header: http.Header{"Content-Type": []string{"text/html"}},
					Body:   []byte("hello, world!"),
				},
			},
		},
	}
}

func TestWriteAndRead(t *testing.T) {
	bundle := createTestBundle()

	var buf bytes.Buffer
	n, err := bundle.WriteTo(&buf)
	if err != nil {
		t.Errorf("Bundle.WriteTo unexpectedly failed: %v", err)
	}
	if n != int64(buf.Len()) {
		t.Errorf("Bundle.WriteTo returned %d, but wrote %d bytes", n, buf.Len())
	}

	deserialized, err := Read(&buf)
	if err != nil {
		t.Errorf("Bundle.Read unexpectedly failed: %v", err)
	}
	if !reflect.DeepEqual(deserialized, bundle) {
		t.Errorf("got: %v\nwant: %v", deserialized, bundle)
	}
}
