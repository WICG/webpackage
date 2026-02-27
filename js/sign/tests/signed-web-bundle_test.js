// @ts-check
/** @typedef {import('jasmine')} _ */
import fs from 'fs';
import path from 'path';
import url from 'url';

import {
  getRawPublicKey,
  NodeCryptoSigningStrategy,
  parsePemKey,
  SignedWebBundle,
} from '../lib/wbn-sign.js';

const __dirname = path.dirname(url.fileURLToPath(import.meta.url));
const TEST_ED25519_PRIVATE_KEY = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIB8nP5PpWU7HiILHSfh5PYzb5GAcIfHZ+bw6tcd/LZXh
-----END PRIVATE KEY-----`;
const TEST_ED25519_PRIVATE_KEY_2 = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIKxzyXkSRaUIc6fpI+TYecPQjo4YJTSFPulQY/0lGjs1
-----END PRIVATE KEY-----`;

const TEST_ECDSA_P256_WEB_BUNDLE_ID =
  'amoiebz32b7o24tilu257xne2yf3nkblkploanxzm7ebeglseqpfeaacai';
const TEST_ED25519_WEB_BUNDLE_ID_1 =
  '4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic';

const [STRATEGY_KEY_1, STRATEGY_KEY_2] = [
  TEST_ED25519_PRIVATE_KEY,
  TEST_ED25519_PRIVATE_KEY_2,
].map((key) => new NodeCryptoSigningStrategy(parsePemKey(key)));

const UNSIGNED_BUNDLE_PATH = path.resolve(__dirname, 'testdata/unsigned.wbn');
const UNSIGNED_WEB_BUNDLE_BYTES = Uint8Array.from(
  fs.readFileSync(UNSIGNED_BUNDLE_PATH)
);

describe('Signed Web Bundle - ', function () {
  it('fromWebBundle() - bundle id got from first key by default', async function () {
    const double_signed_bundle = await SignedWebBundle.fromWebBundle(
      UNSIGNED_WEB_BUNDLE_BYTES,
      [STRATEGY_KEY_1, STRATEGY_KEY_2]
    );
    expect(double_signed_bundle.getWebBundleId()).toEqual(
      TEST_ED25519_WEB_BUNDLE_ID_1
    );
  });

  it('fromWebBundle() - bundle id successfully overridden', async function () {
    const double_signed_bundle = await SignedWebBundle.fromWebBundle(
      UNSIGNED_WEB_BUNDLE_BYTES,
      [STRATEGY_KEY_1, STRATEGY_KEY_2],
      { webBundleId: TEST_ECDSA_P256_WEB_BUNDLE_ID }
    );
    expect(double_signed_bundle.getWebBundleId()).toEqual(
      TEST_ECDSA_P256_WEB_BUNDLE_ID
    );
  });

  it('addSignature() - the same result as signing with two keys', async function () {
    // I'm using only ed25519 keys on purpose: Ecdsa signing algorithm is not deterministic and adds a random nonce
    // so I could not just simply verify if two integrity blocks are same after different sequence of operations
    const bundle_signed_by_first_key_then_other = await (
      await SignedWebBundle.fromWebBundle(UNSIGNED_WEB_BUNDLE_BYTES, [
        STRATEGY_KEY_1,
      ])
    ).addSignature(STRATEGY_KEY_2);
    const bundle_signed_by_two_keys_at_once =
      await SignedWebBundle.fromWebBundle(UNSIGNED_WEB_BUNDLE_BYTES, [
        STRATEGY_KEY_1,
        STRATEGY_KEY_2,
      ]);
    const bundle_signed_by_second_key_then_other = await (
      await SignedWebBundle.fromWebBundle(UNSIGNED_WEB_BUNDLE_BYTES, [
        STRATEGY_KEY_2,
      ])
    ).addSignature(STRATEGY_KEY_1);
    const bundle_signed_by_two_keys_at_once_reversed_order =
      await SignedWebBundle.fromWebBundle(UNSIGNED_WEB_BUNDLE_BYTES, [
        STRATEGY_KEY_2,
        STRATEGY_KEY_1,
      ]);

    expect(bundle_signed_by_first_key_then_other).toEqual(
      bundle_signed_by_two_keys_at_once
    );
    expect(bundle_signed_by_second_key_then_other).toEqual(
      bundle_signed_by_two_keys_at_once_reversed_order
    );
    expect(bundle_signed_by_first_key_then_other).not.toEqual(
      bundle_signed_by_second_key_then_other
    );
  });

  it('removeSignature() - bundle signed with key A and B + removing A = bundle singed with key B', async function () {
    const bundle_signed_by_second_key = await SignedWebBundle.fromWebBundle(
      UNSIGNED_WEB_BUNDLE_BYTES,
      [STRATEGY_KEY_2],
      { webBundleId: TEST_ED25519_WEB_BUNDLE_ID_1 }
    );
    const bundle_signed_by_two_keys_second_removed = (
      await SignedWebBundle.fromWebBundle(UNSIGNED_WEB_BUNDLE_BYTES, [
        STRATEGY_KEY_1,
        STRATEGY_KEY_2,
      ])
    ).removeSignature(getRawPublicKey(await STRATEGY_KEY_1.getPublicKey()));
    expect(bundle_signed_by_two_keys_second_removed).toEqual(
      bundle_signed_by_second_key
    );
  });
});
