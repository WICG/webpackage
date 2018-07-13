package certurl_test

import (
	"golang.org/x/crypto/ocsp"
	"io/ioutil"
	"testing"

	. "github.com/WICG/webpackage/go/signedexchange/certurl"
	"github.com/WICG/webpackage/go/signedexchange"
)

func TestOCSP(t *testing.T) {
	expectedResponderURL := "http://ocsp.digicert.com"

	pem, err := ioutil.ReadFile("test-cert.pem")
	if err != nil {
		t.Errorf("Cannot read test-cert.pem: %v", err)
		return
	}
	certs, err := signedexchange.ParseCertificates(pem)
	if err != nil {
		t.Errorf("Cannot parse test-cert.pem: %v", err)
		return
	}

	req, err := CreateOCSPRequest(certs)
	if err != nil {
		t.Errorf("CreateOCSPRequest failed: %v", err)
		return
	}
	if req.Method != "POST" {
		t.Errorf("OCSP request Method:\ngot: %q,\nwant: %q", req.Method, "POST")
	}
	if req.URL.String() != expectedResponderURL {
		t.Errorf("OCSP request URL:\ngot: %q,\nwant: %q", req.URL.String(), expectedResponderURL)
	}
	if req.Header.Get("Content-Type") != "application/ocsp-request" {
		t.Errorf("OCSP request Content-Type header:\ngot: %q,\nwant: %q", req.Header.Get("Content-Type"), "application/ocsp-request")
	}
	if req.Header.Get("Accept") != "application/ocsp-response" {
		t.Errorf("OCSP request Accept header:\ngot: %q,\nwant: %q", req.Header.Get("Accept"), "application/ocsp-response")
	}

	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Cannot read request body: %v", err)
		return
	}
	if _, err = ocsp.ParseRequest(body); err != nil {
		t.Errorf("Cannot parse request body as an OCSP request: %v", err)
	}
}
