# go/signedexchange
This directory contains a reference implementation of [Signed HTTP Exchanges](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html) format generator.

## Overview
We currently provide two command-line tools: `gen-signedexchange` and `gen-certurl`.

`gen-signedexchange` generates a signed exchange file. The `gen-signedexchange` command constructs an HTTP request and response pair from given command line flags, attach the cryptographic signature of the pair, and serializes the result to an output file.

`gen-certurl` converts an X.509 certificate chain, an OCSP response, and an SCT (if one isn't already included in the certificate or OCSP response) to `application/cert-chain+cbor` format, which is defined in the [Section 3.3 of the Signed HTTP Exchanges spec](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#rfc.section.3.3).

You are also welcome to use the code as a Go lib (e.g. `import "github.com/WICG/webpackage/go/signedexchange"`), but please be aware that the API is not yet stable and is subject to change any time.

## Getting Started

### Prerequisite
The Go environment needs to be set up in prior to using the tool. We are testing the tool on the latest version of Go. Please refer to the [Go Getting Started documentation](https://golang.org/doc/install) for the details.

### Installation
We recommend using `go get` to install the command-line tool.

```
go get -u github.com/WICG/webpackage/go/signedexchange/cmd/...
```

### Creating our first signed exchange
In this section, we guide you to create a signed exchange file that is signed using a self-signed certificate pair.

Here, we assume that you have an access to an HTTPS server capable of serving static content. [1] Please substitute `https://yourcdn.example.net/` URLs to your web server URL, and `example.org` to the domain for which you want to sign the exchange.

1. Prepare a file to be enclosed in the signed exchange. This serves as the payload of the HTTP response in the signed exchange.
    ```
    echo "<h1>hi</h1>" > payload.html
    ```

1. Prepare a certificate and private key pair to use for signing the exchange. As of July 2018, we need to use self-signed certificate for testing, since there are no CA that issues certificate with ["CanSignHttpExchanges" extension](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cross-origin-cert-req). To generate a signed-exchange-compatible self-signed key pair with OpenSSL, invoke:
    ```
    # Generate prime256v1 ecdsa private key.
    openssl ecparam -out priv.key -name prime256v1 -genkey
    # Create a certificate signing request for the private key.
    openssl req -new -sha256 -key priv.key -out cert.csr \
      -subj '/CN=example.org/O=Test/C=US'
    # Self-sign the certificate with "CanSignHttpExchanges" extension.
    openssl x509 -req -days 360 -in cert.csr -signkey priv.key -out cert.pem \
      -extfile <(echo -e "1.3.6.1.4.1.11129.2.1.22 = ASN1:NULL\nsubjectAltName=DNS:example.org")
    ```

1. Convert the PEM certificate to `application/cert-chain+cbor` format using `gen-certurl` tool. This command will show warnings about OCSP and SCT, but you can ignore them.
    ```
    # Fill in dummy data for OCSP, since the certificate is self-signed.
    gen-certurl -pem cert.pem -ocsp <(echo ocsp) > cert.cbor
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

1. Host the signed exchange file `example.org.hello.sxg` on the HTTPS server. Configure the resource to be served with `Content-Type: application/signed-exchange;v=b2` HTTP header.
    - Note: If you are using [Firebase Hosting](https://firebase.google.com/docs/hosting/) as your HTTPS server, see an example config [here](https://github.com/WICG/webpackage/blob/master/examples/firebase.json).

1. Navigate to the signed exchange URL using a web browser supporting signed exchanges.
    - As of July 2018, you can use Chrome M69 [Dev](https://www.google.com/chrome/?extra=devchannel)/[Canary](https://www.google.com/chrome/browser/canary.html) versions with a command-line flag to enable signed exchange support.
      ```
      # Launch chrome dev set to ignore certificate errors of the self-signed certificate,
      # with an experimental feature of signed exchange support enabled.
      google-chrome-unstable \
        --user-data-dir=/tmp/udd \
        --ignore-certificate-errors-spki-list=`openssl x509 -noout -pubkey -in cert.pem | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | base64` \
        --enable-features=SignedHTTPExchange \
        https://yourcdn.example.net/example.org.hello.sxg
      ```

[1]: You can deploy your own HTTPS server or use a cloud hosting service. Note that the server must support configuring "Content-Type" HTTP headers, like [Firebase Hosting](https://firebase.google.com/docs/hosting/).

### Creating a signed exchange using a trusted certificate

In this section, you will create a signed exchange using a certificate issued by a publicly trusted CA.

As of July 2018, there are no CA that issues certificate with ["CanSignHttpExchanges" extension](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cross-origin-cert-req). So, created signed exchange can be used only for testing.

1. Get a certificate from a CA. You have to use prime256v1 ecdsa keys, as you did in the previous section. Please follow the CA's instructions.

   Assume you got a server certificate `server.pem` and an intermediate certificate `intermediates.pem`. The tools need all certificates in a single file, so concatenate them.
    ```
    cat server.pem intermediates.pem > cert-chain.pem
    ```

1. Convert the PEM certificate to `application/cert-chain+cbor` format using `gen-certurl` tool.
    ```
    gen-certurl -pem cert-chain.pem > cert.cbor
    ```
    If you got the warning message `Warning: Neither cert nor OCSP have embedded SCT list. Use -sctDir flag to add SCT from files.`, you need to prepare SCT files.

    Otherwise, you can skip the next step.

1. To get SCTs, submit your certificate chain to [Certificate Transparency](http://www.certificate-transparency.org/) log servers.

   1. Install a tool to submit certificates to log servers.
      ```
      go get github.com/grahamedgecombe/ct-submit
      ```
   2. Submit your cert chain to logs as appropriate, and write out SCTs:
      ```
      mkdir scts
      ct-submit ct.googleapis.com/logs/argon2018 < cert-chain.pem > scts/argon2018.sct
      ct-submit ct.cloudflare.com/logs/nimbus2018 < cert-chain.pem > scts/nimbus2018.sct
      ```
   3. Create a cert-chain with the obtained SCTs.
      ```
      gen-certurl -pem cert-chain.pem -sctDir scts > cert.cbor
      ```

1. Host `cert.cbor` on a HTTPS server. Please see the previous section for details.

1. Generate the signed exchange using `gen-signedexchange` tool. `priv.key` is the private key used to create your certificate.
    ```
    gen-signedexchange \
      -uri https://example.org/hello.html \
      -content ./payload.html \
      -certificate cert-chain.pem \
      -privateKey priv.key \
      -certUrl https://yourcdn.example.net/cert.cbor \
      -validityUrl https://example.org/resource.validity.msg \
      -o example.org.hello.sxg
    ```

1. Host `example.org.hello.sxg` on a HTTPS server. Please see the previous section for details.

1. Navigate to the signed exchange URL using a web browser supporting signed exchanges.
    - As of July 2018, you can use Chrome M69 [Dev](https://www.google.com/chrome/?extra=devchannel)/[Canary](https://www.google.com/chrome/browser/canary.html) versions with the following two flags enabled:
      - chrome://flags/#enable-signed-http-exchange
      - chrome://flags/#allow-sxg-certs-without-extension

### Dump a signed exchange file

You can dump the content of your sxg file by `dump-signedexchange`. If you want to see the content of the signed exchange file `example.org.hello.sxg` you created above, run this command.

```
dump-signedexchange -i example.org.hello.sxg
```

If `-verify` command-line flag is specified, `dump-signedexchange` checks if the signed exchange is valid.

By default, `dump-signedexchange` fetches certificate chain from network (from the URL you specified as `-certUrl` parameter of `gen-signedexchange`). But if `-cert filename` flag is given, `dump-signedexchange` reads certificates from `filename`.

For example, If you want to verify `example.org.hello.sxg` using certificates in `cert.cbor`, run this command.
```
dump-signedexchange -i example.org.hello.sxg -verify -cert cert.cbor
```
