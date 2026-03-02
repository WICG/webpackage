import { KeyObject } from 'crypto';

export interface ISigningStrategy {
  sign(data: Uint8Array): Promise<Uint8Array>;
  getPublicKey(): Promise<KeyObject>;

  // TODO(sonkkeli): Implement.
  // isDevSigning(): Promise<boolean>;
}
