package main

import (
	"crypto"
	"flag"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/bundle/signature"
	"github.com/WICG/webpackage/go/signedexchange"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

var (
	flagInput        = flag.String("i", "in.wbn", "Webbundle input file")
	flagOutput       = flag.String("o", "out.wbn", "Webbundle output file")
	flagCertificate  = flag.String("certificate", "cert.cbor", "Certificate chain CBOR file")
	flagPrivateKey   = flag.String("privateKey", "cert-key.pem", "Private key PEM file")
	flagValidityUrl  = flag.String("validityUrl", "https://example.com/resource.validity.msg", "The URL where resource validity info is hosted at.")
	flagDate         = flag.String("date", "", "Datetime for the signature in RFC3339 format (2006-01-02T15:04:05Z). (default: current time)")
	flagExpire       = flag.Duration("expire", 1*time.Hour, "Validity duration of the signature")
	flagMIRecordSize = flag.Int("miRecordSize", 4096, "Record size of Merkle Integrity Content Encoding")
)

func readCertChainFromFile(path string) (certurl.CertChain, error) {
	fi, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	return certurl.ReadCertChain(fi)
}

func readPrivateKeyFromFile(path string) (crypto.PrivateKey, error) {
	privkeytext, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return signedexchange.ParsePrivateKey(privkeytext)
}

func readBundleFromFile(path string) (*bundle.Bundle, error) {
	fi, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	return bundle.Read(fi)
}

func writeBundleToFile(b *bundle.Bundle, path string) error {
	fo, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fo.Close()
	_, err = b.WriteTo(fo)
	return err
}

func addSignature(b *bundle.Bundle, signer *signature.Signer) error {
	for _, e := range b.Exchanges {
		if !signer.CanSignForURL(e.Request.URL) {
			continue
		}
		payloadIntegrityHeader, err := e.AddPayloadIntegrity(b.Version, *flagMIRecordSize)
		if err != nil {
			return err
		}
		if err := signer.AddExchange(e, payloadIntegrityHeader); err != nil {
			return err
		}
	}

	newSignatures, err := signer.UpdateSignatures(b.Signatures)
	if err != nil {
		return err
	}
	b.Signatures = newSignatures
	return nil
}

func main() {
	flag.Parse()

	certs, err := readCertChainFromFile(*flagCertificate)
	if err != nil {
		log.Fatalf("%s: %v", *flagCertificate, err)
	}

	privKey, err := readPrivateKeyFromFile(*flagPrivateKey)
	if err != nil {
		log.Fatalf("%s: %v", *flagPrivateKey, err)
	}

	validityUrl, err := url.Parse(*flagValidityUrl)
	if err != nil {
		log.Fatalf("failed to parse validity URL %q: %v", *flagValidityUrl, err)
	}

	var date time.Time
	if *flagDate == "" {
		date = time.Now()
	} else {
		var err error
		date, err = time.Parse(time.RFC3339, *flagDate)
		if err != nil {
			log.Fatalf("failed to parse date %q: %v", *flagDate, err)
		}
	}

	b, err := readBundleFromFile(*flagInput)
	if err != nil {
		log.Fatalf("%s: %v", *flagInput, err)
	}

	signer, err := signature.NewSigner(b.Version, certs, privKey, validityUrl, date, *flagExpire)
	if err != nil {
		log.Fatal(err)
	}

	if err := addSignature(b, signer); err != nil {
		log.Fatal(err)
	}

	if err := writeBundleToFile(b, *flagOutput); err != nil {
		log.Fatalf("%s: %v", *flagOutput, err)
	}
}
