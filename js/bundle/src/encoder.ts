import * as CBOR from 'cbor';
import {URL} from 'url';

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

export class BundleBuilder {
  private sectionLengths: Array<{ name: string; length: number }> = [];
  private sections: CBORValue[] = [];
  private responses: Uint8Array[][] = [];
  private index: Map<string, [Uint8Array, number, number]> = new Map();
  private currentResponsesOffset = 0;

  constructor(private primaryURL: string) {
    validateExchangeURL(primaryURL);
  }

  // TODO: Provide async version of this.
  createBundle(): Buffer {
    if (!this.index.has(this.primaryURL)) {
      throw new Error(
        `Exchange for primary URL (${this.primaryURL}) does not exist`
      );
    }

    this.addSection('index', this.fixupIndex());
    this.addSection('responses', this.responses);

    const estimatedBundleSize = this.sectionLengths.reduce(
      (size, s) => size + s.length,
      16384 // For headers (including primary URL)
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
  ) {
    validateExchangeURL(url);
    if (typeof payload === 'string') {
      payload = byteString(payload);
    }
    this.addIndexEntry(
      url,
      this.addResponse(new HeaderMap(status, headers), payload)
    );
  }

  setManifestURL(url: string) {
    validateExchangeURL(url);
    this.addSection('manifest', url);
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
    this.index.set(url, [
      new Uint8Array(0), // variants-value
      this.currentResponsesOffset,
      responseLength,
    ]);
    this.currentResponsesOffset += responseLength;
  }

  private fixupIndex() {
    // Adjust the offsets by the length of the response section's CBOR header.
    const responsesHeaderSize = encodedLength(this.responses.length);
    for (const value of this.index.values()) {
      value[1] += responsesHeaderSize;
    }
    return this.index;
  }

  private createTopLevel(): CBORValue {
    const sectionLengths: Array<string | number> = [];
    for (const s of this.sectionLengths) {
      sectionLengths.push(s.name, s.length);
    }
    return [
      byteString('🌐📦'),
      byteString('b1\0\0'),
      this.primaryURL,
      new Uint8Array(encodeCanonical(sectionLengths)),
      this.sections,
      new Uint8Array(8), // Length (to be filled in later)
    ];
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

// Throws an error if `urlString` is not a valid exchange URL, i.e. it must:
// - be absolute,
// - have http: or https: protocol, and
// - have no credentials (user:password@) or hash (#fragment).
function validateExchangeURL(urlString: string): void {
  const url = new URL(urlString);
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new Error("Exchange URL's protocol must be http(s): " + urlString);
  }
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
