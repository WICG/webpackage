import assert from 'assert';
import crypto, { KeyObject } from 'crypto';

import { encode } from 'cborg';

import { checkDeterministic } from '../cbor/deterministic.js';
import {
  IntegrityBlock,
  type SignatureAttributes,
} from '../core/integrity-block.js';
import {
  checkIsValidKey,
  getPublicKeyAttributeName,
  getRawPublicKey,
  isPureWebBundle,
  verifySignature,
} from '../utils/utils.js';
import { ISigningStrategy } from './signing-strategy-interface.js';

// This class were previously exported, but now is going to be used only internally.
// Therefore its methods are marked as deprecated to discourage using them while still keeping backward compatible.
// Later @deprecated markers may change to @internal to make it fully internal.
export class IntegrityBlockSigner {
  private readonly integrityBlock: IntegrityBlock;
  private readonly webBundle: Uint8Array;
  private readonly signingStrategies: Array<ISigningStrategy>;

  /**
   * @internal This constructor is only internal, use `SignedWebBundle` for adding new
   * signatures instead.
   *
   * First argument can be only a pure web bundle (without integrity block).
   */
  constructor(
    webBundle: Uint8Array,
    integrityBlock: IntegrityBlock,
    signingStrategies: Array<ISigningStrategy>
  );
  /**
   * @deprecated External access to `IntegrityBlockSigner` will be removed in
   * a future version, use `SignedWebBundle.from*` instead.
   *
   * First argument can be only a pure web bundle (without integrity block).
   */
  constructor(
    webBundle: Uint8Array,
    webBundleId: string,
    signingStrategies: Array<ISigningStrategy>
  );
  constructor(
    webBundle: Uint8Array,
    arg2: string | IntegrityBlock,
    signingStrategies: Array<ISigningStrategy>
  ) {
    this.webBundle = webBundle;
    assert(isPureWebBundle(this.webBundle), 'Wrong argument');
    // arg2: Web bundle id
    if (typeof arg2 === 'string') {
      this.integrityBlock = new IntegrityBlock() as IntegrityBlock;
      this.integrityBlock.setWebBundleId(arg2);
    }
    // arg2: IntegrityBlock
    else {
      assert(arg2 instanceof IntegrityBlock, 'Wrong argument');
      this.integrityBlock = arg2;
    }
    this.signingStrategies = signingStrategies;
  }

  /** @deprecated This class will become only internal in a future release, use `SignedWebBundle` instead*/
  async sign(): Promise<{
    integrityBlock: Uint8Array;
    signedWebBundle: Uint8Array;
  }> {
    const newIntegrityBlock = await this.signAndGetIntegrityBlock();
    const signedIbCbor = newIntegrityBlock.toCbor();

    const signedWebBundle = Buffer.concat([signedIbCbor, this.webBundle]);

    return {
      integrityBlock: signedIbCbor,
      signedWebBundle,
    };
  }

  // This method is expected to replace 'sign' which now keeps previous return type for backward compatibility
  /** @internal */
  async signAndGetIntegrityBlock(): Promise<IntegrityBlock> {
    const ibCbor = this.integrityBlock.toStrippedCbor();
    checkDeterministic(ibCbor);
    const webBundleHash = this.calcWebBundleHash();

    // Append new signatures to the old stack
    for (const signingStrategy of this.signingStrategies) {
      const publicKey = await signingStrategy.getPublicKey();
      checkIsValidKey('public', publicKey);
      const newAttributes: SignatureAttributes = {
        [getPublicKeyAttributeName(publicKey)]: getRawPublicKey(publicKey),
      };

      const attrCbor = encode(newAttributes);
      checkDeterministic(attrCbor);

      const dataToBeSigned = this.generateDataToBeSigned(
        webBundleHash,
        ibCbor,
        attrCbor
      );

      const signature = await signingStrategy.sign(dataToBeSigned);
      this.verifySignature(dataToBeSigned, signature, publicKey);

      // Cached stripped CBOR (without any singatures) is used in each loop independently as a part
      // of data to be signed, so directly adding signatures to integrity block do not harm.
      this.integrityBlock.addIntegritySignature({
        signature,
        signatureAttributes: newAttributes,
      });
    }

    const signedIbCbor = this.integrityBlock.toCbor();
    checkDeterministic(signedIbCbor);

    return this.integrityBlock;
  }

  // TODO: Remove this method, not needed anymore in this class
  /** @deprecated This class will become only internal in a future release, use `SignedWebBundle` instead*/
  readWebBundleLength(): number {
    // The length of the web bundle is contained in the last 8 bytes of the web
    // bundle, represented as BigEndian.
    const buffer = Buffer.from(this.webBundle.slice(-8));
    // Number is big enough to represent 4GB which is the limit for the web
    // bundle size which can be contained in a Buffer, which is the format
    // in which it's passed back to e.g. Webpack.
    return Number(buffer.readBigUint64BE());
  }

  // TODO: Move this method to SignedWebBundle/WebBundle class, signer do not need this, especially externally
  /** @deprecated This method will not be supported in a future release. */
  calcWebBundleHash(): Uint8Array {
    const hash = crypto.createHash('sha512');
    const data = hash.update(this.webBundle);
    return new Uint8Array(data.digest());
  }

  /** @internal */
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
    const buffer = Buffer.alloc(totalLength);

    let offset = 0;
    dataParts.forEach((d) => {
      buffer.writeBigInt64BE(BigInt(d.length), offset);
      offset += bigEndianNumLength;

      Buffer.from(d).copy(buffer, offset);
      offset += d.length;
    });

    return new Uint8Array(buffer);
  }

  /** @deprecated  Moved to utils */
  verifySignature(
    data: Uint8Array,
    signature: Uint8Array,
    publicKey: KeyObject
  ): void {
    if (!verifySignature(data, signature, publicKey)) {
      throw new Error(
        'IntegrityBlockSigner: Signature cannot be verified. Your keys might be corrupted or not corresponding each other.'
      );
    }
  }
}
