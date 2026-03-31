// @ts-check
/** @typedef {import('jasmine')} _ */

import { IntegrityBlock } from '../lib/wbn-sign.js';

describe('Integrity Block', () => {
  it('fromCbor(toCbor(a)) == a', () => {
    const ib = new IntegrityBlock();
    ib.setWebBundleId(
      '4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic'
    );
    ib.addIntegritySignature({
      signatureAttributes: { ed25519PublicKey: new Uint8Array(32).fill(1) },
      signature: Buffer.from(new Uint8Array(64).fill(2)),
    });

    const ib2 = IntegrityBlock.fromCbor(ib.toCbor());
    expect(ib2).toEqual(ib);
  });

  it('toCbor(fromCbor(bytes)) == bytes', () => {
    const ib = new IntegrityBlock();
    ib.setWebBundleId(
      'amoiebz32b7o24tilu257xne2yf3nkblkploanxzm7ebeglseqpfeaacai'
    );
    ib.addIntegritySignature({
      signatureAttributes: {
        ecdsaP256SHA256PublicKey: new Uint8Array(33).fill(3),
      },
      signature: Buffer.from(new Uint8Array(64).fill(4)),
    });

    const bytes = ib.toCbor();
    const ib2 = IntegrityBlock.fromCbor(bytes);
    expect(ib2.toCbor()).toEqual(bytes);
  });

  it('fromCbor(toCbor(a)) == a with multiple signatures', () => {
    const ib = new IntegrityBlock();
    ib.setWebBundleId(
      '4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic'
    );
    ib.addIntegritySignature({
      signatureAttributes: { ed25519PublicKey: new Uint8Array(32).fill(1) },
      signature: Buffer.from(new Uint8Array(64).fill(2)),
    });
    ib.addIntegritySignature({
      signatureAttributes: {
        ecdsaP256SHA256PublicKey: new Uint8Array(33).fill(3),
      },
      signature: Buffer.from(new Uint8Array(64).fill(4)),
    });

    const ib2 = IntegrityBlock.fromCbor(ib.toCbor());
    expect(ib2).toEqual(ib);
    expect(ib2.getSignatureStack().length).toEqual(2);
  });

  it('fails to parse with invalid magic', () => {
    const ib = new IntegrityBlock();
    ib.setWebBundleId(
      '4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic'
    );
    ib.addIntegritySignature({
      signatureAttributes: { ed25519PublicKey: new Uint8Array(32).fill(1) },
      signature: Buffer.from(new Uint8Array(64).fill(2)),
    });

    const bytes = ib.toCbor();
    bytes[5] = 0x00; // Corrupt magic
    expect(() => IntegrityBlock.fromCbor(bytes)).toThrowError(
      /Invalid integrity block: Invalid magic bytes/
    );
  });

  it('fails to parse with invalid version', () => {
    const ib = new IntegrityBlock();
    ib.setWebBundleId(
      '4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic'
    );
    ib.addIntegritySignature({
      signatureAttributes: { ed25519PublicKey: new Uint8Array(32).fill(1) },
      signature: Buffer.from(new Uint8Array(64).fill(2)),
    });

    const bytes = ib.toCbor();
    // Version should be kept in bytes: 12-16
    bytes[14] = 0xff;
    expect(() => IntegrityBlock.fromCbor(bytes)).toThrowError(
      /Invalid integrity block: Invalid version/
    );
  });

  it('fails to parse without webBundleId attribute', () => {
    const ib = new IntegrityBlock();
    // Don't set webBundleId
    ib.addIntegritySignature({
      signatureAttributes: { ed25519PublicKey: new Uint8Array(32).fill(1) },
      signature: Buffer.from(new Uint8Array(64).fill(2)),
    });

    const bytes = ib.toCbor();
    expect(() => IntegrityBlock.fromCbor(bytes)).toThrowError(
      /Invalid integrity block: Missing webBundleId attribute/
    );
  });

  it('fails to parse with unknown signature attribute key', () => {
    const ib = new IntegrityBlock();
    ib.setWebBundleId(
      '4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic'
    );
    ib.addIntegritySignature({
      signatureAttributes: { unknownAttribute: new Uint8Array(32).fill(1) },
      signature: Buffer.from(new Uint8Array(64).fill(2)),
    });

    const bytes = ib.toCbor();
    expect(() => IntegrityBlock.fromCbor(bytes)).toThrowError(
      /Invalid integrity block: .* key type/
    );
  });

  it('fails to parse with multiple signature attributes', () => {
    const ib = new IntegrityBlock();
    ib.setWebBundleId(
      '4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic'
    );
    ib.addIntegritySignature({
      signatureAttributes: {
        ed25519PublicKey: new Uint8Array(32).fill(1),
        extraAttribute: new Uint8Array(32).fill(2),
      },
      signature: Buffer.from(new Uint8Array(64).fill(2)),
    });

    const bytes = ib.toCbor();
    expect(() => IntegrityBlock.fromCbor(bytes)).toThrowError(
      /Invalid integrity block: Invalid signature attributes/
    );
  });
});
