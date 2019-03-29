package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/WICG/webpackage/go/bundle"
)

func fromDir(dir string, baseURL string, startURL string, manifestURL string) error {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("Failed to parse base URL. err: %v", err)
	}
	parsedStartURL, err := parsedBaseURL.Parse(startURL)
	if err != nil {
		return fmt.Errorf("Failed to parse start URL. err: %v", err)
	}
	var parsedManifestURL *url.URL
	if len(manifestURL) > 0 {
		parsedManifestURL, err = parsedBaseURL.Parse(manifestURL)
		if err != nil {
			return fmt.Errorf("Failed to parse manifest URL. err: %v", err)
		}
	}

	fo, err := os.OpenFile(*flagOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Failed to open output file %q for writing. err: %v", *flagOutput, err)
	}
	defer fo.Close()

	es, err := createExchangesFromDir(dir, parsedBaseURL)
	if err != nil {
		return err
	}
	b := &bundle.Bundle{Exchanges: es, ManifestURL: parsedManifestURL}
	// Move the startURL entry to first.
	for i, e := range b.Exchanges {
		if e.Request.URL.String() == parsedStartURL.String() {
			b.Exchanges[0], b.Exchanges[i] = b.Exchanges[i], b.Exchanges[0]
			break
		}
	}

	if _, err := b.WriteTo(fo); err != nil {
		return fmt.Errorf("Failed to write exchange. err: %v", err)
	}
	return nil
}

func createExchangesFromDir(baseDir string, baseURL *url.URL) ([]*bundle.Exchange, error) {
	es := []*bundle.Exchange{}
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		url, err := convertPathToURL(path, baseDir, baseURL)
		if err != nil {
			return err
		}
		if info.IsDir() {
			// For a directory, create an entry only if it contains index.html
			// (otherwise http.ServeFile generates a directory list).
			if _, err := os.Stat(filepath.Join(path, "index.html")); err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return fmt.Errorf("Stat(%s) failed. err: %v", path, err)
			}
			if !strings.HasSuffix(url, "/") {
				url += "/"
			}
		}
		e, err := createExchange(path, url)
		if err != nil {
			return err
		}
		es = append(es, e)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Error walking the path %s. err: %v", baseDir, err)
	}
	return es, nil
}

func convertPathToURL(path string, baseDir string, baseURL *url.URL) (string, error) {
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return "", fmt.Errorf("Cannot make relative path for %q: %v", path, err)
	}
	url, err := baseURL.Parse(filepath.ToSlash(relPath))
	if err != nil {
		return "", fmt.Errorf("Failed to construct URL for %s. err: %v", path, err)
	}
	return url.String(), nil
}

// responseWriter implements http.ResponseWriter.
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

// createExchange creates a bundle.Exchange whose request URL is url
// and response body is the contents of the file. Internally, it uses
// http.ServeFile to generate a realistic HTTP response for the file.
func createExchange(file string, url string) (*bundle.Exchange, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("http.newRequest failed: %v", err)
	}
	log.Printf("Creating exchange: %s -> %s", file, req.URL)

	w := newResponseWriter()
	http.ServeFile(w, req, file)

	return &bundle.Exchange{
		Request: bundle.Request{
			URL:    req.URL,
			Header: req.Header,
		},
		Response: bundle.Response{
			Status: w.status,
			Header: w.header,
			Body:   w.Bytes(),
		},
	}, nil
}
