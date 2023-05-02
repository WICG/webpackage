import crypto, { KeyObject } from 'crypto';
import * as cborg from 'cborg';
import {
  ED25519_PK_SIGNATURE_ATTRIBUTE_NAME,
  INTEGRITY_BLOCK_MAGIC,
  VERSION_B1,
} from '../utils/constants';
import { checkDeterministic } from '../cbor/deterministic';
import { getRawPublicKey } from '../utils/utils';

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

  constructor(webBundle: Uint8Array, opts: IntegrityBlockSignerOptions) {
    if (opts.key.asymmetricKeyType !== 'ed25519') {
      throw new Error(
        `IntegrityBlockSigner: Only ed25519 keys are currently supported. Your key's type is ${opts.key.asymmetricKeyType}.`
      );
    }
    this.key = opts.key;
    this.webBundle = webBundle;
  }

  sign(): { integrityBlock: Uint8Array; signedWebBundle: Uint8Array } {
    const integrityBlock = this.obtainIntegrityBlock().integrityBlock;

    const publicKey = crypto.createPublicKey(this.key);

    const newAttributes: SignatureAttributes = {
      [ED25519_PK_SIGNATURE_ATTRIBUTE_NAME]: getRawPublicKey(publicKey),
    };

    const ibCbor = integrityBlock.toCBOR();
    const attrCbor = cborg.encode(newAttributes);
    checkDeterministic(ibCbor);
    checkDeterministic(attrCbor);

    const dataToBeSigned = this.generateDataToBeSigned(
      this.calcWebBundleHash(),
      ibCbor,
      attrCbor
    );

    const signature = this.signAndVerify(dataToBeSigned, this.key, publicKey);

    integrityBlock.addIntegritySignature({
      signature,
      signatureAttributes: newAttributes,
    });

    const signedIbCbor = integrityBlock.toCBOR();
    checkDeterministic(signedIbCbor);
    return {
      integrityBlock: signedIbCbor,
      signedWebBundle: new Uint8Array(
        Buffer.concat([signedIbCbor, this.webBundle])
      ),
    };
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
      throw new Error(
        'IntegrityBlockSigner: Re-signing signed bundles is not supported yet.'
      );
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
        'IntegrityBlockSigner: Signature cannot be verified. Your keys might be corrupted.'
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
