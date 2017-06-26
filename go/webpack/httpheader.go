package webpack

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/http2/hpack"
)

func httpHeader(name, value string) hpack.HeaderField {
	return hpack.HeaderField{Name: strings.ToLower(name), Value: value}
}

type HTTPHeaders []hpack.HeaderField

func ParseHTTPHeader(line string) (hpack.HeaderField, error) {
	split := strings.SplitN(line, ": ", 2)
	if len(split) == 1 {
		return hpack.HeaderField{}, fmt.Errorf("Malformed HTTP header: %q", line)
	}
	return httpHeader(split[0], split[1]), nil
}

func (headers HTTPHeaders) WriteHTTP1(f io.Writer) error {
	for _, header := range headers {
		if _, err := fmt.Fprintf(f, "%s: %s\r\n", header.Name, header.Value); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(f, "\r\n"); err != nil {
		return err
	}
	return nil
}

func (h HTTPHeaders) EncodeHPACK() []byte {
	var buf bytes.Buffer
	encoder := hpack.NewEncoder(&buf)
	for _, field := range h {
		if err := encoder.WriteField(field); err != nil {
			panic(err)
		}
	}
	return buf.Bytes()
}
