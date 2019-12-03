package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mrichman/hargo"

	"github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/signedexchange"
)

func ReadHar(r io.Reader) (*hargo.Har, error) {
	dec := json.NewDecoder(r)
	var har hargo.Har
	if err := dec.Decode(&har); err != nil {
		return nil, fmt.Errorf("Failed to parse har. err: %v", err)
	}
	return &har, nil
}

func ReadHarFromFile(path string) (*hargo.Har, error) {
	fi, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open input file %q for reading. err: %v", path, err)
	}
	defer fi.Close()
	return ReadHar(fi)
}

func nvpToHeader(nvps []hargo.NVP, predBanned func(string) bool) (http.Header, error) {
	h := make(http.Header)
	for _, nvp := range nvps {
		// Drop HTTP/2 pseudo headers.
		if strings.HasPrefix(nvp.Name, ":") {
			continue
		}
		if predBanned(nvp.Name) {
			log.Printf("Dropping banned header: %q", nvp.Name)
			continue
		}
		h.Add(nvp.Name, nvp.Value)
	}
	return h, nil
}

func contentToBody(c *hargo.Content) ([]byte, error) {
	if c.Encoding == "base64" {
		return base64.StdEncoding.DecodeString(c.Text)
	}
	return []byte(c.Text), nil
}

func fromHar(harPath string) ([]*bundle.Exchange, error) {
	har, err := ReadHarFromFile(harPath)
	if err != nil {
		return nil, err
	}

	es := []*bundle.Exchange{}
	hasVariants := make(map[string]bool)

	for _, e := range har.Log.Entries {
		log.Printf("Processing entry: %q", e.Request.URL)

		parsedUrl, err := url.Parse(e.Request.URL) // TODO(kouhei): May be this should e.Respose.RedirectURL?
		if err != nil {
			return nil, fmt.Errorf("Failed to parse request URL %q. err: %v", e.Request.URL, err)
		}
		reqh, err := nvpToHeader(e.Request.Headers, signedexchange.IsStatefulRequestHeader)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse request header for the request %q. err: %v", e.Request.URL, err)
		}
		resh, err := nvpToHeader(e.Response.Headers, signedexchange.IsUncachedHeader)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse response header for the request %q. err: %v", e.Request.URL, err)
		}
		body, err := contentToBody(&e.Response.Content)
		if err != nil {
			return nil, fmt.Errorf("Failed to extract body from response content for the request %q. err: %v", e.Request.URL, err)
		}

		if e.Request.Method != http.MethodGet {
			log.Printf("Dropping the entry: non-GET request method (%s)", e.Request.Method)
			continue
		}
		if e.Response.Status < 100 || e.Response.Status > 999 {
			log.Printf("Dropping the entry: invalid response status (%d)", e.Response.Status)
			continue
		}

		// Allow multiple entries for single URL only if all responses have
		// Variants: header.
		_, thisHasVariants := resh["Variants"]
		othersHaveVariants, hasMultipleEntries := hasVariants[parsedUrl.String()]
		if hasMultipleEntries && (!thisHasVariants || !othersHaveVariants) {
			log.Printf("Dropping the entry: exchange for this URL already exists, and has no Variants header")
			continue
		}
		hasVariants[parsedUrl.String()] = thisHasVariants

		e := &bundle.Exchange{
			Request: bundle.Request{
				URL:    parsedUrl,
				Header: reqh,
			},
			Response: bundle.Response{
				Status: e.Response.Status,
				Header: resh,
				Body:   body,
			},
		}
		es = append(es, e)
	}

	return es, nil
}
