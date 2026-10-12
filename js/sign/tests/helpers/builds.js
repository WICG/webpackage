import { createRequire } from 'module';

import * as esmBuild from '../../lib/wbn-sign.js';

const require = createRequire(import.meta.url);

// The package ships two builds of the same sources: ESM (`lib/**/*.js`, built
// by tsc) and CommonJS (`lib/wbn-sign.cjs`, bundled by esbuild).
const BUILDS = [
  ['ESM', esmBuild],
  ['CommonJS', require('../../lib/wbn-sign.cjs')],
];

/**
 * Defines the suite once for each build, so that both builds have to pass it.
 * @param {string} description
 * @param {(wbnSign: typeof esmBuild) => void} specDefinitions Receives the
 *     exports of the build under test.
 */
export function describeEachBuild(description, specDefinitions) {
  for (const [buildName, wbnSign] of BUILDS) {
    describe(`[${buildName}] ${description}`, () => specDefinitions(wbnSign));
  }
}
