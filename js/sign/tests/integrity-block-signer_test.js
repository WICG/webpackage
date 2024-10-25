import * as wbnSign from '../lib/wbn-sign.js';
import * as constants from '../lib/utils/constants.js';
import * as utils from '../lib/utils/utils.js';
import * as fs from 'fs';
import * as path from 'path';
import * as crypto from 'crypto';
import * as cborg from 'cborg';
import url from 'url';

const __dirname = path.dirname(url.fileURLToPath(import.meta.url));
const TEST_WEB_BUNDLE_HASH =
  '95f8713d382ffefb8f1e4f464e39a2bf18280c8b26434d2fcfc08d7d710c8919ace5a652e25e66f9292cda424f20e4b53bf613bf9488140272f56a455393f7e6';
const EMPTY_INTEGRITY_BLOCK_HEX = '8448f09f968bf09f93a64432620000a080';
const TEST_ED25519_PRIVATE_KEY = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIB8nP5PpWU7HiILHSfh5PYzb5GAcIfHZ+bw6tcd/LZXh
-----END PRIVATE KEY-----`;
const TEST_ECDSA_P256_PRIVATE_KEY = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIG6HAXvoG+dOP20rbyPuGC21od4DAZCKBkPy/1902xPnoAoGCCqGSM49
AwEHoUQDQgAEHIIHO9B+7XJoXTXf3aTWC7aoK1PW4Db5Z8gSGXIkHlLrucUI4lyx
DttYYhi36vrg5nR6zrfdhe7+8F1MoTvLuw==
-----END EC PRIVATE KEY-----`;
const TEST_ED25519_WEB_BUNDLE_ID =
  '4tkrnsmftl4ggvvdkfth3piainqragus2qbhf7rlz2a3wo3rh4wqaaic';
const TEST_ECDSA_P256_WEB_BUNDLE_ID =
  'amoiebz32b7o24tilu257xne2yf3nkblkploanxzm7ebeglseqpfeaacai';

const IWA_SCHEME = 'isolated-app://';

describe('Web Bundle ID', () => {
  const ed25519PrivateKey = wbnSign.parsePemKey(TEST_ED25519_PRIVATE_KEY);
  const ecdsaP256PrivateKey = wbnSign.parsePemKey(TEST_ECDSA_P256_PRIVATE_KEY);
  const testKeys = [
    ed25519PrivateKey,
    crypto.createPublicKey(ed25519PrivateKey),
    ecdsaP256PrivateKey,
    crypto.createPublicKey(ecdsaP256PrivateKey),
  ];

  testKeys.forEach((key, index) => {
    it(`calculates the ID and isolated web app origin correctly with key #${index}.`, () => {
      const expectedWebBundleId = (() => {
        switch (utils.getSignatureType(key)) {
          case constants.SignatureType.Ed25519:
            return TEST_ED25519_WEB_BUNDLE_ID;
          case constants.SignatureType.EcdsaP256SHA256:
            return TEST_ECDSA_P256_WEB_BUNDLE_ID;
        }
      })();
      expect(expectedWebBundleId).toEqual(
        new wbnSign.WebBundleId(key).serialize()
      );
      expect(`${IWA_SCHEME}${expectedWebBundleId}/`).toEqual(
        new wbnSign.WebBundleId(key).serializeWithIsolatedWebAppOrigin()
      );
    });
  });
});

