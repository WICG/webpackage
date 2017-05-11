package webpack

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/http2/hpack"
)

func httpHeader(name, value string) hpack.HeaderField {
	return hpack.HeaderField{Name: strings.ToLower(name), Value: value}
}

type HttpHeaders []hpack.HeaderField

func ParseHttpHeader(line string) (hpack.HeaderField, error) {
	split := strings.SplitN(line, ": ", 2)
	if len(split) == 1 {
		return hpack.HeaderField{}, fmt.Errorf("Malformed HTTP header: %q", line)
	}
	return httpHeader(split[0], split[1]), nil
}

func (headers HttpHeaders) WriteHttp1(f io.Writer) (err error) {
	for _, header := range headers {
		if _, err = fmt.Fprintf(f, "%s: %s\r\n", header.Name, header.Value); err != nil {
			return
		}
	}
	if _, err = fmt.Fprintf(f, "\r\n"); err != nil {
		return
	}
	return
}

func (HttpHeaders) WriteHpack(f io.Writer) {
	panic("Unimplemented")
}
