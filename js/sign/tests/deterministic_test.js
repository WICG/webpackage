import * as det from '../lib/cbor/deterministic.js';
import * as ainfo from '../lib/cbor/additionalinfo.js';
import * as fs from 'fs';
import * as path from 'path';
import url from 'url';
const __dirname = path.dirname(url.fileURLToPath(import.meta.url));

const uInts = {
  0: new Uint8Array([0x00]),
  10: new Uint8Array([0x0a]),
  23: new Uint8Array([0x17]),
  24: new Uint8Array([0x18, 0x18]),
  45: new Uint8Array([0x18, 0x2d]),
  255: new Uint8Array([0x18, 0xff]),
  256: new Uint8Array([0x19, 0x01, 0x00]),
  5000: new Uint8Array([0x19, 0x13, 0x88]),
  65535: new Uint8Array([0x19, 0xff, 0xff]),
  65536: new Uint8Array([0x1a, 0x00, 0x01, 0x00, 0x00]),
  4294967: new Uint8Array([0x1a, 0x00, 0x41, 0x89, 0x37]),
  4294967295: new Uint8Array([0x1a, 0xff, 0xff, 0xff, 0xff]),
  // This is also the max length of Uint8Array on node.
  4294967296: new Uint8Array([
    0x1b, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
  ]),
  // Number.MAX_SAFE_INTEGER
  9007199254740991: new Uint8Array([
    0x1b, 0x00, 0x1f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
  ]),
  '9223372036854775807': new Uint8Array([
    0x1b, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
  ]),
  // Max CBOR supported value 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF.
  '18446744073709551615': new Uint8Array([
    0x1b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
  ]),
};

describe('Deterministic check - Unsigned Integers', () => {
  it('works for positive integers, both single and sequences.', () => {
    for (const testCase of Object.values(uInts)) {
      det.checkDeterministic(testCase);
    }
    // CBOR sequence of positive integers
    det.checkDeterministic(Buffer.concat(Object.values(uInts)));
  });

  it("doesn't throw error for empty byte array.", () => {
    det.checkDeterministic(new Uint8Array());
  });

  it('calculates the value of an unsigned integer correctly.', () => {
    for (const testCase of Object.entries(uInts)) {
      const [expectedValue, valueInBytes] = testCase;
      const calculatedValue = det.getUnsignedIntegerValue(
        valueInBytes,
        ainfo.convertToAdditionalInfo(valueInBytes[0])
      );
      expect(calculatedValue).toBe(BigInt(expectedValue));
    }
  });

  // If the value of the unsigned integer is kept the same but there are
  // additional non-necessary empty bytes added in-front of the bytes of the
  // actual value, it's no longer considered as deterministically encoded CBOR
  // and should throw an error.
  it('fails for non-deterministically encoded positive integers.', () => {
    // The additional information values for representing a number with
    // 1, 2, 4 or 8 bytes.
    const firstBytes = [0x18, 0x19, 0x1a, 0x1b];

    for (const testCase of Object.values(uInts)) {
      for (const firstByte of firstBytes) {
        const ai = ainfo.convertToAdditionalInfo(firstByte);
        const newLength = ainfo.getAdditionalInfoLength(ai);

        // Cannot represent too big number with too little amount of bytes.
        if (testCase.length > newLength) {
          continue;
        }

        const nonDeterministicBytes = convertToNonDeterministicUintHelper(
          testCase,
          firstByte
        );
        const valueOfOriginalByteArr = det.getUnsignedIntegerValue(
          testCase,
          ainfo.convertToAdditionalInfo(testCase[0])
        );
        const valueOfNonDeterministicByteArr = det.getUnsignedIntegerValue(
          nonDeterministicBytes,
          ainfo.convertToAdditionalInfo(nonDeterministicBytes[0])
        );
        if (valueOfOriginalByteArr !== valueOfNonDeterministicByteArr) {
          throw new Error(
            'Values of original bytes and non-deterministic bytes should match.'
          );
        }

        expect(() =>
          det.checkDeterministic(nonDeterministicBytes)
        ).toThrowError();
      }
    }
  });
});

