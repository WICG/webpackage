import * as wbnSign from '../lib/wbn-sign.js';
import * as fs from 'fs';
import * as path from 'path';
import * as crypto from 'crypto';
import url from 'url';
const __dirname = path.dirname(url.fileURLToPath(import.meta.url));

describe('Integrity Block', () => {
  function initSignerWithTestWebBundleAndKeys(keyType = 'ed25519') {
    const keypair = crypto.generateKeyPairSync(keyType);
    const file = path.resolve(__dirname, 'testdata/unsigned.wbn');
    const contents = fs.readFileSync(file);
    const signer = new wbnSign.IntegrityBlockSigner(contents, {
      key: keypair.privateKey,
    });
    return signer;
  }

  it('accepts only ed25519 type of key.', () => {
    const signer = initSignerWithTestWebBundleAndKeys();
    const integrityBlock = signer.sign();

    for (const keyType of ['rsa', 'dsa', 'ec', 'ed448', 'x25519', 'x448']) {
      expect(() => initSignerWithTestWebBundleAndKeys(keyType)).toThrowError();
    }
  });
});
