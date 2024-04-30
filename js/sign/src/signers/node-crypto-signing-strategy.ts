import crypto, { KeyObject } from 'crypto';
import { checkIsValidKey } from '../utils/utils.js';
import { ISigningStrategy } from './signing-strategy-interface.js';

// Class to be used when signing with parsed `crypto.KeyObject` private key
// provided directly in the constructor.
export class NodeCryptoSigningStrategy implements ISigningStrategy {
  constructor(private readonly privateKey: KeyObject) {
    checkIsValidKey('private', privateKey);
  }

  async sign(data: Uint8Array): Promise<Uint8Array> {
    // For ECDSA P-256 keys the algorithm is implicitly selected as SHA-256.
    return crypto.sign(/*algorithm=*/ undefined, data, this.privateKey);
  }

  async getPublicKey(): Promise<KeyObject> {
    return crypto.createPublicKey(this.privateKey);
  }
}
