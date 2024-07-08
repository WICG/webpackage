import crypto, { KeyObject } from 'crypto';
import read from 'read';
import assert from 'assert';
import {
  PUBLIC_KEY_ATTRIBUTE_NAME_MAPPING,
  SignatureType,
} from './constants.js';

// A helper function that can be used to read the passphrase to decrypt a
// password-decrypted private key.
export async function readPassphrase(description: string): Promise<string> {
  try {
    const passphrase = await read({
      prompt: `Passphrase for the key ${description}: `,
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
