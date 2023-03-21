import * as wbn from '../lib/wbn.js';
import * as cborg from 'cborg';
import * as constants from '../lib/constants.js';

// Tests for webbundle format version b2

describe('Bundle Builder', () => {
  const defaultHeaders = { 'Content-Type': 'text/plain' };
  const defaultContent = 'Hello, world!';
  const validURLs = [
    'https://example.com/',
    'relative/url',
    '', // An empty string is a valid relative URL.
  ];
  const exampleURL = validURLs[0];
  const invalidURLs = [
    'https://example.com/#fragment',
    'https://user:pass@example.com/',
  ];

  it('builds', () => {
    const builder = new wbn.BundleBuilder();
    const buf = builder.createBundle();
    // Just checks the result is a valid CBOR array.
    expect(cborg.decode(buf)).toBeInstanceOf(Array);
  });

  describe('addExchange', () => {
    it('returns the builder itself', () => {
      const builder = new wbn.BundleBuilder();
      expect(
        builder.addExchange(exampleURL, 200, defaultHeaders, defaultContent)
      ).toBe(builder);
    });

    it('accepts valid URLs', () => {
      const builder = new wbn.BundleBuilder();
      validURLs.forEach((url) => {
        expect(
          builder.addExchange(url, 200, defaultHeaders, defaultContent)
        ).toBe(builder);
      });
    });

    it('rejects invalid URLs', () => {
      const builder = new wbn.BundleBuilder();
      invalidURLs.forEach((url) => {
        expect(() =>
          builder.addExchange(url, 200, defaultHeaders, defaultContent)
        ).toThrowError();
      });
    });

    it('requires content-type for non-empty resources', () => {
      const builder = new wbn.BundleBuilder();
      expect(() =>
        builder.addExchange(exampleURL, 200, {}, defaultContent)
      ).toThrowError();
      builder.addExchange(exampleURL, 200, {}, ''); // This is accepted
    });
  });

  describe('setPrimaryURL', () => {
    it('returns the builder itself', () => {
      const builder = new wbn.BundleBuilder();
      expect(builder.setPrimaryURL(exampleURL)).toBe(builder);
    });

    it('rejects invalid URLs', () => {
      const builder = new wbn.BundleBuilder();
      invalidURLs.forEach((url) => {
        expect(() => builder.setPrimaryURL(url)).toThrowError();
      });
    });

    it('rejects double call', () => {
      const builder = new wbn.BundleBuilder();
      builder.setPrimaryURL(exampleURL);
      expect(() => builder.setPrimaryURL(exampleURL)).toThrowError();
    });
  });

  it('builds large bundle', () => {
    const builder = new wbn.BundleBuilder();
    builder.addExchange(
      exampleURL,
      200,
      defaultHeaders,
      new Uint8Array(1024 * 1024)
    );
    const buf = builder.createBundle();
    // Just checks the result is a valid CBOR array.
    expect(cborg.decode(buf)).toBeInstanceOf(Array);
  });

  describe('overriding headers -', () => {
    const failingOverrideHeadersTestCases = [
      {
        testName: 'same header name with different value throws an error',
        headers: { 'Content-Type': 'text/plain' },
        overrides: { 'Content-Type': 'text/html' },
      },
      {
        testName:
          'different header name capitalization with different value throws an error',
        headers: { 'Content-Type': 'text/plain' },
        overrides: { 'content-type': 'text/html' },
      },
    ];

    for (const testCase of failingOverrideHeadersTestCases) {
      it(testCase.testName, () => {
        expect(() =>
          wbn.getCombinedHeaders(testCase.headers, testCase.overrides)
        ).toThrowError();
      });
    }

    const succeedingOverrideHeadersTestCases = [
      {
        testName: 'same header with same value only gets added once',
        headers: { 'Content-Type': 'text/plain' },
        overrides: { 'Content-Type': 'text/plain' },
        combinedResult: { 'content-type': 'text/plain' },
      },
      {
        testName: "different headers don't interfere and both get added",
        headers: { 'Content-Type': 'text/plain' },
        overrides: { 'another-header': 'another-value' },
        combinedResult: {
          'content-type': 'text/plain',
          'another-header': 'another-value',
        },
      },
      {
        testName:
          'same header name with different capitalization in the header name only gets added once (header names are case-insensitive)',
        headers: { 'Content-Type': 'text/plain' },
        overrides: { 'content-type': 'text/plain' },
        combinedResult: { 'content-type': 'text/plain' },
      },
      {
        testName:
          'same header name with different capitalization in the header value is considered as one and override header value overrides the original value',
        headers: { 'content-type': 'Text/Plain' },
        overrides: { 'content-type': 'text/plain' },
        combinedResult: { 'content-type': 'text/plain' },
      },
      {
        testName:
          'same header name with different capitalization in the header value is considered as one and override header value overrides the original value',
        headers: { 'content-type': 'text/plain' },
        overrides: { 'content-type': 'Text/Plain' },
        combinedResult: { 'content-type': 'Text/Plain' },
      },
    ];

    for (const testCase of succeedingOverrideHeadersTestCases) {
      it(testCase.testName, () => {
        expect(
          wbn.getCombinedHeaders(testCase.headers, testCase.overrides)
        ).toEqual(testCase.combinedResult);
      });
    }

    it('BundleBuilder works with both `OverrideHeadersOption`s', () => {
      const headersMap = { 'content-type': 'text/plain' };

      const overridesMap = { 'another-header': 'another-value' };
      function getOverrides(path) {
        return overridesMap;
      }

      const simpleBuilder = new wbn.BundleBuilder(constants.DEFAULT_VERSION);
      simpleBuilder.addExchange(exampleURL, 200, headersMap, defaultContent);
      const simpleBundle = simpleBuilder.createBundle();

      for (const testCase of [overridesMap, getOverrides]) {
        const builderWithOverrides = new wbn.BundleBuilder(
          constants.DEFAULT_VERSION,
          testCase
        );
        builderWithOverrides.addExchange(
          exampleURL,
          200,
          headersMap,
          defaultContent
        );
        // A bundle with more headers has more bytes.
        expect(builderWithOverrides.createBundle().length).toBeGreaterThan(
          simpleBundle.length
        );
      }
    });
  });
});
