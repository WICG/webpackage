---
coding: utf-8

title: Origin-signed HTTP Responses
docname: draft-yasskin-http-origin-signed-responses-latest
category: std

ipr: trust200902

stand_alone: yes
pi: [comments, sortrefs, strict, symrefs, toc]

author:
 -
    name: Jeffrey Yasskin
    organization: Google
    email: jyasskin@chromium.org

normative:
  FETCH:
    target: https://fetch.spec.whatwg.org/
    title: Fetch
    author:
      org: WHATWG
    date: Living Standard
  POSIX:
    target: http://pubs.opengroup.org/onlinepubs/9699919799/basedefs/
    title: The Open Group Base Specifications Issue 7
    author:
      - org: IEEE
      - org: The Open Group
    seriesinfo:
      name: IEEE
      value: 1003.1-2008, 2016 Edition
    date: 2016

informative:
  SRI: W3C.REC-SRI-20160623

--- abstract

This document explores how a server can send particular responses that are
authoritative for an origin, when the server itself is not authoritative for
that origin. For now, the appendices containing use cases and requirements
should be treated as more confident than the proposal itself.

--- note_Note_to_Readers

Discussion of this draft takes place on the HTTP working group mailing list
(ietf-http-wg@w3.org), which is archived
at <https://lists.w3.org/Archives/Public/ietf-http-wg/>.

The source code and issues list for this draft can be found
in <https://github.com/WICG/webpackage>.

--- middle

# Introduction

