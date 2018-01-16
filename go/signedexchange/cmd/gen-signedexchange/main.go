package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/url"
	"os"

	se "github.com/WICG/webpackage/go/signedexchange"
)

var (
	flagUri             = flag.String("uri", "https://example.com/index.html", "The URI of the resource represented in the exchange")
	flagResponseStatus  = flag.Int("status", 200, "The status of the response represented in the exchange")
	flagContent         = flag.String("content", "index.html", "Source file to be used as the exchange payload")
	flagCertificatePath = flag.String("certificate", "cert.pem", "Certificate chain PEM file of the origin")
	flagPrivateKeyPath  = flag.String("privateKey", "cert-key.pem", "Private key PEM file of the origin")
	flagOutput          = flag.String("o", "out.wpk", "Signed exchange output file")
)

func main() {
	flag.Parse()

	payload, err := ioutil.ReadFile(*flagContent)
	if err != nil {
		log.Printf("Failed to read content from source file \"%s\". err: %v", *flagContent, err)
		os.Exit(1)
	}

	f, err := os.OpenFile(*flagOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Failed to open output file \"%s\" for writing. err: %v", *flagOutput, err)
		os.Exit(1)
	}
	defer f.Close()

	parsedUrl, err := url.Parse(*flagUri)
	if err != nil {
		log.Printf("Failed to parse URL \"%s\". err: %v", *flagUri, err)
	}
	input := &se.Input{
		RequestUri:     parsedUrl,
		ResponseStatus: *flagResponseStatus,
		ResponseHeaders: []se.ResponseHeader{
			// FIXME
			se.ResponseHeader{Name: "Content-Type", Value: "text/html; charset=utf-8"},
		},
		Payload: payload,
	}

	if err := se.WriteExchange(f, input); err != nil {
		log.Printf("Failed to write exchange. err: %v", err)
		os.Exit(1)
	}
	log.Println("Done!")
}
