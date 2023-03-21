import commander from 'commander';
import { BundleBuilder } from './encoder.js';
import * as fs from 'fs';
import * as path from 'path';
import mime from 'mime';
import {
  APPROVED_VERSIONS,
  B1,
  B2,
  DEFAULT_VERSION,
  FormatVersion,
} from './constants.js';

export function addFile(builder: BundleBuilder, url: string, file: string) {
  const headers = {
    'Content-Type': mime.getType(file) || 'application/octet-stream',
  };
  builder.addExchange(url, 200, headers, fs.readFileSync(file));
}

export function addFilesRecursively(
  builder: BundleBuilder,
  baseURL: string,
  dir: string
) {
  if (baseURL !== '' && !baseURL.endsWith('/')) {
    throw new Error("Non-empty baseURL must end with '/'.");
  }
  const files = fs.readdirSync(dir);
  files.sort(); // Sort entries for reproducibility.
  for (const file of files) {
    const filePath = path.join(dir, file);
    if (fs.statSync(filePath).isDirectory()) {
      addFilesRecursively(builder, baseURL + file + '/', filePath);
    } else if (file === 'index.html') {
      // If the file name is 'index.html', create an entry for baseURL itself
      // and another entry for baseURL/index.html which redirects to baseURL.
      // This matches the behavior of gen-bundle.
      addFile(builder, baseURL, filePath);
      builder.addExchange(baseURL + file, 301, { Location: './' }, '');
    } else {
      addFile(builder, baseURL + file, filePath);
    }
  }
}

function readOptions() {
  // TODO(sonkkeli): Add support for overriding headers.

  return commander
    .requiredOption('-d, --dir <directory>', 'input root directory (required)')
    .option('-b, --baseURL <URL>', 'base URL')
    .option(
      '-f, --formatVersion <formatVersion>',
      'webbundle format version, possible values are ' +
        APPROVED_VERSIONS.map((v) => `"${v}"`).join(' and ') +
        ' (default: "' +
        DEFAULT_VERSION +
        '")',
      DEFAULT_VERSION
    )
    .option(
      '-p, --primaryURL <URL>',
      'primary URL (defaults to base URL, only valid with format version "' +
        B1 +
        '")'
    )
    .option(
      '-m, --manifestURL <URL>',
      'manifest URL (only valid with format version "' + B1 + '")'
    )
    .option('-o, --output <file>', 'webbundle output file', 'out.wbn')
    .parse(process.argv);
}

function validateOptions(options: any): string | null {
  if (options.baseURL && !options.baseURL.endsWith('/')) {
    return "error: baseURL must end with '/'.";
  }

  if (
    options.formatVersion !== undefined &&
    !APPROVED_VERSIONS.includes(options.formatVersion)
  ) {
    return (
      'error: invalid format version (must be ' +
      APPROVED_VERSIONS.map((val) => `"${val}"`).join(' or ') +
      ').'
    );
  }

  if (
    options.formatVersion === B1 &&
    !(options.primaryURL || options.baseURL)
  ) {
    return 'error: Primary URL is required.';
  }

  return null;
}

export function main() {
  const options = readOptions();
  const errorMsg = validateOptions(options);
  if (errorMsg) {
    console.error(errorMsg);
    process.exit(1);
  }

  const version: FormatVersion =
    options.formatVersion === undefined
      ? DEFAULT_VERSION
      : options.formatVersion;

  const builder = new BundleBuilder(version);

  switch (version) {
    case B1:
      builder.setPrimaryURL(options.primaryURL || options.baseURL);
      if (options.manifestURL) {
        builder.setManifestURL(options.manifestURL);
      }
      break;

    case B2:
      if (options.primaryURL) {
        builder.setPrimaryURL(options.primaryURL);
      }
      break;
  }

  addFilesRecursively(builder, options.baseURL || '', options.dir);
  fs.writeFileSync(options.output, builder.createBundle());
}
