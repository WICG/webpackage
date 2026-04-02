# Signing with Integrity Block

This is a Node.js module for signing
[Web Bundles](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html)
with [integrityblock](../../explainers/integrity-signature.md).

The module takes an existing bundle file and an ed25519 or ecdsa-p256 private key, and emits a
new bundle file with cryptographic signature(s) added to the integrity block.

The module also support other operations on Integrity block like adding/removing/replacing signatures.

## Installation

Using npm:

```bash
npm install wbn-sign
```

## Requirements

This plugin requires Node v22.13.0+.

## Usage

Please be aware that the APIs are not stable yet and are subject to change at
any time.

### Signing a Web Bundle

The recommended way to sign a web bundle is using the `SignedWebBundle` class.

```javascript
import * as fs from 'fs';
import * as wbnSign from 'wbn-sign';

// Read an existing web bundle file or generate using "wbn" npm package.
const webBundle = fs.readFileSync('./path/to/webbundle.wbn');

const privateKey = wbnSign.parsePemKey(
  fs.readFileSync('./path/to/privatekey.pem', 'utf-8')
);

// Sign with a single key.
const signedWebBundle = await wbnSign.SignedWebBundle.fromWebBundle(
  webBundle,
  [new wbnSign.NodeCryptoSigningStrategy(privateKey)]
);

// Get the signed bytes to save to a file.
const signedBytes = signedWebBundle.getSignedWebBundleBytes();
fs.writeFileSync('./path/to/signed.swbn', signedBytes);

// Get the Web Bundle ID (App ID).
console.log('Web Bundle ID:', signedWebBundle.getWebBundleId());
```

### Advanced Usage: Multiple Signatures

You can sign a bundle with multiple keys at once, or add signatures to an already signed bundle.

```javascript
// Sign with multiple keys at once.
const multiSigned = await wbnSign.SignedWebBundle.fromWebBundle(
  webBundle,
  [strategy1, strategy2]
);

// Or add a signature to an existing SignedWebBundle instance.
await multiSigned.addSignature(strategy3);

// You can also load an existing signed web bundle from bytes.
const existingSigned = wbnSign.SignedWebBundle.fromBytes(
  fs.readFileSync('./path/to/already_signed.swbn')
);

// And remove a signature by providing the public key.
existingSigned.removeSignature(publicKeyBytes);
```

### Custom Signing Strategies

You can implement your own signing strategy by implementing the `ISigningStrategy` interface.

```javascript
const customStrategy = new (class {
  async sign(data: Uint8Array): Promise<Uint8Array> {
    // Connect to an external signing service.
  }
  async getPublicKey(): Promise<KeyObject> {
    // Return the public key from the service.
  }
})();

const signedWebBundle = await wbnSign.SignedWebBundle.fromWebBundle(
  webBundle,
  [customStrategy]
);
```

### Calculating Web Bundle ID

