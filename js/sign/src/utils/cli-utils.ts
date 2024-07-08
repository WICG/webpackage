import tty from 'tty';
import fs from 'fs';
import { KeyObject } from 'crypto';
import path from 'path';
import { parsePemKey, readPassphrase } from '../wbn-sign.js';

// Parses either an unencrypted or encrypted private key. For encrypted keys, it
// reads the passphrase to decrypt them from either the
// `WEB_BUNDLE_SIGNING_PASSPHRASE` environment variable, or, if not set, prompts
// the user for the passphrase.
export async function parseMaybeEncryptedKeyFromFile(
  filePath: string
): Promise<KeyObject> {
  return parseMaybeEncryptedKey(
    fs.readFileSync(filePath),
    path.basename(filePath)
  );
}

// Exported for testing.
export async function parseMaybeEncryptedKey(
  data: Buffer,
  description: string = ''
): Promise<KeyObject> {
  // Read unencrypted private key.
  try {
    return parsePemKey(data);
  } catch (e) {
    console.warn('This key is probably an encrypted private key.');
  }

  const hasEnvVarSet =
    process.env.WEB_BUNDLE_SIGNING_PASSPHRASE &&
    process.env.WEB_BUNDLE_SIGNING_PASSPHRASE !== '';

  // Read encrypted private key.
  try {
    return parsePemKey(
      data,
      hasEnvVarSet
        ? process.env.WEB_BUNDLE_SIGNING_PASSPHRASE
        : await readPassphrase(description)
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

export function greenConsoleLog(text: string): void {
  const logColor = { green: '\x1b[32m', reset: '\x1b[0m' };
  const fileDescriptor: number = process.stdout.fd;

  // If the log is used for non-terminal (fd != 1), e.g., setting an environment
  // variable, it shouldn't have any formatting.
  console.log(
    tty.isatty(fileDescriptor)
      ? `${logColor.green}${text}${logColor.reset}`
      : text
  );
}
