package signature

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/WICG/webpackage/go/bundle"
	"github.com/WICG/webpackage/go/bundle/version"
	"github.com/WICG/webpackage/go/internal/signingalgorithm"
	"github.com/WICG/webpackage/go/signedexchange/cbor"
	"github.com/WICG/webpackage/go/signedexchange/certurl"
)

type Signer struct {
	Version version.Version
	Certs   certurl.CertChain
	PrivKey crypto.PrivateKey
	Rand    io.Reader
	SignedSubset
}

// SignedSubset represents the "sisnged-subset" structure.
// https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#signatures-section
type SignedSubset struct {
	ValidityUrl  *url.URL
	AuthSha256   []byte
	Date         time.Time
	Expires      time.Time
	SubsetHashes map[string]*ResponseHashes
}

type ResponseHashes struct {
	VariantsValue string
	Hashes        []*ResourceIntegrity
}

type ResourceIntegrity struct {
	HeaderSha256           []byte
	PayloadIntegrityHeader string
}

// Encode encodes s as a CBOR item.
func (s *SignedSubset) Encode() ([]byte, error) {
	mes := []*cbor.MapEntryEncoder{
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("validity-url")
			valueE.EncodeTextString(s.ValidityUrl.String())
		}),
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("auth-sha256")
			valueE.EncodeByteString(s.AuthSha256)
		}),
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("date")
			valueE.EncodeInt(s.Date.Unix())
		}),
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("expires")
			valueE.EncodeInt(s.Expires.Unix())
		}),
		cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
			keyE.EncodeTextString("subset-hashes")
			subsetHashes := []*cbor.MapEntryEncoder{}
			for url, rh := range s.SubsetHashes {
				subsetHashes = append(subsetHashes,
					cbor.GenerateMapEntry(func(keyE *cbor.Encoder, valueE *cbor.Encoder) {
						keyE.EncodeTextString(url)
						valueE.EncodeArrayHeader(1 + len(rh.Hashes)*2)
						valueE.EncodeTextString(rh.VariantsValue)
						for _, ri := range rh.Hashes {
							valueE.EncodeByteString(ri.HeaderSha256)
							valueE.EncodeTextString(ri.PayloadIntegrityHeader)
						}
					}))
			}
			valueE.EncodeMap(subsetHashes)
		}),
	}
	var buf bytes.Buffer
	enc := cbor.NewEncoder(&buf)
	if err := enc.EncodeMap(mes); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func NewSigner(ver version.Version, certs certurl.CertChain, privKey crypto.PrivateKey, validityUrl *url.URL, date time.Time, duration time.Duration) (*Signer, error) {
	if ver == version.Unversioned {
		return nil, errors.New("signature: unversioned bundles cannot be signed")
	}

	if err := certs.Validate(); err != nil {
		return nil, err
	}
	authSha256 := certs[0].CertSha256()

	return &Signer{
		Version: ver,
		Certs:   certs,
		PrivKey: privKey,
		Rand:    rand.Reader,
		SignedSubset: SignedSubset{
			ValidityUrl:  validityUrl,
			AuthSha256:   authSha256,
			Date:         date,
			Expires:      date.Add(duration),
			SubsetHashes: make(map[string]*ResponseHashes),
		},
	}, nil
}

// CanSignForURL returns true iff this signer can sign for a resource with given URL.
func (s *Signer) CanSignForURL(u *url.URL) bool {
	return s.Certs[0].Cert.VerifyHostname(u.Hostname()) == nil
}

// AddExchange adds resource integrity information of e for signing.
func (s *Signer) AddExchange(e *bundle.Exchange, payloadIntegrityHeader string) error {
	headerSha256, err := e.Response.HeaderSha256()
	if err != nil {
		return err
	}
	ri := &ResourceIntegrity{HeaderSha256: headerSha256, PayloadIntegrityHeader: payloadIntegrityHeader}

	if _, ok := s.SubsetHashes[e.Request.URL.String()]; ok {
		// TODO: Fix this when we implement variants.
		return errors.New("signature: multiple exchanges for single URL is not supported")
	}

	s.SubsetHashes[e.Request.URL.String()] = &ResponseHashes{
		VariantsValue: "",
		Hashes:        []*ResourceIntegrity{ri},
	}
	return nil
}

// UpdateSignatures updates bundle.Signatures by adding the cert chain and
// the signature of exchanges added with AddExchange.
func (s *Signer) UpdateSignatures(signatures *bundle.Signatures) (*bundle.Signatures, error) {
	if signatures == nil {
		signatures = &bundle.Signatures{}
	}

	authorityIndex := len(signatures.Authorities)
	signatures.Authorities = append(signatures.Authorities, s.Certs...)
	signedSubsetBytes, err := s.SignedSubset.Encode()
	if err != nil {
		return nil, err
	}
	sig, err := s.sign(signedSubsetBytes)
	if err != nil {
		return nil, err
	}
	signatures.VouchedSubsets = append(signatures.VouchedSubsets,
		&bundle.VouchedSubset{
			Authority: uint64(authorityIndex),
			Sig:       sig,
			Signed:    signedSubsetBytes,
		})
	return signatures, err
}

func (s *Signer) sign(signed []byte) ([]byte, error) {
	alg, err := signingalgorithm.SigningAlgorithmForPrivateKey(s.PrivKey, s.Rand)
	if err != nil {
		return nil, err
	}

	return alg.Sign(generateSignedMessage(signed, s.Version))
}

// https://github.com/WICG/webpackage/issues/472#issuecomment-520080192
// TODO: Update the above reference once we have spec text for this.
func generateSignedMessage(signed []byte, ver version.Version) []byte {
	// The message is the concatenation of:
	var buf bytes.Buffer
	// 1. A string that consists of octet 32 (0x20) repeated 64 times.
	for i := 0; i < 64; i++ {
		buf.WriteByte(0x20)
	}
	// 2. A context string: "Web Package <version>".
	buf.WriteString(ver.SignatureContextString())
	// 3. A single 0 byte which serves as a separator.
	buf.WriteByte(0)
	// 4. The signed bstr.
	buf.Write(signed)
	return buf.Bytes()
}
