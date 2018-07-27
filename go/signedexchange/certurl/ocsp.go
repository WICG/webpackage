package certurl

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"golang.org/x/crypto/ocsp"
	"io/ioutil"
	"net/http"
)

func CreateOCSPRequest(certs []*x509.Certificate) (*http.Request, error) {
	if len(certs) < 2 {
		return nil, fmt.Errorf("Could not fetch OCSP response: Issuer certificate not found")
	}
	cert := certs[0]
	if len(cert.OCSPServer) == 0 {
		return nil, fmt.Errorf("Could not fetch OCSP response: No OCSP responder field")
	}
	ocspUrl := cert.OCSPServer[0]
	issuer := certs[1]

	ocspRequest, err := ocsp.CreateRequest(cert, issuer, &ocsp.RequestOptions{})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", ocspUrl, bytes.NewReader(ocspRequest))
	if err != nil {
		return nil, err
	}
	request.Header.Add("Content-Type", "application/ocsp-request")
	request.Header.Add("Accept", "application/ocsp-response")
	return request, nil
}

func FetchOCSPResponse(certs []*x509.Certificate) ([]byte, error) {
	request, err := CreateOCSPRequest(certs)
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
