package webpack

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
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
	file    *os.File
	content *bytes.Reader
}

func (c *PackPartContent) WriteTo(w io.Writer) (int64, error) {
	if c.file != nil {
		return io.Copy(w, c.file)
	} else {
		return io.Copy(w, c.content)
	}
}

func (c *PackPartContent) Read(p []byte) (int, error) {
	if c.file != nil {
		return c.file.Read(p)
	} else {
		return c.content.Read(p)
	}
}

func (c *PackPartContent) Close() {
	if c.file != nil {
		c.file.Close()
	}
}

func (p *PackPart) Content() (*PackPartContent, error) {
	if p.contentFilename != "" {
		file, err := os.Open(p.contentFilename)
		return &PackPartContent{file, nil}, err
	}
	if p.content != nil {
		return &PackPartContent{nil, bytes.NewReader(p.content)}, nil
	}
	return nil, fmt.Errorf("Part %v had no filename and no content.", p.url)
}

func (p *PackPart) HashContent(h hash.Hash) (string, error) {
	var result []byte
	file, err := os.Open(p.contentFilename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(result)), nil
}

func (part *PackPart) Read(p []byte) (n int, err error) {
	return 0, errors.New("Unimplemented.")
}
