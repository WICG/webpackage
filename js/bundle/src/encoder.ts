import * as cborg from 'cborg';
import { encodedLength } from 'cborg/length';
import {
  isApprovedVersion,
  B1,
  B2,
  DEFAULT_VERSION,
  FormatVersion,
} from './constants.js';

type CBORValue = unknown;
interface Headers {
  [key: string]: string;
}
type OverrideHeadersFunction = (filepath: string) => Headers;
type OverrideHeadersOption = Headers | OverrideHeadersFunction | undefined;

interface CompatAdapter {
  formatVersion: FormatVersion;
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
  private overrideHeadersOption: OverrideHeadersOption;

  constructor(
    formatVersion: FormatVersion = DEFAULT_VERSION,
    overrideHeadersOption?: OverrideHeadersOption
  ) {
    if (!isApprovedVersion(formatVersion)) {
      throw new Error(`Invalid webbundle format version`);
    }
    this.compatAdapter = this.createCompatAdapter(formatVersion);
    this.overrideHeadersOption = overrideHeadersOption;
  }

  createBundle(): Uint8Array {
    this.compatAdapter.onCreateBundle();

    this.addSection('index', this.fixupIndex());
    this.addSection('responses', this.responses);

    const wbn = cborg.encode(this.createTopLevel());

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
    const overrideHeaders = this.getOverrideHeaders(url);

    validateExchangeURL(url);
    if (typeof payload === 'string') {
      payload = byteString(payload);
    }
    this.addIndexEntry(
      url,
      this.addResponse(
        new HeaderMap(status, getCombinedHeaders(headers, overrideHeaders)),
        payload
      )
    );
    return this;
  }

  private getOverrideHeaders(url: string): Headers {
    if (!this.overrideHeadersOption) return {};

    return typeof this.overrideHeadersOption == 'function'
      ? this.overrideHeadersOption(url)
      : this.overrideHeadersOption;
  }

  setPrimaryURL(url: string): BundleBuilder {
    return this.compatAdapter.setPrimaryURL(url);
  }

  setManifestURL(url: string): BundleBuilder {
    return this.compatAdapter.setManifestURL(url);
  }

  private addSection(name: string, content: CBORValue) {
    if (this.sectionLengths.some((s) => s.name === name)) {
      throw new Error('Duplicated section: ' + name);
    }
    let length = encodedLength(content);
    this.sectionLengths.push({ name, length });
    this.sections.push(content);
  }

  // Adds a response to `this.response`, and returns its length in the
  // responses section.
  private addResponse(headerMap: HeaderMap, payload: Uint8Array): number {
    if (payload.length > 0 && !headerMap.has('content-type')) {
      throw new Error('Non-empty exchange must have Content-Type header');
    }

    const response = [headerMap.toCBOR(), payload];
    this.responses.push(response);
    return encodedLength(response);
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

  get formatVersion(): FormatVersion {
    return this.compatAdapter.formatVersion;
  }

  // Behaviour that is specific to particular versions of the format.
  private createCompatAdapter(formatVersion: FormatVersion): CompatAdapter {
    if (formatVersion === B1) {
      // format version b1
      return new (class implements CompatAdapter {
        formatVersion: FormatVersion = B1;
        private index: Map<string, [Uint8Array, number, number]> = new Map();
        private primaryURL: string | null = null;

        constructor(private bundleBuilder: BundleBuilder) {}

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
            throw new Error('Primary URL is already set');
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

        setIndexEntry(url: string, responseLength: number): void {
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
            cborg.encode(sectionLengths),
            this.bundleBuilder.sections,
            new Uint8Array(8), // Length (to be filled in later)
          ];
        }
      })(this);
    } else {
      // format version b2
      return new (class implements CompatAdapter {
        formatVersion: FormatVersion = B2;
        private index: Map<string, [number, number]> = new Map();

        constructor(private bundleBuilder: BundleBuilder) {}

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

        setIndexEntry(url: string, responseLength: number): void {
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
            cborg.encode(sectionLengths),
            this.bundleBuilder.sections,
            new Uint8Array(8), // Length (to be filled in later)
          ];
        }
      })(this);
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

  toCBOR(): Uint8Array {
    // Convert keys and values to Uint8Array, as the CBOR representation of
    // header map is {bytestring => bytestring}.
    const m = new Map<Uint8Array, Uint8Array>();
    for (const [key, value] of this.entries()) {
      m.set(byteString(key), byteString(value));
    }
    return cborg.encode(m);
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

function byteString(s: string): Uint8Array {
  return new TextEncoder().encode(s);
}

function headerNamesToLowerCase(headers: Headers) {
  const lowerCaseHeaders: Headers = {};
  for (const key of Object.keys(headers)) {
    lowerCaseHeaders[key.toLowerCase()] = headers[key];
  }
  return lowerCaseHeaders;
}

// Combines the headers in case no duplicates are found. Header names are also
// converted into lower case as they are case insensitive. Header values with
// differing casing are considered equal and the casing in the override takes
// action. If the header name is duplicate but the value is same, we don't
// throw an error and only add the header once.
export function getCombinedHeaders(headers: Headers, overrideHeaders: Headers) {
  const lowerCaseHeaders = headerNamesToLowerCase(headers);
  const lowerCaseOverrideHeaders = headerNamesToLowerCase(overrideHeaders);

  Object.keys(overrideHeaders).filter((overrideKey) => {
    const lowerCaseOverrideKey = overrideKey.toLowerCase();
    const isDuplicate =
      lowerCaseHeaders[lowerCaseOverrideKey] &&
      lowerCaseHeaders[lowerCaseOverrideKey].toLowerCase() !=
        overrideHeaders[overrideKey].toLowerCase();

    if (isDuplicate) {
      throw new Error(
        'Duplicate key with differing values was found: ' + overrideKey
      );
    }
    return isDuplicate;
  });

  return { ...lowerCaseHeaders, ...lowerCaseOverrideHeaders };
}
