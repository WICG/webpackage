// Compile-only test, run with `npm run test:types`. It type-checks the emitted
// declarations the way a consumer sees them: 'wbn-sign' is resolved through the
// `types` field of package.json and `skipLibCheck` is disabled, so broken or
// incomplete `lib/**/*.d.ts` files fail the check.
import {
  getBundleId,
  getRawPublicKey,
  IntegrityBlock,
  IntegrityBlockSigner,
  NodeCryptoSigningStrategy,
  parsePemKey,
  readPassphrase,
  SignedWebBundle,
  WebBundleId,
  type IntegritySignature,
  type ISigningStrategy,
} from 'wbn-sign';

export const runtimeExports = [
  getBundleId,
  getRawPublicKey,
  IntegrityBlock,
  IntegrityBlockSigner,
  NodeCryptoSigningStrategy,
  parsePemKey,
  readPassphrase,
  SignedWebBundle,
  WebBundleId,
];

export type TypeOnlyExports = [IntegritySignature, ISigningStrategy];

// `IntegrityBlockSigner` is deprecated, but still part of the public API.
export function createSigner(
  webBundle: Uint8Array,
  webBundleId: string,
  signingStrategies: ISigningStrategy[]
): IntegrityBlockSigner {
  return new IntegrityBlockSigner(webBundle, webBundleId, signingStrategies);
}
