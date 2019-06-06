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

	if *flagJSON {
		jsonPrintHeaders(e, time.Now(), os.Stdout)
		return nil
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

func initCertFetcher() (func(url string) ([]byte, error), error) {
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

func verify(e *signedexchange.Exchange) error {
	certFetcher, err := initCertFetcher()
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	verificationTime := time.Now()
	if decodedPayload, ok := e.Verify(verificationTime, certFetcher, log.New(os.Stdout, "", 0)); ok {
		e.Payload = decodedPayload
		fmt.Println("The exchange has a valid signature.")
	}
	return nil
}

func jsonPrintHeaders(e *signedexchange.Exchange, verificationTime time.Time, w io.Writer) {
	certFetcher, err := initCertFetcher()
	if err != nil {
		log.Println(err.Error())
		return
	}
	_, valid := e.Verify(verificationTime, certFetcher, log.New(ioutil.Discard, "", 0))
	shv, ok := structuredheader.ParseParameterisedList(e.SignatureHeaderValue)
	var sig structuredheader.Parameters
	if ok == nil && len(shv) > 0 {
		sig = shv[0].Params
	} else {
		sig = structuredheader.Parameters{}
	}
	f := struct {
		Payload              []byte                      `json:",omitempty"`
		SignatureHeaderValue structuredheader.Parameters `json:",omitempty"`
		Valid                bool
		Signature            structuredheader.Parameters
		*signedexchange.Exchange
	}{
		nil, // omitted via "omitempty"
		nil, // omitted via "omitempty"
		valid,
		sig,
		e,
	}
	s, _ := json.MarshalIndent(f, "", "  ")
	w.Write(s)
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
