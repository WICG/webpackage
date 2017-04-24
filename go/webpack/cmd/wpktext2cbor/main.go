// Converts a text-format web package to cbor-format.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/dimich-g/webpackage/go/webpack"
)

var (
	Error = log.New(os.Stderr, "", 0)

	manifestFilename = flag.String("i", "", "A filename to write the CBOR-format package to. No defaults")
	outFlag          = flag.String("o", "", "A filename to write the CBOR-format package to. Defaults to STDOUT")
)

func main() {
	flag.Parse()

	out := os.Stdout
	var err error
	if *outFlag != "" {
		out, err = os.Create(*outFlag)
		if err != nil {
			Error.Printf("Can't create output file: %v", err)

			flag.Usage()
			os.Exit(1)
		}
	}

	if *manifestFilename == "" {
		Error.Print("Must specify -i manifestFile")
		flag.Usage()
		os.Exit(1)
	}

	pack, err := webpack.ParseText(*manifestFilename)
	if err != nil {
		Error.Fatal(err)
	}

	cbor, err := webpack.WriteCbor(&pack)
	if err != nil {
		Error.Fatal(err)
	}

	n, err := out.Write(cbor)
	if err != nil {
		Error.Fatal(err)
	}
	if n != len(cbor) {
		Error.Fatal("Failed to write the whole package.")
	}
}
