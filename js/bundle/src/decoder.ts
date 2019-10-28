import * as CBOR from 'cbor';

export class Bundle {
    private version_: Uint8Array;
    private primaryURL_: string;
    private sections: {[key:string]: unknown} = {};
    private responses: {[key:number]: Response} = {};  // Offset-in-responses -> resp

    constructor(buffer: Buffer) {
        const wbn = asArray(CBOR.decode(buffer));
        if (wbn.length != 6) {
            throw new Error('Wrong toplevel structure');
        }
        const [magic, version, primaryURL, sectionLengthsCBOR, sections, length] = wbn;
        if (bytestringToString(magic) !== '🌐📦') {
            throw new Error('Wrong magic');
        }
        this.version_ = asBytestring(version);
        this.primaryURL_ = asString(primaryURL);
        const sectionLengths = asArray(CBOR.decode(asBytestring(sectionLengthsCBOR)))
        const sectionsArray = asArray(sections);
        if (sectionLengths.length !== sectionsArray.length * 2) {
            throw new Error("Number of elements in section-lengths and in sections don't match");
        }
        for (let i = 0; i < sectionsArray.length; i++) {
            this.sections[asString(sectionLengths[i*2])] = sectionsArray[i];
        }
        // The index section records (offset, length) of each response, but our
        // CBOR decoder doesn't preserve location information. So, recalculate
        // offset and length of each response here. This is inefficient, but works.
        const responses = asArray(this.sections['responses']);
        let offsetInResponses = encodedLength(responses.length);
        for (let resp of responses) {
            this.responses[offsetInResponses] = new Response(asArray(resp));
            offsetInResponses += encodedLength(resp);
        }
    }

    get version(): string {
        // Strip off the '\0' paddings.
        return bytestringToString(this.version_).replace(/\0+$/, '');
    }

    get primaryURL(): string {
        return this.primaryURL_;
    }

    get manifestURL(): string|null {
        if (this.sections['manifest']) {
            return asString(this.sections['manifest']);
        }
        return null;
    }

    get urls(): string[] {
        return Object.keys(this.indexSection);
    }

    getResponse(url: string): Response {
        let indexEntry = asArray(this.indexSection[url]);
        if (!indexEntry) {
            throw new Error('No entry for ' + url);
        }
        let [variants, offset, length] = indexEntry;
        if (asBytestring(variants).length !== 0) {
            throw new Error('Variants are not supported');
        }
        if (indexEntry.length !== 3) {
            throw new Error('Unexpected length of index entry for ' + url);
        }
        let resp = this.responses[asNumber(offset)];
        if (!resp) {
            throw new Error(`Response for ${url} is not found (broken index)`);
        }
        return resp;
    }

    private get indexSection(): {[key:string]: unknown} {
        return asMap(this.sections['index']);
    }
};

export class Response {
    public status: number;
    public headers: {[key:string]: string};
    public body: Buffer;

    constructor(responsesSectionItem: unknown[]) {
        if (responsesSectionItem.length != 2) {
            throw new Error('Wrong response structure');
        }
        let {status, headers} = decodeResponseMap(asBytestring(responsesSectionItem[0]));
        this.status = status;
        this.headers = headers;
        this.body = asBytestring(responsesSectionItem[1]);
    }
}

function encodedLength(value: any): number {
    return CBOR.encode(value).byteLength;
}

function decodeResponseMap(cbor: Buffer): {status: number, headers: {[key:string]: string}} {
    let decoded = CBOR.decode(cbor);
    if (!(decoded instanceof Map)) {
        throw new Error('Wrong header map structure');
    }
    let status: number | null = null;
    let headers: {[key:string]: string} = {};
    for (let [key, val] of decoded.entries()) {
        key = bytestringToString(key);
        val = bytestringToString(val);
        if (key === ':status') {
            status = parseInt(val);
        } else if (key.startsWith(':')) {
            throw new Error('Unknown psuedo header ' + key);
        } else {
            headers[key] = val;
        }
    }
    if (!status) {
        throw new Error('No :status in response header map');
    }
    return {status, headers};
}

// Type assertions and conversions for CBOR-decoded objects.

function asArray(x: unknown): unknown[] {
    if (x instanceof Array) {
        return x;
    }
    throw new Error('Array expected, but got ' + typeof x);
}

function asMap(x: unknown): {[key: string]: unknown} {
    if (typeof x === 'object' && x !== null && !(x instanceof Array)) {
        return x as {[key: string]: unknown};
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
