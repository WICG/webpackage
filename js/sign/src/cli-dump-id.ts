import commander from 'commander';
import { WebBundleId } from './wbn-sign.js';
import * as fs from 'fs';
import { greenConsoleLog, parseMaybeEncryptedKey } from './utils/cli-utils.js';
import { KeyObject } from 'crypto';

const program = new commander.Command()
  .name('wbn-dump-id')
  .description(
    'A simple CLI tool to dump the Web Bundle ID matching to the given private key.'
  );

function readOptions() {
  return program
    .requiredOption(
      '-k, --privateKey <file>',
      'Reads an ed25519 private key from the given path. (required)'
    )
    .option(
      '-s, --withIwaScheme',
      'Dumps the Web Bundle ID with isolated-app:// scheme. By default it only dumps the ID. (optional)',
      /*defaultValue=*/ false
    )
    .parse(process.argv);
}

export async function main() {
  const options = readOptions();
  const parsedPrivateKey: KeyObject = await parseMaybeEncryptedKey(
    fs.readFileSync(options.privateKey)
  );

  const webBundleId: string = options.withIwaScheme
    ? new WebBundleId(parsedPrivateKey).serializeWithIsolatedWebAppOrigin()
    : new WebBundleId(parsedPrivateKey).serialize();

  greenConsoleLog(webBundleId);
}
