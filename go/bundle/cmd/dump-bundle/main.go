package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/WICG/webpackage/go/bundle"
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

func run() error {
	b, err := ReadBundleFromFile(*flagInput)
	if err != nil {
		return err
	}

	if b.ManifestURL != nil {
		fmt.Printf("Manifest URL: %v\n", b.ManifestURL)
	}

	for _, e := range b.Exchanges {
		if err := e.Dump(os.Stdout, *flagDumpContentText); err != nil {
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
