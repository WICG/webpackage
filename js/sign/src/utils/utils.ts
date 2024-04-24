import crypto, { KeyObject } from 'crypto';
import read from 'read';
import assert from 'assert';
import {
  ECDSA_P256_SHA256_PK_SIGNATURE_ATTRIBUTE_NAME,
  ED25519_PK_SIGNATURE_ATTRIBUTE_NAME,
} from './constants.js';

// A helper function that can be used to read the passphrase to decrypt a
// password-decrypted private key.
export async function readPassphrase(): Promise<string> {
  try {
    const passphrase = await read({
      prompt: 'Passphrase for the key: ',
      silent: true,
      replace: '*',
      // Output must be != `stdout`. Otherwise saving the `wbn-dump-id`
      // result into a file or an environment variable also includes the prompt.
      output: process.stderr,
    });
    return passphrase;
  } catch (er) {
    throw new Error('Reading passphrase failed.');
  }
}

// A helper function which can be used to parse string formatted keys to
// KeyObjects.
export function parsePemKey(
  unparsedKey: Buffer,
  passphrase?: string
): KeyObject {
  return crypto.createPrivateKey({
    key: unparsedKey,
    passphrase,
  });
}

export function isAsymmetricKeyTypeSupported(key: crypto.KeyObject) {
  return (
    key.asymmetricKeyType === 'ed25519' ||
    (key.asymmetricKeyType === 'ec' &&
      key.asymmetricKeyDetails!.namedCurve === 'secp256k1')
  );
}

export enum SignatureType {
  Ed25519,
  EcdsaP256SHA256,
}

export function getSignatureType(key: crypto.KeyObject) {
  assert(
    isAsymmetricKeyTypeSupported(key),
    'Expected either "Ed25519" or "ECDSA P-256" key.'
  );
  if (key.asymmetricKeyType === 'ed25519') {
    return SignatureType.Ed25519;
  }
  return SignatureType.EcdsaP256SHA256;
}

export function getPublicKeyAttributeName(key: crypto.KeyObject) {
  switch (getSignatureType(key)) {
    case SignatureType.Ed25519:
      return ED25519_PK_SIGNATURE_ATTRIBUTE_NAME;
    case SignatureType.EcdsaP256SHA256:
      return ECDSA_P256_SHA256_PK_SIGNATURE_ATTRIBUTE_NAME;
  }
}

export function getRawPublicKey(publicKey: crypto.KeyObject) {
  const exportedKey = publicKey.export({ type: 'spki', format: 'der' });
  switch (getSignatureType(publicKey)) {
    case SignatureType.Ed25519:
      // Currently this is the only way for us to get the raw 32 bytes of the public key.
      return new Uint8Array(exportedKey.slice(-32));
    case SignatureType.EcdsaP256SHA256: {
      // The last 65 bytes are the raw bytes of the ECDSA P-256 public key.
      // For the purposes of signing, we'd like to convert it to its compressed form that takes only 33 bytes.
      const uncompressedHex = exportedKey.slice(-65).toString('hex');
      const compressedHex = crypto.ECDH.convertKey(
        uncompressedHex,
        'secp256k1',
        'hex',
        'hex',
        'compressed'
      ) as string;
      return Buffer.from(compressedHex, 'hex');
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
