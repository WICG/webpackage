import * as wbn from '../lib/wbn.js';
import * as fs from 'fs';
import * as path from 'path';
import url from 'url';
const __dirname = path.dirname(url.fileURLToPath(import.meta.url));

// Tests for webbundle format version b2

describe('Bundle', () => {
  const bundleBuffer = (() => {
    const builder = new wbn.BundleBuilder();
    builder.setPrimaryURL('https://example.com/');
    builder.addExchange(
      'https://example.com/',
      200,
      { 'Content-Type': 'text/plain' },
      'Hello, world!'
    );
    builder.addExchange(
      'https://example.com/ja/',
      200,
      { 'Content-Type': 'text/plain', 'Content-Language': 'ja' },
      'こんにちは世界'
    );
    return builder.createBundle();
  })();

  it('has expected fields', () => {
    const b = new wbn.Bundle(bundleBuffer);
    expect(b.version).toBe('b2');
    expect(b.urls).toEqual(['https://example.com/', 'https://example.com/ja/']);
    expect(b.primaryURL).toBe('https://example.com/');
  });

  describe('getResponse', () => {
    it('returns Response with expected fields', () => {
      const b = new wbn.Bundle(bundleBuffer);
      const resp1 = b.getResponse('https://example.com/');
      const resp2 = b.getResponse('https://example.com/ja/');
      expect(resp1.status).toBe(200);
      expect(resp2.status).toBe(200);
      expect(resp1.headers).toEqual({ 'content-type': 'text/plain' });
      expect(resp2.headers).toEqual({
        'content-type': 'text/plain',
        'content-language': 'ja',
      });
      expect(new TextDecoder('utf-8').decode(resp1.body)).toBe('Hello, world!');
      expect(new TextDecoder('utf-8').decode(resp2.body)).toBe(
        'こんにちは世界'
      );
    });

    it('throws if URL is not found', () => {
      const b = new wbn.Bundle(bundleBuffer);
      expect(() =>
        b.getResponse('https://example.com/nonexistent')
      ).toThrowError();
    });
  });

  it('parses pregenerated bundle', () => {
    const buf = fs.readFileSync(
      path.resolve(__dirname, 'testdata/hello_b2.wbn')
    );
    const b = new wbn.Bundle(buf);
    expect(b.primaryURL).toBe(null);
    expect(b.urls).toEqual(['https://example.com/hello.html']);
    const resp = b.getResponse('https://example.com/hello.html');
    expect(resp.status).toBe(200);
    expect(resp.headers['content-type']).toBe('text/html; charset=utf-8');
    expect(new TextDecoder('utf-8').decode(resp.body)).toBe(
      '<html>Hello, Web Bundle!</html>\n'
    );
  });

  it('throws if an unknown section is marked as critical', () => {
    const builder = new wbn.BundleBuilder();
    builder.addExchange(
      'https://example.com/',
      200,
      { 'Content-Type': 'text/plain' },
      'Hello, world!'
    );
    builder.addSection('critical', ['unknown']);
    const buf = builder.createBundle();
    expect(() => new wbn.Bundle(buf)).toThrowError();
  });

  it('does not throw if all names in the critical section are known', () => {
    const builder = new wbn.BundleBuilder();
    builder.addExchange(
      'https://example.com/',
      200,
      { 'Content-Type': 'text/plain' },
      'Hello, world!'
    );
    builder.addSection('critical', [
      'critical',
      'index',
      'responses',
      'signatures',
    ]);
    const buf = builder.createBundle();
    expect(() => new wbn.Bundle(buf)).not.toThrowError();
  });
});
