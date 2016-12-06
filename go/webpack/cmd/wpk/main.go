package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"

	"github.com/dimich-g/webpackage/go/webpack"
)

var (
	Error = log.New(os.Stderr, "E: ", log.Ldate|log.Ltime)

	manifestFlag  = flag.String("manifest", "", "A filename that specifies the manifest for the package to generate.  Defaults to STDIN.")
	packFlag      = flag.String("pack", "", "The name of the package to generate. Defaults to STDOUT.")
	locationFlag  = flag.String("location", "", "The Content-Location for the package. Required.")
	describedFlag = flag.String("describedby", "", "The resource within the package that describes the package.")
	signkeyFlag   = flag.String("signkey", "", "If present, sign the package with the given private key (PEM format). Implies -index.")
	indexFlag     = flag.Bool("index", false, "Whether to generate an index part.")
)

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(file io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func getFile(name string, defaultFile *os.File) (f *os.File, err error) {
	if name == "" {
		return defaultFile, nil
	}

	return os.Open(name)
}

func getWriteableFile(name string, defaultFile *os.File) (f *os.File, err error) {
	if name == "" {
		return defaultFile, nil
	}

	return os.Create(name)
}

func main() {
	flag.Parse()

	switch {
	case *locationFlag == "":
		fmt.Fprintf(os.Stderr, "Location required but not provided.\n")
		flag.Usage()
		os.Exit(2)

	// Some flags are not implemented.  Bail if they are used.
	case *indexFlag != false:
		fmt.Fprintf(os.Stderr, "-index is not implemented.")
		flag.Usage()
		os.Exit(2)
	}

	pack, err := getWriteableFile(*packFlag, os.Stdout)
	if err != nil {
		Error.Println("Error opening for output - ", err)
		os.Exit(1)
	}

	url, err := url.Parse(*locationFlag)
	if err != nil {
		flag.Usage()
		log.Fatal("Couldn't parse location")
	}

	p := webpack.NewPackage(url)
	if *signkeyFlag != "" {
		p.SetSigningKey(*signkeyFlag)
	}
	p.SetDescribedBy(*describedFlag)

	mf, err := getFile(*manifestFlag, os.Stdin)
	defer mf.Close()
	if err != nil {
		log.Fatal(err)
	}

	partNames, err := readLines(bufio.NewReader(mf))
	for _, partText := range partNames {
		p.AddPart(partText, nil)
	}

	_, err = p.WriteTo(pack)
	if err != nil {
		log.Fatal(err)
	}
}
