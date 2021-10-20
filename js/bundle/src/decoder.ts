import * as CBOR from 'cbor';

interface Headers {
  [key: string]: string;
}

const knownSections = [
  'critical',
  'index',
  'manifest', // only defined in version b1, might be present anyway
  'responses',
  'signatures',
];

interface CompatAdapter {
  wbnLength: number;
  getResponse(url: string): Response;
  destructureBundle(wbn: unknown[]): [any, any, any, any, any, any];
}

/** This class represents parsed Web Bundle. */
export class Bundle {
  version: string;
  private b1PrimaryURL: string | null = null; // only valid in format version b1
  private sections: { [key: string]: unknown } = {};
  private responses: { [key: number]: Response } = {}; // Offset-in-responses -> resp
  private compatAdapter: CompatAdapter;

  constructor(buffer: Buffer) {
    const wbn = asArray(CBOR.decode(buffer));

    let peekVersion = bytestringToString(wbn[1]).replace(/\0+$/, '');
    this.compatAdapter = this.createCompatAdapter(peekVersion);

    if (wbn.length !== this.compatAdapter.wbnLength) {
      throw new Error(`Wrong toplevel structure ${peekVersion} ${wbn.length} ${this.compatAdapter.wbnLength}`);
    }
    const [
      magic,
      version,
      primaryURL, // null in format version b2
      sectionLengthsCBOR,
      sections,
      length,
    ] = this.compatAdapter.destructureBundle(wbn);
    if (bytestringToString(magic) !== '🌐📦') {
      throw new Error('Wrong magic');
    }
    this.version = bytestringToString(version).replace(/\0+$/, ''); // Strip off the '\0' paddings.
    if (primaryURL) {
      this.b1PrimaryURL = asString(primaryURL);
    }
    const sectionLengths = asArray(
      CBOR.decode(asBytestring(sectionLengthsCBOR))
    );
    const sectionsArray = asArray(sections);
    if (sectionLengths.length !== sectionsArray.length * 2) {
      throw new Error(
        "Number of elements in section-lengths and in sections don't match"
      );
    }
    for (let i = 0; i < sectionsArray.length; i++) {
      this.sections[asString(sectionLengths[i * 2])] = sectionsArray[i];
    }

    if (this.sections['critical']) {
      for (const name of asArray(this.sections['critical'])) {
        if (!knownSections.includes(asString(name))) {
          throw new Error(`unknown section ${name} is marked as critical`);
        }
      }
    }

    // The index section records (offset, length) of each response, but our
    // CBOR decoder doesn't preserve location information. So, recalculate
    // offset and length of each response here. This is inefficient, but works.
    const responses = asArray(this.sections['responses']);
    let offsetInResponses = encodedLength(responses.length);
    for (const resp of responses) {
      this.responses[offsetInResponses] = new Response(asArray(resp));
      offsetInResponses += encodedLength(resp);
    }
  }

  get manifestURL(): string | null {
    // the manifest section is not part of the b2 spec but it might have been defined anyway
    if (this.sections['manifest']) {
      return asString(this.sections['manifest']);
    }
    return null;
  }

  get urls(): string[] {
    return Object.keys(this.indexSection);
  }

  get primaryURL(): string | null {
    if (this.version === 'b1') {
      return this.b1PrimaryURL;
    } else {
      // TODO format version does not support a primary URL, should we throw an error?
      return null;
    }
  }

  getResponse(url: string): Response {
    return this.compatAdapter.getResponse(url);
  }

  private get indexSection(): { [key: string]: unknown } {
    return asMap(this.sections['index']);
  }

