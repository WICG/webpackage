import crypto, { KeyObject } from 'crypto';
import * as cborg from 'cborg';
import {
  INTEGRITY_BLOCK_MAGIC,
  VERSION_B2,
} from '../utils/constants.js';
import { checkDeterministic } from '../cbor/deterministic.js';
import {
  getRawPublicKey,
  checkIsValidKey,
  getPublicKeyAttributeName,
} from '../utils/utils.js';
import { ISigningStrategy } from './signing-strategy-interface.js';

type SignatureAttributeKey = string;
type SignatureAttributes = { [SignatureAttributeKey: string]: Uint8Array };

type IntegritySignature = {
  signatureAttributes: SignatureAttributes;
  signature: Uint8Array;
};

export class IntegrityBlockSigner {
  constructor(
    private readonly webBundle: Uint8Array,
    private readonly webBundleId: string,
    private readonly signingStrategies: Array<ISigningStrategy>,
  ) {}

  async sign(): Promise<{
    integrityBlock: Uint8Array;
    signedWebBundle: Uint8Array;
  }> {
    const integrityBlock = this.obtainIntegrityBlock().integrityBlock;
    integrityBlock.setWebBundleId(this.webBundleId);

    const signatures = new Array<IntegritySignature>();
    for (const signingStrategy of this.signingStrategies) {
      const publicKey = await signingStrategy.getPublicKey();
      checkIsValidKey('public', publicKey);
      const newAttributes: SignatureAttributes = {
        [getPublicKeyAttributeName(publicKey)]: getRawPublicKey(publicKey),
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

      const signature = await signingStrategy.sign(dataToBeSigned);
      this.verifySignature(dataToBeSigned, signature, publicKey);

      // The signatures are calculated independently, and thus not appended to
      // the integrity block here to not affect subsequent calculations.
      signatures.push({
        signature,
        signatureAttributes: newAttributes,
      });
    }

    for (const signature of signatures) {
      integrityBlock.addIntegritySignature(signature);
    }

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

  verifySignature(
    data: Uint8Array,
    signature: Uint8Array,
    publicKey: KeyObject
  ): void {
    // For ECDSA P-256 keys the algorithm is implicitly selected as SHA-256.
    const isVerified = crypto.verify(
      /*algorithm=*/ undefined,
      data,
      publicKey,
      signature
    );

    if (!isVerified) {
      throw new Error(
        'IntegrityBlockSigner: Signature cannot be verified. Your keys might be corrupted or not corresponding each other.'
      );
    }
  }
}

export class IntegrityBlock {
  private attributes: Map<string, string> = new Map();
  private signatureStack: IntegritySignature[] = [];

  constructor() {}

  setWebBundleId(webBundleId: string) {
    this.attributes.set('webBundleId', webBundleId);
  }

  addIntegritySignature(is: IntegritySignature) {
    this.signatureStack.push(is);
  }

  toCBOR(): Uint8Array {
    return cborg.encode([
      INTEGRITY_BLOCK_MAGIC,
      VERSION_B2,
      this.attributes,
      this.signatureStack.map((integritySig) => {
        // The CBOR must have an array of length 2 containing the following:
        // (0) attributes and (1) signature. The order is important.
        return [integritySig.signatureAttributes, integritySig.signature];
      }),
    ]);
  }
}
