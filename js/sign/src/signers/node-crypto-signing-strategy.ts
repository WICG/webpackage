import crypto, { KeyObject } from 'crypto';

import { checkIsValidKey, getDigestAlgorithm } from '../utils/utils.js';
import { ISigningStrategy } from './signing-strategy-interface.js';

// Class to be used when signing with parsed `crypto.KeyObject` private key
// provided directly in the constructor.
export class NodeCryptoSigningStrategy implements ISigningStrategy {
  constructor(private readonly privateKey: KeyObject) {
    checkIsValidKey('private', privateKey);
  }

  async sign(data: Uint8Array): Promise<Uint8Array> {
    return crypto.sign(
      getDigestAlgorithm(this.privateKey),
      data,
      this.privateKey
    );
  }

  async getPublicKey(): Promise<KeyObject> {
    return crypto.createPublicKey(this.privateKey);
  }
}
