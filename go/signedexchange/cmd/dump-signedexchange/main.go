package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/WICG/webpackage/go/signedexchange/structuredheader"

	"github.com/WICG/webpackage/go/signedexchange"
)

var (
	flagInput     = flag.String("i", "", "Signed-exchange input file")
	flagSignature = flag.Bool("signature", false, "Print only signature value")
	flagVerify    = flag.Bool("verify", false, "Perform signature verification")
	flagCert      = flag.String("cert", "", "Certificate CBOR file. If specified, used instead of fetching from signature's cert-url")
	flagJSON      = flag.Bool("json", false, "Print output as JSON")
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

	certFetcher, err := initCertFetcher()
	if err != nil {
		return err
	}
	verificationTime := time.Now() // TODO: add a flag to override this

	if *flagJSON {
		return jsonPrintHeaders(e, certFetcher, verificationTime, os.Stdout)
	}

	if *flagSignature {
		fmt.Println(e.SignatureHeaderValue)
		return nil
	}

	if *flagVerify {
		if err := verify(e, certFetcher, verificationTime); err != nil {
			return err
		}
		fmt.Println()
	}

	e.PrettyPrintHeaders(os.Stdout)
	e.PrettyPrintPayload(os.Stdout)

	return nil
}

func initCertFetcher() (signedexchange.CertFetcher, error) {
	certFetcher := signedexchange.DefaultCertFetcher
	if *flagCert != "" {
		f, err := os.Open(*flagCert)
		if err != nil {
			return nil, fmt.Errorf("could not %v", err)
		}
		defer f.Close()
		certBytes, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("could not %v", err)
		}
		certFetcher = func(_ string) ([]byte, error) {
			return certBytes, nil
		}
	}
	return certFetcher, nil
}

func verify(e *signedexchange.Exchange, certFetcher signedexchange.CertFetcher, verificationTime time.Time) error {
	if decodedPayload, ok := e.Verify(verificationTime, certFetcher, log.New(os.Stdout, "", 0)); ok {
		e.Payload = decodedPayload
		fmt.Println("The exchange has a valid signature.")
	}
	return nil
}

func jsonPrintHeaders(e *signedexchange.Exchange, certFetcher signedexchange.CertFetcher, verificationTime time.Time, w io.Writer) error {
	// TODO: Add verification error messages to the output.
	_, valid := e.Verify(verificationTime, certFetcher, log.New(ioutil.Discard, "", 0))

	sigs, err := structuredheader.ParseParameterisedList(e.SignatureHeaderValue)
	if err != nil {
		return err
	}

	f := struct {
		Payload              []byte `json:",omitempty"` // hides Payload in nested signedexchange.Exchange
		SignatureHeaderValue []byte `json:",omitempty"` // hides SignatureHeaderValue in nested signedexchange.Exchange
		Valid                bool
		Signatures           structuredheader.ParameterisedList
		*signedexchange.Exchange
	}{
		nil, // omitted via "omitempty"
		nil, // omitted via "omitempty"
		valid,
		sigs,
		e,
	}
	s, _ := json.MarshalIndent(f, "", "  ")
	w.Write(s)

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
