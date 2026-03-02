// Major type RFC: https://www.rfc-editor.org/rfc/rfc8949.html#name-major-types.

export enum MajorType {
  PosInt = 0,
  NegInt = 1,
  ByteString = 2,
  Text = 3,
  Array = 4,
  Map = 5,
  Tag = 6,
  Other = 7,
}

// Returns the first 3 bits of the first byte representing cbor's major type.
export function getMajorType(b: number): MajorType {
  return (b & 0xff) >> 5;
}
