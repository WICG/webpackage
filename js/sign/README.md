# Signing with Integrity Block

This is a Node.js module for signing
[Web Bundles](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html)
with [integrityblock](../../explainers/integrity-signature.md).

The module takes an existing bundle file and an ed25519 private key, and emits a
new bundle file with cryptographic signature added to the integrity block.

```
wbn-sign \
-i ./wbncli/webbundle.wbn \
-k ~/path/to/ed25519key.pem
```

An ed25519 type of a private key can be generated with:

```
openssl genpkey -algorithm Ed25519 -out ed25519key.pem
```
