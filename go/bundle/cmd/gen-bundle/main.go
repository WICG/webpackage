package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/mrichman/hargo"

	"github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/signedexchange"
)

var (
	flagInput  = flag.String("i", "in.har", "HTTP Archive (HAR) input file")
	flagOutput = flag.String("o", "out.webbundle", "Webbundle output file")
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

func nvpToHeader(nvps []hargo.NVP, isStatefulHeader func(string) bool) (http.Header, error) {
	h := make(http.Header)
	for _, nvp := range nvps {
		if isStatefulHeader(nvp.Name) {
			log.Printf("Dropping banned header: %q", nvp.Name)
			continue
		}
		h.Add(nvp.Name, nvp.Value)
	}
	return h, nil
}

func contentToPayload(c *hargo.Content) ([]byte, error) {
	if c.Encoding == "base64" {
		return base64.StdEncoding.DecodeString(c.Text)
	}
	return []byte(c.Text), nil
}

func run() error {
	har, err := ReadHarFromFile(*flagInput)
	if err != nil {
		return err
	}

	fo, err := os.OpenFile(*flagOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Failed to open output file %q for writing. err: %v", *flagOutput, err)
	}
	defer fo.Close()

	es := []*signedexchange.Exchange{}

	for _, e := range har.Log.Entries {
		log.Printf("Processing entry: %q", e.Request.URL)

		parsedUrl, err := url.Parse(e.Request.URL) // TODO(kouhei): May be this should e.Respose.RedirectURL?
		if err != nil {
			return fmt.Errorf("Failed to parse request URL %q. err: %v", e.Request.URL, err)
		}
		reqh, err := nvpToHeader(e.Request.Headers, signedexchange.IsStatefulRequestHeader)
		if err != nil {
			return fmt.Errorf("Failed to parse request header for the request %q. err: %v", e.Request.URL, err)
		}
		resh, err := nvpToHeader(e.Response.Headers, signedexchange.IsStatefulResponseHeader)
		if err != nil {
			return fmt.Errorf("Failed to parse response header for the request %q. err: %v", e.Request.URL, err)
		}
		payload, err := contentToPayload(&e.Response.Content)
		if err != nil {
			return fmt.Errorf("Failed to extract payload from response content for the request %q. err: %v", e.Request.URL, err)
		}

		se, err := signedexchange.NewExchange(parsedUrl, reqh, e.Response.Status, resh, payload)
		if err != nil {
			return err
		}
		es = append(es, se)
	}

	b := &bundle.Bundle{Exchanges: es}

	if _, err := b.WriteTo(fo); err != nil {
		return fmt.Errorf("Failed to write exchange. err: %v", err)
	}
	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
