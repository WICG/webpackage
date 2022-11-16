import {
  AdditionalInfo,
  convertToAdditionalInfo,
  getAdditionalInfoDirectValue,
  getAdditionalInfoLength,
  getAdditionalInfoValueLowerLimit,
} from './additionalinfo.js';
import { getMajorType, MajorType } from './majortype.js';

// This function loops through the given Uint8Array containing CBOR sequence and
// checks that they follow the deterministic principles described here:
// https://www.rfc-editor.org/rfc/rfc8949.html#name-deterministically-encoded-c.
// Throws an error if non-deterministic CBOR is encountered.
export function checkDeterministic(input: Uint8Array) {
  let index = 0;
  while (index < input.length) {
    index += deterministicRec(input, index);
  }
  if (index > input.length) {
    throw new Error(
      `Last CBOR item was incomplete. Index ${index} out of bounds of input of length ${input.length}`
    );
  }
}

// A recursively called helper function to check the deterministicity of the
// CBOR item starting from the given index. Returns the length of the CBOR item
// in bytes.
function deterministicRec(input: Uint8Array, index: number): number {
  const b = input[index];

  switch (getMajorType(b)) {
    case MajorType.PosInt:
      const { lengthInBytes } = unsignedIntegerDeterministic(input, index);
      return lengthInBytes + 1;

    case MajorType.ByteString:
    case MajorType.Text:
      return textOrByteStringDeterministic(input, index) + 1;

    case MajorType.Array:
      return arrayDeterministic(input, index);

    case MajorType.Map:
      return mapDeterministic(input, index);

    default:
      throw new Error('Missing implementation for a major type.');
  }
}

// unsignedIntegerDeterministic calculates the value of the unsigned integer to ensure that
// the right amount of bytes is used in the CBOR encoding. It returns both,
// the length of the bytes array and the value of the unsigned integer.
function unsignedIntegerDeterministic(
  input: Uint8Array,
  index: number
): { lengthInBytes: number; value: BigInt } {
  const info = convertToAdditionalInfo(input[index]);
  const lengthInBytes = getAdditionalInfoLength(info);
  const value = getUnsignedIntegerValue(
    input.slice(index, index + lengthInBytes + /*add info byte*/ 1),
    info
  );

  if (value < getAdditionalInfoValueLowerLimit(info)) {
    throw new Error(
      `${value} should not be represented with ${lengthInBytes} bytes in deterministic CBOR.`
    );
  }

  return { lengthInBytes, value };
}

export function getUnsignedIntegerValue(
  input: Uint8Array,
  info: AdditionalInfo
): bigint {
  // The additional information is on the first byte, which is not part of the
  // number to be read.
  const offset = 1;

  switch (info) {
    case AdditionalInfo.Direct:
      // Get the value from the last 5 bits of the byte.
      return BigInt(getAdditionalInfoDirectValue(input[0]));

    case AdditionalInfo.OneByte:
      return BigInt(Buffer.from(input).readUInt8(offset));

    case AdditionalInfo.TwoBytes:
      return BigInt(Buffer.from(input).readUInt16BE(offset));

    case AdditionalInfo.FourBytes:
      return BigInt(Buffer.from(input).readUInt32BE(offset));

    case AdditionalInfo.EightBytes:
      return Buffer.from(input).readBigUInt64BE(offset);

    default:
      throw new Error(`${info} is not supported.`);
  }
}

// Returns length of the text or byte string element in bytes for which its
// deterministicy has been ensured.
function textOrByteStringDeterministic(
  input: Uint8Array,
  index: number
): number {
  // lengthInBytes is the length of the number representing the length of the
  // text/byte string.
  const { lengthInBytes, value } = unsignedIntegerDeterministic(input, index);
  const totalLength = lengthInBytes + Number(value);

  if (totalLength >= input.length - index) {
    throw new Error(
      "Text or byte string's length cannot exceed the number of bytes left on the input array."
    );
  }

  return totalLength;
}

// Returns length of the CBOR array in bytes and checks that each item on it
// follows the deterministic principles.
function arrayDeterministic(input: Uint8Array, index: number): number {
  const { lengthInBytes, value } = unsignedIntegerDeterministic(input, index);

  // Skip the starter byte and the bytes stating the number of elements the array has.
  let startIndexOfNextElement = index + 1 + lengthInBytes;

  for (var /*element on the array*/ i = 0; i < Number(value); i++) {
    if (startIndexOfNextElement >= input.length) {
      throw new Error(
        'Number of items on CBOR array is less than the number of items it claims.'
      );
    }
    startIndexOfNextElement += deterministicRec(input, startIndexOfNextElement);
  }

  return startIndexOfNextElement - index;
}

// Returns the length of the CBOR map in bytes and checks that each item on it
// follows the deterministic principles. Additionally to ensure deterministicity
// of a map:
// 1) It cannot have duplicate keys.
// 2) Keys must be sorted in the bytewise lexicographic order of their
// deterministic encodings as described here:
// https://www.rfc-editor.org/rfc/rfc8949#section-4.2.1. In our use case it
// doesn't matter whether the ordering would instead follow the "length-first"
// map key ordering, because we don't have any use case where we would mix keys
// with different major types.
function mapDeterministic(input: Uint8Array, index: number): number {
  const { lengthInBytes, value } = unsignedIntegerDeterministic(input, index);

  // Skip the starter byte and the bytes stating the amount of element pairs the map has.
  let startIndexOfNextElement = index + 1 + lengthInBytes;
  let lastSeenKey = new Uint8Array();

  for (var mapItemIndex = 0; mapItemIndex < Number(value) * 2; mapItemIndex++) {
    if (startIndexOfNextElement >= input.length) {
      throw new Error(
        'Number of items on CBOR map is less than the number of items it claims.'
      );
    }

    const itemLength = deterministicRec(input, startIndexOfNextElement);

    // Every even item on the CBOR map is a key, which has to be unique and ordered.
    if (mapItemIndex % 2 == 0) {
      const keyCborBytes = input.slice(
        startIndexOfNextElement,
        startIndexOfNextElement + itemLength
      );

      // To be lexicographically ordered, the previous key must have been lexicographically smaller
      // than the current key. If the keys are equal (and ordered), their comparison returns 0.
      const ordering = Buffer.compare(lastSeenKey, keyCborBytes);
      if (ordering == 0) {
        throw new Error('CBOR map contains duplicate keys.');
      } else if (ordering > 0) {
        throw new Error('CBOR map keys are not in lexicographical order.');
      }

      lastSeenKey = keyCborBytes;
    }

    startIndexOfNextElement += itemLength;
  }

  return startIndexOfNextElement - index;
}
