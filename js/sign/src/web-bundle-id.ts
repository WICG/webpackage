import crypto, { KeyObject } from 'crypto';
import base32Encode from 'base32-encode';
import { getRawPublicKey } from './utils/utils';

// Web Bundle ID is a base32-encoded (without padding) ed25519 public key
// transformed to lowercase. More information:
// https://github.com/WICG/isolated-web-apps/blob/main/Scheme.md#signed-web-bundle-ids
export class WebBundleId {
  // https://github.com/WICG/isolated-web-apps/blob/main/Scheme.md#suffix
  private readonly appIdSuffix = [0x00, 0x01, 0x02];
  private readonly scheme = 'isolated-app://';
  private readonly key: KeyObject;

  constructor(ed25519key: KeyObject) {
    if (ed25519key.asymmetricKeyType !== 'ed25519') {
      throw new Error(
        `WebBundleId: Only ed25519 keys are currently supported. Your key's type is ${ed25519key.asymmetricKeyType}.`
      );
    }

    if (ed25519key.type === 'private') {
      this.key = crypto.createPublicKey(ed25519key);
    } else {
      this.key = ed25519key;
    }
  }

  serialize() {
    return base32Encode(
      new Uint8Array([...getRawPublicKey(this.key), ...this.appIdSuffix]),
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
