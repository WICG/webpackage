package main

import (
	"flag"
	"log"
	"os"

	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

var (
	flagInput = flag.String("i", "", "Cerrt-chain CBOR input file")
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

	chain, err := certurl.ReadCertChain(in)
	if err != nil {
		return err
	}
	chain.PrettyPrint(os.Stdout)

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
