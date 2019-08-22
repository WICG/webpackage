# go/bundle
This directory contains a reference implementation of [Bundled HTTP Exchanges](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html) spec.

## Overview
We currently provide three command-line tools: `gen-bundle`, `sign-bundle` and `dump-bundle`.

`gen-bundle` command is a bundle generator tool. `gen-bundle` consumes a set of http exchanges (currently in the form of [HAR format](https://w3c.github.io/web-performance/specs/HAR/Overview.html) or static files in a local directory), and emits a bundled exchange file.

`sign-bundle` command attaches a signature to a bundle. `sign-bundle` takes an existing bundle file, a certificate and a private key, and emits a new bundle file with cryptographic signature for the bundled exchanges added.

`dump-bundle` command is a bundle inspector tool. `dump-bundle` dumps the enclosed http exchanges of a given bundled exchange file in a human readable form.

You are also welcome to use the code as golang lib (e.g. `import "github.com/WICG/webpackage/go/bundle"`), but please be aware that the API is not yet stable and is subject to change any time.

## Getting Started

### Prerequisite
golang environment needs to be set up in prior to using the tool. We are testing the tool on latest golang. Please refer to [Go Getting Started documentation](https://golang.org/doc/install) for the details.

### Installation
We recommend using `go get` to install the command-line tool.

```
go get -u github.com/WICG/webpackage/go/bundle/cmd/...
```

## Usage

### gen-bundle
`gen-bundle` generates a bundled exchange file from a HAR file.

One convenient way to generate HAR file is via Chrome Devtools. Navigate to "Network" panel, and right-click on any resource and select "Save as HAR with content".
![generating har with devtools](https://raw.githubusercontent.com/WICG/webpackage/master/go/bundle/har-devtools.png)

Once you have the har file, generate the bundled exchange file via:
```
gen-bundle -har foo.har -o foo.wbn
```

You can also create a bundle from a local directory. For example, if you have the necessary files for the site `https://www.example.com/` in `static/` directory, run:
```
gen-bundle -dir static -baseURL https://www.example.com/ -o foo.wbn
```
You can use `-startURL` command-line flag to specify the entry point of the bundle, as a relative URL from `-baseURL`. Currently, this just makes the exchange for `-startURL` the first entry in the bundled exchange file.

### sign-bundle
`sign-bundle` updates a bundle attaching a cryptographic signature of its exchanges. To use this tool, you need a pair of a private key and a certificate in the `application/cert-chain+cbor` format. See [go/signedexchange](../signedexchange/README.md) for more information on how to create a key and certificate pair.

Assuming you have a key and certificate pair for `example.org`, thils command will sign all exchanges in `unsigned.wbn` whose URL's hostname is `example.org`, and writes a new bundle to `signed.wbn`.

```
sign-bundle \
  -i unsigned.wbn \
  -certificate cert.cbor \
  -privateKey priv.key \
  -validityUrl https://example.org/resource.validity.msg \
  -o signed.wbn
```

### dump-bundle
`dump-bundle` dumps the content of a bundled exchange in a human readable form. To display content of a har file, invoke:
```
dump-bundle -i foo.har
```
