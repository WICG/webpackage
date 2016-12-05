package webpack

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"sort"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

type Package struct {
	headers     textproto.MIMEHeader
	url         *url.URL
	describedBy string
	parts       []PackPart
	privateKey  *rsa.PrivateKey
}

const (
	contentType     string = "application/package"
	signatureHeader string = "Package-Signature"
)

func NewPackage(url *url.URL) *Package {
	p := Package{textproto.MIMEHeader{
		"Content-Type": {contentType},
	}, url, "", make([]PackPart, 0, 100), nil}
	return &p
}

func (p *Package) SetSigningKey(filename string) {
	var err error

	pemBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("Error setting signing key ", err)
	}
	eb, _ := pem.Decode(pemBytes)

	var keyBytes []byte

	if x509.IsEncryptedPEMBlock(eb) {
		var pw []byte
		fmt.Fprintf(os.Stderr, "Need passord for private key at "+filename+".")
		pw, err = terminal.ReadPassword(syscall.Stdin)
		keyBytes, err = x509.DecryptPEMBlock(eb, pw)
		if err != nil {
			log.Fatal("Error decrypting pem block ", err)
		}
	} else {
		keyBytes = eb.Bytes
	}
	pk, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		log.Fatal("Error parsing pkcs1 ", err)
	}

	p.privateKey = pk
}

func (p *Package) SetDescribedBy(d string) {
	p.describedBy = d
}

func (p *Package) AddPart(filename string, headers *textproto.MIMEHeader) {
	pp := NewPackPart()
	pp.SetFilename(filename)
	p.parts = append(p.parts, *pp)
}

func (p *Package) signHeader() (signature []byte, err error) {
	if p.privateKey == nil {
		return nil, nil
	}

	hasher := sha256.New()
	printHeaders(p.headers, hasher)

	var headerHash []byte = hasher.Sum(nil)
	return p.privateKey.Sign(rand.Reader, headerHash, crypto.SHA256)
}

func (p *Package) WriteTo(w io.Writer) (nOut int64, err error) {
	if p.describedBy == "" {
		p.describedBy = p.parts[0].filename
	}
	p.headers.Add("Link", fmt.Sprintf("<%s>; rel=describedby", p.describedBy))

	signBytes, err := p.signHeader()
	if err != nil {
		log.Fatal("Signing error: ", err)
	}
	if signBytes != nil {
		fmt.Fprintf(w, "%s: %s\r\n", signatureHeader, hex.EncodeToString(signBytes))
	}

	printHeaders(p.headers, w)
	mw := multipart.NewWriter(w)
	defer mw.Close()

	for _, part := range p.parts {
		mw.CreatePart(*part.Headers())
		f, err := part.File()
		if err != nil {
			log.Fatal("Error creating part file for: "+part.filename, err)
		}
		_, err = io.Copy(w, f)
		if err != nil {
			log.Fatal("Error copying part file for: "+part.filename, err)
		}
	}
	return 0, nil
}

func printHeaders(headers textproto.MIMEHeader, f io.Writer) {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range (headers)[k] {
			fmt.Fprintf(f, "%s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(f, "\r\n")
}
