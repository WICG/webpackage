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

const getMajorType = (b: number): MajorType => {
  return (b & 0xff) >> 5;
};

// This function loops through the given Uint8Array containing CBOR sequence and
// checks that they follow the deterministic principles described here:
// https://www.rfc-editor.org/rfc/rfc8949.html#name-deterministically-encoded-c.
export const deterministic = (input: Uint8Array): boolean => {
  let index = 0;

  while (index < input.length) {
    index += deterministicRec(input, index);
  }
  return true;
};

// A recursively called helper function to check the deterministicy of the CBOR
// item starting from the given index. Returns the length of the CBOR item in
// bytes.
const deterministicRec = (input: Uint8Array, index: number): number => {
  const b = input[index];

  switch (getMajorType(b)) {
    case MajorType.PosInt:
      // TODO(sonkkeli): Implement.
      return 0;

    case MajorType.ByteString:
    case MajorType.Text:
      // TODO(sonkkeli): Implement.
      return 0;

    case MajorType.Array:
      // TODO(sonkkeli): Implement.
      return 0;

    case MajorType.Map:
      // TODO(sonkkeli): Implement.
      return 0;

    default:
      throw new Error('Missing implementation for major type.');
  }
};
