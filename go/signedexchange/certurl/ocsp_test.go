package certurl_test

import (
	"github.com/WICG/webpackage/go/signedexchange"
	. "github.com/WICG/webpackage/go/signedexchange/certurl"
	"golang.org/x/crypto/ocsp"
	"io/ioutil"
	"testing"
)

func TestCreateOCSPRequestSamll(t *testing.T) {
	expectedRequestURL := "http://ocsp.digicert.com/MFEwTzBNMEswSTAJBgUrDgMCGgUABBTPJvUY%2Bsl%2Bj4yzQuAcL2oQno5fCgQUUWj%2FkK8CB3U8zNllZGKiErhZcjsCEA5kxfvCNq3hSxcq60HHjLA%3D"

	pem, err := ioutil.ReadFile("test-cert.pem")
	if err != nil {
		t.Fatalf("Cannot read test-cert.pem: %v", err)
	}
	certs, err := signedexchange.ParseCertificates(pem)
	if err != nil {
		t.Fatalf("Cannot parse test-cert.pem: %v", err)
	}

	req, err := CreateOCSPRequest(certs, true)
	if err != nil {
		t.Fatalf("CreateOCSPRequest failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("OCSP request Method:\ngot: %q,\nwant: %q", req.Method, "GET")
	}
	if req.URL.String() != expectedRequestURL {
		t.Errorf("OCSP request URL:\ngot: %q,\nwant: %q", req.URL.String(), expectedRequestURL)
	}
	if req.Header.Get("Accept") != "application/ocsp-response" {
		t.Errorf("OCSP request Accept header:\ngot: %q,\nwant: %q", req.Header.Get("Accept"), "application/ocsp-response")
	}
}

func TestCreateOCSPRequestLarge(t *testing.T) {
	// TODO(tomokinat): create an appropriate test cert and remove the line below.
	t.SkipNow()

	expectedResponderURL := "http://very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-long-ocsp.example.com"

	pem, err := ioutil.ReadFile("test-cert-long.pem")
	if err != nil {
		t.Fatalf("Cannot read test-cert-long.pem: %v", err)
	}
	certs, err := signedexchange.ParseCertificates(pem)
	if err != nil {
		t.Fatalf("Cannot parse test-cert-long.pem: %v", err)
	}

	req, err := CreateOCSPRequest(certs, true)
	if err != nil {
		t.Fatalf("CreateOCSPRequest failed: %v", err)
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

	if req.Body == nil {
		t.Fatalf("Empty request body")
	}
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("Cannot read request body: %v", err)
	}
	if _, err = ocsp.ParseRequest(body); err != nil {
		t.Errorf("Cannot parse request body as an OCSP request: %v", err)
	}
}
