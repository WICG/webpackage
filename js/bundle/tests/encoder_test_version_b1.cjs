const wbn = require('../lib/wbn.cjs');
const cborg = require('cborg');

// Backwards compatibility tests for webbundle format version b1

describe('Bundle Builder', () => {
  const defaultHeaders = { 'Content-Type': 'text/plain' };
  const defaultContent = 'Hello, world!';
  const validURLs = ['https://example.com/', 'relative/url', ''];
  const primaryURL = validURLs[0];
  const invalidURLs = [
    'https://example.com/#fragment',
    'https://user:pass@example.com/',
  ];

  it('builds', () => {
    const builder = new wbn.BundleBuilder('b1');
    builder.setPrimaryURL(primaryURL);
    builder.addExchange(primaryURL, 200, defaultHeaders, defaultContent);
    const buf = builder.createBundle();
    // Just checks the result is a valid CBOR array.
    expect(cborg.decode(buf)).toBeInstanceOf(Array);
  });

  describe('addExchange', () => {
    it('returns the builder itself', () => {
      const builder = new wbn.BundleBuilder('b1');
      builder.setPrimaryURL(primaryURL);
      expect(
        builder.addExchange(primaryURL, 200, defaultHeaders, defaultContent)
      ).toBe(builder);
    });

    it('accepts valid URLs', () => {
      const builder = new wbn.BundleBuilder('b1');
      builder.setPrimaryURL(primaryURL);
      validURLs.forEach((url) => {
        expect(
          builder.addExchange(url, 200, defaultHeaders, defaultContent)
        ).toBe(builder);
      });
    });

    it('rejects invalid URLs', () => {
      const builder = new wbn.BundleBuilder('b1');
      builder.setPrimaryURL(primaryURL);
      invalidURLs.forEach((url) => {
        expect(() =>
          builder.addExchange(url, 200, defaultHeaders, defaultContent)
        ).toThrowError();
      });
    });

    it('requires content-type for non-empty resources', () => {
      const builder = new wbn.BundleBuilder('b1');
      builder.setPrimaryURL(primaryURL);
      expect(() =>
        builder.addExchange(primaryURL, 200, {}, defaultContent)
      ).toThrowError();
      builder.addExchange(primaryURL, 200, {}, ''); // This is accepted
    });
  });

  describe('setPrimaryURL', () => {
    it('returns the builder itself', () => {
      const builder = new wbn.BundleBuilder('b1');
      expect(builder.setPrimaryURL(primaryURL)).toBe(builder);
    });

    it('must be called before createBundle', () => {
      invalidURLs.forEach((url) => {
        const builder = new wbn.BundleBuilder('b1');
        expect(() => builder.createBundle()).toThrowError();
      });
    });

    it('rejects invalid URLs', () => {
      invalidURLs.forEach((url) => {
        const builder = new wbn.BundleBuilder('b1');
        expect(() => builder.setPrimaryURL(url)).toThrowError();
      });
    });

    it('rejects double call', () => {
      const builder = new wbn.BundleBuilder('b1');
      builder.setPrimaryURL(primaryURL);
      expect(() => builder.setPrimaryURL(primaryURL)).toThrowError();
    });
  });

  describe('setManifestURL', () => {
    it('returns the builder itself', () => {
      const builder = new wbn.BundleBuilder('b1');
      builder.setPrimaryURL(primaryURL);
      expect(builder.setManifestURL(primaryURL)).toBe(builder);
    });

    it('rejects invalid URLs', () => {
      const builder = new wbn.BundleBuilder('b1');
      builder.setPrimaryURL(primaryURL);
      invalidURLs.forEach((url) => {
        expect(() => builder.setManifestURL(url)).toThrowError();
      });
    });

    it('rejects double call', () => {
      const builder = new wbn.BundleBuilder('b1');
      builder.setPrimaryURL(primaryURL);
      builder.setManifestURL('https://example.com/manifest.json');
      expect(() =>
        builder.setManifestURL('https://example.com/manifest.json')
      ).toThrowError();
    });
  });

  it('builds large bundle', () => {
    const builder = new wbn.BundleBuilder('b1');
    builder.setPrimaryURL(primaryURL);
    builder.addExchange(
      primaryURL,
      200,
      defaultHeaders,
      new Uint8Array(1024 * 1024)
    );
    const buf = builder.createBundle();
    // Just checks the result is a valid CBOR array.
    expect(cborg.decode(buf)).toBeInstanceOf(Array);
  });
});
