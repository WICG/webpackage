import * as CBOR from 'cbor';

declare module 'cbor' {
    // TODO: upstream this to @types/cbor
    export function encodeCanonical(input: any): Buffer;
}

type CBORValue = any;

export class BundleBuilder {
    private sectionLengths: (string|number)[] = [];
    private sections: CBORValue[] = [];
    private responses: Uint8Array[][] = [];
    private index: {[key:string]: [Uint8Array, number, number]} = {};
    private currentResponsesOffset = 0;

    constructor(private primaryURL: string) {
        validateExchangeURL(primaryURL);
    }

    createBundle(): Buffer {
        if (!this.index[this.primaryURL]) {
            throw new Error(`Exchange for primary URL (${this.primaryURL}) does not exist`);
        }

        this.addSection("index", this.fixupIndex());
        this.addSection("responses", this.responses);
        let wbn = CBOR.encodeCanonical(this.createTopLevel());
        // Fill in the length field.
        let view = new DataView(wbn.buffer, wbn.byteOffset + wbn.length - 8);
        view.setUint32(0, Math.floor(wbn.length / 0x100000000));
        view.setUint32(4, wbn.length & 0xffffffff);
        return wbn;
    }

    addExchange(url: string, status: number, headers: {[key:string]: string}, payload: Uint8Array|string) {
        validateExchangeURL(url);
        if (typeof payload === 'string')
            payload = byteString(payload);
        this.addIndexEntry(url, this.addResponse(new HeaderMap(status, headers), payload));
    }

    setManifestURL(url: string) {
        validateExchangeURL(url);
        this.addSection('manifest', url);
    }

    private addSection(name: string, content: CBORValue) {
        if (this.sectionLengths.includes(name)) {
            throw new Error('Duplicated section: ' + name);
        }
        this.sectionLengths.push(name);
        this.sectionLengths.push(encodedLength(content));
        this.sections.push(content);
    }

    private addResponse(headerMap: HeaderMap, payload: Uint8Array): number {
        if (payload.length > 0 && !headerMap.has('content-type')) {
            throw new Error('Non-empty exchange must have Content-Type header');
        }

        let response = [new Uint8Array(CBOR.encodeCanonical(headerMap)), payload];
        this.responses.push(response);
        return encodedLength(response);
    }

    private addIndexEntry(url: string, responseLength: number) {
        this.index[url] = [
            new Uint8Array(0), // variants-value
            this.currentResponsesOffset,
            responseLength,
        ];
        this.currentResponsesOffset += responseLength;
    }

    private fixupIndex() {
        // Adjust the offsets by the length of the response section's CBOR header.
        let responsesHeaderSize = encodedLength(this.responses.length);
        for (let key in this.index) {
            this.index[key][1] += responsesHeaderSize;
        }
        return this.index;
    }

    private createTopLevel(): CBORValue {
        return [
            byteString('🌐📦'),
            byteString('b1\0\0'),
            this.primaryURL,
            new Uint8Array(CBOR.encodeCanonical(this.sectionLengths)),
            this.sections,
            new Uint8Array(8),  // Length (to be filled in later)
        ];
    }
};

class HeaderMap extends Map<string, string> {
    constructor(status: number, headers: {[key:string]: string}) {
        super();
        if (status < 100 || status > 999) {
            throw new Error('Invalid status code');
        }

        this.set(':status', status.toString());
        for (let key in headers) {
            this.set(key.toLowerCase(), headers[key]);
        }
    }

    // This tells the CBOR library how to serialize this object.
    encodeCBOR(encoder: CBOR.Encoder) {
        // Convert keys and values to Uint8Array, as the CBOR representation of
        // header map is {bytestring => bytestring}.
        let m = new Map<Uint8Array, Uint8Array>();
        for (let [key, value] of this.entries()) {
            m.set(byteString(key), byteString(value));
        }
        return encoder.pushAny(m);
    }
}

// Throws an error if `urlString` is not a valid exchange URL.
function validateExchangeURL(urlString: string): void {
    let url = new URL(urlString);
    if (url.username !== '' || url.password !== '') {
        throw new Error('Exchange URL must not have credentials: ' + urlString);
    }
    if (url.hash !== '') {
        throw new Error('Exchange URL must not have a hash: ' + urlString);
    }
}

// Returns the length of `value` when CBOR-encoded.
function encodedLength(value: any): number {
    // We don't need to use canonical encoding here.
    return CBOR.encode(value).byteLength;
}

function byteString(s: string): Uint8Array {
    return Buffer.from(s, 'utf-8');
}