describe('Deterministic check - Additional information', () => {
  it('converts additional info correctly.', () => {
    for (let i = 0; i <= 23; i++) {
      expect(ainfo.convertToAdditionalInfo(i)).toEqual(
        ainfo.AdditionalInfo.Direct
      );
    }

    for (const entry of Object.entries({
      24: ainfo.AdditionalInfo.OneByte,
      25: ainfo.AdditionalInfo.TwoBytes,
      26: ainfo.AdditionalInfo.FourBytes,
      27: ainfo.AdditionalInfo.EightBytes,
    })) {
      const [key, value] = entry;
      expect(ainfo.convertToAdditionalInfo(key)).toEqual(value);
    }

    for (const notSupportedAdditionalInfo of [28, 29, 30, 31]) {
      expect(() =>
        ainfo.convertToAdditionalInfo(notSupportedAdditionalInfo)
      ).toThrowError();
    }
  });
});

describe('Deterministic check - ByteString and Text', () => {
  it('works for byte strings.', () => {
    const testBytes = new Uint8Array([0x43, 0x1a, 0x2b, 0x3c]);
    det.checkDeterministic(testBytes);
    det.checkDeterministic(new Uint8Array([...testBytes, ...testBytes]));
  });

  it('works for text.', () => {
    const testBytes = new Uint8Array([0x63, 0x1a, 0x2b, 0x3c]);
    det.checkDeterministic(testBytes);
    det.checkDeterministic(new Uint8Array([...testBytes, ...testBytes]));
  });

  it('works for long text and byte strings.', () => {
    const longStr =
      'olipakerrankilpikonnajakissajotkajuoksivatkilpaajakilpikonnavoitti';

    const testBytesAsByteString = new Uint8Array([
      0x58,
      longStr.length,
      ...new Uint8Array(Buffer.from(longStr)),
    ]);

    const testBytesAsTextString = new Uint8Array([
      0x78,
      longStr.length,
      ...new Uint8Array(Buffer.from(longStr)),
    ]);

    det.checkDeterministic(testBytesAsByteString);
    det.checkDeterministic(testBytesAsTextString);
    det.checkDeterministic(
      new Uint8Array([...testBytesAsByteString, ...testBytesAsByteString])
    );
    det.checkDeterministic(
      new Uint8Array([...testBytesAsTextString, ...testBytesAsTextString])
    );
  });

  it('detects non-deterministic byte string and text.', () => {
    // Claiming they would be followed by 3 bytes but there are either 2 or 4.
    const b01000011 = 0x43;
    const b01100011 = 0x63;
    const twoRandomBytes = [0x1a, 0x2b];
    const fourRandomBytes = [0x1a, 0x2b, 0x1a, 0x2b];

    expect(() =>
      det.checkDeterministic(new Uint8Array([b01000011, ...twoRandomBytes]))
    ).toThrowError();
    expect(() =>
      det.checkDeterministic(new Uint8Array([b01000011, ...fourRandomBytes]))
    ).toThrowError();

    expect(() =>
      det.checkDeterministic(new Uint8Array([b01100011, ...twoRandomBytes]))
    ).toThrowError();
    expect(() =>
      det.checkDeterministic(new Uint8Array([b01100011, ...fourRandomBytes]))
    ).toThrowError();
  });
});

describe('Deterministic check - Arrays', () => {
  it('works for arrays.', () => {
    const testBytes = new Uint8Array([
      0x85,
      ...uInts[23],
      ...uInts[24],
      ...uInts[45],
      ...uInts[4294967295],
      ...uInts['18446744073709551615'],
    ]);

    det.checkDeterministic(testBytes);
    det.checkDeterministic(new Uint8Array([...testBytes, ...testBytes]));
  });

  it('works for empty arrays.', () => {
    det.checkDeterministic(new Uint8Array([0x80]));
  });

  it('works for long arrays.', () => {
    const numOfItems = 24;
    let buffer = Buffer.alloc(2 + numOfItems * uInts[45].length);
    buffer[0] = 0x98; // ARR (4) + ONE_BYTE (24)
    buffer[1] = numOfItems;
    for (var i = 0; i < numOfItems; i++) {
      Buffer.from(uInts[45]).copy(buffer, /*offset=*/ 2 + i * uInts[45].length);
    }
    const testBytes = new Uint8Array(buffer);

    det.checkDeterministic(testBytes);
    det.checkDeterministic(new Uint8Array([...testBytes, ...testBytes]));
  });

  it('detects arrays containing non-deterministic CBOR.', () => {
    const testBytes = new Uint8Array([
      0x85,
      ...uInts[23],
      ...uInts[24],
      ...uInts[45],
      ...convertToNonDeterministicUintHelper(uInts[255], 0x19),
      ...uInts['18446744073709551615'],
    ]);

    expect(() => det.checkDeterministic(testBytes)).toThrowError(
      '255 should not be represented with 2 bytes in deterministic CBOR.'
    );
  });

  it('detects arrays containing too little amount of items.', () => {
    // The additional information claims that there are 5 but there are only 4.
    const testBytes = new Uint8Array([
      0x85,
      ...uInts[23],
      ...uInts[24],
      ...uInts[45],
      ...uInts[4294967295],
    ]);

    expect(() => det.checkDeterministic(testBytes)).toThrowError(
      'Number of items on CBOR array is less than the number of items it claims.'
    );
  });
});

