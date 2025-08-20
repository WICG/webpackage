import crypto, { KeyObject } from 'crypto';
import base32Encode from 'base32-encode';
import {
  getRawPublicKey,
  isAsymmetricKeyTypeSupported,
  getSignatureType,
} from './utils/utils.js';
import { SignatureType } from './utils/constants.js';
import * as cborg from 'cborg';

type decodedSignedWebBundle = [
  [Uint8Array, Uint8Array, { webBundleId: string }, any[][]],
  Uint8Array
];
// Web Bundle ID is a base32-encoded (without padding) ed25519 public key
// transformed to lowercase. More information:
// https://github.com/WICG/isolated-web-apps/blob/main/Scheme.md#signed-web-bundle-ids
export class WebBundleId {
  // https://github.com/WICG/isolated-web-apps/blob/main/Scheme.md#suffix
  private readonly TYPE_SUFFIX_MAPPING = new Map<SignatureType, number[]>([
    [SignatureType.Ed25519, [0x00, 0x01, 0x02]],
    [SignatureType.EcdsaP256SHA256, [0x00, 0x02, 0x02]],
  ]);
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

    this.typeSuffix = this.TYPE_SUFFIX_MAPPING.get(getSignatureType(this.key))!;
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
  getBundleId(signedWebBundle: Uint8Array) {
    try {
      const decodedData: decodedSignedWebBundle =
        cborg.decodeFirst(signedWebBundle);
      const attributes = decodedData[0][2];
      return attributes.webBundleId;
    } catch (e) {
      throw Error(
        'Failed to get webBundleId from ${signedWebBundle}, cause: ${e}'
      );
    }
  }
}
