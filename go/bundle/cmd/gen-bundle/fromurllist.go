package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/WICG/webpackage/go/bundle"
)

func fromURLList(urlListFile string) ([]*bundle.Exchange, error) {
	input, err := os.Open(urlListFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to open %q: %v", urlListFile, err)
	}
	defer input.Close()
	scanner := bufio.NewScanner(input)

	es := []*bundle.Exchange{}
	seen := make(map[string]struct{})
	for scanner.Scan() {
		rawURL := strings.TrimSpace(scanner.Text())
		// Skip blank lines and comments.
		if len(rawURL) == 0 || rawURL[0] == '#' {
			continue
		}
		if _, ok := seen[rawURL]; ok {
			log.Printf("Skipping duplicated URL %q", rawURL)
			continue
		}

		seen[rawURL] = struct{}{}
		log.Printf("Processing %q", rawURL)

		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse URL %q: %v", rawURL, err)
		}
		resp, err := http.Get(rawURL)
		if err != nil {
			return nil, fmt.Errorf("Failed to fetch %q: %v", rawURL, err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Error reading response body of %q: %v", rawURL, err)
		}
		e := &bundle.Exchange{
			Request: bundle.Request{
				URL: parsedURL,
			},
			Response: bundle.Response{
				Status: resp.StatusCode,
				Header: resp.Header,
				Body:   body,
			},
		}
		es = append(es, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Error reading %q: %v", urlListFile, err)
	}

	return es, nil
}
