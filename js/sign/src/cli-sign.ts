import crypto, { KeyObject } from 'crypto';
import * as fs from 'fs';
import { createRequire } from 'module';

import { Command } from 'commander';

import {
  errorLog,
  greenConsoleLog,
  infoLog,
  parseMaybeEncryptedKey,
  parseMaybeEncryptedKeyFromFile,
  warnLog,
} from './utils/cli-utils.js';
import {
  getRawPublicKey,
  isPureWebBundle,
  isSignedWebBundle,
} from './utils/utils.js';
import { NodeCryptoSigningStrategy, SignedWebBundle } from './wbn-sign.js';

const require = createRequire(import.meta.url);
const { name, version } = require('../package.json');

// Get output file path depending on options: bundle path if in-place and provided path otherwise.
function getOutputPath(
  bundlePath: string,
  options: { inPlace?: boolean; output?: string }
) {
  if (options.inPlace && options.output != null) {
    throw new Error(
      "Options '--in-place' and '--output' are mutually exclusive."
    );
  }
  if (!options.inPlace && options.output == null) {
    throw new Error("One of '--in-place' and '--output' options must be used.");
  }
  if (options?.output && fs.existsSync(options.output)) {
    warnLog(
      `The file in output path ${options.output} already exists. Overwriting.`
    );
  }
  return options.output ?? bundlePath;
}

// That key may be either provided by Bytes encoded in base64 or path to file with key. This function checks that and parses key.
async function parseRemovalKey(keyInput: string): Promise<Uint8Array> {
  if (keyInput.endsWith('.pem') || fs.existsSync(keyInput)) {
    if (!fs.existsSync(keyInput)) {
      throw new Error(`The key file "${keyInput}" does not exist.`);
    }
    const data = await fs.promises.readFile(keyInput);
    try {
      // Try parsing as private key (it handles encrypted ones too)
      const privateKey = await parseMaybeEncryptedKey(data, keyInput);
      return getRawPublicKey(privateKey);
    } catch {
      // Try parsing as public key
      try {
        const pubKeyObject = crypto.createPublicKey(data);
        return getRawPublicKey(pubKeyObject);
      } catch (err) {
        throw new Error(
          `Failed to parse key from file "${keyInput}". Ensure it is a valid PEM-encoded Ed25519 or ECDSA P-256 key.`,
          { cause: err }
        );
      }
    }
  } else {
    // Assume Base64 string
    // Base64 regex check (simplified, but sufficient for raw keys)
    if (!/^[A-Za-z0-9+/=]+$/.test(keyInput)) {
      throw new Error(
        `The provided key "${keyInput}" is neither a valid file path nor a proper Base64-encoded string.`
      );
    }
    return new Uint8Array(Buffer.from(keyInput, 'base64'));
  }
}

const program = new Command()
  .name(name)
  .version(version, '-V, --version', 'Display version')
  .description(
    'CLI tool for managing signatures and keys for web bundles. Operates on `wbn` and `swbn` files.'
  )
  .helpOption('-h, --help', 'Display help');

