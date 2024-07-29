# Web Bundles Integrity Block

## Intro

### Problem statement

So far there is no way to check the integrity of a web bundle. For example some responses may be removed from the web bundle or be replaced with compromised ones. To solve this problem it makes sense to have a signature that would guarantee that the web bundle was not modified.

### Terminology

Currently web bundles support signatures of individual responses (not of the whole bundle). In order not to mix up the signature that guarantees integrity of the whole web bundle with the signature that signs just individual responses we will call the first one an "integrity signature".

### Minimal requirements

The integrity signature must guarantee the integrity of the web bundle with one signature, whose public key is trusted.

Please note that the user agent should know which public key to trust as an attacker can modify the app and resign it with another private key. The source of trust can be:

- the trusted public keys can be included in enterprise policy;
- the public key can be obtained from a trusted app distributor;
- any other trusted source.

In future it makes sense to add the ability to have more than one signature (one signature would be set by the developer, another by the app distributor) and key rotation (similar to what APK Signature V3 has). We should take this into account while proposing the solution in order to simplify the future development.

## Design

### Integrity block

We will introduce a data structure called integrity block that will contain:

- integrity signature
- additional relevant to signature information (e.g. public key, certificate, expiry date, etc).

The exact structure of it will be discussed later.

The integrity block will be located before the unsigned web bundle. Basically a web bundle with integrity signature will be a sequence of 2 CBOR objects:

