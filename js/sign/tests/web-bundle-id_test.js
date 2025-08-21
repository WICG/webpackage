import path from 'path';
import { fileURLToPath } from 'url';
import * as fs from 'node:fs/promises';
import * as wbnSign from '../lib/wbn-sign.js';
import * as crypto from 'crypto';
import { getBundleId } from '../lib/web-bundle-id.js';

const TEST_ED25519_PRIVATE_KEY = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIB8nP5PpWU7HiILHSfh5PYzb5GAcIfHZ+bw6tcd/LZXh
-----END PRIVATE KEY-----`;
const EXPECTED_BUNDLE_ID = '4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

describe('Obtaining Bundle ID from signed web bundle', () => {
  it('Signs the .wbn file and obtains Web Bundle ID from the newly created .swbn file', async () => {
    const filePath = path.resolve(__dirname, 'testdata', 'unsigned.wbn');
    const unsignedWebBundle = await fs.readFile(filePath);

    const ed25519PrivateKey = wbnSign.parsePemKey(TEST_ED25519_PRIVATE_KEY);
    const publicKey = crypto.createPublicKey(ed25519PrivateKey);

    const signer = new wbnSign.IntegrityBlockSigner(
      unsignedWebBundle, 
      new wbnSign.WebBundleId(publicKey).serialize(),
      [new wbnSign.NodeCryptoSigningStrategy(ed25519PrivateKey)]
    );

    const { signedWebBundle } = await signer.sign();

    const bundleId = getBundleId(signedWebBundle);

    expect(bundleId).toEqual(EXPECTED_BUNDLE_ID);
  });
});
