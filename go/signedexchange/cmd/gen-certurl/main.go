package main

import (
	"bytes"
	"flag"
	"fmt"
	"golang.org/x/crypto/ocsp"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/WICG/webpackage/go/signedexchange"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

var (
	pemFilepath  = flag.String("pem", "", "PEM filepath")
	ocspFilepath = flag.String("ocsp", "", "DER-encoded OCSP response file. If omitted, fetched from network")
	sctDirpath   = flag.String("sctDir", "", "Directory containing .sct files")
)

func run(pemFilePath, ocspFilePath, sctDirPath string) error {
	certChain := certurl.CertChain{}

	pem, err := ioutil.ReadFile(pemFilePath)
	if err != nil {
		return err
	}
	certs, err := signedexchange.ParseCertificates(pem)
	if err != nil {
		return err
	}
	if len(certs) == 0 {
		return fmt.Errorf("input file %q has no certificates.", pemFilePath)
	}
	for _, cert := range certs {
		certChain = append(certChain, &certurl.CertChainItem{Cert: cert})
	}

	var ocspDer []byte
	if *ocspFilepath == "" {
		ocspDer, err = certurl.FetchOCSPResponse(certs)
		if err != nil {
			return err
		}
	} else {
		ocspDer, err = ioutil.ReadFile(ocspFilePath)
		if err != nil {
			return err
		}
	}
	parsedOcsp, err := ocsp.ParseResponse(ocspDer, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: ocsp is not a correct DER-encoded OCSP response.")
	}
	certChain[0].OCSPResponse = ocspDer

	if sctDirPath != "" {
		files, err := filepath.Glob(filepath.Join(sctDirPath, "*.sct"))
		if err != nil {
			return err
		}
		scts := [][]byte{}
		for _, file := range files {
			sct, err := ioutil.ReadFile(file)
			if err != nil {
				return err
			}
			scts = append(scts, sct)
		}
		certChain[0].SCTList, err = certurl.SerializeSCTList(scts)
		if err != nil {
			return err
		}
	} else {
		if !certurl.HasEmbeddedSCT(certs[0], parsedOcsp) {
			fmt.Fprintln(os.Stderr, "Warning: Neither cert nor OCSP have embedded SCT list. Use -sctDir flag to add SCT from files.")
		}
	}

	buf := &bytes.Buffer{}
	if err := certChain.Write(buf); err != nil {
		return err
	}

	if _, err := buf.WriteTo(os.Stdout); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	if *pemFilepath == "" {
		flag.Usage()
		return
	}

	if err := run(*pemFilepath, *ocspFilepath, *sctDirpath); err != nil {
		log.Fatal(err)
	}
}
