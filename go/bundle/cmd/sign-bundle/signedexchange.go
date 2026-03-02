package main

import (
	"crypto"
	"crypto/ecdsa"
	"errors"
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
		payloadIntegrityHeader, err := e.AddPayloadIntegrity(b.Version, *sxgFlagMIRecordSize)
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

func SignExchanges() error {
	privKey, err := readPrivateKeyFromFile(*sxgFlagPrivateKey)
	if err != nil {
		return fmt.Errorf("%s: %v", *sxgFlagPrivateKey, err)
	}

	if _, ok := privKey.(*ecdsa.PrivateKey); !ok {
		return errors.New("Private key is not ECDSA type.")
	}

	certs, err := readCertChainFromFile(*sxgFlagCertificate)
	if err != nil {
		return fmt.Errorf("%s: %v", *sxgFlagCertificate, err)
	}

	validityUrl, err := url.Parse(*sxgFlagValidityUrl)
	if err != nil {
		return fmt.Errorf("failed to parse validity URL %q: %v", *sxgFlagValidityUrl, err)
	}

	var date time.Time
	if *sxgFlagDate == "" {
		date = time.Now()
	} else {
		var err error
		date, err = time.Parse(time.RFC3339, *sxgFlagDate)
		if err != nil {
			return fmt.Errorf("failed to parse date %q: %v", *sxgFlagDate, err)
		}
	}

	b, err := readBundleFromFile(*sxgFlagInput)
	if err != nil {
		return fmt.Errorf("%s: %v", *sxgFlagInput, err)
	}

	signer, err := signature.NewSigner(b.Version, certs, privKey, validityUrl, date, *sxgFlagExpire)
	if err != nil {
		return err
	}

	if err := addSignature(b, signer); err != nil {
		return err
	}

	if err := writeBundleToFile(b, *sxgFlagOutput); err != nil {
		return fmt.Errorf("%s: %v", *sxgFlagOutput, err)
	}
	return nil
}
