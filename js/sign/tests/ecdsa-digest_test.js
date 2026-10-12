import crypto from 'crypto';
import fs from 'fs';
import path from 'path';
import url from 'url';

import { describeEachBuild } from './helpers/builds.js';

const __dirname = path.dirname(url.fileURLToPath(import.meta.url));

const UNSIGNED_WEB_BUNDLE = Uint8Array.from(
  fs.readFileSync(path.resolve(__dirname, 'testdata/unsigned.wbn'))
);

const KEY_PAIR_GENERATORS = {
  'ECDSA P-256': () =>
    crypto.generateKeyPairSync('ec', { namedCurve: 'prime256v1' }),
  Ed25519: () => crypto.generateKeyPairSync('ed25519'),
};

// Node.js builds that use BoringSSL instead of OpenSSL have no default digest
// for EC keys, so `crypto.sign()` and `crypto.verify()` throw for ECDSA keys if
// no digest is given. Emulate that to make sure that it is always passed.
function rejectMissingDigestForEcKey(algorithm, key) {
  if (algorithm == null && key.asymmetricKeyType === 'ec') {
    throw new Error('No default digest for EC keys (emulated BoringSSL)');
  }
}

describeEachBuild('No default digest for EC keys - ', function (wbnSign) {
  const { NodeCryptoSigningStrategy, SignedWebBundle } = wbnSign;

  beforeEach(function () {
    const { sign, verify } = crypto;
    spyOn(crypto, 'sign').and.callFake((algorithm, data, key) => {
      rejectMissingDigestForEcKey(algorithm, key);
      return sign(algorithm, data, key);
    });
    spyOn(crypto, 'verify').and.callFake((algorithm, data, key, signature) => {
      rejectMissingDigestForEcKey(algorithm, key);
      return verify(algorithm, data, key, signature);
    });
  });

  for (const [keyName, generateKeyPair] of Object.entries(
    KEY_PAIR_GENERATORS
  )) {
    it(`signs, adds a signature and validates with ${keyName} keys`, async function () {
      const [firstStrategy, secondStrategy] = [
        generateKeyPair(),
        generateKeyPair(),
      ].map(({ privateKey }) => new NodeCryptoSigningStrategy(privateKey));

      const signedWebBundle = await (
        await SignedWebBundle.fromWebBundle(UNSIGNED_WEB_BUNDLE, [
          firstStrategy,
        ])
      ).addSignature(secondStrategy);

      expect(signedWebBundle.validateSignatures()).toEqual([
        jasmine.objectContaining({ status: 'success', isValid: true }),
        jasmine.objectContaining({ status: 'success', isValid: true }),
      ]);
    });
  }
});
