import assert from 'assert';
import crypto, { KeyObject } from 'crypto';

import { decode } from 'cborg';

import {
  INTEGRITY_BLOCK_MAGIC,
  PUBLIC_KEY_ATTRIBUTE_NAME_MAPPING,
  SignatureType,
} from './constants.js';

// A helper function which can be used to parse string formatted keys to
// KeyObjects.
export function parsePemKey(
  unparsedKey: string | Buffer<ArrayBufferLike>,
  passphrase?: string
): KeyObject {
  return crypto.createPrivateKey({
    key: unparsedKey,
    passphrase,
  });
}

function maybeGetSignatureType(key: crypto.KeyObject): SignatureType | null {
  switch (key.asymmetricKeyType) {
    case 'ed25519':
      return SignatureType.Ed25519;
    case 'ec':
      if (key.asymmetricKeyDetails?.namedCurve === 'prime256v1') {
        return SignatureType.EcdsaP256SHA256;
      }
      break;
    default:
      break;
  }
  return null;
}

export function isAsymmetricKeyTypeSupported(key: crypto.KeyObject): boolean {
  return maybeGetSignatureType(key) !== null;
}

// 'Pure' = not signed web bundles (without integrity block)
export function isPureWebBundle(bundle: Uint8Array): boolean {
  let parsedBundle: Uint8Array[];
  try {
    parsedBundle = decode(bundle, { useMaps: true }) as Uint8Array[];
    if (new TextDecoder('utf-8').decode(parsedBundle[0]) !== '🌐📦') {
      return false;
    }
    // WebBundles have their length in the last cbor section
    const buffer = Buffer.from(bundle.slice(-8));
    if (bundle.length != Number(buffer.readBigUint64BE())) {
      return false;
    }
  } catch {
    return false;
  }
  return true;
}

// Just checks magic bytes at the begging, does not check if valid/parsable
export function isSignedWebBundle(bundle: Uint8Array): boolean {
  // First CBOR byte: Array of length ...
  // Second CBOR byte: String of length ...
  // and then 8 bytes of magic string
  return (
    bundle.length >= 10 &&
    (bundle[1] & 0b00011111) == 8 &&
    Buffer.from(bundle.slice(2, 10)).equals(INTEGRITY_BLOCK_MAGIC)
  );
}

export function getSignatureType(key: crypto.KeyObject): SignatureType {
  const signatureType = maybeGetSignatureType(key);
  assert(
    signatureType !== null,
    'Expected either "Ed25519" or "ECDSA P-256" key.'
  );
  return signatureType;
}

export function getPublicKeyAttributeName(key: crypto.KeyObject) {
  return PUBLIC_KEY_ATTRIBUTE_NAME_MAPPING.get(getSignatureType(key))!;
}

export function getRawPublicKey(publicKey: crypto.KeyObject) {
  const exportedKey = publicKey.export({ type: 'spki', format: 'der' });
  switch (getSignatureType(publicKey)) {
    case SignatureType.Ed25519:
      // Currently this is the only way for us to get the raw 32 bytes of the public key.
      return new Uint8Array(exportedKey.subarray(-32));
    case SignatureType.EcdsaP256SHA256: {
      // The last 65 bytes are the raw bytes of the ECDSA P-256 public key.
      // For the purposes of signing, we'd like to convert it to its compressed form that takes only 33 bytes.
      const uncompressedKey = exportedKey.subarray(-65);
      const compressedKey = crypto.ECDH.convertKey(
        uncompressedKey,
        'prime256v1',
        /*inputEncoding=*/ undefined,
        /*outputEncoding=*/ undefined,
        'compressed'
      ) as Buffer;
      return new Uint8Array(compressedKey);
    }
  }
}

// Throws an error if the key is not a valid Ed25519 or ECDSA P-256 key of the specified type.
export function checkIsValidKey(
  expectedKeyType: crypto.KeyObjectType,
  key: KeyObject
) {
  if (key.type !== expectedKeyType) {
    throw new Error(
      `Expected key type to be ${expectedKeyType}, but it was "${key.type}".`
    );
  }

  if (!isAsymmetricKeyTypeSupported(key)) {
    throw new Error(`Expected either "Ed25519" or "ECDSA P-256" key.`);
  }
}

export function verifySignature(
  data: Uint8Array,
  signature: Uint8Array,
  publicKey: KeyObject
): boolean {
  // For ECDSA P-256 keys the algorithm is implicitly selected as SHA-256.
  const isVerified = crypto.verify(
    /*algorithm=*/ undefined,
    data,
    publicKey,
    signature
  );
  return isVerified;
}
