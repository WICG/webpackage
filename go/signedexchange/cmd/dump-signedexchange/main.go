package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/WICG/webpackage/go/signedexchange"
)

var (
	flagInput     = flag.String("i", "", "Signed-exchange input file")
	flagSignature = flag.Bool("signature", false, "Print only signature value")
	flagVerify    = flag.Bool("verify", false, "Perform signature verification")
	flagCert      = flag.String("cert", "", "Certificate CBOR file. If specified, used instead of fetching from signature's cert-url")
)

func run() error {
	in := os.Stdin
	if *flagInput != "" {
		var err error
		in, err = os.Open(*flagInput)
		if err != nil {
			return err
		}
		defer in.Close()
	}

	e, err := signedexchange.ReadExchange(in)
	if err != nil {
		return err
	}

	if *flagSignature {
		fmt.Println(e.SignatureHeaderValue)
		return nil
	}

	if *flagVerify {
		if err := verify(e); err != nil {
			return err
		}
		fmt.Println()
	}

	e.PrettyPrintHeaders(os.Stdout)
	e.PrettyPrintPayload(os.Stdout)

	return nil
}

func verify(e *signedexchange.Exchange) error {
	certFetcher := signedexchange.DefaultCertFetcher
	if *flagCert != "" {
		f, err := os.Open(*flagCert)
		if err != nil {
			return fmt.Errorf("could not open %s: %v\n", *flagCert, err)
		}
		defer f.Close()
		certBytes, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("Could not read %s: %v\n", *flagCert, err)
		}
		certFetcher = func(_ string) ([]byte, error) {
			return certBytes, nil
		}
	}

	verificationTime := time.Now()
	if decodedPayload, ok := e.Verify(verificationTime, certFetcher, log.New(os.Stdout, "", 0)); ok {
		e.Payload = decodedPayload
		fmt.Println("The exchange has valid signature.")
	}
	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
