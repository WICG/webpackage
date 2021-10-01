import * as commander from 'commander';
import { BundleBuilder } from './encoder';
import * as fs from 'fs';

const options = commander
  .version(require('../package.json').version)
  .requiredOption('-d, --dir <directory>', 'input root directory (required)')
  .requiredOption('-b, --baseURL <URL>', 'base URL (required)')
  .option('-o, --output <file>', 'webbundle output file', 'out.wbn')
  .parse(process.argv);

if (!options.baseURL.endsWith('/')) {
  console.error("error: baseURL must end with '/'.");
  process.exit(1);
}
const builder = new BundleBuilder();
builder.addFilesRecursively(options.baseURL, options.dir);
fs.writeFileSync(options.output, builder.createBundle());
