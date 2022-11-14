// 5 bits of the first byte on CBOR contain the additional information which can
// be either the actual value or a value telling us how to get the value.

// The meaning of additional information depends on the major type. For example,
// in major type 0, the argument is the value of the data item itself (and in
// major type 1, the value of the data item is computed from the argument); in
// major type 2 and 3, it gives the length of the string data in bytes that
// follow; and in major types 4 and 5, it is used to determine the number of
// data items enclosed.

// https://www.rfc-editor.org/rfc/rfc8949.html#name-specification-of-the-cbor-e

export enum AdditionalInfo {
  Direct, // 0->23
  OneByte, // 24
  TwoBytes, // 25
  FourBytes, // 26
  EightBytes, // 27
  Reserved, // 28->30
  Indefinite, // 31
}

export const convertToAdditionalInfo = (b: number): AdditionalInfo => {
  switch (b & 0b00011111) {
    case 24:
      return AdditionalInfo.OneByte;
    case 25:
      return AdditionalInfo.TwoBytes;
    case 26:
      return AdditionalInfo.FourBytes;
    case 27:
      return AdditionalInfo.EightBytes;
    case 28:
    case 29:
    case 30:
      throw new Error('Reserved is not used in deterministic CBOR.');
    case 31:
      throw new Error('Indefinite is not used in deterministic CBOR.');
    default:
      return AdditionalInfo.Direct;
  }
};

// Returns the value of the bits following the additional info when
// AdditionalInfo is of type Direct.
export const getAdditionalInfoDirectValue = (b: number): number => {
  return b & 0b00011111;
};

// Returns the length of the byte array following the additional info byte from
// which to read the unsigned integer's bytes.
export const getAdditionalInfoLength = (info: AdditionalInfo): number => {
  switch (info) {
    case AdditionalInfo.Direct:
      return 0;
    case AdditionalInfo.OneByte:
      return 1;
    case AdditionalInfo.TwoBytes:
      return 2;
    case AdditionalInfo.FourBytes:
      return 4;
    case AdditionalInfo.EightBytes:
      return 8;
    default:
      throw new Error(`${info} is not supported.`);
  }
};

// Returns the unsigned integer limit which is the lowest that should be using
// the AdditionalInfo in question. If the unsigned integer is smaller than the
// limit, it should use less bytes than it is currently using meaning it is not
// following the deterministic principles.
export const getAdditionalInfoValueLowerLimit = (
  info: AdditionalInfo
): bigint => {
  switch (info) {
    case AdditionalInfo.Direct:
      return 0n;
    case AdditionalInfo.OneByte:
      // This is a special case and defined by the deterministic CBOR standard.
      // Anything <= 23L should be directly the value of the additional info.
      return 24n;
    case AdditionalInfo.TwoBytes:
      // 1 << 8 aka 0b11111111 +1
      return 256n;
    case AdditionalInfo.FourBytes:
      // 1 << 16 aka 0b11111111_11111111 +1
      return 65536n;
    case AdditionalInfo.EightBytes:
      // 1 << 32 aka 0b11111111_11111111_11111111_11111111 +1
      return 4294967296n;
    default:
      throw new Error(`Invalid additional information value: ${info}`);
  }
};
