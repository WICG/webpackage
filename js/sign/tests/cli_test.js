import * as cli from '../lib/cli.js';
import * as mockStdin from 'mock-stdin';

const TEST_UNENCRYPTED_PRIVATE_KEY = Buffer.from(
  '-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEIB8nP5PpWU7HiILHSfh5PYzb5GAcIfHZ+bw6tcd/LZXh\n-----END PRIVATE KEY-----'
);
const TEST_ENCRYPTED_PRIVATE_KEY = Buffer.from(
  '-----BEGIN ENCRYPTED PRIVATE KEY-----\nMIGbMFcGCSqGSIb3DQEFDTBKMCkGCSqGSIb3DQEFDDAcBAhOw2E7LxOkzQICCAAw\nDAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEEJZqH2axMFEvdFmJLZlnch4EQMfJ\nAa/4uAmWqu2N5aOn2yIz3Ri+vQ/rzBPrvIaoDxYUUxwujJFujSbr3lnagHlOPptU\n7XhjbPbeOqidLqyv5rA=\n-----END ENCRYPTED PRIVATE KEY-----'
);
const PASSPHRASE = 'helloworld';

// Helper method to test async errors to be thrown, as Jasmine doesn't have
// support for such matcher.
async function expectToThrowErrorAsync(f) {
  if (typeof f !== 'function') {
    throw new Error(
      `expectToThrowErrorAsync: Type of parameter "f" is not "function" but "${typeof f}" instead.`
    );
  }

  let errorMessage = '';
  try {
    await f();
  } catch (error) {
    errorMessage = error.message;
    console.log(errorMessage);
  }

  expect(errorMessage).not.toBe('');
}

describe('CLI key parsing', () => {
  afterEach(() => {
    if (process.env.WEB_BUNDLE_SIGNING_PASSPHRASE) {
      delete process.env.WEB_BUNDLE_SIGNING_PASSPHRASE;
    }
  });

  it('works for unencrypted key.', async () => {
    const key = await cli.parseMaybeEncryptedKey(TEST_UNENCRYPTED_PRIVATE_KEY);
    expect(key.type).toEqual('private');
  });

  it('works for encrypted key read from `WEB_BUNDLE_SIGNING_PASSPHRASE`.', async () => {
    process.env.WEB_BUNDLE_SIGNING_PASSPHRASE = PASSPHRASE;
    const key = await cli.parseMaybeEncryptedKey(TEST_ENCRYPTED_PRIVATE_KEY);
    expect(key.type).toEqual('private');
  });

  it('works for encrypted key read from a prompt.', async () => {
    const stdin = mockStdin.stdin();
    const keyPromise = cli.parseMaybeEncryptedKey(TEST_ENCRYPTED_PRIVATE_KEY);
    stdin.send(`${PASSPHRASE}\n`);
    expect((await keyPromise).type).toEqual('private');
  });

  it('fails for faulty `WEB_BUNDLE_SIGNING_PASSPHRASE`.', async () => {
    process.env.WEB_BUNDLE_SIGNING_PASSPHRASE = 'helloworld1';

    await expectToThrowErrorAsync(() =>
      cli.parseMaybeEncryptedKey(TEST_ENCRYPTED_PRIVATE_KEY)
    );
  });

  it('fails for faulty passphrase read from a prompt.', async () => {
    process.env.WEB_BUNDLE_SIGNING_PASSPHRASE = 'helloworld1';

    await expectToThrowErrorAsync(async () => {
      const stdin = mockStdin.stdin();
      const keyPromise = cli.parseMaybeEncryptedKey(TEST_ENCRYPTED_PRIVATE_KEY);
      stdin.send(`${PASSPHRASE}1\n`);
      await keyPromise;
    });
  });
});
