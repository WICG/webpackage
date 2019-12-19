import * as commander from 'commander';
import { BundleBuilder } from './encoder';
import * as fs from 'fs';

const options = commander
  .version(require('../package.json').version)
  .requiredOption('-d, --dir <directory>', 'input root directory (required)')
  .requiredOption('-b, --baseURL <URL>', 'base URL (required)')
  .option('-p, --primaryURL <URL>', 'primary URL (defaults to base URL)')
  .option('-m, --manifestURL <URL>', 'manifest URL')
  .option('-o, --output <file>', 'webbundle output file', 'out.wbn')
  .parse(process.argv);

if (!options.baseURL.endsWith('/')) {
  console.error("error: baseURL must end with '/'.");
  process.exit(1);
}
const primaryURL = options.primaryURL || options.baseURL;
const builder = new BundleBuilder(primaryURL);
if (options.manifestURL) {
  builder.setManifestURL(options.manifestURL);
}
builder.addFilesRecursively(options.baseURL, options.dir);
fs.writeFileSync(options.output, builder.createBundle());
