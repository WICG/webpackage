import { Command } from 'commander';
import { WebBundleId } from './wbn-sign.js';
import * as fs from 'fs';
import {
  greenConsoleLog,
  parseMaybeEncryptedKeyFromFile,
} from './utils/cli-utils.js';
import { KeyObject, createPublicKey } from 'crypto';

const program = new Command()
  .name('wbn-dump-id')
  .description(
    'A simple CLI tool to dump the Web Bundle ID corresponding to the given key.'
  );

function parsePublicKey(filePath: string): KeyObject {
  return createPublicKey(fs.readFileSync(filePath));
}

async function parseKey(filePath: string): Promise<KeyObject> {
  try {
    return parsePublicKey(filePath);
  } catch (err) {
    // Suppress this error.
  }

  return await parseMaybeEncryptedKeyFromFile(filePath);
}

function readOptions() {
  return program
    .requiredOption(
      '-k, --key <file>',
      'Reads an Ed25519 / ECDSA P-256 private / public key from the given path.'
    )
    .option(
      '-s, --with-iwa-scheme',
      'Dumps the Web Bundle ID with isolated-app:// scheme. By default it only dumps the ID. (optional)',
      /*defaultValue=*/ false
    )
    .option(
      '-t, --with-key-type',
      'Dumps the key type (optional)',
      /*defaultValue=*/ false
    )
    .parse(process.argv)
    .opts();
}

export async function main() {
  const options = readOptions();

  const parsedKey: KeyObject = await parseKey(options.key);
  const webBundleId: WebBundleId = new WebBundleId(parsedKey);
  if (options.withKeyType) {
    greenConsoleLog(webBundleId.getKeyTypeName());
  }
  greenConsoleLog(
    options.withIwaScheme
      ? webBundleId.serializeWithIsolatedWebAppOrigin()
      : webBundleId.serialize()
  );
}
