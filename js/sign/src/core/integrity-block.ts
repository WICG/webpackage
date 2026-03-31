import assert from 'assert';

import { decode, encode } from 'cborg';

import {
  INTEGRITY_BLOCK_MAGIC,
  SIGNATURE_ATTRIBUTE_TO_TYPE_MAPPING,
  VERSION_B2,
  WEB_BUNDLE_ID_ATTRIBUTE_NAME,
  type SignatureType,
} from '../utils/constants.js';

export type SignatureAttributes = {
  [SignatureAttributeKey: string]: Uint8Array;
};

export type IntegritySignature = {
  signatureAttributes: SignatureAttributes;
  signature: Uint8Array;
};

export class IntegrityBlock {
  private attributes: Map<string, string> = new Map();
  private signatureStack: IntegritySignature[] = [];

  /** @internal */
  constructor() {}

  static fromCbor(integrityBlockBytes: Uint8Array): IntegrityBlock {
    const integrityBlock = new IntegrityBlock();
    try {
      const [magic, version, attributes, signatureStack] = decode(
        integrityBlockBytes,
        { useMaps: true }
      );

      assert(magic instanceof Uint8Array, 'Invalid magic bytes');
      assert.deepStrictEqual(
        magic,
        INTEGRITY_BLOCK_MAGIC,
        'Invalid magic bytes'
      );

      assert(version instanceof Uint8Array, 'Invalid version');
      assert.deepStrictEqual(version, VERSION_B2, 'Invalid version');

      assert(attributes instanceof Map, 'Invalid attributes');
      assert(
        attributes.has(WEB_BUNDLE_ID_ATTRIBUTE_NAME),
        `Missing ${WEB_BUNDLE_ID_ATTRIBUTE_NAME} attribute`
      );
      integrityBlock.setWebBundleId(
        attributes.get(WEB_BUNDLE_ID_ATTRIBUTE_NAME)!
      );

      assert(signatureStack instanceof Array, 'Invalid signature stack');
      assert(signatureStack.length > 0, 'Invalid signature stack');

      for (const signatureBlock of signatureStack) {
        assert(signatureBlock instanceof Array, 'Invalid signature');
        assert.strictEqual(signatureBlock.length, 2, 'Invalid signature');

        const [attributes, signature] = signatureBlock;
        assert(attributes instanceof Map, 'Invalid signature attributes');
        assert(signature instanceof Uint8Array, 'Invalid signature');
        assert.equal(attributes.size, 1, 'Invalid signature attributes');

        const [keyType, publicKey] = [...attributes][0];
        assert(
          SIGNATURE_ATTRIBUTE_TO_TYPE_MAPPING.has(keyType),
          'Invalid signature attribute key type'
        );
        assert(
          publicKey instanceof Uint8Array,
          'Invalid signature attribute key'
        );

        integrityBlock.addIntegritySignature({
          signatureAttributes: { [keyType]: publicKey },
          signature: Buffer.from(signature),
        });
      }
      return integrityBlock;
    } catch (err) {
      throw new Error(`Invalid integrity block: ${(err as Error).message}`, {
        cause: err,
      });
    }
  }

  getWebBundleId(): string {
    return this.attributes.get(WEB_BUNDLE_ID_ATTRIBUTE_NAME)!;
  }

  setWebBundleId(webBundleId: string) {
    this.attributes.set(WEB_BUNDLE_ID_ATTRIBUTE_NAME, webBundleId);
  }

  addIntegritySignature(is: IntegritySignature) {
    this.signatureStack.push(is);
  }

  removeIntegritySignature(publicKey: Uint8Array) {
    this.signatureStack = this.signatureStack.filter((integritySignature) => {
      // Uint8Arrays cannot be directly compared, but Buffer can
      return !Buffer.from(
        Object.values(integritySignature.signatureAttributes)[0]
      ).equals(publicKey);
    });
  }

  getSignatureStack(): IntegritySignature[] {
    return this.signatureStack;
  }

  toCbor(): Uint8Array {
    return encode([
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

  // Stripped CBOR does not include signatures and is a part of data which hash is signed
  /** @internal */
  toStrippedCbor(): Uint8Array {
    return encode([INTEGRITY_BLOCK_MAGIC, VERSION_B2, this.attributes, []]);
  }

  private static parseSignatureAttributes(
    attributes: SignatureAttributes
  ): [SignatureType, Uint8Array] {
    assert(
      Object.entries(attributes).length == 1,
      'Invalid signature attributes'
    );
    const [maybeType, publicKey] = Object.entries(attributes)[0];
    const type = SIGNATURE_ATTRIBUTE_TO_TYPE_MAPPING.get(maybeType);
    assert(type != undefined, 'Invalid signature attributes');
    return [type, publicKey];
  }
}
