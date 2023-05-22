import commander from 'commander';
import {
  NodeCryptoSigningStrategy,
  IntegrityBlockSigner,
  WebBundleId,
  parsePemKey,
  readPassphrase,
} from './wbn-sign.js';
import * as fs from 'fs';
import { KeyObject } from 'crypto';

function readOptions() {
  return commander
    .requiredOption(
      '-i, --input <file>',
      'input web bundle to be signed (required)'
    )
    .requiredOption(
      '-k, --privateKey <file>',
      'path to ed25519 private key (required)'
    )
    .option(
      '-o, --output <file>',
      'signed web bundle output file',
      'signed.wbn'
    )
    .parse(process.argv);
}

// Parses either an unencrypted or encrypted private key. For encrypted keys, it
// reads the passphrase to decrypt them from either the
// `WEB_BUNDLE_SIGNING_PASSPHRASE` environment variable, or, if not set, prompts
// the user for the passphrase.
export async function parseMaybeEncryptedKey(
  privateKeyFile: Buffer
): Promise<KeyObject> {
  // Read unencrypted private key.
  try {
    return parsePemKey(privateKeyFile);
  } catch (e) {
    console.warn('This key is probably an encrypted private key.');
  }

  const hasEnvVarSet =
    process.env.WEB_BUNDLE_SIGNING_PASSPHRASE &&
    process.env.WEB_BUNDLE_SIGNING_PASSPHRASE !== '';

  // Read encrypted private key.
  try {
    return parsePemKey(
      privateKeyFile,
      hasEnvVarSet
        ? process.env.WEB_BUNDLE_SIGNING_PASSPHRASE
        : await readPassphrase()
    );
  } catch (e) {
    throw Error(
      `Failed decrypting encrypted private key with passphrase read from ${
        hasEnvVarSet
          ? '`WEB_BUNDLE_SIGNING_PASSPHRASE` environment variable'
          : 'prompt'
      }`
    );
  }
}

export async function main() {
  const options = readOptions();
  const webBundle = fs.readFileSync(options.input);
  const parsedPrivateKey = await parseMaybeEncryptedKey(
    fs.readFileSync(options.privateKey)
  );
  const signer = new IntegrityBlockSigner(
    webBundle,
    new NodeCryptoSigningStrategy(parsedPrivateKey)
  );
  const { signedWebBundle } = await signer.sign();

  const consoleLogColor = { green: '\x1b[32m', reset: '\x1b[0m' };
  console.log(
    `${consoleLogColor.green}${new WebBundleId(parsedPrivateKey)}${
      consoleLogColor.reset
    }`
  );

  fs.writeFileSync(options.output, signedWebBundle);
}
