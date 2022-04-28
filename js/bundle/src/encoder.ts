import * as CBOR from 'cbor';
import * as fs from 'fs';
import * as mime from 'mime';
import * as path from 'path';
import { URL } from 'url';

declare module 'cbor' {
  // TODO: upstream these to @types/cbor
  interface DecoderOptions {
    canonical?: boolean;
    highWaterMark?: number;
  }
  export function encodeOne(obj: CBORValue, opts?: DecoderOptions): Buffer;
}

type CBORValue = unknown;
interface Headers {
  [key: string]: string;
}

interface CompatAdapter {
  formatVersion: string;
  onCreateBundle(): void;
  setPrimaryURL(url: string): BundleBuilder;
  setManifestURL(url: string): BundleBuilder;
  setIndexEntry(url: string, responseLength: number): void;
  updateIndexValues(responsesHeaderSize: number): Map<string, any[]>;
  createTopLevel(): unknown;
}

export class BundleBuilder {
  private sectionLengths: Array<{ name: string; length: number }> = [];
  private sections: CBORValue[] = [];
  private responses: Uint8Array[][] = [];
  private currentResponsesOffset = 0;
  private compatAdapter: CompatAdapter;

  constructor(formatVersion: string = 'b2') {
    if (formatVersion != 'b1' && formatVersion != 'b2') {
      throw new Error(`Invalid webbundle format version`);
    }
    this.compatAdapter = this.createCompatAdapter(formatVersion);
  }

  // TODO: Provide async version of this.
  createBundle(): Buffer {
    this.compatAdapter.onCreateBundle();

    this.addSection('index', this.fixupIndex());
    this.addSection('responses', this.responses);

    const estimatedBundleSize = this.sectionLengths.reduce(
      (size, s) => size + s.length,
      16384 // For headers
    );
    const wbn = encodeCanonical(this.createTopLevel(), estimatedBundleSize);

    // Fill in the length field.
    const view = new DataView(wbn.buffer, wbn.byteOffset + wbn.length - 8);
    view.setUint32(0, Math.floor(wbn.length / 0x100000000));
    view.setUint32(4, wbn.length & 0xffffffff);
    return wbn;
  }

  addExchange(
    url: string,
    status: number,
    headers: Headers,
    payload: Uint8Array | string
  ): BundleBuilder {
    validateExchangeURL(url);
    if (typeof payload === 'string') {
      payload = byteString(payload);
    }
    this.addIndexEntry(
      url,
      this.addResponse(new HeaderMap(status, headers), payload)
    );
    return this;
  }

  addFile(url: string, file: string): BundleBuilder {
    const headers = {
      'Content-Type': mime.getType(file) || 'application/octet-stream',
    };
    this.addExchange(url, 200, headers, fs.readFileSync(file));
    return this;
  }

  addFilesRecursively(baseURL: string, dir: string): BundleBuilder {
    if (!baseURL.endsWith('/')) {
      throw new Error("baseURL must end with '/'.");
    }
    const files = fs.readdirSync(dir);
    files.sort(); // Sort entries for reproducibility.
    for (const file of files) {
      const filePath = path.join(dir, file);
      if (fs.statSync(filePath).isDirectory()) {
        this.addFilesRecursively(baseURL + file + '/', filePath);
      } else if (file === 'index.html') {
        // If the file name is 'index.html', create an entry for baseURL itself
        // and another entry for baseURL/index.html which redirects to baseURL.
        // This matches the behavior of gen-bundle.
        this.addFile(baseURL, filePath);
        this.addExchange(baseURL + file, 301, { Location: './' }, '');
      } else {
        this.addFile(baseURL + file, filePath);
      }
    }
    return this;
  }

  setPrimaryURL(url: string): BundleBuilder {
    return this.compatAdapter.setPrimaryURL(url);
  }

  setManifestURL(url: string): BundleBuilder {
    return this.compatAdapter.setManifestURL(url);
  }

