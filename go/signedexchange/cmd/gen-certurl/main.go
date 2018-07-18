package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"golang.org/x/crypto/ocsp"

	"github.com/WICG/webpackage/go/signedexchange"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

var (
	pemFilepath  = flag.String("pem", "", "PEM filepath")
	ocspFilepath = flag.String("ocsp", "", "DER-encoded OCSP response file. If omitted, fetched from network")
	sctDirpath   = flag.String("sctDir", "", "Directory containing .sct files")
)

func run(pemFilePath, ocspFilePath, sctDirPath string) error {
	pem, err := ioutil.ReadFile(pemFilePath)
	if err != nil {
		return err
	}
	certs, err := signedexchange.ParseCertificates(pem)
	if err != nil {
		return err
	}

	var ocsp_der []byte
	if *ocspFilepath == "" {
		ocsp_der, err = certurl.FetchOCSPResponse(certs)
		if err != nil {
			return err
		}
	} else {
		ocsp_der, err = ioutil.ReadFile(ocspFilePath)
		if err != nil {
			return err
		}
	}
	parsed_ocsp, err := ocsp.ParseResponse(ocsp_der, nil)
	if err != nil {
		log.Println("Warning: ocsp is not a correct DER-encoded OCSP response.")
	}

	var sctList []byte
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
		sctList, err = certurl.SerializeSCTList(scts)
		if err != nil {
			return err
		}
	} else {
		if !certurl.HasEmbeddedSCT(certs[0], parsed_ocsp) {
			log.Println("Warning: Neither cert nor OCSP have embedded SCT list. Use -sctDir flag to add SCT from files.")
		}
	}

	out, err := certurl.CreateCertChainCBOR(certs, ocsp_der, sctList)
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
	if *pemFilepath == "" {
		flag.Usage()
		return
	}

	if err := run(*pemFilepath, *ocspFilepath, *sctDirpath); err != nil {
		log.Fatal(err)
	}
}
