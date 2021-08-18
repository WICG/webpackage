package bundle_test

import (
	"bytes"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	. "github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/bundle/version"
	"github.com/WICG/webpackage/go/signedexchange"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

const pemCerts = `-----BEGIN CERTIFICATE-----
MIIBhjCCAS2gAwIBAgIJAOhR3xtYd5QsMAoGCCqGSM49BAMCMDIxFDASBgNVBAMM
C2V4YW1wbGUub3JnMQ0wCwYDVQQKDARUZXN0MQswCQYDVQQGEwJVUzAeFw0xODEx
MDUwOTA5MjJaFw0xOTEwMzEwOTA5MjJaMDIxFDASBgNVBAMMC2V4YW1wbGUub3Jn
MQ0wCwYDVQQKDARUZXN0MQswCQYDVQQGEwJVUzBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABH1E6odXRm3+r7dMYmkJRmftx5IYHAsqgA7zjsFfCvPqL/fM4Uvi8EFu
JVQM/oKEZw3foCZ1KBjo/6Tenkoj/wCjLDAqMBAGCisGAQQB1nkCARYEAgUAMBYG
A1UdEQQPMA2CC2V4YW1wbGUub3JnMAoGCCqGSM49BAMCA0cAMEQCIEbxRKhlQYlw
Ja+O9h7misjLil82Q82nhOtl4j96awZgAiB6xrvRZIlMtWYKdi41BTb5fX22gL9M
L/twWg8eWpYeJA==
-----END CERTIFICATE-----
`

func urlMustParse(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}

func createTestBundle(t *testing.T, ver version.Version) *Bundle {
	bundle := &Bundle{
		Version: ver,
		Exchanges: []*Exchange{
			&Exchange{
				Request{
					URL: urlMustParse("https://bundle.example.com/"),
				},
				Response{
					Status: 200,
					Header: http.Header{"Content-Type": []string{"text/html"}},
					Body:   []byte("hello, world!"),
				},
			},
		},
		Signatures: &Signatures{
			Authorities: createTestCerts(t),
			VouchedSubsets: []*VouchedSubset{
				&VouchedSubset{Authority: 0, Sig: []byte("sig"), Signed: []byte("sig")},
			},
		},
	}
	if ver == version.Unversioned {
		bundle.Exchanges[0].Request.Header = make(http.Header)
		bundle.Signatures = nil // Unversioned bundle cannot have signatures.
	}
	if ver.HasPrimaryURLFieldInHeader() {
		bundle.PrimaryURL = urlMustParse("https://bundle.example.com/")
	}
	return bundle
}

func createTestBundleWithVariants(ver version.Version) *Bundle {
	primaryURL := urlMustParse("https://variants.example.com/")
	return &Bundle{
		Version:    ver,
		PrimaryURL: primaryURL,
		Exchanges: []*Exchange{
			&Exchange{
				Request{URL: primaryURL},
				Response{
					Status: 200,
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
						"Variants":     []string{"Accept-Language;en;ja"},
						"Variant-Key":  []string{"en"},
					},
					Body: []byte("Hello, world!"),
				},
			},
			&Exchange{
				Request{URL: primaryURL},
				Response{
					Status: 200,
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
						"Variants":     []string{"Accept-Language;en;ja"},
						"Variant-Key":  []string{"ja"},
					},
					Body: []byte("こんにちは世界"),
				},
			},
		},
	}
}

func createTestCerts(t *testing.T) []*certurl.AugmentedCertificate {
	certs, err := signedexchange.ParseCertificates([]byte(pemCerts))
	if err != nil {
		t.Fatal(err)
	}
	var acs []*certurl.AugmentedCertificate
	for _, c := range certs {
		acs = append(acs, &certurl.AugmentedCertificate{Cert: c})
	}
	return acs
}

func TestWriteAndRead(t *testing.T) {
	for _, ver := range version.AllVersions {
		bundle := createTestBundle(t, ver)

		var buf bytes.Buffer
		n, err := bundle.WriteTo(&buf)
		if err != nil {
			t.Errorf("Bundle.WriteTo unexpectedly failed: %v", err)
		}
		if n != int64(buf.Len()) {
			t.Errorf("Bundle.WriteTo returned %d, but wrote %d bytes", n, buf.Len())
		}

		deserialized, err := Read(&buf)
		if err != nil {
			t.Errorf("Bundle.Read unexpectedly failed: %v", err)
		}
		if !reflect.DeepEqual(deserialized, bundle) {
			t.Errorf("got: %v\nwant: %v", deserialized, bundle)
		}
	}
}

func TestWriteAndReadWithVariants(t *testing.T) {
	for _, ver := range version.AllVersions {
		if !ver.SupportsVariants() {
			continue
		}
		bundle := createTestBundleWithVariants(ver)

		var buf bytes.Buffer
		if _, err := bundle.WriteTo(&buf); err != nil {
			t.Errorf("Bundle.WriteTo unexpectedly failed: %v", err)
		}
		deserialized, err := Read(&buf)
		if err != nil {
			t.Errorf("Bundle.Read unexpectedly failed: %v", err)
		}
		if !reflect.DeepEqual(deserialized, bundle) {
			t.Errorf("got: %v\nwant: %v", deserialized, bundle)
		}
	}
}
