package main

import (
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/WICG/webpackage/go/signedexchange"
)

type headerArgs []string

func (h *headerArgs) String() string {
	return fmt.Sprintf("%v", *h)
}

func (h *headerArgs) Set(value string) error {
	*h = append(*h, value)
	return nil
}

var (
	flagUri            = flag.String("uri", "https://example.com/index.html", "The URI of the resource represented in the exchange")
	flagResponseStatus = flag.Int("status", 200, "The status of the response represented in the exchange")
	flagContent        = flag.String("content", "index.html", "Source file to be used as the exchange payload")
	flagCertificate    = flag.String("certificate", "cert.pem", "Certificate chain PEM file of the origin")
	flagCertificateUrl = flag.String("certUrl", "https://example.com/cert.msg", "The URL where the certificate chain is hosted at.")
	flagValidityUrl    = flag.String("validityUrl", "https://example.com/resource.validity.msg", "The URL where resource validity info is hosted at.")
	flagPrivateKey     = flag.String("privateKey", "cert-key.pem", "Private key PEM file of the origin")
	flagOutput         = flag.String("o", "out.htxg", "Signed exchange output file")
	flagMIRecordSize   = flag.Int("miRecordSize", 4096, "The record size of Merkle Integrity Content Encoding")
	flagExpire         = flag.Duration("expire", 1*time.Hour, "The expire time of the signed exchange")

	flagRequestHeader  = headerArgs{}
	flagResponseHeader = headerArgs{}
)

func init() {
	flag.Var(&flagRequestHeader, "requestHeader", "Request header arguments")
	flag.Var(&flagResponseHeader, "responseHeader", "Response header arguments")
}

func run() error {
	payload, err := ioutil.ReadFile(*flagContent)
	if err != nil {
		return fmt.Errorf("failed to read content from payload source file \"%s\". err: %v", *flagContent, err)
	}

	certtext, err := ioutil.ReadFile(*flagCertificate)
	if err != nil {
		return fmt.Errorf("failed to read certificate file %q. err: %v", *flagCertificate, err)

	}
	certs, err := signedexchange.ParseCertificates(certtext)
	if err != nil {
		return fmt.Errorf("failed to parse certificate file %q. err: %v", *flagCertificate, err)
	}

	certUrl, err := url.Parse(*flagCertificateUrl)
	if err != nil {
		return fmt.Errorf("failed to parse certificate URL %q. err: %v", *flagCertificateUrl, err)
	}
	validityUrl, err := url.Parse(*flagValidityUrl)
	if err != nil {
		return fmt.Errorf("failed to parse validity URL %q. err: %v", *flagValidityUrl, err)
	}

	privkeytext, err := ioutil.ReadFile(*flagPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to read private key file %q. err: %v", *flagPrivateKey, err)
	}

	parsedPrivKey, _ := pem.Decode(privkeytext)
	if parsedPrivKey == nil {
		return fmt.Errorf("invalid private key")
	}
	privkey, err := signedexchange.ParsePrivateKey(parsedPrivKey.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key file %q. err: %v", *flagPrivateKey, err)
	}

	f, err := os.OpenFile(*flagOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file %q for writing. err: %v", *flagOutput, err)
		os.Exit(1)
	}
	defer f.Close()

	parsedUrl, err := url.Parse(*flagUri)
	if err != nil {
		return fmt.Errorf("failed to parse URL %q. err: %v", *flagUri, err)
	}

	reqHeader := http.Header{}
	for _, h := range flagRequestHeader {
		chunks := strings.SplitN(h, ":", 2)
		reqHeader.Add(strings.TrimSpace(chunks[0]), strings.TrimSpace(chunks[1]))
	}

	resHeader := http.Header{}
	for _, h := range flagResponseHeader {
		chunks := strings.SplitN(h, ":", 2)
		resHeader.Add(strings.TrimSpace(chunks[0]), strings.TrimSpace(chunks[1]))
	}
	if resHeader.Get("content-type") == "" {
		resHeader.Add("content-type", "text/html; charset=utf-8")
	}
	e, err := signedexchange.NewExchange(parsedUrl, reqHeader, *flagResponseStatus, resHeader, payload, *flagMIRecordSize)
	if err != nil {
		return err
	}

	s := &signedexchange.Signer{
		Date:        time.Now(),
		Expires:     time.Now().Add(*flagExpire),
		Certs:       certs,
		CertUrl:     certUrl,
		ValidityUrl: validityUrl,
		PrivKey:     privkey,
	}
	if err := e.AddSignatureHeader(s); err != nil {
		return err
	}

	if err := signedexchange.WriteExchangeFile(f, e); err != nil {
		return fmt.Errorf("failed to write exchange. err: %v", err)
	}
	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
