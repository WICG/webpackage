package signedexchange

import (
	"bytes"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/WICG/webpackage/go/signedexchange/certurl"
	"github.com/WICG/webpackage/go/signedexchange/internal/signingalgorithm"
	"github.com/WICG/webpackage/go/signedexchange/mice"
	"github.com/WICG/webpackage/go/signedexchange/structuredheader"
	"github.com/WICG/webpackage/go/signedexchange/version"
)

// draft-yasskin-http-origin-signed-responses.html#signature-validity
// Step 8. "If validating integrity using the selected header field requires
// the client to process records larger than 16384 bytes, return "invalid"."
const maxMIRecordSize = 16384

type Signature struct {
	Label       structuredheader.Token
	Sig         []byte
	Integrity   string
	CertUrl     string
	CertSha256  []byte
	ValidityUrl string
	Date        int64
	Expires     int64
}

func extractSignatureFields(pi structuredheader.ParameterisedIdentifier) (*Signature, error) {
	sig := &Signature{Label: pi.Label}
	params := pi.Params
	var ok bool
	if sig.Sig, ok = params["sig"].([]byte); !ok {
		return nil, errors.New("verify: no valid 'sig' value")
	}
	if sig.Integrity, ok = params["integrity"].(string); !ok {
		return nil, errors.New("verify: no valid 'integrity' value")
	}
	if sig.CertUrl, ok = params["cert-url"].(string); !ok {
		return nil, errors.New("verify: no valid 'cert-url' value")
	}
	if sig.CertSha256, ok = params["cert-sha256"].([]byte); !ok {
		return nil, errors.New("verify: no valid 'cert-sha256' value")
	}
	if sig.ValidityUrl, ok = params["validity-url"].(string); !ok {
		return nil, errors.New("verify: no valid 'validity-url' value")
	}
	if sig.Date, ok = params["date"].(int64); !ok {
		return nil, errors.New("verify: no valid 'date' value")
	}
	if sig.Expires, ok = params["expires"].(int64); !ok {
		return nil, errors.New("verify: no valid 'expires' value")
	}
	return sig, nil
}

// CertFetcher takes certificat URL and returns certificate bytes in
// application/cert-chain+cbor format.
type CertFetcher = func(url string) ([]byte, error)

// DefaultCertFetcher fetches certificates using http.Get.
func DefaultCertFetcher(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("verify: could not fetch %q: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("verify: could not read response body of %q: %v", url, err)
	}
	return body, nil
}

// Verify validates the Exchange by running the algorithm described in
// https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cross-origin-trust.
// Signature timestamps are checked against verificationTime.
// Certificates for signatures are fetched using certFetcher.
// Errors encountered during verification are logged to l.
// If successful, it returns the decoded payload and true. otherwise it returns
// nil and false.
func (e *Exchange) Verify(verificationTime time.Time, certFetcher CertFetcher, l *log.Logger) ([]byte, bool) {
	// draft-yasskin-http-origin-signed-responses.html#cross-origin-trust

	// "The client MUST parse the Signature header into a list of signatures
	// according to the instructions in Section 3.5, ..."
	signatures, err := structuredheader.ParseParameterisedList(e.SignatureHeaderValue)
	if err != nil {
		l.Printf("Could not parse signature header: %v", err)
		return nil, false
	}
	// "...and run the following algorithm for each signature, stopping at the
	// first one that returns "valid". If any signature returns "valid", return
	// "valid". Otherwise, return "invalid"."
	for _, item := range signatures {
		signature, err := extractSignatureFields(item)
		if err != nil {
			l.Printf("Invalid signature: %v", err)
			continue
		}
		// Step 1: "If the signature's "validity-url" parameter is not
		//         same-origin with requestUrl, return "invalid"."
		validityUrl, err := url.Parse(signature.ValidityUrl)
		if err != nil {
			l.Printf("Cannot parse validity-url: %q", signature.ValidityUrl)
			continue
		}
		requestURI, err := url.Parse(e.RequestURI)
		if err != nil {
			l.Printf("Cannot parse request URI: %q", e.RequestURI)
			continue
		}
		if !isSameOrigin(validityUrl, requestURI) {
			l.Printf("validity-url (%s) is not same-origin with request URL (%v)", signature.ValidityUrl, e.RequestURI)
			continue
		}

		// Step 2: "Use Section 3.5 to determine the signature's validity for
		//         requestUrl, responseHeaders, and payload, getting
		//         certificate-chain back. If this returned "invalid" or didn't
		//         return a certificate chain, return "invalid"."
		_, decodedPayload, err := verifySignature(e, verificationTime, certFetcher, signature)
		if err != nil {
			l.Printf("Verification of sinature %q failed: %v", signature.Label, err)
			continue
		}

		// Step 3: "Let response be the exchange metadata and headers parsed out
		//         of responseHeaders."
		// `e` contains the exchange metadata and headers.

		if e.Version == version.Version1b1 || e.Version == version.Version1b2 {
			// Version 1b1 and 1b2 only -- Step 4 of
			// https://tools.ietf.org/html/draft-yasskin-httpbis-origin-signed-exchanges-impl-02#section-4:
			// "If exchange's request method is not safe (Section 4.2.1 of
			// [RFC7231]) or not cacheable (Section 4.2.3 of [RFC7231]),
			// return "invalid"."
			// Per [RFC7231], only GET and HEAD are safe and cacheable.
			if e.RequestMethod != http.MethodGet && e.RequestMethod != http.MethodHead {
				l.Printf("Request method %q is not safe or not cacheable.", e.RequestMethod)
				continue
			}
		}

		// Step 4: If Section 3 of [RFC7234] forbids a shared cache from storing
		//         response, return "invalid".
		if e.Version != version.Version1b1 && e.Version != version.Version1b2 && !e.IsCacheable(l) {
			continue
		}

		// Step 5: "If response's headers contain an uncached header field, as
		//         defined in Section 4.1, return "invalid"."
		if err := verifyHeaders(e); err != nil {
			l.Printf("Header validation failed: %v", err)
			continue
		}

		// TODO: Implement Step 6 and 7 (certificate verification).

		// Step 8: "Return "valid"."
		return decodedPayload, true
	}
	return nil, false
}