- integrity-block (defined below), which contains a signature and related data;
- web-bundle (defined [here](https://www.ietf.org/archive/id/draft-yasskin-wpack-bundled-exchanges-04.html)) which is the content of the signed web bundle.

In the binary form this sequence will be stored as a concatenation of the serialized integrity-block and web-bundle CBOR objects. Unfortunately there is [no CDDL notation for CBOR sequences](https://datatracker.ietf.org/doc/html/rfc8742#section-4.1) in the top level.

### Structure of the integrity block

The integrity block structure will contain the following information:
Attributes in the form of a CBOR map. The attributes must always contain exactly one of the following two entries:

- A 32-byte Ed25519 public key (the attributes map key is `ed25519PublicKey`);
- A 33-byte Ecdsa P-256 public key for the Ecdsa P-256 SHA-256 signing scheme (the attributes map key is `ecdsaP256SHA256PublicKey`).

In the future there can be more attributes, e.g. expiry date, key rotation attribute, etc. The keys in the map will always be strings.

### Signature bytes.

CDDL of the signature is:

```
integrity-signature-ed25519 = [
  attributes: {
    "ed25519PublicKey" => bstr .size 32, ; 32-bytes long Ed25519 public key
  },
  signature: bstr .size 64, ; 64 bytes long Ed25519 signature
]

integrity-signature-ecdsa-p256-sha256 = [
  attributes: {
    "ecdsaP256SHA256PublicKey" => bstr .size 33, ; 33-bytes long Ecdsa P-256 public key
  },
  signature: bstr, ; Ecdsa P-256 SHA-256 signature
]

integrity-signature = integrity-signature-ed25519 / integrity-signature-ecdsa-p256-sha256
```

Since web bundles might be signed by more than one signature, `integrity-signature` objects are wrapped in a signature list which is represented as a CBOR array.

The CDDL notation of the integrity block in its final state with the signature list is as follows:

```
integrity-block = [
  magic: h'F0 9F 96 8B F0 9F 93 A6',
  version: bstr .size 4, ; Version value is '2\0\0\0' for release.
  attributes: {
    "webBundleId" => tstr
  }
  signature-list: [ +integrity-signature ],
]
```

Please note that during the signing processes the signature list may be empty, but in the final state (when the web bundle is signed and is ready for validation) there should be at least one integrity signature there.

To be able to recognize the signed bundle by first several bytes of the file we will add a magic number as the first element of the integrity block. [Similar to Web Bundles](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html#section-4.1) let’s make integrity-block magic equals to the following hex numbers: 0xF0 0x9F 0x96 0x8B 0xF0 0x9F 0x93 0xA6 which are UTF-8 encoded “🖋📦” (U+1F58B, U+1F4E6)

The integrity block will also contain a version field. Also [similar to Web Bundles](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html#section-4.1) we will make it a bytestring that must be 32 00 00 00 in base 16 (an ASCII "2" followed by 3 0s) for this version of integrity signature. If the recipient doesn't support the version in this field, it must reject the validation of the integrity signature and return an error.

### Signing process

The input for signing is an unsigned web bundle file that doesn have an `integrity-block`. We assume that there is a private key and the corresponding public key. The process of signing is following:

1. Generate a minimal valid integrity-block. At this point the `integrity-block` must be deterministically encoded CBOR (see below) and consists of:

   - Magic.
   - Version.
   - Attributes.
   - Empty signature list.

2. Calculate a SHA-512 hash of the serialized content of the `web-bundle`.
3. Create a separate empty signature list (`temp-signature-list`) to store intermediate signing results.

4. For each signing key:

- Generate signature `attributes` CBOR map with respect to the `integrity-signature-ed25519.attributes` or `integrity-signature-ecdsa-p256-sha256.attributes` specification accordingly.
- Set `data-to-be-signed` as a concatenation of 6 elements listed below (the order must persist):
  - 64 bit big-endian integer length of the web bundle hash from step 2;
  - web bundle hash from step 2;
  - 64 bit big-endian integer length of the serialized `integrity-block` from step 1;
  - serialized `integrity-block` from step 1;
  - 64 bit big-endian integer length of signature `attributes`;
  - serialized signature `attributes`.
- Compute the binary signature of `data-to-be-signed`.
- Build an `integrity-signature` object that will be an array of 2 elements:
  - attributes from step 2;
  - the binary signature from the previous step.
- Add the `integrity-signature` to `temp-signature-list`.

6. Assign `integrity-block.signature-list` from `temp-signature-list`.
7. Combine the serialized `integrity-block` with the web bundle file in a [CBOR sequence](https://datatracker.ietf.org/doc/html/rfc8742) (basically store `integrity-block` and after it append the contents of the web bundle file).

### Integrity Block Validation

The main assumption is that we can place the whole integrity-block into RAM.

In order for an integrity block to be deemed valid, it should pass both the signature validation requirements as well as identity validation requirements.

#### Signature validation

Validation of the signature consists of the following steps:

1. Read the `integrity-block` from the web bundle with the integrity signature file.
2. Check that version and magic have expected values.
3. Calculate a SHA-512 hash of the serialized content of the `web-bundle`.
4. Create a minimal valid integrity block (`integrity-block-min`) by clearing `signature-list`.
5. For each signature in the original integrity block's `signature-list`:

- Verify that the signature conforms to the definition of `integrity-signature`. If not, this signature is considered unknown and silently ignored.
- If the `attributes` map contains any entries unknown to the current client, these are also silently discarded and do not affect the validity check.
- Read the public key from `attributes`.
- Validate the `signed-data` with the public key and signature obtained from `integrity-signature`. The `signed-data` is a concatenation of:
  - 64 bit big-endian integer length of the web bundle hash from step 3;
  - web bundle hash from step 3;
  - 64 bit big-endian integer length of the `integrity-block` from step 4;
  - serialized `integrity-block` from step 4;
  - 64 bit big-endian integer length of the attributes;
  - serialized attributes of the signature.

6. If there are no invalid signatures and at least one known signature, the signature validation process succeeds.

#### Identity Validation

Signature validation establishes the _trust_ of an signed web bundle; however, its _identity_ is defined via `integrity-block.attributes.webBundleId` and has to be validated separately to allow the embedder to perform key rotations.

The standard validation process goes as follows:

1. Query the embedder regarding the expected public key `K` for the given `web-bundle-id`.
2. If `K` is defined, ensure that `K` is present in `integrity-block.signature-list`.

- If yes, the identity vaidation succeeds; if not, fails.

3. If there's no expected public key for `web-bundle-id`, check whether this id can be obtained from any public key in `integrity-block.signature-list` according to the conversion process outlined in the Signed Web Bundle IDs [explainer](https://github.com/WICG/isolated-web-apps/blob/main/Scheme.md#signed-web-bundle-ids).

- If yes, the identity validation succeeds; if not, fails.

### Signing algorithm

The supported signing algorithms are

- Ed25519 ([RFC8032](https://datatracker.ietf.org/doc/html/rfc8032));
- Ecdsa P-256 SHA-256 ([RFC4754](https://datatracker.ietf.org/doc/html/rfc4754)).

Note that the signatures are not required to be homogeneous -- `signature-list` might mix in signatures of different kinds.