  // Behaviour that is specific to particular versions of the format.
  private createCompatAdapter(formatVersion: string): CompatAdapter {
    if (formatVersion === 'b1') {
      // format version b1
      return new class implements CompatAdapter {
        wbnLength: number = 6;

        constructor(private bundle: Bundle) {
        }

        getResponse(url: string): Response {
          const indexEntry = asArray(this.bundle.indexSection[url]);
          if (!indexEntry) {
            throw new Error('No entry for ' + url);
          }
          const [variants, offset, length] = indexEntry;
          if (asBytestring(variants).length !== 0) {
            throw new Error('Variants are not supported');
          }
          if (indexEntry.length !== 3) {
            throw new Error('Unexpected length of index entry for ' + url);
          }
          const resp = this.bundle.responses[asNumber(offset)];
          if (!resp) {
            throw new Error(`Response for ${url} is not found (broken index)`);
          }
          return resp;
        }

        destructureBundle(wbn: unknown[]): [any, any, any, any, any, any] {
          return [wbn[0], wbn[1], wbn[2], wbn[3], wbn[4], wbn[5]];
        }
      }(this);
    } else {
      // format version b2
      return new class implements CompatAdapter {
        wbnLength: number = 5;

        constructor(private bundle: Bundle) {
        }

        getResponse(url: string): Response {
          const indexEntry = asArray(this.bundle.indexSection[url]);
          if (!indexEntry) {
            throw new Error('No entry for ' + url);
          }
          const [offset, length] = indexEntry;
          if (indexEntry.length !== 2) {
            throw new Error('Unexpected length of index entry for ' + url);
          }
          const resp = this.bundle.responses[asNumber(offset)];
          if (!resp) {
            throw new Error(`Response for ${url} is not found (broken index)`);
          }
          return resp;
        }

        destructureBundle(wbn: unknown[]): [any, any, any, any, any, any] {
          return [wbn[0], wbn[1], null, wbn[2], wbn[3], wbn[4]];
        }
      }(this);
    }
  }
}

/** This class represents an HTTP resource in Web Bundle. */
export class Response {
  status: number;
  headers: Headers;
  body: Buffer;

  constructor(responsesSectionItem: unknown[]) {
    if (responsesSectionItem.length !== 2) {
      throw new Error('Wrong response structure');
    }
    const { status, headers } = decodeResponseMap(
      asBytestring(responsesSectionItem[0])
    );
    this.status = status;
    this.headers = headers;
    this.body = asBytestring(responsesSectionItem[1]);
  }
}

function encodedLength(value: unknown): number {
  return CBOR.encode(value).byteLength;
}

function decodeResponseMap(cbor: Buffer): { status: number; headers: Headers } {
  const decoded = CBOR.decode(cbor);
  if (!(decoded instanceof Map)) {
    throw new Error('Wrong header map structure');
  }
  let status: number | null = null;
  const headers: Headers = {};
  for (let [key, val] of decoded.entries()) {
    key = bytestringToString(key);
    val = bytestringToString(val);
    if (key === ':status') {
      status = Number(val);
    } else if (key.startsWith(':')) {
      throw new Error('Unknown psuedo header ' + key);
    } else {
      headers[key] = val;
    }
  }
  if (!status) {
    throw new Error('No :status in response header map');
  }
  return { status, headers };
}

// Type assertions and conversions for CBOR-decoded objects.

function asArray(x: unknown): unknown[] {
  if (x instanceof Array) {
    return x;
  }
  throw new Error('Array expected, but got ' + typeof x);
}

function asMap(x: unknown): { [key: string]: unknown } {
  if (typeof x === 'object' && x !== null && !(x instanceof Array)) {
    return x as { [key: string]: unknown };
  }
  throw new Error('Map expected, but got ' + typeof x);
}

function asNumber(x: unknown): number {
  if (typeof x === 'number') {
    return x;
  }
  throw new Error('Number expected, but got ' + typeof x);
}

function asString(x: unknown): string {
  if (typeof x === 'string') {
    return x;
  }
  throw new Error('String expected, but got ' + typeof x);
}

function asBytestring(x: unknown): Buffer {
  if (x instanceof Buffer) {
    return x;
  }
  throw new Error('Bytestring expected, but got ' + typeof x);
}

function bytestringToString(bstr: unknown): string {
  if (!(bstr instanceof Buffer)) {
    throw new Error('Bytestring expected');
  }
  return bstr.toString('utf-8');
}
