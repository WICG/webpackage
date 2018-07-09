package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

var (
	pemFilepath  = flag.String("pem", "", "PEM filepath")
	ocspFilepath = flag.String("ocsp", "", "OCSP filepath")
	sctFilepath  = flag.String("sct", "", "SCT filepath")
)

func run(pemFilePath, ocspFilePath, sctFilePath string) error {
	in, err := ioutil.ReadFile(pemFilePath)
	if err != nil {
		return err
	}

	ocsp, err := ioutil.ReadFile(ocspFilePath)
	if err != nil {
		return err
	}

	var sct []byte
	if sctFilePath != "" {
		sct, err = ioutil.ReadFile(sctFilePath)
		if err != nil {
			return err
		}
	}

	out, err := certurl.CertificateMessageFromPEM(in, ocsp, sct)
	if err != nil {
		return err
	}

	if _, err := os.Stdout.Write(out); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	if *pemFilepath == "" || *ocspFilepath == "" {
		flag.Usage()
		return
	}

	if err := run(*pemFilepath, *ocspFilepath, *sctFilepath); err != nil {
		log.Fatal(err)
	}
}