  private addSection(name: string, content: CBORValue) {
    if (this.sectionLengths.some(s => s.name === name)) {
      throw new Error('Duplicated section: ' + name);
    }
    let length: number;
    if (name === 'responses') {
      // The responses section can be large, so avoid using encodedLength()
      // with the entire section's content.
      // Here, this.currentResponsesOffset holds the sum of all response
      // lengths. Adding encodedLength(this.responses.length) to this gives
      // the same result as encodedLength(content).
      length =
        this.currentResponsesOffset + encodedLength(this.responses.length);
    } else {
      length = encodedLength(content);
    }
    this.sectionLengths.push({ name, length });
    this.sections.push(content);
  }

  // Adds a response to `this.response`, and returns its length in the
  // responses section.
  private addResponse(headerMap: HeaderMap, payload: Uint8Array): number {
    if (payload.length > 0 && !headerMap.has('content-type')) {
      throw new Error('Non-empty exchange must have Content-Type header');
    }

    const response = [new Uint8Array(encodeCanonical(headerMap)), payload];
    this.responses.push(response);
    // This should be the same as encodedLength(response).
    return response.reduce(
      (len, buf) => len + encodedLength(buf.length) + buf.length,
      1
    );
  }

  private addIndexEntry(url: string, responseLength: number) {
    this.compatAdapter.setIndexEntry(url, responseLength);
    this.currentResponsesOffset += responseLength;
  }

  private fixupIndex() {
    // Adjust the offsets by the length of the response section's CBOR header.
    const responsesHeaderSize = encodedLength(this.responses.length);
    return this.compatAdapter.updateIndexValues(responsesHeaderSize);
  }

  private createTopLevel(): CBORValue {
    return this.compatAdapter.createTopLevel();
  }

  get formatVersion(): string {
    return this.compatAdapter.formatVersion;
  }

  // Behaviour that is specific to particular versions of the format.
  private createCompatAdapter(formatVersion: string): CompatAdapter {
    if (formatVersion === 'b1') {
      // format version b1
      return new class implements CompatAdapter {
        formatVersion: string = 'b1';
        private index: Map<string, [Uint8Array, number, number]> = new Map();
        private primaryURL: string | null = null;

        constructor(private bundleBuilder: BundleBuilder) {
        }

        onCreateBundle(): void {
          if (this.primaryURL === null) {
            throw new Error('Primary URL is not set');
          }
          if (!this.index.has(this.primaryURL)) {
            throw new Error(
              `Exchange for primary URL (${this.primaryURL}) does not exist`
            );
          }
        }

        setPrimaryURL(url: string): BundleBuilder {
          if (this.primaryURL !== null) {
            throw new Error('Primary URL is already set')
          }
          validateExchangeURL(url);
          this.primaryURL = url;
          return this.bundleBuilder;
        }

        setManifestURL(url: string): BundleBuilder {
          validateExchangeURL(url);
          this.bundleBuilder.addSection('manifest', url);
          return this.bundleBuilder;
        }

        setIndexEntry(url: string,
          responseLength: number): void {
          this.index.set(url, [
            new Uint8Array(0), // variants-value
            this.bundleBuilder.currentResponsesOffset,
            responseLength,
          ]);
        }

        updateIndexValues(responsesHeaderSize: number): Map<string, any[]> {
          for (const value of this.index.values()) {
            value[1] += responsesHeaderSize;
          }
          return this.index;
        }

        createTopLevel(): unknown {
          const sectionLengths: Array<string | number> = [];
          for (const s of this.bundleBuilder.sectionLengths) {
            sectionLengths.push(s.name, s.length);
          }
          return [
            byteString('🌐📦'),
            byteString(`${formatVersion}\0\0`),
            this.primaryURL,
            new Uint8Array(encodeCanonical(sectionLengths)),
            this.bundleBuilder.sections,
            new Uint8Array(8), // Length (to be filled in later)
          ];
        }
      }(this);
    } else {
      // format version b2
      return new class implements CompatAdapter {
        formatVersion: string = 'b2';
        private index: Map<string, [number, number]> = new Map();

        constructor(private bundleBuilder: BundleBuilder) {
        }

        onCreateBundle(): void {
          // not used
        }

        setPrimaryURL(url: string): BundleBuilder {
          validateExchangeURL(url);
          this.bundleBuilder.addSection('primary', url);
          return this.bundleBuilder;
        }

        setManifestURL(url: string): BundleBuilder {
          throw new Error('setManifestURL(): wrong format version');
        }

        setIndexEntry(url: string,
          responseLength: number): void {
          this.index.set(url, [
            this.bundleBuilder.currentResponsesOffset,
            responseLength,
          ]);
        }

        updateIndexValues(responsesHeaderSize: number): Map<string, any[]> {
          for (const value of this.index.values()) {
            value[0] += responsesHeaderSize;
          }
          return this.index;
        }

        createTopLevel(): unknown {
          const sectionLengths: Array<string | number> = [];
          for (const s of this.bundleBuilder.sectionLengths) {
            sectionLengths.push(s.name, s.length);
          }
          return [
            byteString('🌐📦'),
            byteString(`${formatVersion}\0\0`),
            new Uint8Array(encodeCanonical(sectionLengths)),
            this.bundleBuilder.sections,
            new Uint8Array(8), // Length (to be filled in later)
          ];
        }
      }(this);
    }
  }
}

