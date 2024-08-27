import { Command, Option } from 'commander';
import {
  NodeCryptoSigningStrategy,
  IntegrityBlockSigner,
  WebBundleId,
} from './wbn-sign.js';
import * as fs from 'fs';
import {
  greenConsoleLog,
  parseMaybeEncryptedKeyFromFile,
} from './utils/cli-utils.js';
import { KeyObject } from 'crypto';

const program = new Command()
  .name('wbn-sign')
  .description(
    'A simple CLI tool to sign the given web bundle with the given private key.'
  );

function readOptions() {
  return program
    .addOption(
      new Option('--version <version>').choices(['v1', 'v2']).default('v2')
    )
    .requiredOption(
      '-i, --input <file>',
      'input web bundle to be signed (required)'
    )
    .requiredOption(
      '-k, --private-key <file...>',
      'paths to Ed25519 / ECDSA P-256 private key(s) (required)'
    )
    .option(
      '-o, --output <file>',
      'signed web bundle output file',
      /*defaultValue=*/ 'signed.swbn'
    )
    .option('--web-bundle-id <web-bundle-id>', 'web bundle ID (only for v2)')
    .action((options) => {
      switch (options.version) {
        case 'v1':
          {
            if (options.privateKey.length > 1) {
              throw new Error(
                `It's not allowed to specify more than one private key for v1 signing.`
              );
            }
            if (options.webBundleId) {
              throw new Error(
                `It's not allowed to specify --web-bundle-id for v1 signing.`
              );
            }
          }
          break;
        case 'v2':
          {
            if (options.privateKey.length > 1 && !options.webBundleId) {
              throw new Error(
                `--web-bundle-id must be specified if there's more than 1 signing key involved.`
              );
            }
          }
          break;
      }
    })
    .parse(process.argv)
    .opts();
}

export async function main() {
  const options = readOptions();
  const webBundle = fs.readFileSync(options.input);

  const privateKeys = new Array<KeyObject>();
  for (const privateKey of options.privateKey) {
    privateKeys.push(await parseMaybeEncryptedKeyFromFile(privateKey));
  }

  const webBundleId = options.webBundleId
    ? options.webBundleId
    : new WebBundleId(privateKeys[0]).serialize();
  const signer = new IntegrityBlockSigner(
    webBundle,
    webBundleId,
    privateKeys.map((privateKey) => new NodeCryptoSigningStrategy(privateKey)),
    /*is_v2=*/ options.version === 'v2'
  );
  const { signedWebBundle } = await signer.sign();
  greenConsoleLog(`${webBundleId}`);
  fs.writeFileSync(options.output, signedWebBundle);
}
