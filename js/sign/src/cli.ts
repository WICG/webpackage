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

async function parseMaybeEncryptedKey(
  privateKeyFile: Buffer
): Promise<KeyObject> {
  try {
    return parsePemKey(privateKeyFile);
  } catch (e) {
    console.warn("This might be an encrypted private key, let's try again.");
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
