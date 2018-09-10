package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/WICG/webpackage/go/signedexchange"
)

var (
	flagInput     = flag.String("i", "", "Signed-exchange input file")
	flagSignature = flag.Bool("signature", false, "Print only signature value")
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
	} else {
		e.PrettyPrint(os.Stdout)
	}

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
