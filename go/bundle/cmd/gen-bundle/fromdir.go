package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/WICG/webpackage/go/bundle"
)

func fromDir(dir string, baseURL string, startURL string) error {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("Failed to parse base URL. err: %v", err)
	}
	parsedStartURL, err := parsedBaseURL.Parse(startURL)
	if err != nil {
		return fmt.Errorf("Failed to parse start URL. err: %v", err)
	}

	fo, err := os.OpenFile(*flagOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Failed to open output file %q for writing. err: %v", *flagOutput, err)
	}
	defer fo.Close()

	b := &bundle.Bundle{}
	if err := addEntriesFromDir(dir, parsedBaseURL, b); err != nil {
		return err
	}
	// Move the startURL entry to first.
	for i, e := range b.Exchanges {
		if e.Request.URL.String() == parsedStartURL.String() {
			tmp := b.Exchanges[0]
			b.Exchanges[0] = e
			b.Exchanges[i] = tmp
			break
		}
	}

	if _, err := b.WriteTo(fo); err != nil {
		return fmt.Errorf("Failed to write exchange. err: %v", err)
	}
	return nil
}

func addEntriesFromDir(dir string, baseURL *url.URL, b *bundle.Bundle) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("Error reading directory %s. err: %v", dir, err)
	}
	for _, f := range files {
		name := f.Name()
		pathName := filepath.Join(dir, name)
		if f.IsDir() {
			url, err := baseURL.Parse(name + "/")
			if err != nil {
				return fmt.Errorf("Failed to construct URL for %s. err: %v", name, err)
			}
			if err := addEntriesFromDir(pathName, url, b); err != nil {
				return err
			}
		} else {
			url, err := baseURL.Parse(name)
			if err != nil {
				return fmt.Errorf("Failed to construct URL for %s. err: %v", name, err)
			}
			if err := addEntryFromFile(pathName, url, b); err != nil {
				return err
			}
			if name == "index.html" {
				// Create an entry for the directory itself.
				if err := addEntryFromFile(dir, baseURL, b); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Implements http.ResponseWriter
type responseWriter struct {
	bytes.Buffer
	status int
	header http.Header
}

func newResponseWriter() *responseWriter {
	return &responseWriter{header: make(http.Header)}
}

func (w *responseWriter) Header() http.Header {
	return w.header
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func addEntryFromFile(filePath string, url *url.URL, b *bundle.Bundle) error {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return fmt.Errorf("http.newRequest failed: %v", err)
	}

	w := newResponseWriter()
	http.ServeFile(w, req, filePath)

	e := &bundle.Exchange{
		Request: bundle.Request{
			URL:    req.URL,
			Header: req.Header,
		},
		Response: bundle.Response{
			Status: w.status,
			Header: w.header,
			Body:   w.Bytes(),
		},
	}
	b.Exchanges = append(b.Exchanges, e)

	log.Printf("%s -> %s", filePath, e.Request.URL)
	return nil
}
