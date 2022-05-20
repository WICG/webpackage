const wbn = require('../lib/wbn');
const CBOR = require('cbor');
const fs = require('fs');
const path = require('path');

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
    expect(CBOR.decode(buf)).toBeInstanceOf(Array);
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

  describe('addFile', () => {
    it('returns the builder itself', () => {
      const file = path.resolve(__dirname, 'testdata/encoder_test/index.html');
      const builder = new wbn.BundleBuilder();
      expect(builder.addFile(exampleURL, file)).toBe(builder);
    });

    it('adds an exchange as expected', () => {
      const file = path.resolve(__dirname, 'testdata/encoder_test/index.html');
      const builder = new wbn.BundleBuilder();
      builder.addFile(exampleURL, file);
      const generated = builder.createBundle();

      const refBuilder = new wbn.BundleBuilder();
      refBuilder.addExchange(
        exampleURL,
        200,
        { 'Content-Type': 'text/html' },
        fs.readFileSync(file)
      );
      const expected = refBuilder.createBundle();

      expect(expected.equals(generated)).toBeTrue();
    });

    it('throws on nonexistent file', () => {
      const file = path.resolve(__dirname, 'testdata/hello/nonexistent.html');
      const builder = new wbn.BundleBuilder();
      expect(() => builder.addFile(exampleURL, file)).toThrowError();
    });
  });

  describe('addFilesRecursively', () => {
    it('returns the builder itself', () => {
      const dir = path.resolve(__dirname, 'testdata/encoder_test');
      const builder = new wbn.BundleBuilder();
      expect(builder.addFilesRecursively(exampleURL, dir)).toBe(builder);
    });

    it('adds exchanges as expected', () => {
      const dir = path.resolve(__dirname, 'testdata/encoder_test');
      const baseURL = 'https://example.com/';

      const builder = new wbn.BundleBuilder();
      builder.addFilesRecursively(baseURL, dir);
      const generated = builder.createBundle();

      const refBuilder = new wbn.BundleBuilder();
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

    it('accepts relative base URL', () => {
      const dir = path.resolve(__dirname, 'testdata/encoder_test');
      const baseURL = 'assets/';

      const builder = new wbn.BundleBuilder();
      builder.addFilesRecursively(baseURL, dir);
      const generated = builder.createBundle();

      const refBuilder = new wbn.BundleBuilder();
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

    it('accepts empty base URL', () => {
      const dir = path.resolve(__dirname, 'testdata/encoder_test');

      const builder = new wbn.BundleBuilder();
      builder.addFilesRecursively('', dir);
      const generated = builder.createBundle();

      const refBuilder = new wbn.BundleBuilder();
      refBuilder.addExchange(
        '',
        200,
        { 'Content-Type': 'text/html' },
        fs.readFileSync(path.resolve(dir, 'index.html'))
      );
      refBuilder.addExchange(
        'index.html',
        301,
        { Location: './' },
        ''
      );
      refBuilder.addExchange(
        'resources/style.css',
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
      const builder = new wbn.BundleBuilder();
      expect(() => builder.addFilesRecursively(url, dir)).toThrowError();
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
    expect(CBOR.decode(buf)).toBeInstanceOf(Array);
  });
});