describe('Deterministic check - Maps', () => {
  const hello = new Uint8Array([0x45, ...new Uint8Array(Buffer.from('hello'))]);
  const hello2 = new Uint8Array([
    0x46,
    ...new Uint8Array(Buffer.from('hello2')),
  ]);

  it('works for deterministic maps.', () => {
    const testBytes = new Uint8Array([
      0xa2,
      ...hello,
      ...uInts[24],
      ...hello2,
      ...uInts[4294967295],
    ]);

    det.checkDeterministic(testBytes);
    det.checkDeterministic(new Uint8Array([...testBytes, ...testBytes]));
  });

  it('detects duplicated keys in maps.', () => {
    const testBytes = new Uint8Array([
      0xa2,
      ...hello,
      ...uInts[24],
      ...hello,
      ...uInts[4294967295],
    ]);
    expect(() => det.checkDeterministic(testBytes)).toThrowError(
      'CBOR map contains duplicate keys.'
    );
  });

  it('detects duplicated non-sequential keys in maps.', () => {
    const testBytes = new Uint8Array([
      0xa3,
      ...hello,
      ...uInts[24],
      ...hello2,
      ...uInts[4294967295],
      ...hello,
      ...uInts['18446744073709551615'],
    ]);
    // This is actually impossible to happen without the keys also being in
    // non-lexicographical order.
    expect(() => det.checkDeterministic(testBytes)).toThrowError(
      'CBOR map keys are not in lexicographical order.'
    );
  });

  it('detects unordered keys in maps.', () => {
    // hello: 0b01100101 (0x65) should become before hello2: 0b01100110 (0x66),
    // so this doesn't follow the lexicographical order.
    const testBytes = new Uint8Array([
      0xa2,
      ...hello2,
      ...uInts[24],
      ...hello,
      ...uInts[4294967295],
    ]);
    expect(() => det.checkDeterministic(testBytes)).toThrowError(
      'CBOR map keys are not in lexicographical order.'
    );
  });

  it('detects maps containing too little amount of items.', () => {
    // The additional information claims that there are 2 pairs (=4 items) but
    // there are only 1 pair and 1 item (3 items).
    const testBytes = new Uint8Array([0xa2, ...hello, ...uInts[24], ...hello2]);
    expect(() => det.checkDeterministic(testBytes)).toThrowError(
      'Number of items on CBOR map is less than the number of items it claims.'
    );

    // The additional information claims that there are 2 pairs (=4 items) but
    // there are only 1 pair (2 items).
    const testBytes2 = new Uint8Array([0xa2, ...hello, ...uInts[24]]);
    expect(() => det.checkDeterministic(testBytes2)).toThrowError(
      'Number of items on CBOR map is less than the number of items it claims.'
    );
  });
});

describe('Deterministic check - Complex', () => {
  // The deterministic check is not planned to be used for the whole web bundle
  // and this test is just there to add more "randomness" to the CBOR tests
  // since we know the test web bundle is deterministically encoded. It contains
  // a lot of nested CBOR types so it might catch something that has not been
  // covered with other tests.
  it('works for a deterministically encoded web bundle.', () => {
    const webBundle = fs.readFileSync(
      path.resolve(__dirname, 'testdata', 'unsigned.wbn')
    );

    det.checkDeterministic(webBundle);
  });
});

// Helper functions.

function convertToNonDeterministicUintHelper(
  deterministicUintBytes,
  firstByte
) {
  const newLength = ainfo.getAdditionalInfoLength(
    ainfo.convertToAdditionalInfo(firstByte)
  );

  const ainfoIsDirect =
    ainfo.convertToAdditionalInfo(deterministicUintBytes[0]) ===
    ainfo.AdditionalInfo.Direct;

  // Defines if the first byte is part of the actual value or not.
  const shift = ainfoIsDirect ? 0 : 1;

  const emptyExtraBytes = new Uint8Array(
    newLength - deterministicUintBytes.length + shift
  );

  return new Uint8Array([
    firstByte,
    ...emptyExtraBytes,
    ...deterministicUintBytes.slice(shift),
  ]);
}
