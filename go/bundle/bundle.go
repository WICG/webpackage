package bundle

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/WICG/webpackage/go/bundle/version"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

type Request struct {
	*url.URL
	http.Header
}

type Response struct {
	Status int
	http.Header
	Body []byte
}

func (r Response) String() string {
	return fmt.Sprintf("{Status: %q, Header: %v, body: %v}", r.Status, r.Header, string(r.Body))
}

type Exchange struct {
	Request
	Response
}

type Signatures struct {
	Authorities    []*certurl.AugmentedCertificate
	VouchedSubsets []*VouchedSubset
}

type VouchedSubset struct {
	Authority uint64 // index in Authorities
	Sig       []byte
	Signed    []byte
}

type Bundle struct {
	Version     version.Version
	PrimaryURL  *url.URL
	Exchanges   []*Exchange
	ManifestURL *url.URL
	Signatures  *Signatures
}

// AddPayloadIntegrity encodes the exchange's payload with Merkle Integrity
// content encoding, and adds `Content-Encoding` and `Digest` response headers.
// It returns an identifier for the "payload-integrity-header" field of the
// "resource-integrity" structure. [1]
//
// [1] https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#signatures-section
func (e *Exchange) AddPayloadIntegrity(ver version.Version, recordSize int) (string, error) {
	if e.Response.Header.Get("Digest") != "" {
		return "", errors.New("bundle: the exchange already has the Digest: header")
	}

	encoding := ver.MiceEncoding()
	var buf bytes.Buffer
	digest, err := encoding.Encode(&buf, e.Response.Body, recordSize)
	if err != nil {
		return "", err
	}
	e.Response.Body = buf.Bytes()
	e.Response.Header.Add("Content-Encoding", encoding.ContentEncoding())
	e.Response.Header.Add("Digest", digest)
	return encoding.IntegrityIdentifier(), nil
}
