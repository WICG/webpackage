import { Command } from 'commander';
import { WebBundleId } from './wbn-sign.js';
import * as fs from 'fs';
import {
  greenConsoleLog,
  parseMaybeEncryptedKeyFromFile,
} from './utils/cli-utils.js';
import { KeyObject } from 'crypto';

const program = new Command()
  .name('wbn-dump-id')
  .description(
    'A simple CLI tool to dump the Web Bundle ID matching to the given private key.'
  );

function readOptions() {
  return program
    .requiredOption(
      '-k, --private-key <file>',
      'Reads an Ed25519 / ECDSA P-256 private key from the given path. (required)'
    )
    .option(
      '-s, --with-iwa-scheme',
      'Dumps the Web Bundle ID with isolated-app:// scheme. By default it only dumps the ID. (optional)',
      /*defaultValue=*/ false
    )
    .parse(process.argv)
    .opts();
}

export async function main() {
  const options = readOptions();
  const parsedPrivateKey: KeyObject = await parseMaybeEncryptedKeyFromFile(
    options.privateKey
  );

  const webBundleId: string = options.withIwaScheme
    ? new WebBundleId(parsedPrivateKey).serializeWithIsolatedWebAppOrigin()
    : new WebBundleId(parsedPrivateKey).serialize();

  greenConsoleLog(webBundleId);
}
