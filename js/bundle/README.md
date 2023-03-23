# Web Bundles

This is a Node.js module for serializing and parsing the `application/webbundle`
format defined in the
[Web Bundles](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html)
draft spec.

## Installation

Using npm:

```bash
npm install wbn
```

## Usage

Please be aware that the API is not yet stable and is subject to change any
time.

Creating a Bundle:

```javascript
import * as fs from 'fs';
import * as wbn from 'wbn';

const builder = new wbn.BundleBuilder();
builder.addExchange(
  'https://example.com/', // URL
  200, // response code
  { 'Content-Type': 'text/html' }, // response headers
  '<html>Hello, Web Bundle!</html>' // response body (string or Uint8Array)
);
builder.setPrimaryURL('https://example.com/'); // entry point URL

fs.writeFileSync('out.wbn', builder.createBundle());
```

Reading a Bundle:

```javascript
import * as fs from 'fs';
import * as wbn from 'wbn';

const buf = fs.readFileSync('out.wbn');
const bundle = new wbn.Bundle(buf);
const exchanges = [];
for (const url of bundle.urls) {
  const resp = bundle.getResponse(url);
  exchanges.push({
    url,
    status: resp.status,
    headers: resp.headers,
    body: new TextDecoder('utf-8').decode(resp.body),
  });
}
console.log(
  JSON.stringify(
    {
      version: bundle.version, // format version
      exchanges,
    },
    null,
    2
  )
);
```

## CLI

This package also includes `wbn` command which lets you build a web bundle from
a local directory. For example, if you have all the necessary files for
`https://example.com/` in `static/` directory, run the following command:

```sh
$ wbn --dir static \
      --baseURL https://example.com/ \
      --output out.wbn
```

Run `wbn --help` for full options.

Note: currently this CLI only covers a subset of the functionality offered by
[`gen-bundle`](https://github.com/WICG/webpackage/tree/master/go/bundle#gen-bundle)
Go tool.

### Backwards compatibility

This module supports creating and parsing Web Bundles that follow different
draft versions of the format specification. In particular:

- version `b2` follows
  [the latest version of the Web Bundles spec](https://datatracker.ietf.org/doc/html/draft-ietf-wpack-bundled-responses)
  (default)
- version `b1` follows
  [the previous version of the Web Bundles spec](https://datatracker.ietf.org/doc/html/draft-yasskin-wpack-bundled-exchanges-03)

To create a new bundle with the `b1` format, pass the version value to the
constructor:

```javascript
const builder = (new wbn.BundleBuilder('b1'))
  .setPrimaryURL('https://example.com/')
  .setManifestURL('https://example.com/manifest.json')
  .addExchange(...);

fs.writeFileSync('out_b1.wbn', builder.createBundle());
```

Likewise, the `wbn` command can optionally take a `--formatVersion b1` parameter
when creating a new Web Bundle.

This module also takes care of selecting the right format version automatically
when reading a bundle. Check the property `bundle.version` to know the decoded
bundle's format version.

## Using Bundles

Generated bundles can be opened with web browsers supporting web bundles.

Chrome (79+) experimentally supports navigation to Web Bundles with some
limitations. See
[this document](https://chromium.googlesource.com/chromium/src/+/refs/heads/master/content/browser/web_package/using_web_bundles.md)
for more details.

Chrome (104+) supports
[`<script type=webbundle>`](https://github.com/WICG/webpackage/blob/main/explainers/subresource-loading.md).

## Release Notes

### 0.0.9

- Add support for overriding headers.

### 0.0.8

- Now `wbn` package provides both ES modules and CommonJS exports.
- Dependency on the Node.js API has been removed from the `wbn` module, making
  it easier to use the module in browsers.

### 0.0.7

- Now BundleBuilder accepts relative resource URLs. Relative URLs can be used in
  [`<script type=webbundle>`](https://github.com/WICG/webpackage/blob/main/explainers/subresource-loading.md)
  where relative URLs are resolved using the bundle's URL as base URL.

### 0.0.6

- Now BundleBuilder generates bundles of format `b2` by default. See
  [Backwards compatibility](#backwards-compatibility) section above.
