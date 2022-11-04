import commander from 'commander';
import { IntegrityBlockSigner, parseStringKey } from './integrityblock.js';
import * as fs from 'fs';
import crypto, { KeyObject } from 'crypto';

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

export function main() {
  const options = readOptions();
  const webBundle = fs.readFileSync(options.input);

  const signer = new IntegrityBlockSigner(webBundle, {
    key: parseStringKey(fs.readFileSync(options.privateKey, 'utf-8')),
  });
  const integrityBlock = signer.sign();

  // TODO: Prepend integrity block.
  // fs.writeFileSync(options.output, webBundle);
}