When
I
[presented Web Packaging to DISPATCH](https://datatracker.ietf.org/doc/minutes-99-dispatch/),
folks thought it would make sense to split it into a way to sign individual HTTP
responses as coming from a particular origin, and separately a way to bundle a
collection of HTTP responses. This document explores the constraints on any
method of signing HTTP responses and sketches a possible solution to the
constraints.

# Terminology

Author
: The entity that controls the server for a particular origin {{?RFC6454}}. The
  author can get a CA to issue certificates for their private keys and can run a
  TLS server for their origin.

Exchange (noun)
: An HTTP request/response pair. This can either be a request from a client and
the matching response from a server or the request in a PUSH_PROMISE and its
matching response stream. Defined by Section 8 of {{!RFC7540}}.

Intermediate
: An entity that fetches signed HTTP exchanges from an author or another
  intermediate and forwards them to another intermediate or a client.

Client
: An entity that uses a signed HTTP exchange and needs to be able to prove that
  the author vouched for it as coming from its claimed origin.

Unix time
: Defined by {{POSIX}} [section
4.16](http://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap04.html#tag_04_16).

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL
NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED",
"MAY", and "OPTIONAL" in this document are to be interpreted as
described in BCP 14 {{!RFC2119}} {{!RFC8174}} when, and only when, they
appear in all capitals, as shown here.

# Straw proposal # {#proposal}

As a response to an HTTP request or as a Server Push (Section 8.2 of
{{!RFC7540}}) the server MAY include a `Signed-Headers` header field
({{signed-headers}}) identifying [significant](#significant-headers) header
fields and a `Signature` header field ({{signature-header}}) holding a list of
one or more parameterised signatures that vouch for the content of the response.

The client categorizes each signature as "valid" or "invalid" by validating that
signature with its certificate or public key and other metadata against the
significant headers and content ({{signature-validity}}). This validity then
informs higher-level protocols.

Each signature is parameterised with information to let a client fetch assurance
that a signed exchange is still valid, in the face of revoked certificates and
newly-discovered vulnerabilities. This assurance can be bundled back into the
signed exchange and forwarded to another client, which won't have to re-fetch
this validity information for some period of time.

## The Signed-Headers Header ## {#signed-headers}

The `Signed-Headers` header field identifies an ordered list of response header
fields to include in a signature. The request URL and response status are
included unconditionally. This allows a TLS-terminating intermediate to reorder
headers without breaking the signature. This *can* also allow the intermediate
to add headers that will be ignored by some higher-level protocols, but
{{signature-validity}} provides a hook to let other higher-level protocols
reject such insecure headers.

This header field appears once instead of being incorporated into the
signatures' parameters because the significant header fields need to be
consistent across all signatures of an exchange, to avoid forcing higher-level
protocols to merge the header field lists of valid signatures.

See {{how-much-to-sign}} for a discussion of why only the URL from the request
is included and not other request headers.

`Signed-Headers` is a Structured Header as defined by
{{!I-D.ietf-httpbis-header-structure}}. Its value MUST be a list (Section 4.8 of
{{!I-D.ietf-httpbis-header-structure}}) of lowercase strings (Section 4.2 of
{{!I-D.ietf-httpbis-header-structure}}) naming HTTP response header fields.
Pseudo-header field names (Section 8.1.2.1 of {{!RFC7540}}) MUST NOT appear in
this list.

Higher-level protocols SHOULD place requirements on the minimum set of headers
to include in the `Signed-Headers` header field.

## The Signature Header ## {#signature-header}

The `Signature` header field conveys a list of signatures for an exchange, each
one accompanied by information about how to determine the authority of and
refresh that signature. Each signature directly signs the significant headers of
the exchange and identifies one of those headers that enforces the integrity of
the exchange's payload.

The `Signature` header is a Structured Header as defined by
{{!I-D.ietf-httpbis-header-structure}}. Its value MUST be a list (Section 4.8 of
{{!I-D.ietf-httpbis-header-structure}}) of parameterised labels (Section 4.4 of
{{!I-D.ietf-httpbis-header-structure}}).

Each parameterised label MUST have parameters named "sig", "integrity",
"validityUrl", "date", and "expires", and either "certUrl" and "certSha256"
parameters or an "ed25519Key" parameter. This specification gives no meaning to
the label itself, which can be used as a human-readable identifier for the
signature (see {{parameterised-binary}}). The present parameters MUST have the
following values:

"sig"

: Binary content (Section 4.5 of {{!I-D.ietf-httpbis-header-structure}}) holding
  the signature of most of these parameters and the significant headers of the
  exchange ({{significant-headers}}).

"integrity"

: A string (Section 4.2 of {{!I-D.ietf-httpbis-header-structure}}) containing
  the lowercase name of the response header field that guards the response
  payload's integrity.

"certUrl"

: A string (Section 4.2 of {{!I-D.ietf-httpbis-header-structure}}) containing a
  [valid URL string](https://url.spec.whatwg.org/#valid-url-string).

"certSha256"

: Binary content (Section 4.5 of {{!I-D.ietf-httpbis-header-structure}}) holding
  the SHA-256 hash of the first certificate found at "certUrl".

"ed25519Key"

: Binary content (Section 4.5 of {{!I-D.ietf-httpbis-header-structure}}) holding
  an Ed25519 public key ({{!RFC8032}}).

{:#signature-validityurl} "validityUrl"

: A string (Section 4.2 of {{!I-D.ietf-httpbis-header-structure}}) containing a
  [valid URL string](https://url.spec.whatwg.org/#valid-url-string).

"date" and "expires"

: An unsigned integer (Section 4.1 of {{!I-D.ietf-httpbis-header-structure}})
  representing a Unix time.

The "certUrl" and "validityUrl" parameters are *not* signed, so intermediates can
update them with pointers to cached versions.

### Open Questions

{{?I-D.ietf-httpbis-header-structure}} provides a way to parameterise labels but
not other supported types like binary content. If the `Signature` header field
is notionally a list of parameterised signatures, maybe we should add a
"parameterised binary content" type.
{:#parameterised-binary}

Should the certUrl and validityUrl be lists so that intermediates can offer a
cache without losing the original URLs? Putting lists in dictionary fields is
more complex than {{?I-D.ietf-httpbis-header-structure}} allows, so they're
single items for now.

Should "validityUrl" be signed or optionally signed so that an exchange's author
can prevent an intermediate from removing it, which would prevent clients from
sharing the exchange among themselves without going back to the intermeidate?

## Significant headers of an exchange ## {#significant-headers}

The significant headers of an exchange are:

* The method (Section 4 of {{!RFC7231}}) and effective request URI (Section 5.5
  of {{!RFC7230}}) of the request.
* The response status code (Section 6 of {{!RFC7231}}) and the response header
  fields whose names are listed in that exchange's `Signed-Headers` header field
  ({{signed-headers}}), in the order they appear in that header field. If a
  response header field name from `Signed-Headers` does not appear in the
  exchange's response header fields, the exchange has no significant headers.

If the exchange's `Signed-Headers` header field is not present, doesn't parse as
a Structured Header ({{!I-D.ietf-httpbis-header-structure}}) or doesn't follow
the constraints on its value described in {{signed-headers}}, the exchange has
no significant headers.

### Open Questions

Do the significant headers of an exchange need to include the `Signed-Headers`
header field itself?

## CBOR representation of exchange headers ## {#cbor-representation}

To sign an exchange's headers, they need to be serialized into a byte string.
Since intermediaries and [distributors](#uc-explicit-distributor) might
rearrange, add, or just reserialize headers, we can't use the literal bytes of
the headers as this serialization. Instead, this section defines a CBOR
representation that can be embedded into other CBOR, canonically serialized
({{canonical-cbor}}), and then signed.

The CBOR representation of an exchange `exchange`'s headers is the CBOR
({{!RFC7049}}) array with the following content:

1. The map mapping:
   * The byte string ':method' to the byte string containing `exchange`'s
     request's method.
   * The byte string ':url' to the byte string containing `exchange`'s request's
     effective request URI.
1. The map mapping:
   * the byte string ':status' to the byte string containing `exchange`'s
     response's 3-digit status code, and
   * for each response header field in `exchange`, the header field's name as a
     byte string to the header field's value as a byte string.

### Example ### {#example-cbor-representation}

Given the HTTP exchange:

~~~http
GET https://example.com/ HTTP/1.1
Accept: */*

HTTP/1.1 200
Content-Type: text/html
Digest: SHA-256=20addcf7368837f616d549f035bf6784ea6d4bf4817a3736cd2fc7a763897fe3
Signed-Headers: "content-type", "digest"

<!doctype html>
<html>
...
~~~

The cbor representation consists of the following item, represented using the
extended diagnostic notation from {{?I-D.ietf-cbor-cddl}} appendix G:

~~~cbor-diag
[
  {
    ':url': 'https://example.com/'
    ':method': 'GET',
  },
  {
    'digest': 'SHA-256=20addcf7368837f616d549f035bf6784ea6d4bf4817a3736cd2fc7a763897fe3',
    ':status': '200',
    'content-type': 'text/html'
  }
]
~~~

## Canonical CBOR serialization ## {#canonical-cbor}

Within this specification, the canonical serialization of a CBOR item uses the
following rules derived from Section 3.9 of {{?RFC7049}} with erratum 4964
applied:

* Integers and the lengths of arrays, maps, and strings MUST use the smallest
  possible encoding.
* Items MUST NOT be encoded with indefinite length.
* The keys in every map MUST be sorted in the bytewise lexicographic order of
  their canonical encodings. For example, the following keys are correctly sorted:
  1. 10, encoded as 0A.
  1. 100, encoded as 18 64.
  1. -1, encoded as 20.
  1. "z", encoded as 61 7A.
  1. "aa", encoded as 62 61 61.
  1. \[100], encoded as 81 18 64.
  1. \[-1], encoded as 81 20.
  1. false, encoded as F4.

Note: this specification does not use floating point, tags, or other more
complex data types, so it doesn't need rules to canonicalize those.

## Signature validity ## {#signature-validity}

The client MUST parse the `Signature` header field as the list of parameterised
values described in {{signature-header}} (Section 4.8.1 of
{{!I-D.ietf-httpbis-header-structure}}). If an error is thrown during this
parsing, the exchange has no valid signatures. Otherwise, each member of this
list represents a signature with parameters.

The client MUST use the following algorithm to determine whether each signature
with parameters is invalid or potentially-valid. Potentially-valid results
include:

* The signed headers of the exchange so that higher-level protocols can avoid
  relying on unsigned headers, and
* Either a certificate chain or a public key so that a higher-level protocol can
  determine whether it's actually valid.

This algorithm accepts a `forceFetch` flag that avoids the cache when fetching
URLs. A client that determines that a potentially-valid certificate chain is
actually invalid due to an expired OCSP response MAY retry with `forceFetch` set
to retrieve an updated OCSP from the original server.
{:#force-fetch}

This algorithm also accepts an `allResponseHeaders` flag, which insists that
there are no non-significant response header fields in the exchange.

1. Let `originalExchange` be the signature's exchange.
1. Let `headers` be the significant headers ({{significant-headers}}) of
   `originalExchange`. If `originalExchange` has no significant headers, then
   return "invalid".
1. Let `payload` be the payload body (Section 3.3 of {{!RFC7230}}) of
   `originalExchange`. Note that the payload body is the message body with any
   transfer encodings removed.
1. If `allResponseHeaders` is set and the response header fields in
   `originalExchange` are a proper superset of the response header fields in
   `headers`, then return "invalid".
1. Let:
   * `signature` be the signature (binary content in the parameterised value's
     "sig" parameter).
   * `integrity` be the signature's "integrity" parameter.
   * `certUrl` be the signature's "certUrl" parameter, if any.
   * `certSha256` be the signature's "certSha256" parameter, if any.
   * `ed25519Key` be the signature's "ed25519Key" parameter, if any.
   * `date` be the signature's "date" parameter, interpreted as a Unix time.
   * `expires` be the signature's "expires" parameter, interpreted as a Unix
     time.
1. If `integrity` names a header field that is not present in `headers` or which
   the client cannot use to check the integrity of `payload` (for example, the
   header field is new and hasn't been implemented yet), then return "invalid".
   Clients MUST implement at least the `Digest` ({{!RFC3230}}) and `MI`
   ({{!I-D.thomson-http-mice}}) header fields.
1. If `integrity` is "digest", and the `Digest` header field in `headers`
   contains no digest-algorithms
   (<https://www.iana.org/assignments/http-dig-alg/http-dig-alg.xhtml>) stronger
   than `SHA`, then return "invalid".
1. Set `publicKey` and `signing-alg` depending on which key fields are present:
   1. If `certUrl` is present:
      1. Let `certificate-chain` be the result of fetching ({{FETCH}}) `certUrl`
         and parsing it as a TLS 1.3 Certificate message (Section 4.4.2 of
         {{!I-D.ietf-tls-tls13}}) containing X.509v3 certificates. If
         `forceFetch` is *not* set, the fetch can be fulfilled from a cache
         using normal HTTP semantics {{!RFC7234}}. If this fetch or parse fails,
         return "invalid".

         Parsing notes:
         1. This does not include the 4-byte header that would appear in a
            Handshake message.
         1. Since this fetch is not in response to a CertificateRequest, the
            certificate_request_context MUST be empty, and a non-empty value
            MUST cause the parse to fail.
      1. Let `main-certificate` be the first certificate in `certificate-chain`.
      1. If the SHA-256 hash of `main-certificate`'s `cert_data` is not equal to
         `certSha256`, return "invalid". Note that this intentionally differs
         from TLS 1.3, which signs the entire certificate chain in its
         Certificate Verify (Section 4.4.3 of {{?I-D.ietf-tls-tls13}}), in order
         to allow updating the stapled OCSP response without updating signatures
         at the same time.
      1. Set `publicKey` to `main-certificate`'s public key
      1. The client MUST define a partial function from public key types to
         signing algorithms, and this function must at the minimum include the
         following mappings:

         RSA, 2048 bits:
         : rsa_pss_sha256 as defined in Section 4.2.3 of
           {{!I-D.ietf-tls-tls13}}.

         EC, with the secp256r1 curve:
         : ecdsa_secp256r1_sha256 as defined in Section 4.2.3 of
           {{!I-D.ietf-tls-tls13}}.

         EC, with the secp384r1 curve:
         : ecdsa_secp384r1_sha384 as defined in Section 4.2.3 of
           {{!I-D.ietf-tls-tls13}}.

         Set `signing-alg` to the result of applying this function to type of
         `main-certificate`'s public key. If the function is undefined on this
         input, return "invalid".
   1. If `ed25519Key` is present, set `publicKey` to `ed25519Key` and
      `signing-alg` to ed25519, as defined by {{!RFC8032}}
1. If `expires` is more than 7 days (604800 seconds) after `date`, return
   "invalid".
1. If the current time is before `date` or after `expires`, return "invalid".
1. Let `message` be the concatenation of the following byte strings. This
   matches the {{?I-D.ietf-tls-tls13}} format to avoid cross-protocol attacks
   when TLS certificates are used to sign manifests.
   1. A string that consists of octet 32 (0x20) repeated 64 times.
   1. A context string: the ASCII encoding of “HTTP Exchange”.
   1. A single 0 byte which serves as a separator.
   1. The bytes of the canonical CBOR serialization ({{canonical-cbor}}) of a
      CBOR map mapping:
      1. If `certSha256` is set:
         1. The text string "certSha256" to the byte string `certSha256`.
      1. The text string "date" to the integer value of `date`.
      1. The text string "expires" to the integer value of `expires`.
      1. The text string "headers" to the CBOR representation
         ({{cbor-representation}}) of `exchange`'s headers.
1. If `signature` is `message`'s signature by `main-certificate`'s public key
   using `signing-alg`, return "potentially-valid" with `exchange` and whichever
   is present of `certificate-chain` or `ed25519Key`. Otherwise, return
   "invalid".

   Note that the above algorithm can determine that an exchange's headers are
   potentially-valid before the exchange's payload is received. The client MAY
   process those headers as soon as they are validated. If `integrity`
   identifies a header field like `MI` ({{?I-D.thomson-http-mice}}) that can
   incrementally validate the payload, the client MAY also incrementally process
   the validated parts of the payload as soon as they are validated. The client
   MUST NOT process any part of the headers or payload before it has been
   validated either by the signature or the header field named by `integrity`.

## Updating signature validity ## {#updating-validity}

Both OCSP responses and signatures are designed to expire a short
time after they're signed, so that revoked certificates and signed exchanges
with known vulnerabilities are distrusted promptly.

This specification provides no way to update OCSP responses by themselves.
Instead, [clients need to re-fetch the "certUrl"](#force-fetch) to get a chain
including a newer OCSP response.

The ["validityUrl" parameter](#signature-validityurl) of the signatures provides
a way to fetch new signatures or learn where to fetch a complete updated
package.

Each version of a signed exchange SHOULD have its own validity URLs, since each
version needs different signatures and becomes obsolete at different times.

The resource at a "validityUrl" is "validity data", a CBOR map matching the
following CDDL ({{!I-D.ietf-cbor-cddl}}):

~~~cddl
validity = {
  ? signatures: [ + bytes ]
  ? update: {
    url: text,
    ? size: uint,
  }
]
~~~

The elements of the `signatures` array are header field values meant to replace
the signatures within the `Signature` header field pointing to this validity
data. If the signed exchange contains a bug severe enough that clients need to
stop using the content, the `signatures` array MUST NOT be present.

The `update` map gives a location to update the entire signed exchange and an
estimate of the size of the resource at that URL. If the signed exchange is
currently the most recent version, the `update` SHOULD NOT be present.

If both the `signatures` and `update` fields are present, clients can use the
estimated size to decide whether to update the whole resource or just its
signatures.

### Examples ### {#examples-updating-validity}

For example, if a signed exchange has the following `Signature` header field (written as multiple fields for convenience):

~~~http
Signature: sig1;
  sig=*MEUCIQDXlI2gN3RNBlgFiuRNFpZXcDIaUpX6HIEwcZEc0cZYLAIga9DsVOMM+g5YpwEBdGW3sS+bvnmAJJiSMwhuBdqp5UY;
  integrity="mi";
  validityUrl="https://example.com/resource.validity";
  certUrl="https://example.com/certs";
  certSha256=*W7uB969dFW3Mb5ZefPS9Tq5ZbH5iSmOILpjv2qEArmI;
  date=1511128380; expires=1511560380
Signature: sig2;
  sig=*MEQCIGjZRqTRf9iKNkGFyzRMTFgwf/BrY2ZNIP/dykhUV0aYAiBTXg+8wujoT4n/W+cNgb7pGqQvIUGYZ8u8HZJ5YH26Qg;
  integrity="mi";
  validityUrl="https://example.com/resource.validity";
  certUrl="https://example.com/certs";
  certSha256=*kQAA8u33cZRTy7RHMO4+dv57baZL48SYA2PqmYvPPbg;
  date=1511301183; expires=1511905983
Signature: sig3;
  sig=*MEYCIQCNxJzn6Rh2fNxsobktir8TkiaJYQFhWTuWI1i4PewQaQIhAMs2TVjc4rTshDtXbgQEOwgj2mRXALhfXPztXgPupii+;
  integrity="mi";
  validityUrl="https://thirdparty.example.com/resource.validity";
  certUrl="https://thirdparty.example.com/certs";
  certSha256=*UeOwUPkvxlGRTyvHcsMUN0A2oNsZbU8EUvg8A9ZAnNc;
  date=1511301183; expires=1511905983
~~~

https://example.com/resource.validity might contain:

~~~cbor-diag
{
  "signatures": [
    'sig4; '
    'sig=*MEQCIC/I9Q+7BZFP6cSDsWx43pBAL0ujTbON/+7RwKVk+ba5AiB3FSFLZqpzmDJ0NumNwN04pqgJZE99fcK86UjkPbj4jw; '
    'integrity="mi"; '
    'validityUrl="https://example.com/resource.validity"; '
    'certUrl="https://example.com/certs"; '
    'certSha256=*W7uB969dFW3Mb5ZefPS9Tq5ZbH5iSmOILpjv2qEArmI; '
    'date=1511467200; expires=1511985600'
  ],
  "update": {
    "url": "https://example.com/resource",
    "size": 5557452
  }
}
~~~

This indicates that the first two of the original signatures (the ones with a
validityUrl of "https://example.com/resource.validity") can be replaced with a
single new signature. The signatures of the updated signed
exchange would be:

~~~http
Signature: sig4;
  sig=*MEQCIC/I9Q+7BZFP6cSDsWx43pBAL0ujTbON/+7RwKVk+ba5AiB3FSFLZqpzmDJ0NumNwN04pqgJZE99fcK86UjkPbj4jw;
  integrity="mi";
  validityUrl="https://example.com/resource.validity";
  certUrl="https://example.com/certs";
  certSha256=*W7uB969dFW3Mb5ZefPS9Tq5ZbH5iSmOILpjv2qEArmI;
  date=1511467200; expires=1511985600
Signature: sig3;
  sig=*MEYCIQCNxJzn6Rh2fNxsobktir8TkiaJYQFhWTuWI1i4PewQaQIhAMs2TVjc4rTshDtXbgQEOwgj2mRXALhfXPztXgPupii+;
  integrity="mi";
  validityUrl="https://thirdparty.example.com/resource.validity";
  certUrl="https://thirdparty.example.com/certs";
  certSha256=*UeOwUPkvxlGRTyvHcsMUN0A2oNsZbU8EUvg8A9ZAnNc;
  date=1511301183; expires=1511905983
~~~

https://example.com/resource.validity could also expand the set of signatures if
its `signatures` array contained more than 2 elements.

# HTTP/2 extension for cross-origin Server Push # {#cross-origin-push}

To allow servers to Server-Push (Section 8.2 of {{?RFC7540}}) signed exchanges
({{proposal}}) signed by an authority for which the server is not authoritative
(Section 9.1 of {{?RFC7230}}), this section defines an HTTP/2 extension.

## Indicating support for cross-origin Server Push # {#setting}

Clients that might accept signed Server Pushes with an authority for which the
server is not authoritative indicate this using the HTTP/2 SETTINGS parameter
ENABLE_CROSS_ORIGIN_PUSH (0xSETTING-TBD).

An ENABLE_CROSS_ORIGIN_PUSH value of 0 indicates that the client does not
support cross-origin Push. A value of 1 indicates that the client does support
cross-origin Push.

A client MUST NOT send a ENABLE_CROSS_ORIGIN_PUSH setting with a value other
than 0 or 1 or a value of 0 after previously sending a value of 1. If a server
receives a value that violates these rules, it MUST treat it as a connection
error (Section 5.4.1 of {{!RFC7540}}) of type PROTOCOL_ERROR.

The use of a SETTINGS parameter to opt-in to an otherwise incompatible protocol
change is a use of "Extending HTTP/2" defined by Section 5.5 of {{?RFC7540}}. If
a server were to send a cross-origin Push without first receiving a
ENABLE_CROSS_ORIGIN_PUSH setting with the value of 1 it would be a protocol
violation.

## NO_TRUSTED_EXCHANGE_SIGNATURE error code {#error-code}

The signatures on a Pushed cross-origin exchange may be untrusted for several
reasons, for example that the certificate could not be fetched, that the
certificate does not chain to a trusted root, that the signature itself doesn't
validate, that the signature is expired, etc. This draft conflates all of these
possible failures into one error code, NO_TRUSTED_EXCHANGE_SIGNATURE
(0xERROR-TBD).

### Open Questions

How fine-grained should this specification's error codes be?

## Validating a cross-origin Push ## {#validating-cross-origin-push}

If the client has set the ENABLE_CROSS_ORIGIN_PUSH setting to 1, the server MAY
Push a signed exchange for which it is not authoritative, and the client MUST
NOT treat a PUSH_PROMISE for which the server is not authoritative as a stream
error (Section 5.4.2 of {{!RFC7540}}) of type PROTOCOL_ERROR, as described in
Section 8.2 of {{?RFC7540}}.

Instead, the client MUST validate such a PUSH_PROMISE and its response by
parsing the `Signature` header into a list of signatures according to the
instructions in {{signature-validity}}, and searching that list for a valid
signature using the algorithm in {{authority-chain-validation}}. If no valid
signature is found, the client MUST treat the response as a stream error
(Section 5.4.2 of {{!RFC7540}}) of type NO_TRUSTED_EXCHANGE_SIGNATURE.
Otherwise, the client MUST treat the pushed response as if the server were
authoritative for the PUSH_PROMISE's authority.

### Validating a certificate chain for an authority ### {#authority-chain-validation}

1. Run {{signature-validity}} over the signature with the `allResponseHeaders`
   flag set, getting `exchange` and `certificate-chain` back. If this returned
   "invalid" or didn't return a certificate chain, return "invalid".
1. Let `authority` be the host component of `exchange`'s effective request URI.
1. Validate the `certificate-chain` using the following substeps. If any of them
   fail, re-run {{signature-validity}} once over the signature with both the
   `forceFetch` flag and the `allResponseHeaders` flag set, and restart from
   step 2. If a substep fails again, return "invalid".
   1. Use `certificate-chain` to validate that its first entry,
      `main-certificate` is trusted as `authority`'s server certificate
      ({{!RFC5280}} and other undocumented conventions). Let `path` be the path
      that was used from the `main-certificate` to a trusted root, including the
      `main-certificate` but excluding the root.
   1. Validate that `main-certificate` includes a "status_request" extension
      with a valid OCSP response whose lifetime (`nextUpdate - thisUpdate`) is
      less than 7 days ({{!RFC6960}}). Note that this does not check for
      revocation of intermediate certificates, and clients SHOULD implement
      another mechanism for that.
   1. Validate that all certificates in `path` include
      "signed_certificate_timestamp" extensions containing valid SCTs from
      trusted logs. ({{!RFC6962}})
1. Return "valid".

# Security considerations

Authors MUST NOT include confidential information in a signed response that an
untrusted intermediate could forward, since the response is only signed and not
encrypted. Intermediates can read the content.

Relaxing the requirement to consult DNS when determining authority for an origin
means that an attacker who possesses a valid certificate no longer needs to be
on-path to redirect traffic to them; instead of modifying DNS, they need only
convince the user to visit another Web site in order to serve responses signed
as the target. This consideration and mitigations for it are shared by
{{?I-D.ietf-httpbis-origin-frame}}.

Signing a bad response can affect more users than simply serving a bad response,
since a served response will only affect users who make a request while the bad
version is live, while an attacker can forward a signed response until its
signature expires. Authors should consider shorter signature expiration times
than they use for cache expiration times.

An attacker with temporary access to a signing oracle can sign "still valid"
assertions with arbitrary timestamps and expiration times. As a result, when a
signing oracle is removed, the keys it provided access to SHOULD be revoked so
that, even if the attacker used them to sign future-dated package validity
assertions, the key's OCSP assertion will expire, causing the package as a
whole to become untrusted.

## Aspects of the straw proposal

The use of a single `Signed-Headers` header field prevents us from signing
aspects of the request other than its effective request URI (Section 5.5 of
{{?RFC7230}}). For example, if an author signs both `Content-Encoding: br` and
`Content-Encoding: gzip` variants of a response, what's the impact if an
attacker serves the brotli one for a request with `Accept-Encoding: gzip`?

The simple form of `Signed-Headers` also prevents us from signing less than the
full request URL. The SRI use case ({{uc-sri}}) may benefit from being able to
leave the authority less constrained.

{{signature-validity}} can succeed when some delivered headers aren't included
in the signed set. This accommodates current TLS-terminating intermediates and
may be useful for SRI ({{uc-sri}}), but is risky for trusting cross-origin
responses ({{uc-pushed-subresources}}, {{uc-explicit-distributor}}, and
{{uc-offline-websites}}). {{cross-origin-push}} requires all headers to be
included in the signature before trusting cross-origin pushed resources, at Ryan
Sleevi's recommendation.

# Privacy considerations

Normally, when a client fetches `https://o1.com/resource.js`,
`o1.com` learns that the client is interested in the resource. If
`o1.com` signs `resource.js`, `o2.com` serves it as
`https://o2.com/o1resource.js`, and the client fetches it from there,
then `o2.com` learns that the client is interested, and if the client
executes the Javascript, that could also report the client's interest back to
`o1.com`.

Often, `o2.com` already knew about the client's interest, because it's the
entity that directed the client to `o1resource.js`, but there may be cases where
this leaks extra information.

For non-executable resource types, a signed response can improve the privacy
situation by hiding the client's interest from the original author.

# IANA considerations

TODO: possibly register the validityUrl format.

## Signature Header Field Registration

This section registers the `Signature` header field in the "Permanent Message
Header Field Names" registry ({{!RFC3864}}).

Header field name:  `Signature`

Applicable protocol:  http

Status:  standard

Author/Change controller:  IETF

Specification document(s):  {{signature-header}} of this document

## HTTP/2 Settings

This section establishes an entry for the HTTP/2 Settings Registry that was
established by Section 11.3 of {{!RFC7540}}

Name: ENABLE_CROSS_ORIGIN_PUSH

Code: 0xSETTING-TBD

Initial Value: 0

Specification: This document

## HTTP/2 Error code

This section establishes an entry for the HTTP/2 Error Code Registry that was
established by Section 11.4 of {{!RFC7540}}

Name: NO_TRUSTED_EXCHANGE_SIGNATURE

Code: 0xERROR-TBD

Description: The client does not trust the signature for a cross-origin Pushed
signed exchange.

Specification: This document

--- back

# Use cases

## PUSHed subresources {#uc-pushed-subresources}

To reduce round trips, a server might use HTTP/2 PUSH to inject a subresource
from another server into the client's cache. If anything about the subresource
is expired or can't be verified, the client would fetch it from the original
server.

For example, if `https://example.com/index.html` includes

~~~html
<script src="https://jquery.com/jquery-1.2.3.min.js">
~~~

Then to avoid the need to look up and connect to `jquery.com` in the critical
path, `example.com` might PUSH that resource (Section 8.2 of {{?RFC7540}}),
signed by `jquery.com`.

## Explicit use of a content distributor for subresources {#uc-explicit-distributor}

In order to speed up loading but still maintain control over its content, an
HTML page in a particular origin `O.com` could tell clients to load its
subresources from an intermediate content distributor that's not authoritative,
but require that those resources be signed by `O.com` so that the distributor
couldn't modify the resources. This is more constrained than the common CDN case
where `O.com` has a CNAME granting the CDN the right to serve arbitrary content
as `O.com`.

~~~html
<img logicalsrc="https://O.com/img.png"
     physicalsrc="https://distributor.com/O.com/img.png">
~~~

To make it easier to configure the right distributor for a given request,
computation of the `physicalsrc` could be encapsulated in a custom element:

~~~html
<dist-img src="https://O.com/img.png"></dist-img>
~~~

where the `<dist-img>` implementation generates an appropriate `<img>` based on,
for example, a `<meta name="dist-base">` tag elsewhere in the page. However,
this has the downside that the
[preloader](https://calendar.perfplanet.com/2013/big-bad-preloader/) can no
longer see the physical source to download it. The resulting delay might cancel
out the benefit of using a distributor.

This could be used for some of the same purposes as SRI ({{uc-sri}}).

To implement this with the current proposal, the distributor would respond to
the physical request to `https://distributor.com/O.com/img.png` with first a
signed PUSH_PROMISE for `https://O.com/img.png` and then a redirect to
`https://O.com/img.png`.

## Subresource Integrity {#uc-sri}

The W3C WebAppSec group is investigating
[using signatures](https://github.com/mikewest/signature-based-sri) in {{SRI}}.
They need a way to transmit the signature with the response, which this proposal
could provide.

However, their needs also differ in some significant ways:

1. The `integrity="ed25519-[public-key]"` attribute and CSP-based ways of
   expressing a public key don't need the signing key to be also trusted to sign
   arbitrary content for an origin.
2. Some uses of SRI want to constrain subresources to be vouched for by a
   third-party, rather than just being signed by the subresource's author.

While we can design this system to cover both origin-trusted and simple-key
signatures, we should check that this is better than having two separate systems
for the two kinds of signatures.

Note that while the current proposal for SRI describes signing only the content
of a
resource,
[they may need to sign its name as well, to prevent security vulnerabilities](https://github.com/mikewest/signature-based-sri/issues/5).
The details of what they need to sign will affect whether and how they can use
this proposal.

## Offline websites {#uc-offline-websites}

See <https://github.com/WICG/webpackage> and
{{?I-D.yasskin-dispatch-web-packaging}}. This use requires origin-signed
resources to be bundled.

# Requirements

## Proof of origin

To verify that a thing came from a particular origin, for use in the same
context as a TLS connection, we need someone to vouch for the signing key with
as much verification as the signing keys used in TLS. The obvious way to do this
is to re-use the web PKI and CA ecosystem.

### Certificate constraints

If we re-use existing TLS server certificates, we incur the risks that:

1. TLS server certificates must be accessible from online servers, so they're
   easier to steal than an offline key. A package's signing key doesn't need to
   be online.
2. A server using an origin-trusted key for one purpose (e.g. TLS) might
   accidentally sign something that looks like a package, or vice versa.

If these risks are too high, we could define a new Extended Key Usage (Section
4.2.1.12 of {{?RFC5280}}) that requires CAs to issue new keys for this purpose
or a new certificate extension to do the same. A new EKU would probably require
CAs to also issue new intermediate certificates because of how browsers trust
EKUs. Both an EKU and a new extension take a long time to deploy and allow CAs
to charge package-signers more than normal server operators, which will reduce
adoption.

The rest of this document will assume we can re-use existing TLS server
certificates.

### Signature constraints

In order to prevent an attacker who can convince the server to sign some
resource from causing those signed bytes to be interpreted as something else,
signatures here need to:

1. Avoid key types that are used for non-TLS protocols whose output could be
   confused with a signature. That may be just the `rsaEncryption` OID from
   {{?RFC8017}}.
2. Use the same format as TLS's signatures, specified in Section 4.4.3 of
   {{?I-D.ietf-tls-tls13}} , with a context string that's specific to this use.

The specification also needs to define which signing algorithm to use. I expect
to define that as a function from the key type, instead of allowing
attacker-controlled data to specify it.

### Retrieving the certificate {#certificate-chain}

The client needs to be able to find the certificate vouching for the signing
key, a chain from that certificate to a trusted root, and possibly other trust
information like SCTs ({{?RFC6962}}). One approach would be to include the
certificate and its chain in the signature metadata itself, but this wastes
bytes when the same certificate is used for multiple HTTP responses. If we
decide to put the signature in an HTTP header, certificates are also unusually
large for that context.

Another option is to pass a URL that the client can fetch to retrieve the
certificate and chain. To avoid extra round trips in fetching that URL, it could
be [bundled](#uc-offline-websites) with the signed content or
[PUSHed](#uc-pushed-subresources) with it. The risks from the
`client_certificate_url` extension (Section 11.3 of {{RFC6066}}) don't seem to
apply here, since an attacker who can get a client to load a package and fetch
the certificates it references, can also get the client to perform those fetches
by loading other HTML.

To avoid using an unintended certificate with the same public key as the
intended one, the content of the certificate chain should be included in the
signed data, like TLS does (Section 4.4.3 of {{?I-D.ietf-tls-tls13}}).

## How much to sign ## {#how-much-to-sign}

The previous {{?I-D.thomson-http-content-signature}} and
{{?I-D.burke-content-signature}} schemes signed just the content, while
({{?I-D.cavage-http-signatures}} could also sign the response headers and the
request method and path. However, the same path, response headers, and content
may mean something very different when retrieved from a different server.
{{significant-headers}} currently includes the whole request URL in the
signature, but it's possible we need a more flexible scheme to allow some
higher-level protocols to accept a less-signed URL.

The question of whether to include other request headers---primarily the
`accept*` family---is still open. These headers need to be represented so that
clients wanting a different language, say, can avoid using the wrong-language
response, but it's not obvious that there's a security vulnerability if an
attacker can spoof them. For now, the proposal ({{proposal}}) omits other
request headers.

In order to allow multiple clients to consume the same signed exchange, the
exchange shouldn't include the exact request headers that any particular client
sends. For example, a Japanese resource wouldn't include

~~~http
accept-language: ja-JP, ja;q=0.9, en;q=0.8, zh;q=0.7, *;q=0.5
~~~

Instead, it would probably include just

~~~http
accept-language: ja-JP, ja
~~~

and clients would use the same matching logic as
for [PUSH_PROMISE](https://tools.ietf.org/html/rfc7540#section-8.2) frame
headers.

### Conveying the signed headers

HTTP headers are traditionally munged by proxies, making it impossible to
guarantee that the client will see the same sequence of bytes as the author
wrote. In the HTTPS world, we have more end-to-end header integrity, but it's
still likely that there are enough TLS-terminating proxies that the author's
signatures would tend to break before getting to the client.

There's also no way in current HTTP for the response to a client-initiated
request (Section 8.1 of {{RFC7540}}) to convey the request headers it expected
to respond to. A PUSH_PROMISE (Section 8.2 of {{RFC7540}}) does not have this
problem, and it would be possible to introduce a response header to convey the
expected request headers.

Since proxies are unlikely to modify unknown content types, we could wrap the
original exchange into an `application/http2` format and include the
`Cache-Control: no-transform` header when sending it. This format could be as
simple as a series of HTTP/2 frames, or could

1. Allow longer contiguous bodies than [HTTP/2's 16MB frame
   limit](https://tools.ietf.org/html/rfc7540#section-4.2), and
1. Use better compression than {{?RFC7541}} for the non-confidential headers.
   Note that header compression can probably share a compression state across a
   single signed exchange, but needs a mechanism like
   {{?I-D.vkrasnov-h2-compression-dictionaries}} to use any compression state
   from other responses.

To reduce the likelihood of accidental modification by proxies, an
`application/http2` format should be sure to include a file signature that
doesn't collide with other known signatures.

To help the PUSHed subresources use case ({{uc-pushed-subresources}}), we might
also want to extend the `PUSH_PROMISE` frame type to include a signature, and
that could tell intermediates not to change the ensuing headers.

## Response lifespan

A normal HTTPS response is authoritative only for one client, for as long as its
cache headers say it should live. A signed exchange can be re-used for many
clients, and if it was generated while a server was compromised, it can continue
compromising clients even if their requests happen after the server recovers.
This signing scheme needs to mitigate that risk.

### Certificate revocation

Certificates are mis-issued and private keys are stolen, and in response clients
need to be able to stop trusting these certificates as promptly as possible.
Online revocation
checks [don't work](https://www.imperialviolet.org/2012/02/05/crlsets.html), so
the industry has moved to pushed revocation lists and stapled OCSP responses
{{?RFC6066}}.

Pushed revocation lists work as-is to block trust in the certificate signing an
exchange, but the signatures need an explicit strategy to staple OCSP responses.
One option is to extend the certificate download ({{certificate-chain}}) to
include the OCSP response too, perhaps in the
[TLS 1.3 CertificateEntry](https://tlswg.github.io/tls13-spec/draft-ietf-tls-tls13.html#ocsp-and-sct) format.

### Response downgrade attacks {#downgrade}

The signed content in a response might be vulnerable to attacks, such as XSS, or
might simply be discovered to be incorrect after publication. Once the author
fixes those vulnerabilities or mistakes, clients should stop trusting the old
signed content in a reasonable amount of time. Similar to certificate
revocation, I expect the best option to be stapled "this version is still valid"
assertions with short expiration times.

These assertions could be structured as:

1. A signed minimum version number or timestamp for a set of request headers:
   This requires that signed responses need to include a version number or
   timestamp, but allows a server to provide a single signature covering all
   valid versions.
1. A replacement for the whole exchange's signature. This requires the author to
   separately re-sign each valid version and requires each version to include a
   different update URL, but allows intermediates to serve less data. This is
   the approach taken in {{proposal}}.
1. A replacement for the exchange's signature and an update for the embedded
   `expires` and related cache-control HTTP headers {{?RFC7234}}. This naturally
   extends authors' intuitions about cache expiration and the existing cache
   revalidation behavior to signed exchanges. This is sketched and its downsides
   explored in {{validity-with-cache-control}}.

The signature also needs to include instructions to intermediates for how to
fetch updated validity assertions.

# Determining validity using cache control # {#validity-with-cache-control}

This draft could expire signature validity using the normal HTTP cache control
headers ({{?RFC7234}}) instead of embedding an expiration date in the signature
itself. This section specifies how that would work, and describes why I haven't
chosen that option.

The signatures in the `Signature` header field ({{signature-header}}) would no
longer contain "date" or "expires" fields.

The validity-checking algorithm ({{signature-validity}}) would initialize `date`
from the resource's `Date` header field (Section 7.1.1.2 of {{?RFC7231}}) and
initialize `expires` from either the `Expires` header field (Section 5.3 of
{{?RFC7234}}) or the `Cache-Control` header field's `max-age` directive (Section
5.2.2.8 of {{?RFC7234}}) (added to `date`), whichever is present, preferring
`max-age` (or failing) if both are present.

Validity updates ({{updating-validity}}) would include a list of replacement
response header fields. For each header field name in this list, the client
would remove matching header fields from the stored exchange's response header
fields. Then the client would append the replacement header fields to the stored
exchange's response header fields.

## Example of updating cache control

For example, given a stored exchange of:

~~~http
GET https://example.com/ HTTP/1.1
Accept: */*

HTTP/1.1 200
Date: Mon, 20 Nov 2017 10:00:00 UTC
Content-Type: text/html
Date: Tue, 21 Nov 2017 10:00:00 UTC
Expires: Sun, 26 Nov 2017 10:00:00 UTC

<!doctype html>
<html>
...
~~~

And an update listing the following headers:

~~~http
Expires: Fri, 1 Dec 2017 10:00:00 UTC
Date: Sat, 25 Nov 2017 10:00:00 UTC
~~~

The resulting stored exchange would be:

~~~http
GET https://example.com/ HTTP/1.1
Accept: */*

HTTP/1.1 200
Content-Type: text/html
Expires: Fri, 1 Dec 2017 10:00:00 UTC
Date: Sat, 25 Nov 2017 10:00:00 UTC

<!doctype html>
<html>
...
~~~

## Downsides of updating cache control ## {#downsides-of-cache-control}

In an exchange with multiple signatures, using cache control to expire
signatures forces all signatures to initially live for the same period. Worse,
the update from one signature's "validityUrl" might not match the update for
another signature. Clients would need to maintain a current set of headers for
each signature, and then decide which set to use when actually parsing the
resource itself.

This need to store and reconcile multiple sets of headers for a single signed
exchange argues for embedding a signature's lifetime into the signature.

# Acknowledgements

Thanks to Ilari Liusvaara, Mark Nottingham, Mike Bishop, Ryan Sleevi, and Yoav
Weiss for comments that improved this draft.
