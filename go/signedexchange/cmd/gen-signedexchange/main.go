package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/WICG/webpackage/go/signedexchange"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
	"github.com/WICG/webpackage/go/signedexchange/version"
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
	flagMethod         = flag.String("method", http.MethodGet, "Request method")
	flagUri            = flag.String("uri", "https://example.com/index.html", "The URI of the resource represented in the exchange")
	flagVersion        = flag.String("version", "1b3", "The signedexchange version")
	flagResponseStatus = flag.Int("status", 200, "The status of the response represented in the exchange")
	flagContent        = flag.String("content", "index.html", "Source file to be used as the exchange payload")
	flagCertificate    = flag.String("certificate", "cert.pem", "Certificate chain PEM file of the origin")
	flagCertificateUrl = flag.String("certUrl", "https://example.com/cert.msg", "The URL where the certificate chain is hosted at.")
	flagValidityUrl    = flag.String("validityUrl", "https://example.com/resource.validity.msg", "The URL where resource validity info is hosted at.")
	flagPrivateKey     = flag.String("privateKey", "cert-key.pem", "Private key PEM file of the origin")
	flagMIRecordSize   = flag.Int("miRecordSize", 4096, "The record size of Merkle Integrity Content Encoding")
	flagDate           = flag.String("date", "", "The datetime for the signed exchange in RFC3339 format (2006-01-02T15:04:05Z). Use now by default.")
	flagExpire         = flag.Duration("expire", 1*time.Hour, "The expire time of the signed exchange")

	flagDumpSignatureMessage = flag.String("dumpSignatureMessage", "", "Dump signature message bytes to a file for debugging.")
	flagDumpHeadersCbor      = flag.String("dumpHeadersCbor", "", "Dump metadata and headers encoded as a canonical CBOR to a file for debugging.")
	flagOutput               = flag.String("o", "out.sxg", "Signed exchange output file. If value is '-', sxg is written to stdout.")

	flagIgnoreErrors = flag.Bool("ignoreErrors", false, "Do not reject invalid input arguments")

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
	ver, ok := version.Parse(*flagVersion)
	if !ok {
		return fmt.Errorf("failed to parse version %q", *flagVersion)
	}
	privkey, err := signedexchange.ParsePrivateKey(privkeytext)
	if err != nil {
		return fmt.Errorf("failed to parse private key file %q. err: %v", *flagPrivateKey, err)
	}

	var fMsg io.WriteCloser
	if *flagDumpSignatureMessage != "" {
		var err error
		fMsg, err = os.Create(*flagDumpSignatureMessage)
		if err != nil {
			return fmt.Errorf("failed to open signature message dump output file %q for writing. err: %v", *flagDumpSignatureMessage, err)
		}
		defer fMsg.Close()
	}
	var fHdr io.WriteCloser
	if *flagDumpHeadersCbor != "" {
		var err error
		fHdr, err = os.Create(*flagDumpHeadersCbor)
		if err != nil {
			return fmt.Errorf("failed to open signedheaders dump output file %q for writing. err: %v", *flagDumpHeadersCbor, err)
		}
		defer fHdr.Close()
	}

	f := os.Stdout
	if *flagOutput != "-" {
		var err error
		f, err = os.Create(*flagOutput)
		if err != nil {
			return fmt.Errorf("failed to open output file %q for writing. err: %v", *flagOutput, err)
		}
		defer f.Close()
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

	e := signedexchange.NewExchange(ver, *flagUri, *flagMethod, reqHeader, *flagResponseStatus, resHeader, payload)
	if err := e.MiEncodePayload(*flagMIRecordSize); err != nil {
		return err
	}

	var date time.Time
	if *flagDate == "" {
		date = time.Now()
	} else {
		var err error
		date, err = time.Parse(time.RFC3339, *flagDate)
		if err != nil {
			return err
		}
	}

	s := &signedexchange.Signer{
		Date:        date,
		Expires:     date.Add(*flagExpire),
		Certs:       certs,
		CertUrl:     certUrl,
		ValidityUrl: validityUrl,
		PrivKey:     privkey,
	}
	if err := e.AddSignatureHeader(s); err != nil {
		return err
	}

	if !*flagIgnoreErrors {
		// Check if the generated exchange passes Verify().

		// Create a cert fetcher for Verify() that returns `certs` in
		// application/cert-chain+cbor format.
		certFetcher := func(_ string) ([]byte, error) {
			certChain, err := certurl.NewCertChain(certs, []byte("dummy"), nil)
			if err != nil {
				return nil, err
			}
			var certBuf bytes.Buffer
			if err := certChain.Write(&certBuf); err != nil {
				return nil, err
			}
			return certBuf.Bytes(), nil
		}
		var logBuf bytes.Buffer
		if _, ok := e.Verify(date, certFetcher, log.New(&logBuf, "", 0)); !ok {
			return fmt.Errorf("failed to verify generated exchange: %s", logBuf.String())
		}
	}

	if fMsg != nil {
		if err := e.DumpSignedMessage(fMsg, s); err != nil {
			return fmt.Errorf("failed to write signature message dump. err: %v", err)
		}
	}
	if fHdr != nil {
		if err := e.DumpExchangeHeaders(fHdr); err != nil {
			return fmt.Errorf("failed to write headers cbor dump. err: %v", err)
		}
	}
	if err := e.Write(f); err != nil {
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
