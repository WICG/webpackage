package webpack_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"testing"

	"github.com/WICG/webpackage/go/webpack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const msg = "This is a long message that's longer than a valid SHA512 hash. Really, it is."

func signatureTest(t *testing.T, key crypto.Signer) {
	require := require.New(t)
	assert := assert.New(t)

	sig, err := webpack.Sign(key, []byte(msg))
	require.NoError(err)
	assert.NoError(webpack.Verify(key.Public(), []byte(msg), sig))

	assert.Error(webpack.Verify(key.Public(), []byte("Wrong message"), sig))
	sig[0] = ^sig[0]
	assert.Error(webpack.Verify(key.Public(), []byte(msg), sig))
}

func TestSignRsa2048(t *testing.T) {
	// Hard-code the RSA key to save time.
	N, _ := new(big.Int).SetString("afe1df19db81e581ea80c71f3bd3d71d51f81ffeb576442a561eed0a1f200576bc122945a4262915d6ff2ec888aed64135ae0bd459a37e5112123d93bde83376333d828de91950091e1df03d89793361897d3b1c9172f071090a97e34d7052c596fb9051a230a6101a94bb1094d50c050676203fe12efeddcf38c9a047257711a5fe2a3800b58fdda71bc848d3d68d8137e3bd36f62da708b805baa5062661a961dd2c44c727c7148f3b2932a21820d604d0f773436e85beda017e7111edf38e145bfd29dce081b42a50383bf329e8f6749bbc030fb0d1b6d8d21e1efc8d720abff0ab2e50dfc44b13ffc3e0dc659c2b7e2f17a045c1487f2aa126e38318cb11", 16)
	require.Equal(t, 2048, N.BitLen())
	E := 0x10001
	D, _ := new(big.Int).SetString("672f9bc53ecbe18b2bba2b983e705527057d0dc05053b7402350777ed5ade2a6bb45e862cc1ffb40ade6fe5a761e24e3130c2e3281f872563bc4e9cd70bff6d924ccb4786f460377a5eca89261c1f28c09aea7ec65c4ca1d76d17934c8acda52c3f688bfebe8a0b497f3a41fe1417090ce2ea552f4d8ae7c1163de9ea2beef303929ea3b4826cbc157e20b1bf7f4be116d4dae6a37a55aaca95349d83ad29194f543db062ce3d17be54e0f2f57a8f6817f74c77af3fdb5dfb65bca30bfab44986efe380e178f34f0559de73d8884ec4d34b5b39f7397077e9772b5dcd63860c79669cc3d4c190459076de76acfd1e52957ada1a8a4c16ce25ffeaa62598b18e5", 16)
	P, _ := new(big.Int).SetString("defbceb58ea209484bd2fe41f5a878682437268738b627643f670e30c0ba60bb84e2f45d73bc78f379080b1311f8d4ab54427f415d4f2ad7b1c26f6c069d620770db6e5235f6aee3cd8008883f0ea703a7947d88d3d2858bc406dfc8e3de17da650c9dd32b78d8c26f2f57526c82f1a9153795893a8e54026b8729cdb560533f", 16)
	Q, _ := new(big.Int).SetString("c9ecb06b7e0a5d5c9c84bd48734fc77d3bb5832f74ec16732e44b35e5acd784b1787d5f2cab348c0584baedcd28fe63f40176db25f30b3557eaf9e385540eb850e88da8e99e35415d9a54b9c30406e6e4a3740fff9a131e782018169ead0be36fdf0b112bac6a7a8fa34a4d394a08cf05b82c65e230e07fd8afe9157b17e5daf", 16)
	key := &rsa.PrivateKey{PublicKey: rsa.PublicKey{N: N, E: E},
		D: D, Primes: []*big.Int{P, Q}}
	signatureTest(t, key)
}

func TestSignSecp256(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signatureTest(t, key)
}

func TestSignSecp384(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	signatureTest(t, key)
}
