package signature

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/bundle/version"
	"github.com/WICG/webpackage/go/internal/signingalgorithm"
	"github.com/WICG/webpackage/go/signedexchange/cbor"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

// draft-yasskin-http-origin-signed-responses.html#signature-validity
// Step 8. "If validating integrity using the selected header field requires
// the client to process records larger than 16384 bytes, return "invalid"."
const maxMIRecordSize = 16384

type Verifier struct {
	Version               version.Version
	VerifiedSignedSubsets []*VerifiedSignedSubset
}

type VerifiedSignedSubset struct {
	*SignedSubset
	Authority *certurl.AugmentedCertificate
}

// NewVerifier checks the validity of the signatures in sigs at verificationTime,
// and returns a Verifier that can be used to verify responses in the bundle.
//
// Note: this does not check the validity of the certificates in sigs.
func NewVerifier(sigs *bundle.Signatures, verificationTime time.Time, ver version.Version) (*Verifier, error) {
	var verifiedSubsets []*VerifiedSignedSubset
	for _, vs := range sigs.VouchedSubsets {
		verified, err := verifyVouchedSubset(vs, sigs.Authorities, verificationTime, ver)
		if err != nil {
			return nil, err
		}
		verifiedSubsets = append(verifiedSubsets, verified)
	}
	return &Verifier{Version: ver, VerifiedSignedSubsets: verifiedSubsets}, nil
}

type VerifyExchangeResult struct {
	// VerifiedPayload is the verified, MI-decoded payload.
	VerifiedPayload []byte
	// Authority is the certificate used to verify the signature.
	Authority *certurl.AugmentedCertificate
}

// VerifyExchange verifies a bundled response. If successfully verified,
// returns a VerifyExchangeResult. If verification failed, returns an error.
// If e is not covered by a signature, returns (nil, nil).
func (v *Verifier) VerifyExchange(e *bundle.Exchange) (*VerifyExchangeResult, error) {
	rhs, auth := v.findResponseHashes(e.Request.URL.String())
	if rhs == nil || auth == nil {
		return nil, nil
	}
	if len(rhs.VariantsValue) != 0 || len(rhs.Hashes) != 1 {
		return nil, errors.New("signature: signature with variants-value is not supported")
	}
	rh := rhs.Hashes[0]

	// TODO: Use the SHA256 of original header cbor bytes, instead of
	// calculating from parsed-and-reconstructed CBOR header.
	headerSha256, err := e.Response.HeaderSha256()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(headerSha256, rh.HeaderSha256) {
		return nil, errors.New("signature: header sha256 mismatch")
	}

	encoding := v.Version.MiceEncoding()
	if encoding.IntegrityIdentifier() != rh.PayloadIntegrityHeader {
		return nil, errors.New("signature: integrity identifier mismatch")
	}
	digest := e.Response.Header.Get(encoding.DigestHeaderName())
	if digest == "" {
		return nil, errors.New("signature: digest response header not present")
	}
	dec, err := encoding.NewDecoder(bytes.NewReader(e.Response.Body), digest, maxMIRecordSize)
	if err != nil {
		return nil, err
	}
	decoded, err := ioutil.ReadAll(dec)
	if err != nil {
		return nil, err
	}
	return &VerifyExchangeResult{VerifiedPayload: decoded, Authority: auth}, nil
}

func (v *Verifier) findResponseHashes(requestUrl string) (*ResponseHashes, *certurl.AugmentedCertificate) {
	for _, ss := range v.VerifiedSignedSubsets {
		if rh, ok := ss.SubsetHashes[requestUrl]; ok {
			return rh, ss.Authority
		}
	}
	return nil, nil
}

