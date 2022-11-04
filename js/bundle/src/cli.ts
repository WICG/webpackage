import commander from 'commander';
import { BundleBuilder } from './encoder.js';
import * as fs from 'fs';
import * as path from 'path';
import mime from 'mime';

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

export function main() {
  const options = commander
    .requiredOption('-d, --dir <directory>', 'input root directory (required)')
    .option('-b, --baseURL <URL>', 'base URL')
    .option(
      '-f, --formatVersion <formatVersion>',
      'webbundle format version, possible values are "b1" and "b2" (default: "b2")',
      'b2'
    )
    .option(
      '-p, --primaryURL <URL>',
      'primary URL (defaults to base URL, only valid with format version "b1")'
    )
    .option(
      '-m, --manifestURL <URL>',
      'manifest URL (only valid with format version "b1")'
    )
    .option('-o, --output <file>', 'webbundle output file', 'out.wbn')
    .parse(process.argv);

  if (options.baseURL && !options.baseURL.endsWith('/')) {
    console.error("error: baseURL must end with '/'.");
    process.exit(1);
  }

  if (options.formatVersion === undefined || options.formatVersion === 'b2') {
    // webbundle format version b2
    const builder = new BundleBuilder();
    if (options.primaryURL) {
      builder.setPrimaryURL(options.primaryURL);
    }
    addFilesRecursively(builder, options.baseURL || '', options.dir);
    fs.writeFileSync(options.output, builder.createBundle());
  } else if (options.formatVersion === 'b1') {
    // webbundle format version b1
    const primaryURL = options.primaryURL || options.baseURL;
    if (!primaryURL) {
      console.error('error: Primary URL is required.');
      process.exit(1);
    }
    const builder = new BundleBuilder('b1');
    builder.setPrimaryURL(primaryURL);
    if (options.manifestURL) {
      builder.setManifestURL(options.manifestURL);
    }
    addFilesRecursively(builder, options.baseURL || '', options.dir);
    fs.writeFileSync(options.output, builder.createBundle());
  } else {
    console.error('error: invalid format version (must be "b1" or "b2").');
    process.exit(1);
  }
}
