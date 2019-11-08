const wbn = require('../lib/wbn');
const CBOR = require('cbor');

describe('Bundle Builder', () => {
  const primaryURL = 'https://example.com/';
  const defaultHeaders = { 'Content-Type': 'text/plain' };
  const defaultContent = 'Hello, world!';
  const invalidURLs = [
    '',
    'ftp://example.com/',
    'https://example.com/#fragment',
    'https://user:pass@example.com/',
    'relative/url',
  ];

  it('builds', () => {
    const builder = new wbn.BundleBuilder(primaryURL);
    builder.addExchange(primaryURL, 200, defaultHeaders, defaultContent);
    const buf = builder.createBundle();
    // Just checks the result is a valid CBOR array.
    expect(CBOR.decode(buf)).toBeInstanceOf(Array);
  });

  it('rejects invalid primary URLs', () => {
    invalidURLs.forEach(url => {
      expect(() => new wbn.BundleBuilder(url)).toThrowError();
    });
  });

  describe('addExchange', () => {
    it('rejects invalid URLs', () => {
      const builder = new wbn.BundleBuilder(primaryURL);
      invalidURLs.forEach(url => {
        expect(() =>
          builder.addExchange(url, 200, defaultHeaders, defaultContent)
        ).toThrowError();
      });
    });

    it('requires content-type for non-empty resources', () => {
      const builder = new wbn.BundleBuilder(primaryURL);
      expect(() =>
        builder.addExchange(primaryURL, 200, {}, defaultContent)
      ).toThrowError();
      builder.addExchange(primaryURL, 200, {}, ''); // This is accepted
    });
  });

  describe('setManifestURL', () => {
    it('rejects invalid URLs', () => {
      const builder = new wbn.BundleBuilder(primaryURL);
      invalidURLs.forEach(url => {
        expect(() => builder.addManifestURL(url)).toThrowError();
      });
    });

    it('rejects double call', () => {
      const builder = new wbn.BundleBuilder(primaryURL);
      builder.setManifestURL('https://example.com/manifest.json');
      expect(() =>
        builder.setManifestURL('https://example.com/manifest.json')
      ).toThrowError();
    });
  });

  it('builds large bundle', () => {
    const builder = new wbn.BundleBuilder(primaryURL);
    builder.addExchange(primaryURL, 200, defaultHeaders, new Uint8Array(1024*1024));
    const buf = builder.createBundle();
    // Just checks the result is a valid CBOR array.
    expect(CBOR.decode(buf)).toBeInstanceOf(Array);
  });
});
