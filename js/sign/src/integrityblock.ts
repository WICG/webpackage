import crypto, { KeyObject } from 'crypto';

// A helper function which can be used to parse string formatted keys to
// KeyObjects.
export function parseStringKey(unparsedKey: string): KeyObject {
  return crypto.createPrivateKey({
    key: unparsedKey,
  });
}

type IntegrityBlockSignerOptions = {
  key: KeyObject;
};

export class IntegrityBlockSigner {
  private key: KeyObject;
  private webBundle: Uint8Array;

  constructor(webBundle: Uint8Array, opts: IntegrityBlockSignerOptions) {
    if (opts.key.asymmetricKeyType !== 'ed25519') {
      throw new Error('Only ed25519 keys are currently supported.');
    }
    this.key = opts.key;
    this.webBundle = webBundle;
  }

  sign(): Uint8Array {
    // TODO(sonkkeli): All the rest of the signing logic.
    return new Uint8Array();
  }
}
