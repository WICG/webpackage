package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/WICG/webpackage/go/signedexchange"
)

var (
	flagInput = flag.String("i", "out.htxg", "Signed exchange file")
)

func run() error {
	r, err := os.Open(*flagInput)
	if err != nil {
		return fmt.Errorf("Failed to open input file \"%s\". err: %v", *flagInput, err)
	}

	e, err := signedexchange.ReadExchangeFile(r)
	if err != nil {
		return fmt.Errorf("Failed to read exchange file: %v", err)
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
