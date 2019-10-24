# go/bundle
This directory contains a reference implementation of [Bundled HTTP Exchanges](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html) spec.

## Overview
We currently provide three command-line tools: `gen-bundle`, `sign-bundle` and `dump-bundle`.

`gen-bundle` command is a bundle generator tool. `gen-bundle` consumes a set of http exchanges (currently in the form of [HAR format](https://w3c.github.io/web-performance/specs/HAR/Overview.html), URL list file, or static files in a local directory), and emits a bundled exchange file.

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
`gen-bundle` generates a bundled exchange file. There are three ways to provide a set of exchanges to bundle; by a HAR file, by a URL list, and by a local directory.

These command-line flags are common to all the three options:

- `-primaryURL` specifies the bundle's main resource URL. This URL is also used as the fallback destination when browser cannot process the bundle.
- `-manifestURL` specifies the bundle's [manifest](https://www.w3.org/TR/appmanifest/) URL.
- `-o` specifies name of the output bundle file.
- `-headerOverride` adds additional response header to all bundled responses. Existing values of the header are overwritten.

#### From a HAR file

One convenient way to generate HAR file is via Chrome Devtools. Navigate to "Network" panel, and right-click on any resource and select "Save as HAR with content".
![generating har with devtools](https://raw.githubusercontent.com/WICG/webpackage/master/go/bundle/har-devtools.png)

Once you have the har file, generate the bundled exchange file via:
```
gen-bundle -har foo.har -o foo.wbn
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
gen-bundle -dir static -baseURL https://www.example.com/ -o foo.wbn
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
`dump-bundle` dumps the content of a bundled exchange in a human readable form. To display content of a bundle file, invoke:
```
dump-bundle -i foo.wbn
```

## Using Bundles
Bundles generated with `gen-bundle` can be opened with web browsers supporting web bundles.

- Chrome (80+): enable `chrome://flags/#web-bundles` flag to enable experimental support for Web Bundles.

## Dealing with Common Problems in Unsigned Bundles

In Chrome's current experimental implementation, bundles can be loaded from local files. Currently only unsigned bundles are supported and they are loaded in "untrusted mode", where:

- Document's URL is set to the concatenation of bundle's URL and the document's inner URL, e.g. `file://path/to/wbn?https://example.com/article.html`.
- (As a consequence of the above) document's origin is set to an [opaque origin](https://html.spec.whatwg.org/multipage/origin.html#concept-origin-opaque), not the origin of the exchange's URL.
- Document's base URL is set to the exchange's URL, so that relative URL resolution will work correctly.

The following sections list some issues you may encounter when you create an unsigned bundle for a web site.

### Cannot access the origin's resources

Since documents loaded from unsigned bundles have opaque origins, bundled pages cannot access resources of the original site, such as cookies, local storages or service worker caches.

This is a fundamental limitation of unsigned bundles. Origin-signed bundles (not supported by any browser yet) will not have this limitation.

### CORS failures

In unsigned bundles, a document has a synthesized URL like `file://path/to/wbn?https://example.com/` but its subresources are requested with their original url, like `https://example.com/style.css`. That means, **all subresource requests are cross-origin**. So [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS) requests (e.g. fonts, module scripts) will fail if the response doesn’t have the `Access-Control-Allow-Origin` header, even when it was same-origin request in the original site.

A workaround is to inject the `Access-Control-Allow-Origin` header to the responses when generating bundles. To do that with `gen-bundle`, use `-headerOverride 'Access-Control-Allow-Origin: *'`.

### Cannot create web workers

Web worker is created by giving a URL of worker script to `new Worker()`, where the script URL **must be same-origin** (relative URLs are resolved with document's base URL). In unsigned bundles, this doesn't work because the document has an opaque origin.

A workaround to this issue is to create a worker via a blob URL. i.e. define a function like this:
```javascript
function newWorkerViaBlob(script) {
  const scriptURL = new URL(script, document.baseURI);
  const blob = new Blob(['importScripts("' + scriptURL + '");'],
                        {type: 'text/javascript'});
  return new Worker(window.URL.createObjectURL(blob));
}
```
and replace `new Worker(scriptURL)` with `newWorkerViaBlob(scriptURL)`.

Note that you may also have to fix `importScripts()` or `fetch()` from the worker script that use relative URLs, because the worker’s base URL is now `blob://...`.

### Other things that do not work in unsigned bundles
- Scripts using the value of `location.href` may not work (use `document.baseURI`'s value instead).
- Service workers (does not work in file://)
- History API (does not work in opaque origins)