This library also exposes a helper class to calculate the Web Bundle's ID (or
App ID) from the private or public key, which can then be used when
bundling
[Isolated Web Apps](https://github.com/WICG/isolated-web-apps/blob/main/README.md).

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

This package also includes 2 CLI tools:

- `wbn-sign`: A comprehensive tool for signing bundles and managing signatures.
- `wbn-dump-id`: A simple utility to calculate the Web Bundle ID for a given key.

### Running wbn-sign

The base usage is: `wbn-sign [command] [options] <arguments...>`

#### Commands:

- `sign <web_bundle> <private_keys...>`: Signs a web bundle with one or more private keys.
- `add-signature <signed_web_bundle> <private_keys...>`: Adds new signatures to an already signed bundle.
- `remove-signature <signed_web_bundle> <keys...>`: Removes signatures from a bundle. Keys can be public (Base64/.pem) or private (.pem).
- `replace-signature <signed_web_bundle> <old_key> <new_private_key>`: Replaces an existing signature.
- `info <web_bundle>`: Displays information about the integrity block, including the Web Bundle ID and public keys of signers.

For more details, run `wbn-sign help [command]`.

#### Examples:

```bash
# Sign a web bundle with two keys.
wbn-sign sign webbundle.wbn key1.pem key2.pem -o signed.swbn

# Add a signature to an existing swbn.
wbn-sign add-signature signed.swbn key3.pem --in-place

# View information about a signed bundle.
wbn-sign info signed.swbn
```

### Running wbn-dump-id

- `--key <filePath>` (required): Path to ed25519/ecdsaP256 public or private key.
- `--with-iwa-scheme` (`-s`): Dumps the Web Bundle ID with `isolated-app://` scheme.
- `--with-key-type` (`-t`): Outputs the type of the key used (ecdsa/ed25519).

Example:
```bash
wbn-dump-id -s -k path/to/ed25519key.pem
```

This would print the Web Bundle ID calculated from `ed25519key.pem` into the
console with the `isolated-app://` scheme.

If one wants to save the ID into a file or into an environment variable, one can
do the following (respectively):

```bash
wbn-dump-id -k file_enc.pem -s > webbundleid.txt
```

```bash
export DUMP_WEB_BUNDLE_ID="$(wbn-dump-id -k file_enc.pem -s)"
```

The environment variable set like this, can then be used in other scripts, for
example in `--baseURL` when creating a web bundle with
[wbn CLI tool](https://github.com/WICG/webpackage/tree/main/js/bundle#cli).

## Generating Keys

### Ed25519 (recommended)

An unencrypted ed25519 private key can be generated with:

```
openssl genpkey -algorithm Ed25519 -out ed25519key.pem
```

**Note**: We recommend using Ed25519, as it is considered more secure than ECDSA P-256. Unlike ECDSA, which relies on a pseudo-random number generator and is vulnerable to entropy-related flaws, Ed25519 is deterministic and remains secure even if the system's random number generator is compromised.

### ECDSA P-256
```bash
openssl ecparam -name prime256v1 -genkey -noout -out ecdsap256key.pem
```

### Encryption
For better security, one should prefer using passphrase-encrypted
private keys. To encrypt an unencrypted private key (both supported types), run:

```bash
# encrypt the key (will ask for a passphrase, make sure to use a strong one)
openssl pkcs8 -in private_key.pem -topk8 -out encrypted_key.pem
# delete the unencrypted key
rm private_key.pem
```

If you use an encrypted private key to sign a bundle, you will be prompted for its passphrase as
part of the signing process. If you want to use the CLI tool programmatically,
then you can bypass the passphrase prompt by storing the passphrase in an
environment variable named `WEB_BUNDLE_SIGNING_PASSPHRASE`.
## Release Notes

### v0.3.1
- Enhanced **CLI**: New commands `add-signature`, `remove-signature`, `replace-signature`, and `info`.

### v0.3.0
- **Major architectural update**: Introduced `SignedWebBundle` as the primary interface for managing signed bundles.
- Support for **multi-signatures**: Add, remove, and replace signatures in already signed web bundles.
- **NOTICE**: Upcoming breaking change. `IntegrityBlockSigner` is deprecated and will be removed in a future version. Migrate to `SignedWebBundle`.


### v0.2.7
  - The new command line interface that supports commands introduced for wbn-sign tool

### v0.2.6
Mostly developer changes, not affecting usage:

 - Adds eslint to control style consistency and prevent errors with `npx eslint .`
 - Adds prettifier module to keep imports always sorted
 - Updates some devDependecy packages version

### v0.2.5

- Add support for dumping bundle IDs from public keys (used to be private-only).
  Note that --private-key is hence renamed to --key.

### v0.2.3

- Add support for obtaining bundleID from a .swbn file.

### v0.2.2

- BREAKING CHANGE: Removed support for v1 integrity block format.

### v0.2.1

- Moved is_v2 to the last and optional (defaulting to true) argument of
  `IntegrityBlockSigner` constructor. This is a preparation for the future
  removal of the deprecated v1 format.
- CLI signer defaults to v2 format of integrity block now.

### v0.2.0

- Add support for the v2 integrity block format. Now web-bundle-id is no longer
  presumed to be a derivative of the first public key in the stack, but rather
  acts as a separate entry in the integrity block attributes, and multiple
  independent signatures are allowed to facilitate key rotation.

### v0.1.3

- Add support for ECDSA P-256 SHA-256 signatures

### v0.1.2

- Add support for calculating the Web Bundle ID with the CLI tool.

### v0.1.1

- Add support for bypassing the passphrase prompt for encrypted private keys by
  reading the passphrase from the `WEB_BUNDLE_SIGNING_PASSPHRASE` environment
  variable if it is set when using the CLI tool directly.

### v0.1.0

- BREAKING CHANGE: Introducing the support for using different types of signing
  strategies. Will enable users to initialize a SigningStrategy class
  (implementing the newly introduced `ISigningStrategy` interface). Also `sign`
  changes to be an async function.
- Add support for using passphrase-encrypted Ed25519 private keys.

### v0.0.1

- Support for signing web bundles with
  [integrity block](https://github.com/WICG/webpackage/blob/main/explainers/integrity-signature.md)
  added.
