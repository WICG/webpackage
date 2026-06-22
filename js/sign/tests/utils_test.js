import crypto from 'crypto';

import { SignatureType } from '../lib/utils/constants.js';
import * as utils from '../lib/utils/utils.js';

describe('Utils', () => {
  describe('parseRawPublicKey', () => {
    it('correctly parses an Ed25519 raw public key', () => {
      const { publicKey, privateKey } = crypto.generateKeyPairSync('ed25519');
      // Export to raw bytes
      const spki = publicKey.export({ type: 'spki', format: 'der' });
      const rawPublicKey = new Uint8Array(spki.subarray(-32));

      const parsedKey = utils.parseRawPublicKey(
        SignatureType.Ed25519,
        rawPublicKey
      );

      expect(parsedKey.type).toBe('public');
      expect(parsedKey.asymmetricKeyType).toBe('ed25519');
      // Ensure the exported keys match
      expect(parsedKey.export({ type: 'spki', format: 'der' })).toEqual(spki);
    });

    it('correctly parses an ECDSA P-256 raw public key (compressed)', () => {
      const { publicKey, privateKey } = crypto.generateKeyPairSync('ec', {
        namedCurve: 'prime256v1',
      });
      const spki = publicKey.export({ type: 'spki', format: 'der' });
      const uncompressedKey = spki.subarray(-65);
      const rawPublicKey = crypto.ECDH.convertKey(
        uncompressedKey,
        'prime256v1',
        undefined,
        undefined,
        'compressed'
      );

      const parsedKey = utils.parseRawPublicKey(
        SignatureType.EcdsaP256SHA256,
        new Uint8Array(rawPublicKey)
      );

      expect(parsedKey.type).toBe('public');
      expect(parsedKey.asymmetricKeyType).toBe('ec');
      // Check if it reconstructs exactly the same uncompressed SPKI format
      expect(parsedKey.export({ type: 'spki', format: 'der' })).toEqual(spki);
    });
  });

  describe('calcWebBundleHash', () => {
    it('calculates SHA-512 hash correctly', () => {
      const payload = new Uint8Array([1, 2, 3, 4, 5]);
      const expectedHash = crypto.createHash('sha512').update(payload).digest();
      const hash = utils.calcWebBundleHash(payload);

      expect(Buffer.from(hash)).toEqual(expectedHash);
    });
  });

  describe('generateDataToBeSigned', () => {
    it('generates the correct payload for signing', () => {
      const hash = new Uint8Array([0xaa, 0xbb]);
      const ib = new Uint8Array([0x01, 0x02, 0x03]);
      const attr = new Uint8Array([0x04]);

      const data = utils.generateDataToBeSigned(hash, ib, attr);

      // We expect 3 lengths (8 bytes each) followed by the actual data
      const expectedLength = 8 + 2 + 8 + 3 + 8 + 1; // 30 bytes
      expect(data.length).toBe(expectedLength);

      const view = new DataView(data.buffer);
      // First part
      expect(Number(view.getBigInt64(0, false))).toBe(2);
      expect(data.slice(8, 10)).toEqual(hash);

      // Second part
      expect(Number(view.getBigInt64(10, false))).toBe(3);
      expect(data.slice(18, 21)).toEqual(ib);

      // Third part
      expect(Number(view.getBigInt64(21, false))).toBe(1);
      expect(data.slice(29, 30)).toEqual(attr);
    });
  });
});
