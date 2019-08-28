package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/bundle/signature"
)

var (
	flagInput           = flag.String("i", "in.webbundle", "Webbundle input file")
	flagDumpContentText = flag.Bool("contentText", true, "Dump response content if text")
)

func ReadBundleFromFile(path string) (*bundle.Bundle, error) {
	fi, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open input file %q for reading. err: %v", path, err)
	}
	defer fi.Close()
	return bundle.Read(fi)
}

func DumpExchange(e *bundle.Exchange, b *bundle.Bundle, verifier *signature.Verifier) error {
	payload := e.Response.Body
	if verifier != nil {
		result, err := verifier.VerifyExchange(e)
		if err != nil {
			fmt.Printf("[Response verification error: %v]\n", err)
		} else if result != nil {
			payload = result.VerifiedPayload
			for i, auth := range b.Signatures.Authorities {
				if result.Authority == auth {
					fmt.Printf("[Signed with certificate #%d]\n", i)
					break
				}
			}
		} else {
			fmt.Println("[Not signed]")
		}
	}
	if _, err := fmt.Printf("> :url: %v\n", e.Request.URL); err != nil {
		return err
	}
	for k, v := range e.Request.Header {
		if _, err := fmt.Printf("> %v: %v\n", k, v); err != nil {
			return err
		}
	}
	if _, err := fmt.Printf("< :status: %v\n", e.Response.Status); err != nil {
		return err
	}
	for k, v := range e.Response.Header {
		if _, err := fmt.Printf("< %v: %v\n", k, v); err != nil {
			return err
		}
	}
	if _, err := fmt.Printf("< [len(Body)]: %d\n", len(e.Response.Body)); err != nil {
		return err
	}
	if *flagDumpContentText {
		ctype := e.Response.Header.Get("content-type")
		if strings.Contains(ctype, "text") {
			if _, err := fmt.Print(string(payload)); err != nil {
				return err
			}
			if _, err := fmt.Print("\n"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Print("[non-text body]\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func run() error {
	b, err := ReadBundleFromFile(*flagInput)
	if err != nil {
		return err
	}

	fmt.Printf("Version: %v\n", b.Version)

	if b.Version.HasPrimaryURLField() {
		fmt.Printf("Primary URL: %v\n", b.PrimaryURL)
	}
	if b.ManifestURL != nil {
		fmt.Printf("Manifest URL: %v\n", b.ManifestURL)
	}

	var verifier *signature.Verifier
	if b.Signatures != nil {
		fmt.Println("Signatures:")
		for i, ac := range b.Signatures.Authorities {
			fmt.Printf("  Certificate #%d:\n", i)
			fmt.Println("    Subject:", ac.Cert.Subject.CommonName)
			fmt.Println("    Valid from:", ac.Cert.NotBefore)
			fmt.Println("    Valid until:", ac.Cert.NotAfter)
			fmt.Println("    Issuer:", ac.Cert.Issuer.CommonName)
		}
		var err error
		verifier, err = signature.NewVerifier(b.Signatures, time.Now(), b.Version)
		if err != nil {
			fmt.Printf("Signature verification error: %v\n", err)
		}
	}

	for _, e := range b.Exchanges {
		fmt.Println()
		if err := DumpExchange(e, b, verifier); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
