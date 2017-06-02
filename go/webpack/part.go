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
	requestHeaders HTTPHeaders
	// responseHeaders include the :status pseudoheader.
	responseHeaders HTTPHeaders
	contentFilename string
	content         []byte
}

func (p *PackPart) URL() (*url.URL, error) {
	if p.requestHeaders[1].Name != ":scheme" ||
		p.requestHeaders[2].Name != ":authority" ||
		p.requestHeaders[3].Name != ":path" {
		panic(fmt.Sprintf("Request headers don't include the expected pseudoheaders: %#v", p.requestHeaders))
	}
	url, err := url.Parse(p.requestHeaders[3].Value)
	url.Scheme = p.requestHeaders[1].Value
	url.Host = p.requestHeaders[2].Value
	return url, err
}

func (p *PackPart) Hash() (string, error) {
	h := sha256.New()
	p.requestHeaders.WriteHTTP1(h)
	h.Write([]byte{0})
	p.responseHeaders.WriteHTTP1(h)
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
