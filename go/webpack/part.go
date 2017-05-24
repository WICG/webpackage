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
	url             *url.URL
	requestHeaders  HTTPHeaders
	status          int
	responseHeaders HTTPHeaders
	contentFilename string
	content         []byte
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
	return nil, fmt.Errorf("Part %v had no filename and no content.", p.url)
}
