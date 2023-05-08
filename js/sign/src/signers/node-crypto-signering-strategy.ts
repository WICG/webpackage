import crypto, { KeyObject } from 'crypto';
import { checkIsValidEd25519Key } from '../utils/utils.js';
import { ISigningStrategy } from './signing-strategy-interface.js';

// Class to be used when signing with parsed `crypto.KeyObject` private key
// provided directly in the constructor.
export class NodeCryptoSigningStrategy implements ISigningStrategy {
  constructor(private readonly privateKey: KeyObject) {
    checkIsValidEd25519Key('private', privateKey);
  }

  async sign(dataToBeSigned: Uint8Array): Promise<Uint8Array> {
    return crypto.sign(
      /*algorithm=*/ undefined,
      dataToBeSigned,
      this.privateKey
    );
  }

  async getPublicKey(): Promise<KeyObject> {
    return crypto.createPublicKey(this.privateKey);
  }
}
