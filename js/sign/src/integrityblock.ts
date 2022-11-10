import crypto, { KeyObject } from 'crypto';
import * as cborg from 'cborg';
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

type IntegrityBlockSignerOptions = {
  key: KeyObject;
};

export class IntegrityBlockSigner {
  private key: KeyObject;
  private webBundle: Uint8Array;
  private webBundleHash: Uint8Array | undefined;
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

    const publicKey = crypto.createPublicKey(this.key);
    // Currently this is the only way for us to get the raw 32 bytes of the public key.
    const rawPublicKey = new Uint8Array(
      publicKey.export({ type: 'spki', format: 'der' }).slice(-32)
    );

    // TODO(sonkkeli): Add to signatureStack.
    const newAttributes: SignatureAttributes = {
      [ED25519_PK_SIGNATURE_ATTRIBUTE_NAME]: rawPublicKey,
    };

    this.webBundleHash = this.calcWebBundleHash();

    // TODO(sonkkeli): All the rest of the signing logic.
    return this.integrityBlock.toCBOR();
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

  obtainIntegrityBlock(): {
    integrityBlock: IntegrityBlock;
    offset: number;
  } {
    const webBundleLength = this.readWebBundleLength();
    if (webBundleLength !== this.webBundle.length) {
      throw new Error('Re-signing signed bundles is not supported yet.');
    }
    return { integrityBlock: new IntegrityBlock(), offset: 0 };
  }

  calcWebBundleHash(): Uint8Array {
    var hash = crypto.createHash('sha512');
    var data = hash.update(this.webBundle);
    return new Uint8Array(data.digest());
  }
}

export class IntegrityBlock {
  private readonly magic = INTEGRITY_BLOCK_MAGIC;
  private readonly version = VERSION_B1;
  private signatureStack: IntegritySignature[] = [];

  constructor() {}

  addIntegritySignature(is: IntegritySignature) {
    this.signatureStack.unshift(is);
  }

  toCBOR(): Uint8Array {
    return cborg.encode([this.magic, this.version, this.signatureStack]);
  }
}
