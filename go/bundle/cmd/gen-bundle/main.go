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
	"strings"

	"github.com/mrichman/hargo"

	"github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/signedexchange"
)

var (
	flagHar         = flag.String("har", "", "HTTP Archive (HAR) input file")
	flagDir         = flag.String("dir", "", "Input directory")
	flagBaseURL     = flag.String("baseURL", "", "Base URL")
	flagStartURL    = flag.String("startURL", "", "Entry point URL (relative from -baseURL)")
	flagManifestURL = flag.String("manifestURL", "", "Manifest URL (relative from -baseURL)")
	flagOutput      = flag.String("o", "out.wbn", "Webbundle output file")
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

func fromHar(harPath string) error {
	har, err := ReadHarFromFile(harPath)
	if err != nil {
		return err
	}

	fo, err := os.OpenFile(*flagOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Failed to open output file %q for writing. err: %v", *flagOutput, err)
	}
	defer fo.Close()

	es := []*bundle.Exchange{}

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
		resh, err := nvpToHeader(e.Response.Headers, signedexchange.IsUncachedHeader)
		if err != nil {
			return fmt.Errorf("Failed to parse response header for the request %q. err: %v", e.Request.URL, err)
		}
		body, err := contentToBody(&e.Response.Content)
		if err != nil {
			return fmt.Errorf("Failed to extract body from response content for the request %q. err: %v", e.Request.URL, err)
		}

		if e.Request.Method != http.MethodGet {
			log.Printf("Dropping the entry: non-GET request method (%s)", e.Request.Method)
			continue
		}
		if e.Response.Status < 100 || e.Response.Status > 999 {
			log.Printf("Dropping the entry: invalid response status (%d)", e.Response.Status)
			continue
		}

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

	b := &bundle.Bundle{Exchanges: es}

	if _, err := b.WriteTo(fo); err != nil {
		return fmt.Errorf("Failed to write exchange. err: %v", err)
	}
	return nil
}

func main() {
	flag.Parse()
	if *flagHar != "" {
		if *flagBaseURL == "" {
			fmt.Fprintln(os.Stderr, "Warning: -baseURL is ignored when input is HAR.")
		}
		if *flagStartURL == "" {
			fmt.Fprintln(os.Stderr, "Warning: -startURL is ignored when input is HAR.")
		}
		if err := fromHar(*flagHar); err != nil {
			log.Fatal(err)
		}
	} else if *flagDir != "" {
		if *flagBaseURL == "" {
			fmt.Fprintln(os.Stderr, "Please specify -baseURL.")
			flag.Usage()
			return
		}
		if err := fromDir(*flagDir, *flagBaseURL, *flagStartURL, *flagManifestURL); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Please specify -har or -dir.")
		flag.Usage()
	}
}
