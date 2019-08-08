package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/bundle/version"
)

var (
	flagVersion     = flag.String("version", string(version.Unversioned), "The webbundle format version")
	flagHar         = flag.String("har", "", "HTTP Archive (HAR) input file")
	flagDir         = flag.String("dir", "", "Input directory")
	flagBaseURL     = flag.String("baseURL", "", "Base URL (used with -dir)")
	flagPrimaryURL  = flag.String("primaryURL", "", "Primary URL")
	flagManifestURL = flag.String("manifestURL", "", "Manifest URL")
	flagOutput      = flag.String("o", "out.wbn", "Webbundle output file")
)

func main() {
	flag.Parse()

	ver, ok := version.Parse(*flagVersion)
	if !ok {
		log.Fatalf("Error: failed to parse version %q\n", *flagVersion)
	}
	if *flagPrimaryURL == "" {
		fmt.Fprintln(os.Stderr, "Please specify -primaryURL.")
		flag.Usage()
		return
	}
	parsedPrimaryURL, err := url.Parse(*flagPrimaryURL)
	if err != nil {
		log.Fatalf("Failed to parse primary URL. err: %v", err)
	}
	var parsedManifestURL *url.URL
	if len(*flagManifestURL) > 0 {
		parsedManifestURL, err = url.Parse(*flagManifestURL)
		if err != nil {
			log.Fatalf("Failed to parse manifest URL. err: %v", err)
		}
	}

	b := &bundle.Bundle{Version: ver, PrimaryURL: parsedPrimaryURL, ManifestURL: parsedManifestURL}

	if *flagHar != "" {
		if *flagBaseURL != "" {
			fmt.Fprintln(os.Stderr, "Warning: -baseURL is ignored when input is HAR.")
		}
		if *flagPrimaryURL != "" {
			fmt.Fprintln(os.Stderr, "Warning: -primaryURL is ignored when input is HAR.")
		}
		es, err := fromHar(*flagHar)
		if err != nil {
			log.Fatal(err)
		}
		b.Exchanges = es
	} else if *flagDir != "" {
		if *flagBaseURL == "" {
			fmt.Fprintln(os.Stderr, "Please specify -baseURL.")
			flag.Usage()
			return
		}
		parsedBaseURL, err := url.Parse(*flagBaseURL)
		if err != nil {
			log.Fatalf("Failed to parse base URL. err: %v", err)
		}
		es, err := fromDir(*flagDir, parsedBaseURL)
		if err != nil {
			log.Fatal(err)
		}
		b.Exchanges = es
	} else {
		fmt.Fprintln(os.Stderr, "Please specify -har or -dir.")
		flag.Usage()
		return
	}

	fo, err := os.OpenFile(*flagOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to open output file %q for writing. err: %v", *flagOutput, err)
	}
	defer fo.Close()
	if _, err := b.WriteTo(fo); err != nil {
		log.Fatalf("Failed to write exchange. err: %v", err)
	}
}
