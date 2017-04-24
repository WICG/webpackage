package webpack

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

func ParseText(manifestFilename string) (Package, error) {
	contentBase := path.Dir(manifestFilename)
	manifestFile, err := os.Open(manifestFilename)
	if err != nil {
		return Package{}, err
	}
	manifest := bufio.NewReader(manifestFile)
	lines := bufio.NewScanner(manifest)
	for lines.Scan() {
		line := lines.Text()
		if line == "[Content]" {
			break
		}
	}

	// Content:
	parts := make([]*PackPart, 0)
	for lines.Scan() {
		// Request headers:
		url, err := url.Parse(lines.Text())
		if err != nil {
			return Package{}, err
		}
		if !url.IsAbs() {
			return Package{}, fmt.Errorf("Resource URLs must be absolute: %q", lines.Text())
		}
		requestHeaders := make(HttpHeaders, 0)
		for lines.Scan() {
			line := lines.Text()
			if line == "" {
				break
			}
			header, err := ParseHttpHeader(line)
			if err != nil {
				return Package{}, err
			}
			requestHeaders = append(requestHeaders, header)
		}

		// Response
		if !lines.Scan() {
			return Package{}, fmt.Errorf("Missing response status for resource %q", url)
		}
		status, err := strconv.Atoi(lines.Text())
		if err != nil {
			return Package{}, fmt.Errorf("Invalid status code: %s", err)
		}
		if status < 100 || status > 999 {
			return Package{}, fmt.Errorf("Invalid status code: %d must be a 3-digit integer.", status)
		}
		responseHeaders := make(HttpHeaders, 0)
		for lines.Scan() {
			line := lines.Text()
			if line == "" {
				break
			}
			header, err := ParseHttpHeader(line)
			if err != nil {
				return Package{}, err
			}
			responseHeaders = append(responseHeaders, header)
		}

		// Body
		if !lines.Scan() {
			return Package{}, fmt.Errorf("Missing body for resource %q", url)
		}
		relativeFilename := lines.Text()
		filename := path.Join(contentBase, relativeFilename)
		// Trailing blank line is optional.
		lines.Scan()
		line := lines.Text()
		if line != "" {
			return Package{}, fmt.Errorf("Body should be a single line: %q", line)
		}

		parts = append(parts, &PackPart{url, requestHeaders, status, responseHeaders, filename, nil})
	}

	return Package{Manifest{}, parts, nil, nil}, lines.Err()
}

// Writes the manifest to base.manifest and the content bodies to base/scheme/domain/path.
// This doesn't support request headers yet.
func WriteTextTo(base string, p *Package) (err error) {
	manifest := base + ".manifest"
	manifestFile, err := os.Create(manifest)
	defer manifestFile.Close()
	if err != nil {
		return
	}
	w := bufio.NewWriter(manifestFile)
	defer w.Flush()
	if _, err = w.WriteString("[Content]\r\n"); err != nil {
		return
	}
	for _, part := range p.parts {
		if err = writePart(w, base, part); err != nil {
			return
		}
	}
	return nil
}

func writePart(w *bufio.Writer, base string, part *PackPart) (err error) {
	if _, err = io.WriteString(w, part.url.String()); err != nil {
		return
	}
	if err = part.requestHeaders.WriteHttp1(w); err != nil {
		return
	}
	if _, err = io.WriteString(w, "\r\n"); err != nil {
		return
	}
	if err = part.responseHeaders.WriteHttp1(w); err != nil {
		return
	}

	// Write the content to a file under base/.
	relativeOutContentFilename := filepath.Join(part.url.Scheme, part.url.Host,
		part.url.Path+part.url.RawQuery)
	outContentFilename := filepath.Join(base, relativeOutContentFilename)
	if err = os.MkdirAll(filepath.Dir(outContentFilename), 0755); err != nil {
		return
	}
	outContentFile, err := os.Create(outContentFilename)
	if err != nil {
		return
	}
	defer outContentFile.Close()
	inContent, err := part.Content()
	if err != nil {
		return
	}
	defer inContent.Close()
	io.Copy(outContentFile, inContent)

	if _, err = io.WriteString(w, relativeOutContentFilename); err != nil {
		return
	}
	if _, err = io.WriteString(w, "\r\n"); err != nil {
		return
	}
	return
}
