import * as wbn from '../lib/wbn.js';
import * as cborg from 'cborg';

// Tests for webbundle format version b2

describe('Bundle Builder', () => {
  const defaultHeaders = { 'Content-Type': 'text/plain' };
  const defaultContent = 'Hello, world!';
  const validURLs = [
    'https://example.com/',
    'relative/url',
    '',  // An empty string is a valid relative URL.
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
      validURLs.forEach(url => {
        expect(
          builder.addExchange(url, 200, defaultHeaders, defaultContent)
        ).toBe(builder);
      });
    });

    it('rejects invalid URLs', () => {
      const builder = new wbn.BundleBuilder();
      invalidURLs.forEach(url => {
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
      invalidURLs.forEach(url => {
        expect(() => builder.setPrimaryURL(url)).toThrowError();
      });
    });

    it('rejects double call', () => {
      const builder = new wbn.BundleBuilder();
      builder.setPrimaryURL(exampleURL);
      expect(() =>
        builder.setPrimaryURL(exampleURL)
      ).toThrowError();
    });
  });

  it('builds large bundle', () => {
    const builder = new wbn.BundleBuilder();
    builder.addExchange(exampleURL, 200, defaultHeaders, new Uint8Array(1024 * 1024));
    const buf = builder.createBundle();
    // Just checks the result is a valid CBOR array.
    expect(cborg.decode(buf)).toBeInstanceOf(Array);
  });
});
