import crypto, { KeyObject } from 'crypto';
import read from 'read';

// A helper function that can be used to read the passphrase to decrypt a
// password-decrypted private key.
export async function readPassphrase(): Promise<string> {
  try {
    const passphrase = await read({
      prompt: 'Passphrase for the key: ',
      silent: true,
      replace: '*',
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

export function getRawPublicKey(publicKey: crypto.KeyObject) {
  // Currently this is the only way for us to get the raw 32 bytes of the public key.
  return new Uint8Array(
    publicKey.export({ type: 'spki', format: 'der' }).slice(-32)
  );
}

export function isValidEd25519Key(
  expectedKeyType: crypto.KeyObjectType,
  key: KeyObject
) {
  if (key.type !== expectedKeyType) {
    throw new Error(
      `Expected key type to be ${expectedKeyType}, but it was "${key.type}".`
    );
  }

  if (key.asymmetricKeyType !== 'ed25519') {
    throw new Error(
      `Expected asymmetric key type to be "ed25519", but it was "${key.asymmetricKeyType}".`
    );
  }
}
