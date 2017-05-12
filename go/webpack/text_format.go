package webpack

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// Used to split comma- or semicolon-separated values.
var commaSeparator *regexp.Regexp = regexp.MustCompile(`\s*,\s*`)
var semicolonSeparator *regexp.Regexp = regexp.MustCompile(`\s*;\s*`)

func ParseText(manifestFilename string) (Package, error) {
	contentBase := filepath.Dir(manifestFilename)
	manifestFile, err := os.Open(manifestFilename)
	if err != nil {
		return Package{}, err
	}
	return ParseTextContent(contentBase, manifestFile)
}

func ParseTextContent(baseDir string, manifestReader io.Reader) (pack Package, err error) {
	lines := bufio.NewScanner(manifestReader)
	var parts []*PackPart
	var manifest Manifest
	for lines.Scan() {
		line := lines.Text()
		if line == "[Content]" {
			if parts, err = parseTextParts(lines, baseDir); err != nil {
				return pack, err
			}
		}
		if line == "[Manifest]" {
			if manifest, err = parseTextManifest(lines, baseDir); err != nil {
				return pack, err
			}
		}
	}

	return Package{manifest, parts}, lines.Err()
}

func parseTextManifest(lines *bufio.Scanner, baseDir string) (Manifest, error) {
	manifest := Manifest{}
	for lines.Scan() {
		line := lines.Text()
		if line == "" {
			break
		}
		header, err := ParseHTTPHeader(line)
		if err != nil {
			return manifest, err
		}

		switch header.Name {
		case "hash-algorithms":
			for _, name := range commaSeparator.Split(header.Value, -1) {
				if hash, err := parseHashName(name); err != nil {
					return manifest, err
				} else {
					manifest.hashTypes = append(manifest.hashTypes, hash)
				}
			}
			sort.Slice(manifest.hashTypes, func(i, j int) bool {
				return manifest.hashTypes[i] < manifest.hashTypes[j]
			})
		case "sign-with":
			cert_key := semicolonSeparator.Split(header.Value, -1)
			var certFilename, keyFilename string
			switch len(cert_key) {
			case 2:
				keyFilename = filepath.Join(baseDir, cert_key[1])
				fallthrough
			case 1:
				certFilename = filepath.Join(baseDir, cert_key[0])
			default:
				return manifest, fmt.Errorf("Too many values in sign-with: %q", header.Value)
			}
			signWith, err := LoadSignWith(certFilename, keyFilename)
			if err != nil {
				return manifest, err
			}
			manifest.signatures = append(manifest.signatures, signWith)
		case "certificate-chain":
			filename := filepath.Join(baseDir, header.Value)
			if err := LoadCertificatesInto(filename, &manifest.certificates); err != nil {
				return manifest, err
			}
		case "date":
			date, err := http.ParseTime(header.Value)
			if err != nil {
				return manifest, err
			}
			manifest.metadata.date = date
		case "origin":
			origin, err := url.Parse(header.Value)
			if err != nil {
				return manifest, err
			}
			manifest.metadata.origin = origin
		}
	}
	return manifest, nil
}