async function parseArguments(): Promise<void> {
  program.commandsGroup('General commands');
  program.helpCommand('help [command]', 'Display help for command');

  program
    .command('info')
    .summary('Display integrity block information for a signed web bundle.')
    .description(
      'Display integrity block information for a signed web bundle, including the Web Bundle ID and signatures. \n\n' +
        'WARNING: Experimental. The output format is subject to change. Do not use in scripts or automation.'
    )
    .argument('<web_bundle>', 'Path to a signed web bundle `*.swbn`')
    .action(async (webBundlePath) => {
      const webBundle = await fs.promises.readFile(webBundlePath);
      const signedWebBundle = SignedWebBundle.fromBytes(webBundle);

      signedWebBundle.printInfo();
    });

  program.commandsGroup('Signature management commands');
  program
    .command('add-signature')
    .summary('Add signatures to an existing signed web bundle.')
    .description(
      'Add one or more signatures to the integrity block of a signed web bundle. ' +
        'This updates the bundle’s metadata with new signatures without altering ' +
        'the bundled resources.\n\n' +
        'For encrypted keys, you will be prompted for a password unless the ' +
        'WEB_BUNDLE_SIGNING_PASSPHRASE environment variable is set.'
    )
    .argument('<signed_web_bundle>', 'Path to a signed web bundle (*.swbn).')
    .argument(
      '<private_keys...>',
      '*.pem files containing ecdsaP256 or ed25519 private keys.'
    )
    .option(
      '-i, --in-place',
      'Overwrite the input file with the new signatures. Incompatible with --output.',
      false
    )
    .option(
      '-o, --output <file>',
      'Path for the new signed output file. Incompatible with --in-place.'
    )
    .action(async (webBundlePath, keyFilesPaths, options) => {
      const outputPath = getOutputPath(webBundlePath, options);

      const webBundle = await fs.promises.readFile(webBundlePath);
      const signedWebBundle = SignedWebBundle.fromBytes(webBundle);

      for (const keyPath of keyFilesPaths) {
        const privateKey = await parseMaybeEncryptedKeyFromFile(keyPath);
        await signedWebBundle.addSignature(
          new NodeCryptoSigningStrategy(privateKey)
        );
      }

      await fs.promises.writeFile(
        outputPath,
        signedWebBundle.getSignedWebBundleBytes()
      );
      greenConsoleLog(
        `Signature${keyFilesPaths.length > 1 ? 's' : ''} added successfully.`
      );
    });

  program
    .command('remove-signature')
    .summary('Remove signatures from a signed web bundle.')
    .description(
      'Remove one or more signatures from the integrity block of a signed web bundle. ' +
        'Signatures are identified by providing their associated public or private keys.\n\n' +
        'Tip: Use the `info` command to list the public keys currently present in a bundle.\n\n' +
        'For encrypted private keys, you will be prompted for a password unless ' +
        'the WEB_BUNDLE_SIGNING_PASSPHRASE environment variable is set.'
    )
    .argument('<signed_web_bundle>', 'Path to a signed web bundle (*.swbn).')
    .argument(
      '<keys...>',
      'Public keys (Base64 strings or .pem files) or private keys (.pem) ' +
        'used to identify signatures for removal.'
    )
    .option(
      '-i, --in-place',
      'Overwrite the input file. Incompatible with --output.',
      false
    )
    .option(
      '-o, --output <file>',
      'Path for the new signed output file. Incompatible with --in-place.'
    )
    .action(async (webBundlePath, keyInputs, options) => {
      const outputPath = getOutputPath(webBundlePath, options);

      const webBundle = await fs.promises.readFile(webBundlePath);
      const signedWebBundle = SignedWebBundle.fromBytes(webBundle);

      for (const keyInput of keyInputs) {
        const publicKey = await parseRemovalKey(keyInput);
        signedWebBundle.removeSignature(publicKey);
      }

      await fs.promises.writeFile(
        outputPath,
        signedWebBundle.getSignedWebBundleBytes()
      );
      greenConsoleLog(
        `Signature${keyInputs.length > 1 ? 's' : ''} removed successfully.`
      );
    });

  program
    .command('replace-signature')
    .summary('Replace an existing signature with a new one.')
    .description(
      'Replace a specific signature in the integrity block with a new signature. ' +
        'This is a convenience command equivalent to performing an `add-signature` ' +
        'followed by a `remove-signature`.\n\n' +
        'Tip: Use the `info` command to identify the public key of the signature you wish to replace.\n\n' +
        'For encrypted private keys, you will be prompted for a password unless ' +
        'the WEB_BUNDLE_SIGNING_PASSPHRASE environment variable is set.'
    )
    .argument('<signed_web_bundle>', 'Path to a signed web bundle (*.swbn).')
    .argument(
      '<old_key>',
      'The public key (Base64 or .pem) or private key (.pem) of the signature to replace.'
    )
    .argument(
      '<new_private_key>',
      'The new *.pem file (ecdsaP256 or ed25519) to sign the bundle with.'
    )
    .option(
      '-i, --in-place',
      'Overwrite the input file. Incompatible with --output.',
      false
    )
    .option(
      '-o, --output <file>',
      'Path for the new signed output file. Incompatible with --in-place.'
    )
    .action(async (webBundlePath, oldKeyInput, newKeyPath, options) => {
      const outputPath = getOutputPath(webBundlePath, options);

      const webBundle = await fs.promises.readFile(webBundlePath);
      const signedWebBundle = SignedWebBundle.fromBytes(webBundle);

      const oldPublicKey = await parseRemovalKey(oldKeyInput);
      const newPrivateKey = await parseMaybeEncryptedKeyFromFile(newKeyPath);

      await signedWebBundle.addSignature(
        new NodeCryptoSigningStrategy(newPrivateKey)
      );
      signedWebBundle.removeSignature(oldPublicKey);

      await fs.promises.writeFile(
        outputPath,
        signedWebBundle.getSignedWebBundleBytes()
      );
      greenConsoleLog('Signature replaced successfully.');
    });

  program
    .command('sign')
    .summary(
      'Sign a web bundle with private key(s). Produces signed web bundle.'
    )
    .description(
      `Sign a web bundle using one or more private keys to produce a signed web bundle (.swbn). Only .pem files are supported.`
    )
    .argument('<web_bundle>', 'The *.wbn file to sign')
    .argument(
      '<private_keys...>',
      '*.pem files containing ecdsaP256 or ed25519 private keys.' +
        'For encrypted keys, you will be prompted for a password' +
        'unless the WEB_BUNDLE_SIGNING_PASSPHRASE env var is set.'
    )
    .option(
      '-o, --output <file>',
      'Path for the signed web bundle output file',
      /*defaultValue=*/ 'signed.swbn'
    )
    .option(
      '--web-bundle-id <web-bundle-id>',
      'Web bundle ID. Derived from the first key if not specified.'
    )
    .showHelpAfterError()
    .action(async (webBundlePath, keyFilesPaths, options) => {
      if (!options.webBundleId) {
        infoLog(
          `The bundle id was not specified. It will be derived from the ${
            keyFilesPaths.length > 1 ? 'first ' : ''
          }given key.`
        );
      }
      if (fs.existsSync(options.output)) {
        warnLog(`'${options.output}' file already exists. Overwriting.`);
      }

      await readVerifyAndSignWebBundle(
        webBundlePath,
        keyFilesPaths,
        options.output,
        options.webBundleId
      );
    });

  program
    .command('sing', { hidden: true })
    .argument('[anything...]')
    .action(() => {
      greenConsoleLog('🎶 Never gonna let you down, lalala la lala... 🎶 \n');
      errorLog("Unrecognized command 'sing'. Use 'sign' instead.\n");
      process.exit(1);
    });

  // This default command provides backward compatibility.
  // The tool in the past only supported signing and didn't use commands.
  program
    .command('backward-compatibility-sign', { isDefault: true, hidden: true })
    // That's the workaround for proper error message, when an improper command is used.
    // It's then interpreted as an argument of the default command, so falls here.
    .argument('[...]', '')
    .option('-i, --input <file>', 'input web bundle to be signed (required)')
    .option(
      '-k, --private-key <file...>',
      'paths to Ed25519 / ECDSA P-256 private key(s) (required)'
    )
    .option(
      '-o, --output <file>',
      'signed web bundle output file',
      /*defaultValue=*/ 'signed.swbn'
    )
    .option('--web-bundle-id <web-bundle-id>', 'web bundle ID')
    // Command-specific error message on parsing error (e.g. no value, or incorrect option)
    .showHelpAfterError()
    .action(async (args, options, command) => {
      // Wrong command
      if (args.length > 0) {
        // Help quits the program, so not need to do it manually.
        program.help();
      }

      // Does it seem like backward-compatible format usage? If not just show help.
      if (
        !('input' in options) &&
        !('privateKey' in options) &&
        !('webBundleId' in options)
      ) {
        program.help();
      }

      // Backward-compatible mode
      warnLog(
        'This `wbn-sign` usage is deprecated. Please check `wbn-sign help`. This CLI usage form may be not supported in the future.'
      );

      if (!('input' in options) || !('privateKey' in options)) {
        errorLog(
          `input and private key options are required! Please, consider using new cli (see \`wbn-sign help\`)`
        );
        command.help();
      }

      if (options.privateKey.length > 1 && !options.webBundleId) {
        errorLog(
          `--web-bundle-id must be specified if there's more than 1 signing key involved.`
        );
        command.help();
      }

      await readVerifyAndSignWebBundle(
        options.input,
        options.privateKey,
        options.output,
        options.webBundleId
      );
    });

  // All errors that happen during command execution are caught here
  try {
    await program.parseAsync(process.argv);
  } catch (err) {
    if (err instanceof Error) {
      errorLog(err.message);
    }
    process.exit(1);
  }
}

