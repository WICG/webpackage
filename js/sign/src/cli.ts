import commander from 'commander';
import {
  IntegrityBlockSigner,
  WebBundleId,
  parsePemKey,
} from './integrityblock.js';
import * as fs from 'fs';

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
  const parsedPrivateKey = parsePemKey(
    fs.readFileSync(options.privateKey, 'utf-8')
  );
  const signer = new IntegrityBlockSigner(webBundle, {
    key: parsedPrivateKey,
  });
  const { signedWebBundle } = signer.sign();

  const consoleLogColor = { green: '\x1b[32m', reset: '\x1b[0m' };
  console.log(
    `${consoleLogColor.green}${new WebBundleId(parsedPrivateKey)}${
      consoleLogColor.reset
    }`
  );

  fs.writeFileSync(options.output, signedWebBundle);
}
