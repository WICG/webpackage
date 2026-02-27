export enum SignatureType {
  Ed25519,
  EcdsaP256SHA256,
}

export const PUBLIC_KEY_ATTRIBUTE_NAME_MAPPING = new Map<SignatureType, string>(
  [
    [SignatureType.Ed25519, 'ed25519PublicKey'],
    [SignatureType.EcdsaP256SHA256, 'ecdsaP256SHA256PublicKey'],
  ]
);

export const WEB_BUNDLE_ID_ATTRIBUTE_NAME = 'webBundleId';

// Reversed map above, useful for parsing Integrity Block
export const SIGNATURE_ATTRIBUTE_TO_TYPE_MAPPING = new Map(
  Array.from(PUBLIC_KEY_ATTRIBUTE_NAME_MAPPING, ([k, v]) => [v, k])
);

export const INTEGRITY_BLOCK_MAGIC = new Uint8Array([
  0xf0, 0x9f, 0x96, 0x8b, 0xf0, 0x9f, 0x93, 0xa6,
]); // 🖋📦
export const VERSION_B2 = new Uint8Array([0x32, 0x62, 0x00, 0x00]); // 2b\0\0
