package main

import (
	"flag"
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
	pem, err := ioutil.ReadFile(pemFilePath)
	if err != nil {
		return err
	}
	certs, err := signedexchange.ParseCertificates(pem)
	if err != nil {
		return err
	}

	var ocsp []byte
	if *ocspFilepath == "" {
		ocsp, err = certurl.FetchOCSPResponse(certs)
		if err != nil {
			return err
		}
	} else {
		ocsp, err = ioutil.ReadFile(ocspFilePath)
		if err != nil {
			return err
		}
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
	}

	out, err := certurl.CreateCertChainCBOR(certs, ocsp, sctList)
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
