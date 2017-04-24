// Decodes hex-encoded HPACK data given as command-line arguments into
// Name: Value pairs written to stdout.
package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"golang.org/x/net/http2/hpack"
)

func main() {
	dec := hpack.NewDecoder(1<<10, func(f hpack.HeaderField) {
		fmt.Fprintf(os.Stdout, "%s: %s\n", f.Name, f.Value)
	})
	for _, arg := range os.Args[1:] {
		binary, err := hex.DecodeString(arg)
		if err != nil {
			log.Fatalf("Invalid input: %v", err)
		}
		_, err = dec.Write(binary)
		if err != nil {
			log.Fatalf("Invalid input: %v", err)
		}
	}
}
