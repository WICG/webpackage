package main

import (
	"crypto"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/bundle/signature"
	"github.com/WICG/webpackage/go/internal/signingalgorithm"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

func readCertChainFromFile(path string) (certurl.CertChain, error) {
	fi, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	return certurl.ReadCertChain(fi)
}

func readPrivateKeyFromFile(path string) (crypto.PrivateKey, error) {
	privkeytext, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return signingalgorithm.ParsePrivateKey(privkeytext)
}

func readBundleFromFile(path string) (*bundle.Bundle, error) {
	fi, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	return bundle.Read(fi)
}

func writeBundleToFile(b *bundle.Bundle, path string) error {
	fo, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fo.Close()
	_, err = b.WriteTo(fo)
	return err
}

func addSignature(b *bundle.Bundle, signer *signature.Signer) error {
	for _, e := range b.Exchanges {
		if !signer.CanSignForURL(e.Request.URL) {
			continue
		}
		payloadIntegrityHeader, err := e.AddPayloadIntegrity(b.Version, *flagMIRecordSize)
		if err != nil {
			return err
		}
		if err := signer.AddExchange(e, payloadIntegrityHeader); err != nil {
			return err
		}
	}

	newSignatures, err := signer.UpdateSignatures(b.Signatures)
	if err != nil {
		return err
	}
	b.Signatures = newSignatures
	return nil
}

func SignExchanges(privKey crypto.PrivateKey) error {
	certs, err := readCertChainFromFile(*flagCertificate)
	if err != nil {
		return fmt.Errorf("%s: %v", *flagCertificate, err)
	}

	validityUrl, err := url.Parse(*flagValidityUrl)
	if err != nil {
		return fmt.Errorf("failed to parse validity URL %q: %v", *flagValidityUrl, err)
	}

	var date time.Time
	if *flagDate == "" {
		date = time.Now()
	} else {
		var err error
		date, err = time.Parse(time.RFC3339, *flagDate)
		if err != nil {
			return fmt.Errorf("failed to parse date %q: %v", *flagDate, err)
		}
	}

	b, err := readBundleFromFile(*flagInput)
	if err != nil {
		return fmt.Errorf("%s: %v", *flagInput, err)
	}

	signer, err := signature.NewSigner(b.Version, certs, privKey, validityUrl, date, *flagExpire)
	if err != nil {
		return err
	}

	if err := addSignature(b, signer); err != nil {
		return err
	}

	if err := writeBundleToFile(b, *flagOutput); err != nil {
		return fmt.Errorf("%s: %v", *flagOutput, err)
	}
	return nil
}
