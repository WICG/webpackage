import crypto, { KeyObject } from 'crypto';
import base32Encode from 'base32-encode';
import {
  getRawPublicKey,
  isAsymmetricKeyTypeSupported,
  getSignatureType,
  SignatureType,
} from './utils/utils.js';

// Web Bundle ID is a base32-encoded (without padding) ed25519 public key
// transformed to lowercase. More information:
// https://github.com/WICG/isolated-web-apps/blob/main/Scheme.md#signed-web-bundle-ids
export class WebBundleId {
  // https://github.com/WICG/isolated-web-apps/blob/main/Scheme.md#suffix
  private readonly kEd25519PublicKeyTypeSuffix = [0x00, 0x01, 0x02];
  private readonly kEcdsaP256SHA256PublicKeyTypeSuffix = [0x00, 0x02, 0x02];
  private readonly scheme = 'isolated-app://';
  private readonly key: KeyObject;
  private readonly typeSuffix: number[];

  constructor(key: KeyObject) {
    if (!isAsymmetricKeyTypeSupported(key)) {
      throw new Error(
        `WebBundleId: Only Ed25519 and ECDSA P-256 keys are currently supported.`
      );
    }

    if (key.type === 'private') {
      this.key = crypto.createPublicKey(key);
    } else {
      this.key = key;
    }

    this.typeSuffix = (() => {
      switch (getSignatureType(this.key)) {
        case SignatureType.Ed25519:
          return this.kEd25519PublicKeyTypeSuffix;
        case SignatureType.EcdsaP256SHA256:
          return this.kEcdsaP256SHA256PublicKeyTypeSuffix;
      }
    })();
  }

  serialize() {
    return base32Encode(
      new Uint8Array([...getRawPublicKey(this.key), ...this.typeSuffix]),
      'RFC4648',
      { padding: false }
    ).toLowerCase();
  }

  serializeWithIsolatedWebAppOrigin() {
    return `${this.scheme}${this.serialize()}/`;
  }

  toString() {
    return `\
  Web Bundle ID: ${this.serialize()}
  Isolated Web App Origin: ${this.serializeWithIsolatedWebAppOrigin()}`;
  }
}
