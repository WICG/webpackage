# go/bundle
This directory contains a reference implementation of [Signed HTTP Exchanges](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html) spec.

## Overview
We currently provide two command-line tools: `gen-signedexchange` and `gen-certurl`.

`gen-signedexchange` generates a signed exchange file. The `gen-signedexchange` command constructs HTTP request and response pair from given command line flags, attach a cryptographic signature of the pair, and serializes the result to output file.

`gen-certurl` converts a X.509 certificate chain to `application/cert-chain+cbor` format, which is defined in the [Section 3.3 of the Signed HTTP Exchanges spec](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#rfc.section.3.3).

You are also welcome to use the code as golang lib (e.g. `import "github.com/WICG/webpackage/go/signedexchange"`), but please be aware that the API is not yet stable and is subject to change any time.

## Getting Started

### Prerequisite
golang environment needs to be set up in prior to using the tool. We are testing the tool on latest golang. Please refer to [Go Getting Started documentation](https://golang.org/doc/install) for the details.

### Installation
We recommend using `go get` to install the command-line tool.

```
go get -u github.com/WICG/webpackage/go/signedexchange/cmd/...
```

### Creating our first signed exchange
In this section, we guide you to create a signed exchange file, signed using self-signed certificate pair.

Here, we assume that you have an access to a HTTPS server capable of serving static content. [1] Please substitute `https://yourcdn.example.net/` URLs to your web server URL.

1. Prepare a file to be enclosed in the signed exchange. This serves as the content of the HTTP response in the signed exchange.
    ```
    echo "<h1>hi</h1>" > payload.html
    ```

1. Prepare a certificate and private key pair to use for signing the exchange. As of July 2018, we need to use self-signed certificate for testing, since there are no CA that issues certificate with ["CanSignHttpExchanges" extension](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cross-origin-cert-req). To generate a signed exchange compatible self-signed key pair with OpenSSL, invoke:
    ```
    # Generate prime256v1 ecdsa private key.
    openssl ecparam -out priv.key -name prime256v1 -genkey
    # Create a certificate signing request for the private key.
    openssl req -new -sha256 -key priv.key -out cert.csr \
      -subj '/CN=example.org/O=Test/C=US'
    # Self-sign the certificate with "CanSignHttpExchanges" extension.
    openssl x509 -req -days 360 -in cert.csr -signkey priv.key -out cert.pem \
      -extfile <(echo "1.3.6.1.4.1.11129.2.1.22 = ASN1:NULL\nsubjectAltName=DNS:example.org")
    ```

1. Convert the PEM certificate to `application/cert-chain+cbor` format using `gen-certurl` tool.
    ```
    # Fill in dummy data for OCSP/SCT, since the certificate is self-signed.
    gen-certurl -pem cert.pem -ocsp <(echo ocsp) -sct <(echo sct) > cert.cbor
    ```

1. Host the `application/cert-chain+cbor` created in Step 3 on the HTTPS server. Configure the resource to be served with `Content-Type: application/cert-chain+cbor` HTTP header. The steps below assume the `cert.cbor` is hosted at `https://yourcdn.example.net/cert.cbor`, so substitute the URL to the actual URL in below steps.
    - Note: If you are using [Firebase Hosting](https://firebase.google.com/docs/hosting/) as your HTTPS server, see an example config [here](https://github.com/WICG/webpackage/blob/master/examples/firebase.json).

1. Generate the signed exchange using `gen-signedexchange` tool.
    ```
    gen-signedexchange \
      -uri https://example.org/hello.html \
      -content ./payload.html \
      -certificate cert.pem \
      -privateKey priv.key \
      -certUrl https://yourcdn.example.net/cert.cbor \
      -validityUrl https://example.org/resource.validity.msg \
      -o example.org.hello.sxg
    ```

1. Host the signed exchange file `example.org.hello.sxg` on the HTTPS server. Configure the resource to be served with `Content-Type: application/signed-exchange;v=b1` HTTP header.
    - Note: If you are using [Firebase Hosting](https://firebase.google.com/docs/hosting/) as your HTTPS server, see an example config [here](https://github.com/WICG/webpackage/blob/master/examples/firebase.json).

1. Navigate to the signed exchange URL using a web browser supporting signed exchange.
    - As of July 2018, you can use Chrome [Dev](https://www.google.com/chrome/?extra=devchannel)/[Canary](https://www.google.com/chrome/browser/canary.html) versions with a command-line flag to enable signed exchange support.
      ```
      # Launch chrome dev set to ignore certificate errors of the self-signed certificate,
      # with an experimental feature of signed exchange support enabled.
      google-chrome-unstable \
        --user-data-dir=/tmp/udd \
        --ignore-certificate-errors-spki-list=`openssl x509 -noout -pubkey -in cert.pem| openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | base64` \
        --enable-features=SignedHTTPExchange \
        https://yourcdn.example.net/example.org.hello.sxg
      ```

[1]: You can deploy your own HTTPS server or use a cloud hosting service. Note that the server must support configuring "Content-Type" HTTP headers, like [Firebase Hosting](https://firebase.google.com/docs/hosting/).
