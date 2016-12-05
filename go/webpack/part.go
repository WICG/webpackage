package webpack

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"mime"
	"net/textproto"
	"os"
	"path/filepath"
)

type PackPart struct {
	headers   textproto.MIMEHeader
	filename  string
	file      *os.File
	readIndex int64
}

func NewPackPart() *PackPart {
	pp := new(PackPart)
	pp.headers = make(textproto.MIMEHeader)
	return pp
}

func (p *PackPart) Headers() *textproto.MIMEHeader {
	h, err := p.HashContent(sha256.New())
	if err == nil {
		p.headers.Add("X-Content-Hash", h)
	}
	return &p.headers
}

func (p *PackPart) File() (*os.File, error) {
	if p.file != nil {
		return p.file, nil
	}

	if p.filename == "" {
		return nil, errors.New("Part had no file and no filename.")
	}
	return os.Open(p.filename)
}

func (p *PackPart) SetFilename(n string) {
	p.filename = n
	p.headers.Set("Content-Location", n)
	p.headers.Set("Content-Type", mime.TypeByExtension(filepath.Ext(n)))
}

func (p *PackPart) HashContent(h hash.Hash) (string, error) {
	var result []byte
	file, err := os.Open(p.filename)
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
