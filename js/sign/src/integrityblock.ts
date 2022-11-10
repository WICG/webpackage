import crypto, { KeyObject } from 'crypto';
import {
  ED25519_PK_SIGNATURE_ATTRIBUTE_NAME,
  INTEGRITY_BLOCK_MAGIC,
  VERSION_B1,
} from './constants.js';

// A helper function which can be used to parse string formatted keys to
// KeyObjects.
export function parseStringKey(unparsedKey: string): KeyObject {
  return crypto.createPrivateKey({
    key: unparsedKey,
  });
}

type SignatureAttributeKey = typeof ED25519_PK_SIGNATURE_ATTRIBUTE_NAME;
type SignatureAttributes = { [SignatureAttributeKey: string]: Uint8Array };

type IntegritySignature = {
  signatureAttributes: SignatureAttributes;
  signature: Uint8Array;
};

type IntegrityBlock = {
  magic: Uint8Array;
  version: Uint8Array;
  signatureStack: IntegritySignature[];
};

type IntegrityBlockSignerOptions = {
  key: KeyObject;
};

export class IntegrityBlockSigner {
  private key: KeyObject;
  private webBundle: Uint8Array;
  private integrityBlock: IntegrityBlock | undefined;

  constructor(webBundle: Uint8Array, opts: IntegrityBlockSignerOptions) {
    if (opts.key.asymmetricKeyType !== 'ed25519') {
      throw new Error('Only ed25519 keys are currently supported.');
    }
    this.key = opts.key;
    this.webBundle = webBundle;
  }

  sign(): Uint8Array {
    this.integrityBlock = this.obtainIntegrityBlock().integrityBlock;

    // TODO(sonkkeli): All the rest of the signing logic.
    return new Uint8Array();
  }

  readWebBundleLength(): number {
    // The length of the web bundle is contained in the last 8 bytes of the web
    // bundle, represented as BigEndian.
    let buffer = Buffer.from(this.webBundle.slice(-8));
    // Number is big enough to represent 4GB which is the limit for the web
    // bundle size which can be contained in a Buffer, which is the format
    // in which it's passed back to e.g. Webpack.
    return Number(buffer.readBigUint64BE());
  }

  getEmptyIntegrityBlock = (): IntegrityBlock => {
    return {
      magic: INTEGRITY_BLOCK_MAGIC,
      version: VERSION_B1,
      signatureStack: [],
    };
  };

  obtainIntegrityBlock(): {
    integrityBlock: IntegrityBlock;
    offset: number;
  } {
    const webBundleLength = this.readWebBundleLength();
    if (webBundleLength !== this.webBundle.length) {
      throw new Error('Re-signing signed bundles is not supported yet.');
    }
    return { integrityBlock: this.getEmptyIntegrityBlock(), offset: 0 };
  }
}
