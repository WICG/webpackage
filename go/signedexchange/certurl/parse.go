package certurl

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func writeHead(w *bytes.Buffer, n int, size int) error {
	for i := 0; i < size; i++ {
		if err := w.WriteByte(byte(n >> (8 * uint(size-i-1)))); err != nil {
			return err
		}
	}
	return nil
}

// ParsePEM parses a PEM formatted content to a certUrl content.
// See https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html for the spec.
func ParsePEM(pemFileContent []byte) ([]byte, error) {
	b := pemFileContent

	entries := []*x509.Certificate{}
	totalLength := 0
	for {
		block, rest := pem.Decode(b)
		if block == nil && len(rest) > 0 {
			return nil, fmt.Errorf("failed to parse PEM file")
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		entries = append(entries, c)
		totalLength += len(c.Raw)

		if len(rest) == 0 {
			break
		}
		b = rest
	}

	buf := &bytes.Buffer{}

	const (
		certificateRequestContextHeadLength = 1
		certificateListHeadLength           = 3
		certDataHeadLength                  = 3
		extensionsHeadLength                = 2
	)

	// certificate_request_context is always empty, so just write the length '0' in 1 byte.
	// See https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#rfc.section.3.6
	if err := writeHead(buf, 0, certificateRequestContextHeadLength); err != nil {
		return nil, err
	}

	if err := writeHead(buf, totalLength+(certDataHeadLength+extensionsHeadLength)*len(entries), certificateListHeadLength); err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if err := writeHead(buf, len(entry.Raw), certDataHeadLength); err != nil {
			return nil, err
		}
		if _, err := buf.Write(entry.Raw); err != nil {
			return nil, err
		}
		if err := writeHead(buf, 0, extensionsHeadLength); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
