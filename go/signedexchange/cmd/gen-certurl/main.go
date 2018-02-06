package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

func showUsage(w io.Writer) {
	fmt.Fprintf(w, "Usage: cert-url [pem-file] > [certurlFile]\n")
}

func run(pemFilePath string) error {
	in, err := ioutil.ReadFile(pemFilePath)
	if err != nil {
		return err
	}

	out, err := certurl.CertificateMessageFromPEM(in)
	if err != nil {
		return err
	}

	if _, err := os.Stdout.Write(out); err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) != 2 {
		showUsage(os.Stderr)
		os.Exit(1)
	}
	if err := run(os.Args[1]); err != nil {
		log.Fatal(err)
	}
}
