// Converts a text-format web package to cbor-format.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/WICG/webpackage/go/webpack"
	"golang.org/x/crypto/ssh/terminal"
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

	for i := range pack.Manifest.Signatures {
		signWith := &pack.Manifest.Signatures[i]
		if signWith.KeyFilename == "" {
			fmt.Printf("Where is the signing key for %s? ", signWith.CertFilename)
			keyFilename, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				Error.Fatal(err)
			}
			signWith.LoadKey(keyFilename)
		}
		if signWith.Key == nil {
			fmt.Printf("%s is encrypted. Please enter its password: ", signWith.KeyFilename)
			password, err := terminal.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				Error.Fatal(err)
			}
			if err := signWith.GivePassword(password); err != nil {
				Error.Fatal(err)
			}
		}
	}

	err = webpack.WriteCBOR(&pack, out)
	if err != nil {
		Error.Fatal(err)
	}
}
