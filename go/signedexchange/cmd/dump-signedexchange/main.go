package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/WICG/webpackage/go/signedexchange/structuredheader"

	"github.com/WICG/webpackage/go/signedexchange"
	"github.com/WICG/webpackage/go/signedexchange/version"
)

type headerArgs []string

func (h *headerArgs) String() string {
	return fmt.Sprintf("%v", *h)
}

func (h *headerArgs) Set(value string) error {
	*h = append(*h, value)
	return nil
}

var latestVersion = string(version.AllVersions[len(version.AllVersions)-1])

var (
	flagCert      = flag.String("cert", "", "Certificate CBOR file. If specified, used instead of fetching from signature's cert-url")
	flagHeaders   = flag.Bool("headers", true, "Print headers")
	flagFilename  = flag.String("i", "", "Signed-exchange input file")
	flagJSON      = flag.Bool("json", false, "Print output as JSON")
	flagPayload   = flag.Bool("payload", true, "Print payload")
	flagSignature = flag.Bool("signature", false, "Print only signature value")
	flagURI       = flag.String("uri", "", "Signed-exchange uri")
	flagVerify    = flag.Bool("verify", false, "Perform signature verification")
	flagVersion   = flag.String("version", latestVersion, "Signed exchange version")

	flagRequestHeader = headerArgs{}
)

func init() {
	flag.Var(&flagRequestHeader, "requestHeader", "Request header arguments")
}

func run() error {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return err
	}
	var e *signedexchange.Exchange
	var in io.Reader = nil
	if *flagFilename != "" { // read sxg from filename
		f, err := os.Open(*flagFilename)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	} else if *flagURI != "" { // read sxg from network
		client := http.DefaultClient
		req, err := http.NewRequest("GET", *flagURI, nil)
		if err != nil {
			return err
		}
		ver, ok := version.Parse(*flagVersion)
		if !ok {
			return fmt.Errorf("failed to parse version %q", *flagVersion)
		}
		mimeType := ver.MimeType()
		req.Header.Add("Accept", mimeType)
		for _, h := range flagRequestHeader {
			chunks := strings.SplitN(h, ":", 2)
			req.Header.Add(strings.TrimSpace(chunks[0]), strings.TrimSpace(chunks[1]))
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		respMimeType := resp.Header.Get("Content-Type")
		if respMimeType != mimeType {
			return fmt.Errorf("GET %q responded with unexpected content type %q", *flagURI, respMimeType)
		}
		in = resp.Body
		defer resp.Body.Close()
	} else if (fi.Mode() & os.ModeCharDevice) == 0 { // read sxg from pipe
		in = os.Stdin
	}

	if in == nil {
		flag.PrintDefaults()
		return nil
	}

	e, err = signedexchange.ReadExchange(in)
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
	} else {
		if *flagHeaders {
			e.PrettyPrintHeaders(os.Stdout)
			if err = e.PrettyPrintHeaderIntegrity(os.Stdout); err != nil {
				return err
			}
		}

		if *flagPayload {
			e.PrettyPrintPayload(os.Stdout)
		}

		if *flagVerify {
			fmt.Println()
			if err := verify(e, certFetcher, verificationTime); err != nil {
				return err
			}
		}
	}

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
	headerIntegrity, err := e.ComputeHeaderIntegrity()
	if err != nil {
		return err
	}

	f := struct {
		Payload              []byte `json:",omitempty"` // hides Payload in nested signedexchange.Exchange
		SignatureHeaderValue []byte `json:",omitempty"` // hides SignatureHeaderValue in nested signedexchange.Exchange
		Valid                bool
		HeaderIntegrity      string
		Signatures           structuredheader.ParameterisedList
		*signedexchange.Exchange
	}{
		nil, // omitted via "omitempty"
		nil, // omitted via "omitempty"
		valid,
		headerIntegrity,
		sigs,
		e,
	}
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "   ")
	if err := enc.Encode(&f); err != nil {
		return err
	}
	w.Write(buf.Bytes())

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
