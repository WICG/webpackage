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

export const INTEGRITY_BLOCK_MAGIC = new Uint8Array([
  0xf0, 0x9f, 0x96, 0x8b, 0xf0, 0x9f, 0x93, 0xa6,
]); // 🖋📦
export const VERSION_B1 = new Uint8Array([0x31, 0x62, 0x00, 0x00]); // 1b\0\0