// IsCacheable returns true if Exchange is cacheable by a shared cache
// (Section 3 of [RFC7234]).
func (e *Exchange) IsCacheable(l *log.Logger) bool {
	if e.Version == version.Version1b1 || e.Version == version.Version1b2 {
		panic("IsCacheable is only applicable to version b3 or later")
	}

	// "A cache MUST NOT store a response to any request, unless:"
	//
	// "o  The request method is understood by the cache and defined as being
	//     cacheable, and"

	// Version b3 and later don't have a request method.

	// "o  the response status code is understood by the cache, and"

	// Check if the status code is understood by the net/http package.
	if http.StatusText(e.ResponseStatus) == "" {
		l.Printf("Unknown response status %d", e.ResponseStatus)
		return false
	}

	cacheDirectives := parseCacheControlDirectives(e.ResponseHeaders.Get("Cache-Control"))

	// "o  the "no-store" cache directive (see Section 5.2) does not appear
	//     in request or response header fields, and"
	if _, ok := cacheDirectives["no-store"]; ok {
		l.Print("Response has the \"no-store\" cache directive")
		return false
	}

	// "o  the "private" response directive (see Section 5.2.2.6) does not
	//     ppear in the response, if the cache is shared, and"
	if _, ok := cacheDirectives["private"]; ok {
		l.Print("Response has the \"private\" response directive")
		return false
	}

	// "o  the Authorization header field (see Section 4.2 of [RFC7235]) does
	//     not appear in the request, if the cache is shared, unless the
	//     response explicitly allows it (see Section 3.2), and"

	// Version b3 and later don't have request headers.

	// "o  the response either:"
	//
	// "  *  contains an Expires header field (see Section 5.3), or"
	if e.ResponseHeaders.Get("Expires") != "" {
		return true
	}

	// "  *  contains a max-age response directive (see Section 5.2.2.8), or"
	if _, ok := cacheDirectives["max-age"]; ok {
		return true
	}

	// "  *  contains a s-maxage response directive (see Section 5.2.2.9)
	//       and the cache is shared, or"
	if _, ok := cacheDirectives["s-maxage"]; ok {
		return true
	}

	// "  *  contains a Cache Control Extension (see Section 5.2.3) that
	//       allows it to be cached, or"
	// This implementation does not recognize any Cache Control Extension.

	// "  *  has a status code that is defined as cacheable by default (see
	//       Section 4.2.2), or"

	// Status codes that are cachable by default (Section 6.1 of [RFC7231]).
	// The elements must be sorted.
	CacheableStatusCodes := []int{
		200, 203, 204, 206, 300, 301, 404, 405, 410, 414, 501,
	}
	i := sort.SearchInts(CacheableStatusCodes, e.ResponseStatus)
	if i < len(CacheableStatusCodes) && CacheableStatusCodes[i] == e.ResponseStatus {
		return true
	}

	// "  *  contains a public response directive (see Section 5.2.2.5)."
	if _, ok := cacheDirectives["public"]; ok {
		return true
	}

	l.Print("Response is not cacheable by a shared cache")
	return false
}

// parseCacheControlDirectives parses a Cache-Control header value
// (Section 5.2 of [RFC7234]).
func parseCacheControlDirectives(cacheControl string) map[string]string {
	directives := map[string]string{}
	// TODO: correctly handle quoted-string arguments.
	for _, s := range strings.Split(cacheControl, ",") {
		s = strings.TrimSpace(s)
		eq := strings.IndexByte(s, '=')
		if eq >= 0 {
			directives[strings.ToLower(s[:eq])] = s[eq+1:]
		} else {
			directives[strings.ToLower(s)] = ""
		}
	}
	return directives
}

