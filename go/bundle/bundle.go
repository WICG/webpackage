package bundle

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/WICG/webpackage/go/bundle/version"
)

type Request struct {
	*url.URL
	http.Header
}

type Response struct {
	Status int
	http.Header
	Body []byte
}

func (r Response) String() string {
	return fmt.Sprintf("{Status: %q, Header: %v, body: %v}", r.Status, r.Header, string(r.Body))
}

type Exchange struct {
	Request
	Response
}

func (e *Exchange) Dump(w io.Writer, dumpContentText bool) error {
	if _, err := fmt.Fprintf(w, "> :url: %v\n", e.Request.URL); err != nil {
		return err
	}
	for k, v := range e.Request.Header {
		if _, err := fmt.Fprintf(w, "> %v: %v\n", k, v); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "< :status: %v\n", e.Response.Status); err != nil {
		return err
	}
	for k, v := range e.Response.Header {
		if _, err := fmt.Fprintf(w, "< %v: %v\n", k, v); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "< [len(Body)]: %d\n", len(e.Response.Body)); err != nil {
		return err
	}
	if dumpContentText {
		ctype := e.Response.Header.Get("content-type")
		if strings.Contains(ctype, "text") {
			if _, err := fmt.Fprint(w, string(e.Response.Body)); err != nil {
				return err
			}
			if _, err := fmt.Fprint(w, "\n"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprint(w, "[non-text body]\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

type Bundle struct {
	Version     version.Version
	PrimaryURL  *url.URL
	Exchanges   []*Exchange
	ManifestURL *url.URL
}
