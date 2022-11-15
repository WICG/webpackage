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

    const newAttributes: SignatureAttributes = {
      [ED25519_PK_SIGNATURE_ATTRIBUTE_NAME]: this.getRawPublicKey(publicKey),
    };

    // TODO(sonkkeli): Ensure integrity block and attributes are deterministic.

    const dataToBeSigned = this.generateDataToBeSigned(
      this.calcWebBundleHash(),
      this.integrityBlock.toCBOR(),
      cborg.encode(newAttributes)
    );

    const signature = this.signAndVerify(dataToBeSigned, this.key, publicKey);

    this.integrityBlock.addIntegritySignature({
      signature,
      signatureAttributes: newAttributes,
    });

    // TODO(sonkkeli): Ensure integrity block and attributes are deterministic.

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

  generateDataToBeSigned(
    webBundleHash: Uint8Array,
    integrityBlockCborBytes: Uint8Array,
    newAttributesCborBytes: Uint8Array
  ): Uint8Array {
    // The order is critical and must be the following:
    // (0) hash of the bundle,
    // (1) integrity block, and
    // (2) attributes.
    const dataParts = [
      webBundleHash,
      integrityBlockCborBytes,
      newAttributesCborBytes,
    ];

    const bigEndianNumLength = 8;

    const totalLength = dataParts.reduce((previous, current) => {
      return previous + current.length;
    }, /*one big endian num per part*/ dataParts.length * bigEndianNumLength);
    let buffer = Buffer.alloc(totalLength);

    let offset = 0;
    dataParts.forEach((d) => {
      buffer.writeBigInt64BE(BigInt(d.length), offset);
      offset += bigEndianNumLength;

      Buffer.from(d).copy(buffer, offset);
      offset += d.length;
    });

    return new Uint8Array(buffer);
  }

  getRawPublicKey(publicKey: crypto.KeyObject) {
    // Currently this is the only way for us to get the raw 32 bytes of the public key.
    return new Uint8Array(
      publicKey.export({ type: 'spki', format: 'der' }).slice(-32)
    );
  }

  signAndVerify(
    dataToBeSigned: Uint8Array,
    privateKey: crypto.KeyObject,
    publicKey: crypto.KeyObject
  ): Uint8Array {
    const signature = crypto.sign(
      /*algorithm=*/ undefined,
      dataToBeSigned,
      privateKey
    );

    const isVerified = crypto.verify(
      /*algorithm=*/ undefined,
      dataToBeSigned,
      publicKey,
      signature
    );

    if (!isVerified) {
      throw new Error(
        'Signature cannot be verified. Your keys might be corrupted.'
      );
    }
    return signature;
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
    return cborg.encode([
      this.magic,
      this.version,
      this.signatureStack.map((integritySig) => {
        // The CBOR must have an array of length 2 containing the following:
        // (0) attributes and (1) signature. The order is important.
        return [integritySig.signatureAttributes, integritySig.signature];
      }),
    ]);
  }
}
