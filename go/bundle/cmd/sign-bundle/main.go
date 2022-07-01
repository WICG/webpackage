package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"time"
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
	flagSignType     = flag.String("signType", "signaturessection", "Type for signing: signaturessection or integrityblock. Defaulting to signaturessection.")
)

const (
	signTypeSignaturesSection = "signaturessection"
	signTypeIntegrityBlock    = "integrityblock"
)

func run() error {
	privKey, err := readPrivateKeyFromFile(*flagPrivateKey)
	if err != nil {
		return fmt.Errorf("%s: %v", *flagPrivateKey, err)
	}

	if *flagSignType == signTypeSignaturesSection {
		return SignExchanges(privKey)

	} else if *flagSignType == signTypeIntegrityBlock {
		return SignWithIntegrityBlock(privKey)

	} else {
		return errors.New(fmt.Sprintf("Unknown signType, approved flag values are \"%v\" and \"%v\".", signTypeSignaturesSection, signTypeIntegrityBlock))
	}
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