func verifyVouchedSubset(vs *bundle.VouchedSubset, authorities []*certurl.AugmentedCertificate, verificationTime time.Time, ver version.Version) (*VerifiedSignedSubset, error) {
	// TODO: Add refernce once we have spec text.
	if vs.Authority >= uint64(len(authorities)) {
		return nil, errors.New("signature: authority index out of range")
	}
	cert := authorities[vs.Authority]
	verifier, err := signingalgorithm.VerifierForPublicKey(cert.Cert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("signature: unsupported certificate public key: %v", err)
	}
	msg := generateSignedMessage(vs.Signed, ver)
	ok, err := verifier.Verify(msg, vs.Sig)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("signature: signature verification failed")
	}

	ss, err := decodeSignedSubset(vs.Signed)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(ss.AuthSha256, cert.CertSha256()) {
		return nil, errors.New("signature: auth-sha256 doesn't match")
	}
	if ss.Expires.Sub(ss.Date) > 7*24*time.Hour {
		return nil, fmt.Errorf("signature: expires (%v) is more than 7 days (604800 seconds) after date (%v)", ss.Expires, ss.Date)
	}
	if verificationTime.Before(ss.Date) {
		return nil, fmt.Errorf("signature: signature is not yet valid. date=%v", ss.Date)
	}
	if verificationTime.After(ss.Expires) {
		return nil, fmt.Errorf("signature: signature is expired. expires=%v", ss.Expires)
	}

	return &VerifiedSignedSubset{SignedSubset: ss, Authority: cert}, nil
}

// decodeSignedSubset deserializes a "signed-subset" CBOR item.
// https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#signatures-section
func decodeSignedSubset(signed []byte) (*SignedSubset, error) {
	dec := cbor.NewDecoder(bytes.NewBuffer(signed))
	n, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, err
	}
	result := &SignedSubset{}
	for i := uint64(0); i < n; i++ {
		label, err := dec.DecodeTextString()
		if err != nil {
			return nil, err
		}
		switch label {
		case "validity-url":
			validityUrl, err := dec.DecodeTextString()
			if err != nil {
				return nil, err
			}
			result.ValidityUrl, err = url.Parse(validityUrl)
			if err != nil {
				return nil, err
			}
		case "auth-sha256":
			result.AuthSha256, err = dec.DecodeByteString()
			if err != nil {
				return nil, err
			}
		case "date":
			date, err := dec.DecodeUint()
			if err != nil {
				return nil, err
			}
			result.Date = time.Unix(int64(date), 0)
		case "expires":
			expires, err := dec.DecodeUint()
			if err != nil {
				return nil, err
			}
			result.Expires = time.Unix(int64(expires), 0)
		case "subset-hashes":
			result.SubsetHashes, err = decodeSubsetHashes(dec)
			if err != nil {
				return nil, err
			}
		default:
			// The spec allows extra fields in the map, but this implementation fails on it.
			// TODO: Skip single CBOR item and continue.
			return nil, fmt.Errorf("signature: unknown key in signed-subset map: %q", label)
		}
	}
	if result.ValidityUrl == nil || result.AuthSha256 == nil ||
		result.Date.IsZero() || result.Expires.IsZero() ||
		result.SubsetHashes == nil {
		return nil, errors.New("signature: incomplete signed-subset value")
	}
	return result, nil
}

// decodeSignedSubset deserializes a "subset-hashes" CBOR item.
// https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#signatures-section
func decodeSubsetHashes(dec *cbor.Decoder) (map[string]*ResponseHashes, error) {
	n, err := dec.DecodeMapHeader()
	if err != nil {
		return nil, err
	}

	shs := make(map[string]*ResponseHashes)
	for i := uint64(0); i < n; i++ {
		urlString, err := dec.DecodeTextString()
		if err != nil {
			return nil, err
		}
		m, err := dec.DecodeArrayHeader()
		if err != nil {
			return nil, err
		}
		if m < 3 || m%2 != 1 {
			return nil, fmt.Errorf("signature: unexpected length of subset-hashes value array: %d", m)
		}
		rhs := &ResponseHashes{}
		rhs.VariantsValue, err = dec.DecodeByteString()
		if err != nil {
			return nil, err
		}
		for j := uint64(1); j < m; j += 2 {
			headerSha256, err := dec.DecodeByteString()
			if err != nil {
				return nil, err
			}
			payloadIntegrityHeader, err := dec.DecodeTextString()
			if err != nil {
				return nil, err
			}
			rhs.Hashes = append(rhs.Hashes, &ResourceIntegrity{headerSha256, payloadIntegrityHeader})
		}
		shs[urlString] = rhs
	}
	return shs, nil
}
