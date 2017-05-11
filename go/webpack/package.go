package webpack

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

type Package struct {
	manifest   Manifest
	parts      []*PackPart
	indexOrder []*PackPart
	privateKey *rsa.PrivateKey
}

const (
	contentType string = "application/package"
)

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
