import * as commander from 'commander';
import { BundleBuilder } from './encoder';
import * as fs from 'fs';

const options = commander
  .version(require('../package.json').version)
  .requiredOption('-d, --dir <directory>', 'input root directory (required)')
  .requiredOption('-b, --baseURL <URL>', 'base URL (required)')
  .option('-f, --formatVersion <formatVersion>', 'webbundle format version, possible values are "b1" and "b2" (default: "b2")', 'b2')
  .option('-p, --primaryURL <URL>', 'primary URL (defaults to base URL, only valid with format version "b1")')
  .option('-m, --manifestURL <URL>', 'manifest URL (only valid with format version "b1")')
  .option('-o, --output <file>', 'webbundle output file', 'out.wbn')
  .parse(process.argv);

if (!options.baseURL.endsWith('/')) {
  console.error("error: baseURL must end with '/'.");
  process.exit(1);
}

if (options.formatVersion === undefined || options.formatVersion === 'b2') {
  // webbundle format version b2
  const builder = new BundleBuilder();
  if (options.primaryURL) {
    builder.setPrimaryURL(options.primaryURL);
  }
  builder.addFilesRecursively(options.baseURL, options.dir);
  fs.writeFileSync(options.output, builder.createBundle());
} else if (options.formatVersion === 'b1') {
  // webbundle format version b1
  const primaryURL = options.primaryURL || options.baseURL;
  const builder = new BundleBuilder('b1');
  builder.setPrimaryURL(primaryURL);
  if (options.manifestURL) {
    builder.setManifestURL(options.manifestURL);
  }
  builder.addFilesRecursively(options.baseURL, options.dir);
  fs.writeFileSync(options.output, builder.createBundle());
} else {
  console.error('error: invalid format version (must be "b1" or "b2").');
  process.exit(1);
}
