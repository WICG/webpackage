export {
  // TODO: Export `BasicSigner` with its real name when there's more reason to
  // introduce a breaking change.
  BasicSigner as IntegrityBlockSigner,
  IntegrityBlock,
  WebBundleId,
  parsePemKey,
  readPassphrase,
  getRawPublicKey,
} from './integrityblock.js';
