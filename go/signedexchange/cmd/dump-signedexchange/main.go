package main

import (
	"flag"
	"log"
	"os"

	"github.com/WICG/webpackage/go/signedexchange"
)

var (
	flagInput = flag.String("i", "", "Signed-exchange input file")
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
	e.PrettyPrint(os.Stdout)

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
