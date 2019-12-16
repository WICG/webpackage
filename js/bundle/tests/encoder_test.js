const wbn = require('../lib/wbn');
const CBOR = require('cbor');
const fs = require('fs');
const path = require('path');

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

  describe('addFile', () => {
    it('adds an exchange as expected', () => {
      const file = path.resolve(__dirname, 'testdata/encoder_test/index.html');

      const builder = new wbn.BundleBuilder(primaryURL);
      builder.addFile(file, primaryURL);
      const generated = builder.createBundle();

      const refBuilder = new wbn.BundleBuilder(primaryURL);
      refBuilder.addExchange(
        primaryURL,
        200,
        { 'Content-Type': 'text/html' },
        fs.readFileSync(file)
      );
      const expected = refBuilder.createBundle();

      expect(expected.equals(generated)).toBeTrue();
    });

    it('throws on nonexistent file', () => {
      const file = path.resolve(__dirname, 'testdata/hello/nonexistent.html');
      const builder = new wbn.BundleBuilder(primaryURL);
      expect(() => builder.addFile(file, primaryURL)).toThrowError();
    });
  });

  describe('addFilesRecursively', () => {
    it('adds exchanges as expected', () => {
      const dir = path.resolve(__dirname, 'testdata/encoder_test');
      const baseURL = 'https://example.com/';

      const builder = new wbn.BundleBuilder(baseURL);
      builder.addFilesRecursively(dir, baseURL);
      const generated = builder.createBundle();

      const refBuilder = new wbn.BundleBuilder(baseURL);
      refBuilder.addExchange(
        baseURL,
        200,
        { 'Content-Type': 'text/html' },
        fs.readFileSync(path.resolve(dir, 'index.html'))
      );
      refBuilder.addExchange(
        baseURL + 'index.html',
        301,
        { Location: './' },
        ''
      );
      refBuilder.addExchange(
        baseURL + 'resources/style.css',
        200,
        { 'Content-Type': 'text/css' },
        fs.readFileSync(path.resolve(dir, 'resources/style.css'))
      );
      const expected = refBuilder.createBundle();

      expect(expected.equals(generated)).toBeTrue();
    });

    it('throws if baseURL does not end with a slash', () => {
      const dir = path.resolve(__dirname, 'testdata/hello');
      const url = 'https://example.com/hello.html';
      const builder = new wbn.BundleBuilder(url);
      expect(() => builder.addFilesRecursively(dir, url)).toThrowError();
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
