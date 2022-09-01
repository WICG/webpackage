import * as wbn from '../lib/wbn.js';
import * as cli from '../lib/cli.js';
import * as fs from 'fs';
import * as path from 'path';
import url from "url";
const __dirname = path.dirname(url.fileURLToPath(import.meta.url));

describe('CLI', () => {
  const exampleURL = 'https://example.com/';
    describe('addFile', () => {
    it('adds an exchange as expected', () => {
      const file = path.resolve(__dirname, 'testdata/encoder_test/index.html');
      const builder = new wbn.BundleBuilder();
      cli.addFile(builder, exampleURL, file);
      const generated = builder.createBundle();

      const refBuilder = new wbn.BundleBuilder();
      refBuilder.addExchange(
        exampleURL,
        200,
        { 'Content-Type': 'text/html' },
        fs.readFileSync(file)
      );
      const expected = refBuilder.createBundle();

      expect(Buffer.compare(expected, generated)).toBe(0);
    });

    it('throws on nonexistent file', () => {
      const file = path.resolve(__dirname, 'testdata/hello/nonexistent.html');
      const builder = new wbn.BundleBuilder();
      expect(() => cli.addFile(builder, exampleURL, file)).toThrowError();
    });
  });

  describe('addFilesRecursively', () => {
    it('adds exchanges as expected', () => {
      const dir = path.resolve(__dirname, 'testdata/encoder_test');
      const baseURL = 'https://example.com/';

      const builder = new wbn.BundleBuilder();
      cli.addFilesRecursively(builder, baseURL, dir);
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

      expect(Buffer.compare(expected, generated)).toBe(0);
    });

    it('accepts relative base URL', () => {
      const dir = path.resolve(__dirname, 'testdata/encoder_test');
      const baseURL = 'assets/';

      const builder = new wbn.BundleBuilder();
      cli.addFilesRecursively(builder, baseURL, dir);
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

      expect(Buffer.compare(expected, generated)).toBe(0);
    });

    it('accepts empty base URL', () => {
      const dir = path.resolve(__dirname, 'testdata/encoder_test');

      const builder = new wbn.BundleBuilder();
      cli.addFilesRecursively(builder, '', dir);
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

      expect(Buffer.compare(expected, generated)).toBe(0);
    });

    it('throws if baseURL does not end with a slash', () => {
      const dir = path.resolve(__dirname, 'testdata/hello');
      const url = 'https://example.com/hello.html';
      const builder = new wbn.BundleBuilder();
      expect(() => cli.addFilesRecursively(builder, url, dir)).toThrowError();
    });
  });
});