func parseTextParts(lines *bufio.Scanner, baseDir string) ([]*PackPart, error) {
	parts := make([]*PackPart, 0)

	for lines.Scan() {
		// Request headers:
		url, err := url.Parse(lines.Text())
		if err != nil {
			return nil, err
		}
		if !url.IsAbs() {
			return nil, fmt.Errorf("Resource URLs must be absolute: %q", lines.Text())
		}
		requestHeaders := make(HTTPHeaders, 0)
		for lines.Scan() {
			line := lines.Text()
			if line == "" {
				break
			}
			header, err := ParseHTTPHeader(line)
			if err != nil {
				return nil, err
			}
			requestHeaders = append(requestHeaders, header)
		}

		// Response
		if !lines.Scan() {
			return nil, fmt.Errorf("Missing response status for resource %q", url)
		}
		status, err := strconv.Atoi(lines.Text())
		if err != nil {
			return nil, fmt.Errorf("Invalid status code: %s", err)
		}
		if status < 100 || status > 999 {
			return nil, fmt.Errorf("Invalid status code: %d must be a 3-digit integer.", status)
		}
		responseHeaders := make(HTTPHeaders, 0)
		for lines.Scan() {
			line := lines.Text()
			if line == "" {
				break
			}
			header, err := ParseHTTPHeader(line)
			if err != nil {
				return nil, err
			}
			responseHeaders = append(responseHeaders, header)
		}
		if err := checkRequestHeadersInVary(requestHeaders, responseHeaders); err != nil {
			return nil, err
		}

		// Body
		if !lines.Scan() {
			return nil, fmt.Errorf("Missing body for resource %q", url)
		}
		relativeFilename := lines.Text()
		filename := filepath.Join(baseDir, relativeFilename)
		// Trailing blank line is optional.
		lines.Scan()
		line := lines.Text()
		if line != "" {
			return nil, fmt.Errorf("Body should be a single line: %q", line)
		}

		parts = append(parts, &PackPart{url, requestHeaders, status, responseHeaders, filename, nil})
	}
	return parts, nil
}

// checkRequestHeadersInVary returns non-nil if there's a request header that
// doesn't appear in the Vary response header.
func checkRequestHeadersInVary(requestHeaders, responseHeaders HTTPHeaders) error {
	varyHeader := ""
	vary := make(map[string]bool)
	for _, header := range responseHeaders {
		if header.Name == "vary" {
			if header.Value == "*" {
				return errors.New("Cannot have a Vary header of '*'.")
			}
			varyHeader = header.Value
			for _, allowedHeader := range commaSeparator.Split(varyHeader, -1) {
				vary[allowedHeader] = true
			}
			break
		}
	}

	for _, header := range requestHeaders {
		if !vary[header.Name] {
			return fmt.Errorf("Can't include request header %q that's not in Vary header: %q", header.Name, varyHeader)
		}
	}

	return nil
}

// WriteTextTo writes the manifest to base.manifest and the content bodies to
// base/scheme/domain/path. This doesn't support request headers yet.
func WriteTextTo(base string, p *Package) error {
	manifest := base + ".manifest"
	manifestFile, err := os.Create(manifest)
	defer manifestFile.Close()
	if err != nil {
		return err
	}
	w := bufio.NewWriter(manifestFile)
	defer w.Flush()
	if _, err = w.WriteString("[Content]\r\n"); err != nil {
		return err
	}
	for _, part := range p.parts {
		if err = writePart(w, base, part); err != nil {
			return err
		}
	}
	return nil
}

func writePart(w *bufio.Writer, base string, part *PackPart) error {
	if _, err := io.WriteString(w, part.url.String()); err != nil {
		return err
	}
	if err := part.requestHeaders.WriteHTTP1(w); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\r\n"); err != nil {
		return err
	}
	if err := part.responseHeaders.WriteHTTP1(w); err != nil {
		return err
	}

	// Write the content to a file under base/.
	relativeOutContentFilename := filepath.Join(part.url.Scheme, part.url.Host,
		part.url.Path+part.url.RawQuery)
	outContentFilename := filepath.Join(base, relativeOutContentFilename)
	if err := os.MkdirAll(filepath.Dir(outContentFilename), 0755); err != nil {
		return err
	}
	outContentFile, err := os.Create(outContentFilename)
	if err != nil {
		return err
	}
	defer outContentFile.Close()
	inContent, err := part.Content()
	if err != nil {
		return err
	}
	defer inContent.Close()
	io.Copy(outContentFile, inContent)

	if _, err = io.WriteString(w, relativeOutContentFilename); err != nil {
		return err
	}
	if _, err = io.WriteString(w, "\r\n"); err != nil {
		return err
	}
	return nil
}
