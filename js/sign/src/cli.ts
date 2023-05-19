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

// Parses either an unencrypted or encrypted private key. For unencrypted keys,
// reads the passphrase to decrypt an encrypted private key from either
// `WEB_BUNDLE_SIGNING_PASSPHRASE` env var or if not set, it prompts passphrase
// from the user.
async function parseMaybeEncryptedKey(
  privateKeyFile: Buffer
): Promise<KeyObject> {
  // If the env var is provided, that's most probably the right one so let's
  // check this before the unencrypted case.
  if (process.env.WEB_BUNDLE_SIGNING_PASSPHRASE !== '') {
    try {
      return parsePemKey(
        privateKeyFile,
        process.env.WEB_BUNDLE_SIGNING_PASSPHRASE
      );
    } catch (e) {
      console.warn(
        "Passphrase read from `WEB_BUNDLE_SIGNING_PASSPHRASE` environment variable doesn't match with the provided private key."
      );
    }
  }

  try {
    return parsePemKey(privateKeyFile);
  } catch (e) {
    console.warn('This key is probably an encrypted private key.');
  }

  return parsePemKey(privateKeyFile, await readPassphrase());
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