describe('Integrity Block Signer', () => {
  function initSignerWithTestWebBundleAndKeys(
    privateKeys,
    webBundleId = undefined
  ) {
    const file = path.resolve(__dirname, 'testdata/unsigned.wbn');
    const contents = fs.readFileSync(file);
    const signer = new wbnSign.IntegrityBlockSigner(
      contents,
      /*webBundleId=*/ webBundleId ||
        new wbnSign.WebBundleId(privateKeys[0]).serialize(),
      privateKeys.map(
        (privateKey) => new wbnSign.NodeCryptoSigningStrategy(privateKey)
      )
    );
    return signer;
  }

  function createTestSuffix(publicKey) {
    return constants.SignatureType[utils.getSignatureType(publicKey)];
  }

  it('accepts only selected key types.', () => {
    for (const validKey of [
      { keyType: 'ed25519' },
      { keyType: 'ec', options: { namedCurve: 'prime256v1' } },
    ]) {
      const keypairValid = crypto.generateKeyPairSync(
        validKey.keyType,
        validKey.options
      );
      expect(() =>
        initSignerWithTestWebBundleAndKeys([keypairValid.privateKey])
      ).not.toThrowError();
    }

    for (const invalidKey of [
      { keyType: 'rsa', options: { modulusLength: 2048 } },
      { keyType: 'dsa', options: { modulusLength: 1024, divisorLength: 224 } },
      { keyType: 'ec', options: { namedCurve: 'secp256k1' } },
      { keyType: 'ed448' },
      { keyType: 'x25519' },
      { keyType: 'x448' },
    ]) {
      const keypairInvalid = crypto.generateKeyPairSync(
        invalidKey.keyType,
        invalidKey.options
      );
      expect(() =>
        initSignerWithTestWebBundleAndKeys([keypairInvalid.privateKey])
      ).toThrowError();
    }
  });

  it('encodes an empty integrity block CBOR correctly.', () => {
    const integrityBlock = new wbnSign.IntegrityBlock();
    const cbor = integrityBlock.toCBOR();

    expect(cbor).toEqual(
      Uint8Array.from(Buffer.from(EMPTY_INTEGRITY_BLOCK_HEX, 'hex'))
    );

    const decoded = cborg.decode(cbor);
    expect(decoded.length).toEqual(4);
    expect(decoded[0]).toEqual(constants.INTEGRITY_BLOCK_MAGIC);
    expect(decoded[1]).toEqual(constants.VERSION_B2);
    expect(decoded[2]).toEqual({});
    expect(decoded[3]).toEqual([]);
  });

  it('calculates the hash of the web bundle correctly.', () => {
    const keypair = crypto.generateKeyPairSync('ed25519');
    const signer = initSignerWithTestWebBundleAndKeys([keypair.privateKey]);

    expect(signer.calcWebBundleHash()).toEqual(
      Uint8Array.from(Buffer.from(TEST_WEB_BUNDLE_HASH, 'hex'))
    );
  });

  [
    crypto.generateKeyPairSync('ed25519'),
    crypto.generateKeyPairSync('ec', { namedCurve: 'prime256v1' }),
  ].forEach((keypair) => {
    it(`generates the dataToBeSigned correctly with ${createTestSuffix(
      keypair.publicKey
    )}.`, () => {
      const signer = initSignerWithTestWebBundleAndKeys([keypair.privateKey]);
      const rawPubKey = wbnSign.getRawPublicKey(keypair.publicKey);

      const dataToBeSigned = signer.generateDataToBeSigned(
        signer.calcWebBundleHash(),
        new wbnSign.IntegrityBlock().toCBOR(),
        cborg.encode({
          [utils.getPublicKeyAttributeName(keypair.publicKey)]: rawPubKey,
        })
      );

      const attributesCborHex = (() => {
        switch (utils.getSignatureType(keypair.publicKey)) {
          case constants.SignatureType.Ed25519:
            return (
              /*52*/ '0000000000000034' +
              'a170656432353531395075626c69634b65795820' +
              Buffer.from(rawPubKey).toString('hex')
            );
          case constants.SignatureType.EcdsaP256SHA256:
            return (
              /*62*/ '000000000000003e' +
              'a178186563647361503235365348413235365075626c69634b65795821' +
              Buffer.from(rawPubKey).toString('hex')
            );
        }
      })();

      const hexHashString =
        /*64*/ '0000000000000040' +
        TEST_WEB_BUNDLE_HASH +
        /*17*/ '0000000000000011' +
        EMPTY_INTEGRITY_BLOCK_HEX +
        attributesCborHex;

      expect(dataToBeSigned).toEqual(
        Uint8Array.from(Buffer.from(hexHashString, 'hex'))
      );
    });
  });

  [
    crypto.generateKeyPairSync('ed25519'),
    crypto.generateKeyPairSync('ec', { namedCurve: 'prime256v1' }),
  ].forEach((keypair) => {
    it(`generates a valid signature with ${createTestSuffix(
      keypair.publicKey
    )}.`, async () => {
      const signer = initSignerWithTestWebBundleAndKeys([keypair.privateKey]);
      const rawPubKey = wbnSign.getRawPublicKey(keypair.publicKey);
      const sigAttr = {
        [utils.getPublicKeyAttributeName(keypair.publicKey)]: rawPubKey,
      };
      const webBundleId = new wbnSign.WebBundleId(
        keypair.privateKey
      ).serialize();

      const ibWithoutSignatures = new wbnSign.IntegrityBlock();
      ibWithoutSignatures.setWebBundleId(webBundleId);

      const dataToBeSigned = signer.generateDataToBeSigned(
        signer.calcWebBundleHash(),
        ibWithoutSignatures.toCBOR(),
        cborg.encode(sigAttr)
      );

      const ib = cborg.decode((await signer.sign()).integrityBlock);
      expect(ib.length).toEqual(4);

      const [magic, version, attributes, signatureStack] = ib;

      expect(magic).toEqual(constants.INTEGRITY_BLOCK_MAGIC);
      expect(version).toEqual(constants.VERSION_B2);
      expect(attributes).toEqual({
        webBundleId: webBundleId,
      });
      expect(signatureStack.length).toEqual(1);
      expect(signatureStack[0].length).toEqual(2);

      const [signatureAttributes, signature] = signatureStack[0];
      expect(signatureAttributes).toEqual(sigAttr);

      // For ECDSA P-256 keys the algorithm is implicitly selected as SHA-256.
      expect(
        crypto.verify(
          /*algorithm=*/ undefined,
          dataToBeSigned,
          keypair.publicKey,
          signature
        )
      ).toBeTruthy();
    });
  });

  it(`generates valid signatures with multiple keys`, async () => {
    const keyPairs = [
      crypto.generateKeyPairSync('ed25519'),
      crypto.generateKeyPairSync('ec', { namedCurve: 'prime256v1' }),
      crypto.generateKeyPairSync('ed25519'),
      crypto.generateKeyPairSync('ec', { namedCurve: 'prime256v1' }),
    ];

    const webBundleId = 'aaa';

    const signer = initSignerWithTestWebBundleAndKeys(
      keyPairs.map((keyPair) => keyPair.privateKey),
      webBundleId
    );

    const ibData = (await signer.sign()).integrityBlock;
    const ib = cborg.decode(ibData);
    expect(ib.length).toEqual(4);

    const [magic, version, ibAttributes, signatureStack] = ib;

    expect(magic).toEqual(constants.INTEGRITY_BLOCK_MAGIC);
    expect(version).toEqual(constants.VERSION_B2);
    expect(Object.keys(ibAttributes).length).toEqual(1);
    expect(ibAttributes.webBundleId).toEqual(webBundleId);
    expect(signatureStack.length).toEqual(keyPairs.length);
    signatureStack.map((entry) => expect(entry.length).toEqual(2));

    const ibWithoutSignatures = new wbnSign.IntegrityBlock();
    ibWithoutSignatures.setWebBundleId(webBundleId);

    for (let i = 0; i < keyPairs.length; i++) {
      const publicKey = keyPairs[i].publicKey;

      const attrName = utils.getPublicKeyAttributeName(publicKey);
      const rawPubKey = wbnSign.getRawPublicKey(publicKey);

      const [signatureAttributes, signature] = signatureStack[i];
      expect(Object.keys(signatureAttributes).length).toEqual(1);
      expect(signatureAttributes[attrName]).toEqual(rawPubKey);

      const dataToBeSigned = signer.generateDataToBeSigned(
        signer.calcWebBundleHash(),
        ibWithoutSignatures.toCBOR(),
        cborg.encode(signatureAttributes)
      );
      // For ECDSA P-256 keys the algorithm is implicitly selected as SHA-256.
      expect(
        crypto.verify(
          /*algorithm=*/ undefined,
          dataToBeSigned,
          publicKey,
          signature
        )
      ).toBeTruthy();
    }
  });
});