class HeaderMap extends Map<string, string> {
  constructor(status: number, headers: Headers) {
    super();
    if (status < 100 || status > 999) {
      throw new Error('Invalid status code');
    }

    this.set(':status', status.toString());
    for (const key of Object.keys(headers)) {
      this.set(key.toLowerCase(), headers[key]);
    }
  }

  // This tells the CBOR library how to serialize this object.
  encodeCBOR(encoder: CBOR.Encoder) {
    // Convert keys and values to Uint8Array, as the CBOR representation of
    // header map is {bytestring => bytestring}.
    const m = new Map<Uint8Array, Uint8Array>();
    for (const [key, value] of this.entries()) {
      m.set(byteString(key), byteString(value));
    }
    return encoder.pushAny(m);
  }
}

// Throws an error if `urlString` is not a valid exchange URL, i.e. it must not
// have credentials (user:password@) or hash (#fragment).
function validateExchangeURL(urlString: string): void {
  // `urlString` can be relative, so try parsing it with a dummy base URL.
  const url = new URL(urlString, 'https://webbundle.example/');
  if (url.username !== '' || url.password !== '') {
    throw new Error('Exchange URL must not have credentials: ' + urlString);
  }
  if (url.hash !== '') {
    throw new Error('Exchange URL must not have a hash: ' + urlString);
  }
}

// Encodes `value` in canonical CBOR. Throws an error if the result is larger
// than `bufferSize`, because the CBOR encoder silently ignores write errors
// to its internal stream and returns broken data.
function encodeCanonical(value: CBORValue, bufferSize = 1024 * 1024): Buffer {
  const buf = CBOR.encodeOne(value, {
    canonical: true,
    highWaterMark: bufferSize,
  });
  if (buf.length >= bufferSize) {
    throw new Error('CBOR encode error: insufficient buffer size');
  }
  return buf;
}

// Returns the length of `value` when CBOR-encoded.
function encodedLength(value: CBORValue, bufferSize = 1024 * 1024): number {
  // We don't need to use canonical encoding here.
  const len = CBOR.encodeOne(value, { highWaterMark: bufferSize }).byteLength;
  if (len >= bufferSize) {
    throw new Error('CBOR encode error: insufficient buffer size');
  }
  return len;
}

function byteString(s: string): Uint8Array {
  return Buffer.from(s, 'utf-8');
}
