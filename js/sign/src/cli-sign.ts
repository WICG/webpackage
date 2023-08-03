import commander from 'commander';
import {
  NodeCryptoSigningStrategy,
  IntegrityBlockSigner,
  WebBundleId,
} from './wbn-sign.js';
import * as fs from 'fs';
import { greenConsoleLog, parseMaybeEncryptedKey } from './utils/cli-utils.js';
import { KeyObject } from 'crypto';

const program = new commander.Command()
  .name('wbn-sign')
  .description(
    'A simple CLI tool to sign the given web bundle with the given private key.'
  );

function readOptions() {
  return program
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
      /*defaultValue=*/ 'signed.swbn'
    )
    .parse(process.argv);
}

export async function main() {
  const options = readOptions();
  const webBundle = fs.readFileSync(options.input);
  const parsedPrivateKey: KeyObject = await parseMaybeEncryptedKey(
    fs.readFileSync(options.privateKey)
  );
  const signer = new IntegrityBlockSigner(
    webBundle,
    new NodeCryptoSigningStrategy(parsedPrivateKey)
  );
  const { signedWebBundle } = await signer.sign();
  greenConsoleLog(`${new WebBundleId(parsedPrivateKey)}`);
  fs.writeFileSync(options.output, signedWebBundle);
}
