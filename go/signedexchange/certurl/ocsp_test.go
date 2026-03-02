package certurl_test

import (
	"io/ioutil"
	"testing"

	"github.com/WICG/webpackage/go/internal/signingalgorithm"
	. "github.com/WICG/webpackage/go/signedexchange/certurl"
	"golang.org/x/crypto/ocsp"
)

func TestCreateOCSPRequestSmall(t *testing.T) {
	expectedRequestURL := "http://ocsp.digicert.com/MFEwTzBNMEswSTAJBgUrDgMCGgUABBTPJvUY%2Bsl%2Bj4yzQuAcL2oQno5fCgQUUWj%2FkK8CB3U8zNllZGKiErhZcjsCEA5kxfvCNq3hSxcq60HHjLA%3D"

	pem, err := ioutil.ReadFile("test-cert.pem")
	if err != nil {
		t.Fatalf("Cannot read test-cert.pem: %v", err)
	}
	certs, err := signingalgorithm.ParseCertificates(pem)
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
	// $ openssl genrsa -out ca.key 2048
	// $ openssl req -x509 -new -nodes -sha256 -key ca.key -out ca.pem -subj '/CN=example.com/O=Test/C=US'
	// $ openssl ecparam -out test-cert-long.key -name prime256v1 -genkey
	// $ openssl req -new -sha256 -key test-cert-long.key -out test-cert-long.csr -subj /CN=example.com
	// $ openssl x509 -req -in test-cert-long.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out test-cert-long-leaf.pem \
	//     -extfile <(echo -e "1.3.6.1.4.1.11129.2.1.22 = ASN1:NULL\nsubjectAltName=DNS:example.com\nauthorityInfoAccess=OCSP;URI:http://very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-long-ocsp.example.com")
	// $ openssl test-cert-long-leaf.pem ca.pem > test-cert-long.pem

	expectedResponderURL := "http://very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-long-ocsp.example.com"

	pem, err := ioutil.ReadFile("test-cert-long.pem")
	if err != nil {
		t.Fatalf("Cannot read test-cert-long.pem: %v", err)
	}
	certs, err := signingalgorithm.ParseCertificates(pem)
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
