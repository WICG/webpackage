package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/WICG/webpackage/go/integrityblock"
)

const (
	signaturesSectionSubCmdName = "signatures-section"
	integrityBlockSubCmdName    = "integrity-block"
	dumpWebBundleIdSubCmdName   = "dump-id"
)

var (
	signedExchangesCmd  = flag.NewFlagSet(signaturesSectionSubCmdName, flag.ExitOnError)
	sxgFlagInput        = signedExchangesCmd.String("i", "in.wbn", "Webbundle input file")
	sxgFlagOutput       = signedExchangesCmd.String("o", "out.wbn", "Webbundle output file")
	sxgFlagCertificate  = signedExchangesCmd.String("certificate", "cert.cbor", "Certificate chain CBOR file")
	sxgFlagPrivateKey   = signedExchangesCmd.String("privateKey", "cert-key.pem", "Private key PEM file")
	sxgFlagValidityUrl  = signedExchangesCmd.String("validityUrl", "https://example.com/resource.validity.msg", "The URL where resource validity info is hosted at.")
	sxgFlagDate         = signedExchangesCmd.String("date", "", "Datetime for the signature in RFC3339 format (2006-01-02T15:04:05Z). (default: current time)")
	sxgFlagExpire       = signedExchangesCmd.Duration("expire", 1*time.Hour, "Validity duration of the signature")
	sxgFlagMIRecordSize = signedExchangesCmd.Int("miRecordSize", 4096, "Record size of Merkle Integrity Content Encoding")
)

var (
	integrityBlockCmd = flag.NewFlagSet(integrityBlockSubCmdName, flag.ExitOnError)
	ibFlagInput       = integrityBlockCmd.String("i", "in.wbn", "Webbundle input file")
	ibFlagOutput      = integrityBlockCmd.String("o", "out.wbn", "Webbundle output file")
	ibFlagPrivateKey  = integrityBlockCmd.String("privateKey", "privatekey.pem", "Private key PEM file")
)

const flagNamePublicKey = "publicKey"

var (
	dumpWebBundleIdCmd   = flag.NewFlagSet(dumpWebBundleIdSubCmdName, flag.ExitOnError)
	dumpIdFlagPrivateKey = dumpWebBundleIdCmd.String("privateKey", "privatekey.pem", "Private key PEM file whose corresponding Web Bundle ID is wanted.")
	dumpIdFlagPublicKey  = dumpWebBundleIdCmd.String(flagNamePublicKey, "", "Public key PEM file whose corresponding Web Bundle ID is wanted.")
)

// isFlagPassed is a helper function to check if the given flag was provided. Note that this needs to be called after flag.Parse.
func isFlagPassed(flags *flag.FlagSet, name string) bool {
	found := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func run() error {
	switch os.Args[1] {

	case signaturesSectionSubCmdName:
		signedExchangesCmd.Parse(os.Args[2:])
		return SignExchanges()

	case integrityBlockSubCmdName:
		integrityBlockCmd.Parse(os.Args[2:])

		// TODO(sonkkeli): Add parsing for the new `signingStrategy` flag and
		// make another switch case for the different types of signing.

		ed25519privKey, err := readAndParseEd25519PrivateKey(*ibFlagPrivateKey)
		if err != nil {
			return err
		}

		var bss integrityblock.ISigningStrategy = integrityblock.NewParsedEd25519KeySigningStrategy(ed25519privKey)
		return SignWithIntegrityBlockWithCmdFlags(bss)

	case dumpWebBundleIdSubCmdName:
		dumpWebBundleIdCmd.Parse(os.Args[2:])
		return DumpWebBundleId()

	default:
		return errors.New(fmt.Sprintf("Unknown subcommand, try '%s', '%s' or '%s'", signaturesSectionSubCmdName, integrityBlockSubCmdName, dumpWebBundleIdSubCmdName))
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