// verifySignature verifies single signature, as described in
// https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#signature-validity.
// On success, returns a potentially-valid cert chain and decoded payload bytes.
func verifySignature(e *Exchange, verificationTime time.Time, fetch CertFetcher, signature *Signature) (certurl.CertChain, []byte, error) {
	// Step 1: Extract the signature fields
	// |signature| is the parsed signature.

	// Step 2: Fetch cert-url and determine the signing algorithm
	certBytes, err := fetch(signature.CertUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("verify: failed to fetch %q: %v", signature.CertUrl, err)
	}
	certs, err := certurl.ReadCertChain(bytes.NewReader(certBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("verify: could not parse certificate CBOR: %v", err)
	}
	mainCert := certs[0].Cert
	verifier, err := signingalgorithm.VerifierForPublicKey(mainCert.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("verify: unsupported main certificate public key: %v", err)
	}

	// Step 3 and 4: Timestamp checks
	if err := verifyTimestamps(signature, verificationTime); err != nil {
		return nil, nil, err
	}
	// Step 5: Reconstruct the signing message
	certSha256 := calculateCertSha256([]*x509.Certificate{mainCert})
	if certSha256 == nil {
		return nil, nil, errors.New("verify: cannot calculate certificate fingerprint")
	}
	msg, err := serializeSignedMessage(e, certSha256, signature.ValidityUrl, signature.Date, signature.Expires)
	if err != nil {
		return nil, nil, errors.New("verify: cannot reconstruct signed message")
	}
	// Step 6: Cert-sha256 check
	if !bytes.Equal(signature.CertSha256, certSha256) {
		return nil, nil, errors.New("verify: cert-sha256 mismatch")
	}
	// Step 7: Signature verification
	ok, err := verifier.Verify(msg, signature.Sig)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, errors.New("verify: signature verification failed")
	}
	// Step 8: (version >= 1b3) Response headers must contain Content-Type
	if e.Version != version.Version1b1 && e.Version != version.Version1b2 {
		if e.ResponseHeaders.Get("Content-Type") == "" {
			return nil, nil, errors.New("verify: Content-Type response header is absent")
		}
	}
	// Step 9: Payload integrity check
	decodedPayload, err := verifyPayload(e, signature)
	if err != nil {
		return nil, nil, err
	}

	// Step 10: Return "potentially-valid" with certificate-chain.
	return certs, decodedPayload, nil
}

func verifyTimestamps(sig *Signature, verificationTime time.Time) error {
	expiresTime := time.Unix(sig.Expires, 0)
	creationTime := time.Unix(sig.Date, 0)
	if expiresTime.Sub(creationTime) > 7*24*time.Hour {
		return fmt.Errorf("verify: expires (%v) is more than 7 days (604800 seconds) after date (%v)", expiresTime, creationTime)
	}
	if verificationTime.Before(creationTime) {
		return fmt.Errorf("verify: signature is not yet valid. date=%d (%v)", sig.Date, creationTime)
	}
	if verificationTime.After(expiresTime) {
		return fmt.Errorf("verify: signature is expired. expires=%d (%v)", sig.Expires, expiresTime)
	}
	return nil
}

func verifyPayload(e *Exchange, signature *Signature) ([]byte, error) {
	var integrityStr string
	var enc mice.Encoding
	switch e.Version {
	case version.Version1b1:
		enc = mice.Draft02Encoding
		integrityStr = "mi-draft2"
	case version.Version1b2, version.Version1b3:
		enc = mice.Draft03Encoding
		integrityStr = "digest/" + enc.ContentEncoding()
	default:
		panic("not reached")
	}
	if signature.Integrity != integrityStr {
		return nil, fmt.Errorf("verify: unsupported integrity scheme %q", signature.Integrity)
	}
	digest := e.ResponseHeaders.Get(enc.DigestHeaderName())
	if digest == "" {
		return nil, fmt.Errorf("verify: response header %q not present", enc.DigestHeaderName())
	}
	dec, err := enc.NewDecoder(bytes.NewReader(e.Payload), digest, maxMIRecordSize)
	if err != nil {
		return nil, err
	}
	decoded, err := ioutil.ReadAll(dec)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func isSameOrigin(u1, u2 *url.URL) bool {
	return u1.Scheme == u2.Scheme && u1.Host == u2.Host
}

func verifyHeaders(e *Exchange) error {
	for k := range e.RequestHeaders {
		if IsStatefulRequestHeader(k) {
			return fmt.Errorf("exchange has stateful request header %q", k)
		}
	}
	if err := VerifyUncachedHeader(e.ResponseHeaders); err != nil {
		return err
	}
	return nil
}
