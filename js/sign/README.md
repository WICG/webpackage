# Signing with Integrity Block

This is a Node.js module for signing
[Web Bundles](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html)
with [integrityblock](../../explainers/integrity-signature.md).

The module takes an existing bundle file and an ed25519 private key, and emits a
new bundle file with cryptographic signature added to the integrity block.

## Installation

Using npm:

```bash
npm install wbn-sign
```

## Usage

Please be aware that the APIs are not stable yet and are subject to change at
any time.

Signing a web bundle file:

```javascript
import * as fs from 'fs';
import * as wbnSign from 'wbn-sign';

// Read an existing web bundle file or generate using "wbn" npm package.
const webBundle = fs.readFileSync('./path/to/webbundle.wbn');

const privateKey = wbnSign.parsePemKey(
  fs.readFileSync('./path/to/privatekey.pem', 'utf-8')
);

const { signedWebBundle } = new wbnSign.IntegrityBlockSigner(webBundle, {
  key: privateKey,
}).sign();

fs.writeFileSync(signedWebBundle);
```

This library also exposes a helper class to calculate the Web Bundle's ID (or
App ID) from the private or public ed25519 key, which can then be used when
bundling
[Isolated Web Apps](https://github.com/WICG/isolated-web-apps/blob/main/README.md).

Calculating the web bundle ID for Isolated Web Apps:

```javascript
import * as fs from 'fs';
import * as wbnSign from 'wbn-sign';

const privateKey = wbnSign.parsePemKey(
  fs.readFileSync('./path/to/privatekey.pem', 'utf-8')
);

// Web Bundle ID only:
const webBundleId = new wbnSign.WebBundleId(privateKey).serialize();

// With origin, meaning "isolated-app://" combined with Web Bundle ID:
const webBundleIdWithIWAOrigin = new wbnSign.WebBundleId(
  privateKey
).serializeWithIsolatedWebAppOrigin();
```

## CLI

This package also includes a CLI tool `wbn-sign` which lets you sign a web
bundle easily without having to write any additional JavaScript.

Example command:

```bash
wbn-sign \
-i ~/path/to/webbundle.wbn \
-o ~/path/to/signed-webbundle.swbn \
-k ~/path/to/ed25519key.pem
```

An ed25519 type of a private key can be generated with:

```
openssl genpkey -algorithm Ed25519 -out ed25519key.pem
```