async function readVerifyAndSignWebBundle(
  wbnFilePath: string,
  keyFilesPaths: string[],
  outputFilePath: string,
  maybeWebBundleId?: string
) {
  const webBundle = await fs.promises.readFile(wbnFilePath);
  if (isSignedWebBundle(webBundle)) {
    throw new Error(
      'Web bundle already signed. Use `add-signature` command instead.'
    );
  }
  if (!isPureWebBundle(webBundle)) {
    throw new Error('Not a web bundle.');
  }

  const privateKeys = new Array<KeyObject>();
  for (const privateKey of keyFilesPaths) {
    privateKeys.push(await parseMaybeEncryptedKeyFromFile(privateKey));
  }

  const signingStrategies = privateKeys.map(
    (privateKey) => new NodeCryptoSigningStrategy(privateKey)
  );

  const signedWebBundle = await SignedWebBundle.fromWebBundle(
    Uint8Array.from(webBundle),
    signingStrategies,
    maybeWebBundleId ? { webBundleId: maybeWebBundleId } : undefined
  );

  greenConsoleLog(`${signedWebBundle.getWebBundleId()}`);
  await fs.promises.writeFile(
    outputFilePath,
    signedWebBundle.getSignedWebBundleBytes()
  );
}

export async function main() {
  await parseArguments();
}
