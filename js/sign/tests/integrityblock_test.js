import * as wbnSign from '../lib/wbn-sign.js';
import * as constants from '../lib/constants.js';
import * as fs from 'fs';
import * as path from 'path';
import * as crypto from 'crypto';
import * as cborg from 'cborg';
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

  it('encodes an empty integrity block CBOR correctly.', () => {
    const integrityBlock = new wbnSign.IntegrityBlock();
    const cbor = integrityBlock.toCBOR();

    expect(cbor).toEqual(
      Uint8Array.from(Buffer.from('8348F09F968BF09F93A6443162000080', 'hex'))
    );

    const decoded = cborg.decode(cbor);
    expect(decoded.length).toEqual(3);
    expect(decoded[0]).toEqual(constants.INTEGRITY_BLOCK_MAGIC);
    expect(decoded[1]).toEqual(constants.VERSION_B1);
    expect(decoded[2]).toEqual([]);
  });
});
