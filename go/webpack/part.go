package webpack

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
)

type PackPart struct {
	// requestHeaders include the 4 pseudoheaders in
	// http://httpwg.org/specs/rfc7540.html#rfc.section.8.1.2.3 that include
	// the URL.
	RequestHeaders HTTPHeaders
	// responseHeaders include the :status pseudoheader.
	ResponseHeaders HTTPHeaders
	contentFilename string
	content         []byte
}

func (p *PackPart) URL() (*url.URL, error) {
	if p.RequestHeaders[1].Name != ":scheme" ||
		p.RequestHeaders[2].Name != ":authority" ||
		p.RequestHeaders[3].Name != ":path" {
		panic(fmt.Sprintf("Request headers don't include the expected pseudoheaders: %#v", p.RequestHeaders))
	}
	url, err := url.Parse(p.RequestHeaders[3].Value)
	url.Scheme = p.RequestHeaders[1].Value
	url.Host = p.RequestHeaders[2].Value
	return url, err
}

func (p *PackPart) NonPseudoRequestHeaders() HTTPHeaders {
	// There are 4 pseudo-headers in the request.
	return p.RequestHeaders[4:]
}

func (p *PackPart) NonPseudoResponseHeaders() HTTPHeaders {
	// There's 1 pseudo-header, :status, in the response.
	return p.ResponseHeaders[1:]
}

func (p *PackPart) Hash() (string, error) {
	h := sha256.New()
	p.RequestHeaders.WriteHTTP1(h)
	h.Write([]byte{0})
	p.ResponseHeaders.WriteHTTP1(h)
	h.Write([]byte{0})
	content, err := p.Content()
	if err != nil {
		return "", err
	}
	io.Copy(h, content)
	return string(h.Sum(nil)), nil
}

type PackPartContent struct {
	io.ReadCloser
	// size is the number of bytes that will be returned by the Reader.
	size int64
}

func (c *PackPartContent) Size() int64 {
	return c.size
}

func (p *PackPart) Content() (*PackPartContent, error) {
	if p.contentFilename != "" {
		file, err := os.Open(p.contentFilename)
		if err != nil {
			return nil, err
		}
		stat, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, err
		}
		return &PackPartContent{file, stat.Size()}, err
	}
	if p.content != nil {
		return &PackPartContent{
			ReadCloser: ioutil.NopCloser(bytes.NewReader(p.content)),
			size:       int64(len(p.content)),
		}, nil
	}
	return nil, fmt.Errorf("Part %v had no filename and no content.", *p)
}
