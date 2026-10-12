// @ts-check
/** @typedef {import('jasmine')} _ */
import fs from 'fs';
import { createRequire } from 'module';
import path from 'path';
import url from 'url';

const __dirname = path.dirname(url.fileURLToPath(import.meta.url));
const PACKAGE_ROOT = path.resolve(__dirname, '..');
const require = createRequire(import.meta.url);
const packageJson = require('../package.json');

/**
 * Collects all string leaves of a (possibly nested) package.json field.
 * @param {unknown} value
 * @returns {string[]}
 */
function collectPaths(value) {
  if (typeof value === 'string') {
    return [value];
  }
  return Object.values(value ?? {}).flatMap(collectPaths);
}

/**
 * Lists all files below `dir`, relative to it and with `/` as the separator.
 * @param {string} dir
 * @returns {string[]}
 */
function listFiles(dir) {
  return fs
    .readdirSync(dir, { recursive: true, encoding: 'utf8' })
    .filter((file) => fs.statSync(path.join(dir, file)).isFile())
    .map((file) => file.split(path.sep).join('/'));
}

describe('Package - ', function () {
  it('package.json entry points refer to existing files', function () {
    const entryPoints = [
      packageJson.main,
      packageJson.types,
      ...collectPaths(packageJson.exports),
      ...collectPaths(packageJson.bin),
    ];
    for (const entryPoint of entryPoints) {
      expect(fs.existsSync(path.resolve(PACKAGE_ROOT, entryPoint)))
        .withContext(entryPoint)
        .toBeTrue();
    }
  });

  // Builds don't delete outputs of removed or renamed sources, so make sure
  // that none of them end up in a published package.
  it('lib/ contains no stale outputs of removed sources', function () {
    const sources = new Set(listFiles(path.join(PACKAGE_ROOT, 'src')));
    const staleOutputs = listFiles(path.join(PACKAGE_ROOT, 'lib')).filter(
      (output) =>
        !sources.has(output.replace(/\.(d\.ts|js|cjs)(\.map)?$/, '.ts'))
    );
    expect(staleOutputs)
      .withContext('Delete these files from lib/ and rebuild')
      .toEqual([]);
  });
});
