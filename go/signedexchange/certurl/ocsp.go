package certurl

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/ocsp"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

func CreateOCSPRequest(certs []*x509.Certificate, preferGET bool) (*http.Request, error) {
	if len(certs) < 2 {
		return nil, fmt.Errorf("Could not fetch OCSP response: Issuer certificate not found")
	}
	cert := certs[0]
	if len(cert.OCSPServer) == 0 {
		return nil, fmt.Errorf("Could not fetch OCSP response: No OCSP responder field")
	}
	ocspUrl, err := url.Parse(cert.OCSPServer[0])
	if err != nil {
		return nil, fmt.Errorf("Could not fetch OCSP respones: Invalid OCSP responder URL: %s", err)
	}
	issuer := certs[1]

	ocspRequest, err := ocsp.CreateRequest(cert, issuer, &ocsp.RequestOptions{})
	if err != nil {
		return nil, err
	}

	// Use GET for small requests per https://tools.ietf.org/html/rfc5019#section-5.

	// Use QueryEscape here instead of PathEscape to escape not only '/' but also '+' and '='
	// to align with the exmaple in https://tools.ietf.org/html/rfc5019#section-5.
	// Note that the string to be escaped doesn't contain other symbols as it is base64 encoded.
	getUrl, err := ocspUrl.Parse(url.QueryEscape(base64.StdEncoding.EncodeToString(ocspRequest)))
	if preferGET && err == nil && len(getUrl.String()) <= 255 {
		request, err := http.NewRequest("GET", getUrl.String(), nil)
		if err != nil {
			return nil, err
		}
		request.Header.Add("Accept", "application/ocsp-response")
		return request, nil
	} else {
		// Fallback to "POST" if the Parse above fails.
		request, err := http.NewRequest("POST", ocspUrl.String(), bytes.NewReader(ocspRequest))
		if err != nil {
			return nil, err
		}
		request.Header.Add("Content-Type", "application/ocsp-request")
		request.Header.Add("Accept", "application/ocsp-response")
		return request, nil
	}
}

func FetchOCSPResponse(certs []*x509.Certificate) ([]byte, error) {
	request, err := CreateOCSPRequest(certs, true)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	output, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (chain CertChain) prettyPrintOCSP(w io.Writer, OCSPResponse []byte) {
	var issuer *x509.Certificate
	if len(chain) >= 2 {
		issuer = chain[1].Cert
	}
	o, err := ocsp.ParseResponseForCert(OCSPResponse, chain[0].Cert, issuer)
	if err != nil {
		fmt.Fprintln(w, "Error: Invalid OCSP response:", err)
		return
	}
	ocspStatusToString := map[int]string{
		ocsp.Good:    "good",
		ocsp.Revoked: "revoked",
		ocsp.Unknown: "unknown",
	}
	fmt.Fprintf(w, "  Status: %d (%s)\n", o.Status, ocspStatusToString[o.Status])
	fmt.Fprintln(w, "  ProducedAt:", o.ProducedAt)
	fmt.Fprintln(w, "  ThisUpdate:", o.ThisUpdate)
	fmt.Fprintln(w, "  NextUpdate:", o.NextUpdate)

	prettyPrintSCTFromOCSP(w, o)
}
