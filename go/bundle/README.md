# go/bundle
This directory contains a reference implementation of the [Web
Bundles](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html)
spec.

## Overview
We currently provide three command-line tools: `gen-bundle`, `sign-bundle` and `dump-bundle`.

`gen-bundle` command is a bundle generator tool. `gen-bundle` consumes a set of http exchanges (currently in the form of [HAR format](https://w3c.github.io/web-performance/specs/HAR/Overview.html), URL list file, or static files in a local directory), and emits a web bundle.

`sign-bundle` command attaches a signature to a bundle. `sign-bundle` takes an existing bundle file, a certificate and a private key, and emits a new bundle file with cryptographic signature for the bundled resources added.

`dump-bundle` command is a bundle inspector tool. `dump-bundle` dumps the enclosed http exchanges of a given web bundle file in a human readable form.

You are also welcome to use the code as golang lib (e.g. `import "github.com/WICG/webpackage/go/bundle"`), but please be aware that the API is not yet stable and is subject to change any time.

## Getting Started

### Prerequisite
golang environment needs to be set up in prior to using the tool. We are testing the tool on latest golang. Please refer to [Go Getting Started documentation](https://golang.org/doc/install) for the details.

### Installation
We recommend using `go install` to install the command-line tool.

```
go install github.com/WICG/webpackage/go/bundle/cmd/...@latest
```

## Usage

### gen-bundle
`gen-bundle` generates a web bundle. There are three ways to provide a set of exchanges to bundle; by a HAR file, by a URL list, and by a local directory.

These command-line flags are common to all the three options:

- `-primaryURL` specifies the bundle's main resource URL. This URL is also used as the fallback destination when browser cannot process the bundle. This option is required.
- `-manifestURL` specifies the bundle's [manifest](https://www.w3.org/TR/appmanifest/) URL. This option is optional and can be omitted.
- `-o` specifies name of the output bundle file. Default file name if unspecified is `out.wbn`.
- `-headerOverride` adds additional response header to all bundled responses. Existing values of the header are overwritten.

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

`gen-bundle` also accepts `-URLList FILE` flag. `FILE` is a plain text file with one URL on each line. `gen-bundle` fetches these URLs and put the responses into the bundle. For example, you could create `urls.txt` with:

```
# A line starting with '#' is a comment.
https://example.com/
https://example.com/manifest.webmanifest
https://example.com/style.css
https://example.com/script.js
```
then run:
```
gen-bundle -URLList urls.txt \
           -primaryURL https://example.com/ \
           -manifestURL https://example.com/manifest.webmanifest \
           -o example_com.wbn
```

Note that `gen-bundle` does not automatically discover subresources; you have to enumerate all the necessary subresources in the URL list file.

#### From a local directory

You can also create a bundle from a local directory. For example, if you have the necessary files for the site `https://www.example.com/` in `static/` directory, run:
```
gen-bundle -dir static -baseURL https://example.com/ -o foo.wbn -primaryURL https://example.com/
```

### sign-bundle
`sign-bundle` updates a bundle attaching a cryptographic signature of its exchanges. To use this tool, you need a pair of a private key and a certificate in the `application/cert-chain+cbor` format. See [go/signedexchange](../signedexchange/README.md) for more information on how to create a key and certificate pair.

Assuming you have a key and certificate pair for `example.org`, this command will sign all exchanges in `unsigned.wbn` whose URL's hostname is `example.org`, and writes a new bundle to `signed.wbn`.

```
sign-bundle \
  -i unsigned.wbn \
  -certificate cert.cbor \
  -privateKey priv.key \
  -validityUrl https://example.org/resource.validity.msg \
  -o signed.wbn
```

### dump-bundle
`dump-bundle` dumps the content of a web bundle in a human readable form. To
display content of a bundle file, invoke:
```
dump-bundle -i foo.wbn
```

## Using Bundles
Bundles generated with `gen-bundle` can be opened with web browsers supporting web bundles.

Chrome (79+) experimentally supports Web Bundles with some limitations. See [this document](https://chromium.googlesource.com/chromium/src/+/refs/heads/master/content/browser/web_package/using_web_bundles.md) for more details.
