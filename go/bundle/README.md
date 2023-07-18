# go/bundle

This directory contains a reference implementation of the
[Web Bundles](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html)
spec.

## Overview

We currently provide three command-line tools: `gen-bundle`, `sign-bundle` and
`dump-bundle`.

`gen-bundle` command is a bundle generator tool. `gen-bundle` consumes a set of
http exchanges (currently in the form of
[HAR format](https://w3c.github.io/web-performance/specs/HAR/Overview.html), URL
list file, or static files in a local directory), and emits a web bundle.

`sign-bundle` command attaches a signature to a bundle. There are two supported
ways to sign: using signatures section or integrity block. `sign-bundle` takes
an existing bundle file, a private key and possibly a certificate, and emits a
new bundle file with cryptographic signature added.

`dump-bundle` command is a bundle inspector tool. `dump-bundle` dumps the
enclosed http exchanges of a given web bundle file in a human readable form.

You are also welcome to use the code as golang lib (e.g.
`import "github.com/WICG/webpackage/go/bundle"`), but please be aware that the
API is not yet stable and is subject to change any time.

## Getting Started

### Prerequisite

golang environment needs to be set up in prior to using the tool. We are testing
the tool on latest golang. Please refer to
[Go Getting Started documentation](https://golang.org/doc/install) for the
details.

### Installation

We recommend using `go install` (Golang 1.18+) to install the command-line tool.

```
go install github.com/WICG/webpackage/go/bundle/cmd/...@latest
```

## Usage

### gen-bundle

`gen-bundle` generates a web bundle. There are three ways to provide a set of
exchanges to bundle; by a HAR file, by a URL list, and by a local directory.

These command-line flags are common to all the three options:

- `-version` specifies WebBundle format version. Possible values are: `b2`
  (default) and `b1`.
- `-primaryURL` specifies the bundle's main resource URL. This URL is also used
  as the fallback destination when browser cannot process the bundle. If bundle
  format is `b1`, this option is required.
- `-manifestURL` specifies the bundle's
  [manifest](https://www.w3.org/TR/appmanifest/) URL. This option can be
  specified only for bundle version `b1`.
- `-o` specifies name of the output bundle file. Default file name if
  unspecified is `out.wbn`.
- `-headerOverride` adds additional response header to all bundled responses.
  Existing values of the header are overwritten.

#### From a HAR file

One convenient way to generate HAR file is via Chrome Devtools:

1. Open DevTools, navigate to Network panel.
2. Make sure that "Disable cache" box is checked.
3. Reload the page.
4. Right click on any resource and select "Save as HAR with content".

![generating har with devtools](https://raw.githubusercontent.com/WICG/webpackage/main/go/bundle/har-devtools.png)

Once you have the har file, generate the web bundle via:

```
gen-bundle -har foo.har -o foo.wbn -primaryURL https://example.com/
```

#### From a URL list

`gen-bundle` also accepts `-URLList FILE` flag. `FILE` is a plain text file with
one URL on each line. `gen-bundle` fetches these URLs and put the responses into
the bundle. For example, you could create `urls.txt` with:

```
# A line starting with '#' is a comment.
https://example.com/
https://example.com/style.css
https://example.com/script.js
```

then run:

```
gen-bundle -URLList urls.txt \
           -primaryURL https://example.com/ \
           -o example_com.wbn
```

Note that `gen-bundle` does not automatically discover subresources; you have to
enumerate all the necessary subresources in the URL list file.

#### From a local directory

You can also create a bundle from a local directory. For example, if you have
the necessary files for the site `https://www.example.com/` in `static/`
directory, run:

```
gen-bundle -dir static -baseURL https://example.com/ -o foo.wbn -primaryURL https://example.com/
```

If `-baseURL` flag is not specified, resources will have relative URLs in the
generated bundle file.

### sign-bundle

`sign-bundle` is split into the following sub-commands:

- `signatures-section`
- `integrity-block`
- `dump-id`

#### Using `signatures-section` sub-command

`sign-bundle signatures-section` takes an existing bundle file, a certificate
and a private key, and emits a new bundle file with cryptographic signature for
the bundled resources added.

It updates a bundle attaching a cryptographic signature of its exchanges. To use
this tool, you need a pair of a private key and a certificate in the
`application/cert-chain+cbor` format. See
[signatures-section extension](../../extensions/signatures-section.md) and
[go/signedexchange](../signedexchange/README.md) for more information on how to
create a key and certificate pair.

Assuming you have a key and certificate pair for `example.org`, this command
will sign all exchanges in `unsigned.wbn` whose URL's hostname is `example.org`,
and writes a new bundle to `signed.wbn`.

```
sign-bundle signatures-section \
  -i unsigned.wbn \
  -certificate cert.cbor \
  -privateKey priv.key \
  -validityUrl https://example.org/resource.validity.msg \
  -o signed.wbn
```

#### Using `integrity-block` sub-command

`sign-bundle integrity-block` takes an existing bundle file and an ed25519
private key, and emits a new bundle file with cryptographic signature added to
the integrity block.

It updates a bundle prepending the web bundle with an integrity block containing
a stack of signatures over the hash of the web bundle. To use this tool, you
need an ed25519 private key in .pem format.

An unencrypted ed25519 private key can be generated with:

```
openssl genpkey -algorithm Ed25519 -out ed25519key.pem
```

For better security, one should prefer using passphrase-encrypted ed25519
private keys. To encrypt an unencrypted private key, run:

```
# encrypt the key (will ask for a passphrase, make sure to use a strong one)
openssl pkcs8 -in ed25519key.pem -topk8 -out encrypted_ed25519key.pem
# delete the unencrypted key
rm ed25519key.pem
```

```
sign-bundle integrity-block \
  -i unsigned.wbn \
  -privateKey encrypted_ed25519key.pem \
  -o signed.swbn
```

In case you chose to use an encrypted private key, it will be prompted during
the signing process. If one wants to use the tool programmatically (e.g. as
part of their DevOps pipeline), they can bypass the prompt by setting the
password into an environment variable named `WEB_BUNDLE_SIGNING_PASSPHRASE`.

See [integrityblock-explainer](../../explainers/integrity-signature.md) for more
information about what an integrity block is.

#### Using `dump-id` sub-command

`sign-bundle dump-id` is a helper tool to print out the
[Web Bundle ID](https://github.com/WICG/isolated-web-apps/blob/main/Scheme.md#signed-web-bundle-ids)
which is corresponding to the given ed25519 key. The ID is
base32-encoded lowercase representation of the ed25519 public key with a
predefined suffix.

Web Bundle ID can be used for example as the origin of an Isolated Web App.

```
sign-bundle dump-id -privateKey privkey.pem
```
or
```
sign-bundle dump-id -publicKey pubkey.pem
```
### dump-bundle

`dump-bundle` dumps the content of a web bundle in a human readable form. To
display content of a bundle file, invoke:

```
dump-bundle -i foo.wbn
```

`dump-bundle` doesn't support web bundles signed with integrity block.

## Using Bundles

Bundles generated with `gen-bundle` can be opened with web browsers supporting
web bundles.

Chrome (79+) experimentally supports Web Bundles with some limitations. See
[this document](https://chromium.googlesource.com/chromium/src/+/refs/heads/master/content/browser/web_package/using_web_bundles.md)
for more details.
