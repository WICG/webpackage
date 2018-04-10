package certurl_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	. "github.com/WICG/webpackage/go/signedexchange/certurl"
	"github.com/WICG/webpackage/go/signedexchange/internal/testhelper"
)

// readableString converts an arbitrary value v to a string.
//
// readableString does a basically same thing as fmt.Sprintf("%q", v), but
// the difference is that map keys are ordered in alphabetical order so that
// the results are deterministic.
func readableString(v interface{}) string {
	switch v := v.(type) {
	case []interface{}:
		vals := []string{}
		for _, val := range v {
			vals = append(vals, readableString(val))
		}
		return "[" + strings.Join(vals, " ") + "]"
	case map[interface{}]interface{}:
		keys := []string{}
		// Assume that keys are strings.
		for k := range v {
			keys = append(keys, k.(string))
		}
		sort.Strings(keys)
		vals := []string{}
		for _, k := range keys {
			val := v[k]
			vals = append(vals, fmt.Sprintf("%q:", k)+readableString(val))
		}
		return "map[" + strings.Join(vals, " ") + "]"
	case string, []byte:
		return fmt.Sprintf("%q", v)
	default:
		panic(fmt.Sprintf("not supported type: %T", v))
	}
}

func TestParsePEM(t *testing.T) {
	in := []byte(`-----BEGIN CERTIFICATE-----
MIIF8jCCBNqgAwIBAgIQDmTF+8I2reFLFyrrQceMsDANBgkqhkiG9w0BAQsFADBw
MQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3
d3cuZGlnaWNlcnQuY29tMS8wLQYDVQQDEyZEaWdpQ2VydCBTSEEyIEhpZ2ggQXNz
dXJhbmNlIFNlcnZlciBDQTAeFw0xNTExMDMwMDAwMDBaFw0xODExMjgxMjAwMDBa
MIGlMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEUMBIGA1UEBxML
TG9zIEFuZ2VsZXMxPDA6BgNVBAoTM0ludGVybmV0IENvcnBvcmF0aW9uIGZvciBB
c3NpZ25lZCBOYW1lcyBhbmQgTnVtYmVyczETMBEGA1UECxMKVGVjaG5vbG9neTEY
MBYGA1UEAxMPd3d3LmV4YW1wbGUub3JnMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAs0CWL2FjPiXBl61lRfvvE0KzLJmG9LWAC3bcBjgsH6NiVVo2dt6u
Xfzi5bTm7F3K7srfUBYkLO78mraM9qizrHoIeyofrV/n+pZZJauQsPjCPxMEJnRo
D8Z4KpWKX0LyDu1SputoI4nlQ/htEhtiQnuoBfNZxF7WxcxGwEsZuS1KcXIkHl5V
RJOreKFHTaXcB1qcZ/QRaBIv0yhxvK1yBTwWddT4cli6GfHcCe3xGMaSL328Fgs3
jYrvG29PueB6VJi/tbbPu6qTfwp/H1brqdjh29U52Bhb0fJkM9DWxCP/Cattcc7a
z8EXnCO+LK8vkhw/kAiJWPKx4RBvgy73nwIDAQABo4ICUDCCAkwwHwYDVR0jBBgw
FoAUUWj/kK8CB3U8zNllZGKiErhZcjswHQYDVR0OBBYEFKZPYB4fLdHn8SOgKpUW
5Oia6m5IMIGBBgNVHREEejB4gg93d3cuZXhhbXBsZS5vcmeCC2V4YW1wbGUuY29t
ggtleGFtcGxlLmVkdYILZXhhbXBsZS5uZXSCC2V4YW1wbGUub3Jngg93d3cuZXhh
bXBsZS5jb22CD3d3dy5leGFtcGxlLmVkdYIPd3d3LmV4YW1wbGUubmV0MA4GA1Ud
DwEB/wQEAwIFoDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwdQYDVR0f
BG4wbDA0oDKgMIYuaHR0cDovL2NybDMuZGlnaWNlcnQuY29tL3NoYTItaGEtc2Vy
dmVyLWc0LmNybDA0oDKgMIYuaHR0cDovL2NybDQuZGlnaWNlcnQuY29tL3NoYTIt
aGEtc2VydmVyLWc0LmNybDBMBgNVHSAERTBDMDcGCWCGSAGG/WwBATAqMCgGCCsG
AQUFBwIBFhxodHRwczovL3d3dy5kaWdpY2VydC5jb20vQ1BTMAgGBmeBDAECAjCB
gwYIKwYBBQUHAQEEdzB1MCQGCCsGAQUFBzABhhhodHRwOi8vb2NzcC5kaWdpY2Vy
dC5jb20wTQYIKwYBBQUHMAKGQWh0dHA6Ly9jYWNlcnRzLmRpZ2ljZXJ0LmNvbS9E
aWdpQ2VydFNIQTJIaWdoQXNzdXJhbmNlU2VydmVyQ0EuY3J0MAwGA1UdEwEB/wQC
MAAwDQYJKoZIhvcNAQELBQADggEBAISomhGn2L0LJn5SJHuyVZ3qMIlRCIdvqe0Q
6ls+C8ctRwRO3UU3x8q8OH+2ahxlQmpzdC5al4XQzJLiLjiJ2Q1p+hub8MFiMmVP
PZjb2tZm2ipWVuMRM+zgpRVM6nVJ9F3vFfUSHOb4/JsEIUvPY+d8/Krc+kPQwLvy
ieqRbcuFjmqfyPmUv1U9QoI4TQikpw7TZU0zYZANP4C/gj4Ry48/znmUaRvy2kvI
l7gRQ21qJTK5suoiYoYNo3J9T+pXPGU7Lydz/HwW+w0DpArtAaukI8aNX4ohFUKS
wDSiIIWIWJiJGbEeIO0TIFwEVWTOnbNl/faPXpk5IRXicapqiII=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIEsTCCA5mgAwIBAgIQBOHnpNxc8vNtwCtCuF0VnzANBgkqhkiG9w0BAQsFADBs
MQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3
d3cuZGlnaWNlcnQuY29tMSswKQYDVQQDEyJEaWdpQ2VydCBIaWdoIEFzc3VyYW5j
ZSBFViBSb290IENBMB4XDTEzMTAyMjEyMDAwMFoXDTI4MTAyMjEyMDAwMFowcDEL
MAkGA1UEBhMCVVMxFTATBgNVBAoTDERpZ2lDZXJ0IEluYzEZMBcGA1UECxMQd3d3
LmRpZ2ljZXJ0LmNvbTEvMC0GA1UEAxMmRGlnaUNlcnQgU0hBMiBIaWdoIEFzc3Vy
YW5jZSBTZXJ2ZXIgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC2
4C/CJAbIbQRf1+8KZAayfSImZRauQkCbztyfn3YHPsMwVYcZuU+UDlqUH1VWtMIC
Kq/QmO4LQNfE0DtyyBSe75CxEamu0si4QzrZCwvV1ZX1QK/IHe1NnF9Xt4ZQaJn1
itrSxwUfqJfJ3KSxgoQtxq2lnMcZgqaFD15EWCo3j/018QsIJzJa9buLnqS9UdAn
4t07QjOjBSjEuyjMmqwrIw14xnvmXnG3Sj4I+4G3FhahnSMSTeXXkgisdaScus0X
sh5ENWV/UyU50RwKmmMbGZJ0aAo3wsJSSMs5WqK24V3B3aAguCGikyZvFEohQcft
bZvySC/zA/WiaJJTL17jAgMBAAGjggFJMIIBRTASBgNVHRMBAf8ECDAGAQH/AgEA
MA4GA1UdDwEB/wQEAwIBhjAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIw
NAYIKwYBBQUHAQEEKDAmMCQGCCsGAQUFBzABhhhodHRwOi8vb2NzcC5kaWdpY2Vy
dC5jb20wSwYDVR0fBEQwQjBAoD6gPIY6aHR0cDovL2NybDQuZGlnaWNlcnQuY29t
L0RpZ2lDZXJ0SGlnaEFzc3VyYW5jZUVWUm9vdENBLmNybDA9BgNVHSAENjA0MDIG
BFUdIAAwKjAoBggrBgEFBQcCARYcaHR0cHM6Ly93d3cuZGlnaWNlcnQuY29tL0NQ
UzAdBgNVHQ4EFgQUUWj/kK8CB3U8zNllZGKiErhZcjswHwYDVR0jBBgwFoAUsT7D
aQP4v0cB1JgmGggC72NkK8MwDQYJKoZIhvcNAQELBQADggEBABiKlYkD5m3fXPwd
aOpKj4PWUS+Na0QWnqxj9dJubISZi6qBcYRb7TROsLd5kinMLYBq8I4g4Xmk/gNH
E+r1hspZcX30BJZr01lYPf7TMSVcGDiEo+afgv2MW5gxTs14nhr9hctJqvIni5ly
/D6q1UEL2tU2ob8cbkdJf17ZSHwD2f2LSaCYJkJA69aSEaRkCldUxPUd1gJea6zu
xICaEnL6VpPX/78whQYwvwt/Tv9XBZ0k7YXDK/umdaisLRbvfXknsuvCnQsH6qqF
0wGjIChBWUMo0oHjqvbsezt3tkBigAVBRQHvFwY+3sAzm2fTYS5yh+Rp/BIAV0Ae
cPUeybQ=
-----END CERTIFICATE-----
`)

	want := "[\"📜⛓\" map[\"cert\":\"0\\x82\\x05\\xf20\\x82\\x04ڠ\\x03\\x02\\x01\\x02\\x02\\x10\\x0ed\\xc5\\xfb\\xc26\\xad\\xe1K\\x17*\\xebAǌ\\xb00\\r\\x06\\t*\\x86H\\x86\\xf7\\r\\x01\\x01\\v\\x05\\x000p1\\v0\\t\\x06\\x03U\\x04\\x06\\x13\\x02US1\\x150\\x13\\x06\\x03U\\x04\\n\\x13\\fDigiCert Inc1\\x190\\x17\\x06\\x03U\\x04\\v\\x13\\x10www.digicert.com1/0-\\x06\\x03U\\x04\\x03\\x13&DigiCert SHA2 High Assurance Server CA0\\x1e\\x17\\r151103000000Z\\x17\\r181128120000Z0\\x81\\xa51\\v0\\t\\x06\\x03U\\x04\\x06\\x13\\x02US1\\x130\\x11\\x06\\x03U\\x04\\b\\x13\\nCalifornia1\\x140\\x12\\x06\\x03U\\x04\\a\\x13\\vLos Angeles1<0:\\x06\\x03U\\x04\\n\\x133Internet Corporation for Assigned Names and Numbers1\\x130\\x11\\x06\\x03U\\x04\\v\\x13\\nTechnology1\\x180\\x16\\x06\\x03U\\x04\\x03\\x13\\x0fwww.example.org0\\x82\\x01\\\"0\\r\\x06\\t*\\x86H\\x86\\xf7\\r\\x01\\x01\\x01\\x05\\x00\\x03\\x82\\x01\\x0f\\x000\\x82\\x01\\n\\x02\\x82\\x01\\x01\\x00\\xb3@\\x96/ac>%\\xc1\\x97\\xadeE\\xfb\\xef\\x13B\\xb3,\\x99\\x86\\xf4\\xb5\\x80\\vv\\xdc\\x068,\\x1f\\xa3bUZ6vޮ]\\xfc\\xe2\\xe5\\xb4\\xe6\\xec]\\xca\\xee\\xca\\xdfP\\x16$,\\xee\\xfc\\x9a\\xb6\\x8c\\xf6\\xa8\\xb3\\xacz\\b{*\\x1f\\xad_\\xe7\\xfa\\x96Y%\\xab\\x90\\xb0\\xf8\\xc2?\\x13\\x04&th\\x0f\\xc6x*\\x95\\x8a_B\\xf2\\x0e\\xedR\\xa6\\xebh#\\x89\\xe5C\\xf8m\\x12\\x1bbB{\\xa8\\x05\\xf3Y\\xc4^\\xd6\\xc5\\xccF\\xc0K\\x19\\xb9-Jqr$\\x1e^UD\\x93\\xabx\\xa1GM\\xa5\\xdc\\aZ\\x9cg\\xf4\\x11h\\x12/\\xd3(q\\xbc\\xadr\\x05<\\x16u\\xd4\\xf8rX\\xba\\x19\\xf1\\xdc\\t\\xed\\xf1\\x18ƒ/}\\xbc\\x16\\v7\\x8d\\x8a\\xef\\x1boO\\xb9\\xe0zT\\x98\\xbf\\xb5\\xb6ϻ\\xaa\\x93\\u007f\\n\\u007f\\x1fV\\xeb\\xa9\\xd8\\xe1\\xdb\\xd59\\xd8\\x18[\\xd1\\xf2d3\\xd0\\xd6\\xc4#\\xff\\t\\xabmq\\xce\\xda\\xcf\\xc1\\x17\\x9c#\\xbe,\\xaf/\\x92\\x1c?\\x90\\b\\x89X\\xf2\\xb1\\xe1\\x10o\\x83.\\xf7\\x9f\\x02\\x03\\x01\\x00\\x01\\xa3\\x82\\x02P0\\x82\\x02L0\\x1f\\x06\\x03U\\x1d#\\x04\\x180\\x16\\x80\\x14Qh\\xff\\x90\\xaf\\x02\\au<\\xcc\\xd9edb\\xa2\\x12\\xb8Yr;0\\x1d\\x06\\x03U\\x1d\\x0e\\x04\\x16\\x04\\x14\\xa6O`\\x1e\\x1f-\\xd1\\xe7\\xf1#\\xa0*\\x95\\x16\\xe4\\xe8\\x9a\\xeanH0\\x81\\x81\\x06\\x03U\\x1d\\x11\\x04z0x\\x82\\x0fwww.example.org\\x82\\vexample.com\\x82\\vexample.edu\\x82\\vexample.net\\x82\\vexample.org\\x82\\x0fwww.example.com\\x82\\x0fwww.example.edu\\x82\\x0fwww.example.net0\\x0e\\x06\\x03U\\x1d\\x0f\\x01\\x01\\xff\\x04\\x04\\x03\\x02\\x05\\xa00\\x1d\\x06\\x03U\\x1d%\\x04\\x160\\x14\\x06\\b+\\x06\\x01\\x05\\x05\\a\\x03\\x01\\x06\\b+\\x06\\x01\\x05\\x05\\a\\x03\\x020u\\x06\\x03U\\x1d\\x1f\\x04n0l04\\xa02\\xa00\\x86.http://crl3.digicert.com/sha2-ha-server-g4.crl04\\xa02\\xa00\\x86.http://crl4.digicert.com/sha2-ha-server-g4.crl0L\\x06\\x03U\\x1d \\x04E0C07\\x06\\t`\\x86H\\x01\\x86\\xfdl\\x01\\x010*0(\\x06\\b+\\x06\\x01\\x05\\x05\\a\\x02\\x01\\x16\\x1chttps://www.digicert.com/CPS0\\b\\x06\\x06g\\x81\\f\\x01\\x02\\x020\\x81\\x83\\x06\\b+\\x06\\x01\\x05\\x05\\a\\x01\\x01\\x04w0u0$\\x06\\b+\\x06\\x01\\x05\\x05\\a0\\x01\\x86\\x18http://ocsp.digicert.com0M\\x06\\b+\\x06\\x01\\x05\\x05\\a0\\x02\\x86Ahttp://cacerts.digicert.com/DigiCertSHA2HighAssuranceServerCA.crt0\\f\\x06\\x03U\\x1d\\x13\\x01\\x01\\xff\\x04\\x020\\x000\\r\\x06\\t*\\x86H\\x86\\xf7\\r\\x01\\x01\\v\\x05\\x00\\x03\\x82\\x01\\x01\\x00\\x84\\xa8\\x9a\\x11\\xa7ؽ\\v&~R${\\xb2U\\x9d\\xea0\\x89Q\\b\\x87o\\xa9\\xed\\x10\\xea[>\\v\\xc7-G\\x04N\\xddE7\\xc7ʼ8\\u007f\\xb6j\\x1ceBjst.Z\\x97\\x85\\xd0̒\\xe2.8\\x89\\xd9\\ri\\xfa\\x1b\\x9b\\xf0\\xc1b2eO=\\x98\\xdb\\xda\\xd6f\\xda*VV\\xe3\\x113\\xec\\xe0\\xa5\\x15L\\xeauI\\xf4]\\xef\\x15\\xf5\\x12\\x1c\\xe6\\xf8\\xfc\\x9b\\x04!K\\xcfc\\xe7|\\xfc\\xaa\\xdc\\xfaC\\xd0\\xc0\\xbb\\xf2\\x89\\xea\\x91m˅\\x8ej\\x9f\\xc8\\xf9\\x94\\xbfU=B\\x828M\\b\\xa4\\xa7\\x0e\\xd3eM3a\\x90\\r?\\x80\\xbf\\x82>\\x11ˏ?\\xcey\\x94i\\x1b\\xf2\\xdaKȗ\\xb8\\x11Cmj%2\\xb9\\xb2\\xea\\\"b\\x86\\r\\xa3r}O\\xeaW<e;/'s\\xfc|\\x16\\xfb\\r\\x03\\xa4\\n\\xed\\x01\\xab\\xa4#ƍ_\\x8a!\\x15B\\x92\\xc04\\xa2 \\x85\\x88X\\x98\\x89\\x19\\xb1\\x1e \\xed\\x13 \\\\\\x04UdΝ\\xb3e\\xfd\\xf6\\x8f^\\x999!\\x15\\xe2q\\xaaj\\x88\\x82\"] map[\"cert\":\"0\\x82\\x04\\xb10\\x82\\x03\\x99\\xa0\\x03\\x02\\x01\\x02\\x02\\x10\\x04\\xe1\\xe7\\xa4\\xdc\\\\\\xf2\\xf3m\\xc0+B\\xb8]\\x15\\x9f0\\r\\x06\\t*\\x86H\\x86\\xf7\\r\\x01\\x01\\v\\x05\\x000l1\\v0\\t\\x06\\x03U\\x04\\x06\\x13\\x02US1\\x150\\x13\\x06\\x03U\\x04\\n\\x13\\fDigiCert Inc1\\x190\\x17\\x06\\x03U\\x04\\v\\x13\\x10www.digicert.com1+0)\\x06\\x03U\\x04\\x03\\x13\\\"DigiCert High Assurance EV Root CA0\\x1e\\x17\\r131022120000Z\\x17\\r281022120000Z0p1\\v0\\t\\x06\\x03U\\x04\\x06\\x13\\x02US1\\x150\\x13\\x06\\x03U\\x04\\n\\x13\\fDigiCert Inc1\\x190\\x17\\x06\\x03U\\x04\\v\\x13\\x10www.digicert.com1/0-\\x06\\x03U\\x04\\x03\\x13&DigiCert SHA2 High Assurance Server CA0\\x82\\x01\\\"0\\r\\x06\\t*\\x86H\\x86\\xf7\\r\\x01\\x01\\x01\\x05\\x00\\x03\\x82\\x01\\x0f\\x000\\x82\\x01\\n\\x02\\x82\\x01\\x01\\x00\\xb6\\xe0/\\xc2$\\x06\\xc8m\\x04_\\xd7\\xef\\nd\\x06\\xb2}\\\"&e\\x16\\xaeB@\\x9b\\xceܟ\\x9fv\\a>\\xc30U\\x87\\x19\\xb9O\\x94\\x0eZ\\x94\\x1fUV\\xb4\\xc2\\x02*\\xafИ\\xee\\v@\\xd7\\xc4\\xd0;r\\xc8\\x14\\x9e\\uf431\\x11\\xa9\\xae\\xd2ȸC:\\xd9\\v\\v\\xd5Օ\\xf5@\\xaf\\xc8\\x1d\\xedM\\x9c_W\\xb7\\x86Ph\\x99\\xf5\\x8a\\xda\\xd2\\xc7\\x05\\x1f\\xa8\\x97\\xc9ܤ\\xb1\\x82\\x84-ƭ\\xa5\\x9c\\xc7\\x19\\x82\\xa6\\x85\\x0f^DX*7\\x8f\\xfd5\\xf1\\v\\b'2Z\\xf5\\xbb\\x8b\\x9e\\xa4\\xbdQ\\xd0'\\xe2\\xdd;B3\\xa3\\x05(Ļ(̚\\xac+#\\rx\\xc6{\\xe6^q\\xb7J>\\b\\xfb\\x81\\xb7\\x16\\x16\\xa1\\x9d#\\x12M\\xe5ג\\b\\xacu\\xa4\\x9c\\xba\\xcd\\x17\\xb2\\x1eD5e\\u007fS%9\\xd1\\x1c\\n\\x9ac\\x1b\\x19\\x92th\\n7\\xc2\\xc2RH\\xcb9Z\\xa2\\xb6\\xe1]\\xc1ݠ \\xb8!\\xa2\\x93&o\\x14J!A\\xc7\\xedm\\x9b\\xf2H/\\xf3\\x03\\xf5\\xa2h\\x92S/^\\xe3\\x02\\x03\\x01\\x00\\x01\\xa3\\x82\\x01I0\\x82\\x01E0\\x12\\x06\\x03U\\x1d\\x13\\x01\\x01\\xff\\x04\\b0\\x06\\x01\\x01\\xff\\x02\\x01\\x000\\x0e\\x06\\x03U\\x1d\\x0f\\x01\\x01\\xff\\x04\\x04\\x03\\x02\\x01\\x860\\x1d\\x06\\x03U\\x1d%\\x04\\x160\\x14\\x06\\b+\\x06\\x01\\x05\\x05\\a\\x03\\x01\\x06\\b+\\x06\\x01\\x05\\x05\\a\\x03\\x0204\\x06\\b+\\x06\\x01\\x05\\x05\\a\\x01\\x01\\x04(0&0$\\x06\\b+\\x06\\x01\\x05\\x05\\a0\\x01\\x86\\x18http://ocsp.digicert.com0K\\x06\\x03U\\x1d\\x1f\\x04D0B0@\\xa0>\\xa0<\\x86:http://crl4.digicert.com/DigiCertHighAssuranceEVRootCA.crl0=\\x06\\x03U\\x1d \\x0460402\\x06\\x04U\\x1d \\x000*0(\\x06\\b+\\x06\\x01\\x05\\x05\\a\\x02\\x01\\x16\\x1chttps://www.digicert.com/CPS0\\x1d\\x06\\x03U\\x1d\\x0e\\x04\\x16\\x04\\x14Qh\\xff\\x90\\xaf\\x02\\au<\\xcc\\xd9edb\\xa2\\x12\\xb8Yr;0\\x1f\\x06\\x03U\\x1d#\\x04\\x180\\x16\\x80\\x14\\xb1>\\xc3i\\x03\\xf8\\xbfG\\x01Ԙ&\\x1a\\b\\x02\\xefcd+\\xc30\\r\\x06\\t*\\x86H\\x86\\xf7\\r\\x01\\x01\\v\\x05\\x00\\x03\\x82\\x01\\x01\\x00\\x18\\x8a\\x95\\x89\\x03\\xe6m\\xdf\\\\\\xfc\\x1dh\\xeaJ\\x8f\\x83\\xd6Q/\\x8dkD\\x16\\x9e\\xacc\\xf5\\xd2nl\\x84\\x99\\x8b\\xaa\\x81q\\x84[\\xed4N\\xb0\\xb7y\\x92)\\xcc-\\x80j\\xf0\\x8e \\xe1y\\xa4\\xfe\\x03G\\x13\\xea\\xf5\\x86\\xcaYq}\\xf4\\x04\\x96k\\xd3YX=\\xfe\\xd31%\\\\\\x188\\x84\\xa3柂\\xfd\\x8c[\\x981N\\xcdx\\x9e\\x1a\\xfd\\x85\\xcbI\\xaa\\xf2'\\x8b\\x99r\\xfc>\\xaa\\xd5A\\v\\xda\\xd56\\xa1\\xbf\\x1cnGI\\u007f^\\xd9H|\\x03\\xd9\\xfd\\x8bI\\xa0\\x98&B@\\xeb֒\\x11\\xa4d\\nWT\\xc4\\xf5\\x1d\\xd6\\x02^k\\xac\\xeeĀ\\x9a\\x12r\\xfaV\\x93\\xd7\\xff\\xbf0\\x85\\x060\\xbf\\v\\u007fN\\xffW\\x05\\x9d$\\xed\\x85\\xc3+\\xfb\\xa6u\\xa8\\xac-\\x16\\xef}y'\\xb2\\xeb\\u009d\\v\\aꪅ\\xd3\\x01\\xa3 (AYC(ҁ\\xe3\\xaa\\xf6\\xec{;w\\xb6@b\\x80\\x05AE\\x01\\xef\\x17\\x06>\\xde\\xc03\\x9bg\\xd3a.r\\x87\\xe4i\\xfc\\x12\\x00W@\\x1ep\\xf5\\x1eɴ\"]]"

	cert, err := CertificateMessageFromPEM(in, nil, nil)
	if err != nil {
		t.Errorf("failed to parse PEM: %v", err)
	}
	got, err := testhelper.CborBinaryToReadableString(cert)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("CertificateMessageFromPEM:\ngot: %q,\nwant: %q", got, want)
	}
}
