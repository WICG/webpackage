# Web Bundles Integrity Block

## Intro

### Problem statement

So far there is no way to check the integrity of a web bundle. For example some responses may be removed from the web bundle or be replaced with compromised ones. To solve this problem it makes sense to have a signature that would guarantee that the web bundle was not modified.

### Terminology

Currently web bundles support signatures of individual responses (not of the whole bundle). In order not to mix up the signature that guarantees integrity of the whole web bundle with the signature that signs just individual responses we will call the first one an "integrity signature".

### Minimal requirements

The integrity signature must guarantee the integrity of the web bundle with one signature, whose public key is trusted.

Please note that the user agent should know which public key to trust as an attacker can modify the app and resign it with another private key. The source of trust can be:
* the trusted public keys can be included in enterprise policy;
* the public key can be obtained from a trusted app distributor;
* any other trusted source.

In future it makes sense to add the ability to have more than one signature (one signature would be set by the developer, another by the app distributor) and key rotation (similar to what APK Signature V3 has). We should take this into account while proposing the solution in order to simplify the future development.

## Design

### Integrity block

We will introduce a data structure called integrity block that will contain:
* integrity signature 
* additional relevant to signature information (e.g.  public key, certificate, expiry date, etc). 

The exact structure of it will be discussed later.

The integrity block will be located before the unsigned web bundle. Basically a web bundle with integrity signature will be a sequence of 2 CBOR objects:
* integrity-block (defined below), which contains a signature and related data;
* webbundle (defined [here](https://www.ietf.org/archive/id/draft-yasskin-wpack-bundled-exchanges-04.html)) which is the content of the signed web bundle.

In the binary form this sequence will be stored as a concatenation of the serialized integrity-block and webbundle CBOR objects. Unfortunately there is [no CDDL notation for CBOR sequences](https://datatracker.ietf.org/doc/html/rfc8742#section-4.1) in the top level.

### Structure of the integrity block

The integrity block structure will contain the following information:
Attributes in the form of a CBOR map. The attributes must contain a 32-bytes Ed25519 public key (the attributes map key is `ed25519PublicKey`). In future there can be more attributes, e.g. expiry date, key rotation attribute, etc. The keys in the map will always be strings.

### Signature bytes.

CDDL of the signature is:

```
integrity-signature = [ 
  attributes: {
    "ed25519PublicKey" => bstr .size 32, ; 32-bytes long Ed25519 public key
  }, 
  signature: bstr .size 64, ; 64 bytes long Ed25519 signature
]
```

Because in the future we will support more than one signature, we should put an `integrity-signature` object in a signature stack: we will "push" a new integrity-signature after signing a web bundle and "pop" an `integrity-signature` after we verify it. As a container for the signature stack we will use a CBOR array. For now after successful signing the signature stack will have just one `integrity-signature`.

The CDDL notation of the integrity block in its final state with the signature stack is following:

```
integrity-block = [
  magic: h'F0 9F 96 8B F0 9F 93 A6',
  ; Version value is '1b\0\0' for beta and '1\0\0\0' for release.
  version: bstr .size 4,
  ; In v1, each bundle must have exactly one signature.
  signature-stack: [ 1*1 integrity-signature],
]
```

Please note that while signature validation or signing processes the signature stack may be empty, but in the final state (when the web bundle is signed and is ready for validation) there should be one integrity signature there.

To be able to recognize the signed bundle by first several bytes of the file we will add a magic number as the first element of the integrity block. [Similar to Web Bundles](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html#section-4.1) let’s make integrity-block magic equals to the following hex numbers: 0xF0 0x9F 0x96 0x8B 0xF0 0x9F 0x93 0xA6 which are UTF-8 encoded “🖋📦” (U+1F58B, U+1F4E6)

The integrity block will also contain a version field. Also [similar to Web Bundles](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html#section-4.1) we will make it a bytestring that must be 31 00 00 00 in base 16 (an ASCII "1" followed by 3 0s) for this version of integrity signature (for development period we will use `31 62 00 00` in base 16, which is ASCII "1b" followed by 2 0s). If the recipient doesn't support the version in this field, it must reject the validation of the integrity signature and return an error.

### Signing process

The input for signing is an unsigned web bundle file that doesn have an `integrity-block`. We assume that there is a private key and the corresponding public key. The process of signing is following:

1. Generate a minimal valid integrity-block. At this point the `integrity-block` must be deterministically encoded CBOR (see below) and consists of:

    * Magic.
    * Version.
    * Signature stack that contains 0 signatures.

2. Generate signature `attributes` CBOR map and put there a key-value pair where the key is "ed25519PublicKey", and the value is 32-byte long Ed25519 public key in the form of a byte string.
3. Calculate SHA-512 of the serialized content of the `webbundle`.
4. Set data-to-be-signed as a concatenation of 6 elements listed below (the order must persist):
    * 64 bit big-endian integer length of the web bundle hash from step 3;
    * web bundle hash from step 3;
    * 64 bit big-endian integer length of the serialized `integrity-block`;
    * serialized `integrity-block` from step 1;
    * 64 bit big-endian integer length of the attributes;
    * serialized attributes from step 2.
5. Compute signature of the data-to-be-signed. For Ed25519 the signature will be a binary string according to [rfc8410](https://datatracker.ietf.org/doc/html/rfc8410).
6. Build a `integrity-signature` object that will be an array of 2 elements:
    * attributes from step 2;
    * the value of integrity signature from step 5.
7. Add the `integrity-signature` object to the `signature-stack` of the `integrity-block`.
8. Combine serialized `integrity-block` with the web-bundle file in a [CBOR sequence](https://datatracker.ietf.org/doc/html/rfc8742) (basically store `integrity-block` and after it append the contents of the web bundle file).

### Signature validation

The main assumption is that we can place the whole integrity-block in RAM.

Validation of the signature consist of the following steps:

1. Read the `integrity-block` from the web bundle with the integrity signature file.
2. Check that version and magic have expected values.
3. Pop the `integrity-signature` from the `signature-stack` and store it separately. The result of this operation should be a deterministically encoded `integrity-block` with empty `signature-stack` and separately stored `integrity-signature`.
4. Verify that the attributes map of the integrity-signature has only one item with the "ed25519PublicKey" key. For future when we support more attributes: check that we can understand all the attributes. Consider signature as not valid if we don’t know how to process any attribute or the conditions that are set by the attribute are not held (e.g. a “timestamp” attribute is in future).
5. Read the value of the "ed25519PublicKey" attribute. 
6. Calculate SHA-512 of the serialized content of the `webbundle`.
7. Validate the `signed-data` with the public key and signature obtained from `integrity-signature`. The `signed-data` should be concatenation of:
    * 64 bit big-endian integer length of the web bundle hash from step 6;
    * web bundle hash from step 6;
    * 64 bit big-endian integer length of the `integrity-block`;
    * serialized `integrity-block` from step 3;
    * 64 bit big-endian integer length of the attributes;
    * serialized attributes of the signature.

### Signing algorithm

So far we will support only Ed25519 signing algorithm OID ​​1.3.101.112 (RFC 8032). 
