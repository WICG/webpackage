import { assert } from 'console';

import { encode } from 'cborg';

import { IntegrityBlockSigner } from '../signers/integrity-block-signer.js';
import { ISigningStrategy } from '../signers/signing-strategy-interface.js';
import { warnLog } from '../utils/cli-utils.js';
import {
  calcWebBundleHash,
  generateDataToBeSigned,
  isSignedWebBundle,
  parseRawPublicKey,
  verifySignature,
} from '../utils/utils.js';
import { WebBundleId } from '../web-bundle-id.js';
import { IntegrityBlock } from './integrity-block.js';

export type SignatureValidationResult =
  | { status: 'success'; derivedBundleId: string; isValid: boolean }
  | { status: 'error'; error: string };

export class SignedWebBundle {
  private constructor(
    private integrityBlock: IntegrityBlock,
    private pureWebBundle: Uint8Array
  ) {}

  // Use with raw Singed Web Bundle data, for example read from file with fs.readFile('bundle.swbn')
  static fromBytes(signedWebBundle: Uint8Array): SignedWebBundle {
    if (!isSignedWebBundle(signedWebBundle)) {
      throw new Error('Not a signed web bundle.');
    }

    const offset = this.obtainIntegrityOffset(signedWebBundle);

    const integrityBlockBytes = signedWebBundle.slice(0, offset);
    const integrityBlock = IntegrityBlock.fromCbor(integrityBlockBytes);

    const bundle = signedWebBundle.slice(offset);

    return new SignedWebBundle(integrityBlock, bundle);
  }

  // Web bundle ID is derived from the first key if not provided
  static async fromWebBundle(
    webBundle: Uint8Array,
    signingStrategies: Array<ISigningStrategy>,
    options?: { webBundleId?: string }
  ): Promise<SignedWebBundle> {
    assert(signingStrategies.length > 0, 'No signing strategies provided');

    const publicKey = await signingStrategies[0].getPublicKey();
    const webBundleId =
      options?.webBundleId ?? new WebBundleId(publicKey).serialize();

    const { signedWebBundle } = await new IntegrityBlockSigner(
      webBundle,
      webBundleId,
      signingStrategies
    ).sign();

    return SignedWebBundle.fromBytes(signedWebBundle);
  }

  async addSignature(signingStrategy: ISigningStrategy): Promise<this> {
    this.integrityBlock = await new IntegrityBlockSigner(
      this.pureWebBundle,
      this.integrityBlock,
      [signingStrategy]
    ).signAndGetIntegrityBlock();
    return this;
  }

  removeSignature(publicKey: Uint8Array): this {
    this.integrityBlock.removeIntegritySignature(publicKey);
    return this;
  }

  printInfo(): void {
    warnLog(
      'This feature is experimental, the form of output will change in the future. Do not use with scripts.'
    );
    this.integrityBlock.printInfo();
  }

  getIntegrityBlock(): IntegrityBlock {
    return this.integrityBlock;
  }

  validateSignatures(): SignatureValidationResult[] {
    const webBundleHash = calcWebBundleHash(this.pureWebBundle);
    const ibCbor = this.integrityBlock.toStrippedCbor();

    return this.integrityBlock.getSignatureStack().map((signature) => {
      try {
        const [keyType, publicKey] = IntegrityBlock.parseSignatureAttributes(
          signature.signatureAttributes
        );
        const keyObject = parseRawPublicKey(keyType, publicKey);
        const derivedBundleId = new WebBundleId(keyObject).serialize();
        const attrCbor = encode(signature.signatureAttributes);
        const dataToBeSigned = generateDataToBeSigned(
          webBundleHash,
          ibCbor,
          attrCbor
        );
        const isValid = verifySignature(
          dataToBeSigned,
          signature.signature,
          keyObject
        );
        return { status: 'success', isValid, derivedBundleId };
      } catch (err) {
        return {
          status: 'error',
          error: err instanceof Error ? err.message : String(err),
        };
      }
    });
  }

  getWebBundleId(): string {
    return this.integrityBlock.getWebBundleId();
  }

  getSignedWebBundleBytes(): Uint8Array {
    if (this.integrityBlock.getSignatureStack().length == 0) {
      throw new Error(
        'Signed Web Bundle instance is in invalid state (no signatures).'
      );
    }
    if (!this.integrityBlock.getWebBundleId()) {
      throw new Error(
        'Signed Web Bundle instance is in invalid state (bundle id not set)'
      );
    }
    return Buffer.concat([this.integrityBlock.toCbor(), this.pureWebBundle]);
  }

  overrideBundleId(bundleId: string): this {
    this.integrityBlock.setWebBundleId(bundleId);
    return this;
  }

  // private method, static to emphasize pure functional character
  private static obtainIntegrityOffset(bundle: Uint8Array): number {
    const bundleLengthFromInternalData =
      SignedWebBundle.readWebBundleLength(bundle);
    const offset = bundle.length - bundleLengthFromInternalData;
    if (bundleLengthFromInternalData < 0 || offset <= 0) {
      throw new Error(
        'SignedWebBundle::fromBytes: The provided bytes do not represent a signed web bundle.'
      );
    }
    return offset;
  }

  // private method, static to emphasize pure functional character
  private static readWebBundleLength(bundle: Uint8Array): number {
    // The length of the web bundle is contained in the last 8 bytes of the web
    // bundle, represented as BigEndian.
    const buffer = Buffer.from(bundle.slice(-8));
    // Number is big enough to represent 4GB which is the limit for the web
    // bundle size which can be contained in a Buffer, which is the format
    // in which it's passed back to e.g. Webpack.
    return Number(buffer.readBigUint64BE());
  }
}
