---
coding: utf-8

title: Signed HTTP Exchanges Implementation Checkpoints
docname: draft-yasskin-httpbis-origin-signed-exchanges-impl-latest
category: std

ipr: trust200902

stand_alone: yes
pi: [comments, sortrefs, strict, symrefs, toc]

author:
 -
    name: Jeffrey Yasskin
    organization: Google
    email: jyasskin@chromium.org
 -  name: Kouhei Ueno
    organization: Google
    email: kouhei@chromium.org

normative:
  FETCH:
    target: https://fetch.spec.whatwg.org/
    title: Fetch
    author:
      org: WHATWG
    date: Living Standard
  HTML:
    target: https://html.spec.whatwg.org/multipage
    title: HTML
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
  DROWN:
    target: https://drownattack.com/
    title: The DROWN Attack
    author:
      - name: Nimrod Aviram
      - name: Sebastian Schinzel
      - name: Juraj Somorovsky
      - name: Nadia Heninger
      - name: Maik Dankel
      - name: Jens Steube
      - name: Luke Valenta
      - name: David Adrian
      - name: J. Alex Halderman
      - name: Viktor Dukhovni
      - name: Emilia Käsper
      - name: Shaanan Cohney
      - name: Susanne Engels
      - name: Christof Paar
      - name: Yuval Shavitt
    date: 2016
  ROBOT:
    target: https://robotattack.org/
    title: The ROBOT Attack
    author:
      - name: Hanno Böck
      - name: Juraj Somorovsky
      - name: Craig Young
    date: 2017
  SRI: W3C.REC-SRI-20160623

--- abstract

This document describes checkpoints of
{{?I-D.yasskin-http-origin-signed-responses}} to synchronize implementation
between clients, intermediates, and authors.

--- note_Note_to_Readers

Discussion of this draft takes place on the HTTP working group mailing list
(ietf-http-wg@w3.org), which is archived
at <https://lists.w3.org/Archives/Public/ietf-http-wg/>.

The source code and issues list for this draft can be found
in <https://github.com/WICG/webpackage>.

--- middle

# Introduction

Each version of this document describes a checkpoint of
{{?I-D.yasskin-http-origin-signed-responses}} that can be implemented in sync by
clients, intermediates, and authors. It defines a technique to detect which
version each party has implemented so that mismatches can be detected up-front.



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

# Signing an exchange # {#proposal}

In the response of an HTTP exchange the server MAY include a `Signature` header
field ({{signature-header}}) holding a list of one or more parameterised
signatures that vouch for the content of the exchange. Exactly which content the
signature vouches for can depend on how the exchange is transferred
({{transfer}}).

The client categorizes each signature as "valid" or "invalid" by validating that
signature with its certificate or public key and other metadata against the
exchange's headers and content ({{signature-validity}}). This validity then
informs higher-level protocols.

Each signature is parameterised with information to let a client fetch assurance
that a signed exchange is still valid, in the face of revoked certificates and
newly-discovered vulnerabilities. This assurance can be bundled back into the
signed exchange and forwarded to another client, which won't have to re-fetch
this validity information for some period of time.

## The Signature Header ## {#signature-header}

The `Signature` header field conveys a single signature for an exchange,
accompanied by information about how to determine the authority of and
refresh that signature. Each signature directly signs the exchange's headers and
identifies one of those headers that enforces the integrity of the exchange's
payload.

The `Signature` header is a Structured Header as defined by
{{!I-D.ietf-httpbis-header-structure}}. Its value MUST be a list (Section 4.8 of
{{!I-D.ietf-httpbis-header-structure}}) of parameterised labels (Section 4.4 of
{{!I-D.ietf-httpbis-header-structure}}), and the list MUST contain exactly one
element.

Each parameterised label MUST have parameters named "sig", "integrity",
"validityUrl", "date", and "expires". Each parameterised label MUST also have
"certUrl" and "certSha256" parameters. This
specification gives no meaning to the label itself, which can be used as a
human-readable identifier for the signature (see {{parameterised-binary}}). The
present parameters MUST have the following values:

"sig"

: Binary content (Section 4.5 of {{!I-D.ietf-httpbis-header-structure}}) holding
  the signature of most of these parameters and the exchange's headers.

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

{:#signature-validityurl} "validityUrl"

: A string (Section 4.2 of {{!I-D.ietf-httpbis-header-structure}}) containing a
  [valid URL string](https://url.spec.whatwg.org/#valid-url-string).

"date" and "expires"

: An unsigned integer (Section 4.1 of {{!I-D.ietf-httpbis-header-structure}})
  representing a Unix time.

The "certUrl" parameter is *not* signed, so intermediates can update it with a
pointer to a cached version.

### Examples ### {#example-signature-header}

The following header is included in the response for an exchange with effective
request URI `https://example.com/resource.html`. Newlines are added for
readability.

~~~http
Signature:
 sig1;
  sig=*t7LoYw6vwL2FSZRNJPYdNdYjfZSQkaCQeqpBD1whcy/6AAamVJ2OryXoXv6ACVBQgPV13o5de9oOVcOGGMX9fsf2ve1UDw/ITpeimB7n3zcuDEePzIcPbUnicicN2yodZAfr5il7BBJTs8L+V2ZERI16nJfrOZOvUfhvuUaMDGQXx5StIj7XLiX7/caxPz5ctwglgVAwCmoVPhmYFLq391O+hEssHSk2xkY6r/D9V2cKMikBBOTZ+JFyrnS/f2B4li7YASIY0YX64ifCmCw97cQTngXax6Upoie44IAe+6JngOie9JlDgcMF3YZ1uxNGWl9VwlalSwWgi1YA9Ff7mQ;
  integrity="mi";
  validityUrl="https://example.com/resource.validity.1511128380";
  certUrl="https://example.com/certs";
  certSha256=*W7uB969dFW3Mb5ZefPS9Tq5ZbH5iSmOILpjv2qEArmI;
  date=1511128380; expires=1511733180
~~~

The signatures uses a 2048-bit RSA certificate within `https://example.com/`.

It relies on the `MI` response header to guard the integrity of the response
payload.

The signature includes a "validityUrl" that includes the first time the resource
was seen. This allows multiple versions of a resource at the same URL to be
updated with new signatures, which allows clients to avoid transferring extra
data while the old versions don't have known security bugs.

The certificate at `https://example.com/certs` has a `subjectAltName` of `example.com`, meaning
that if it and its signature validate, the exchange can be trusted as
having an origin of `https://example.com/`.

### Open Questions ### {#oq-signature-header}

{{?I-D.ietf-httpbis-header-structure}} provides a way to parameterise labels but
not other supported types like binary content. If the `Signature` header field
is notionally a list of parameterised signatures, maybe we should add a
"parameterised binary content" type.
{:#parameterised-binary}

Should the certUrl and validityUrl be lists so that intermediates can offer a
cache without losing the original URLs? Putting lists in dictionary fields is
more complex than {{?I-D.ietf-httpbis-header-structure}} allows, so they're
single items for now.

## CBOR representation of exchange headers ## {#cbor-representation}

To sign an exchange's headers, they need to be serialized into a byte string.
Since intermediaries and distributors might
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
   * For each request header field in `exchange`, the header field's name as a
     byte string to the header field's value as a byte string.
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
Content-Encoding: mi-sha256
MI: mi-sha256=20addcf7368837f616d549f035bf6784ea6d4bf4817a3736cd2fc7a763897fe3

<0x0000000000004000><!doctype html>
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
    'mi': 'mi-sha256=20addcf7368837f616d549f035bf6784ea6d4bf4817a3736cd2fc7a763897fe3',
    ':status': '200',
    'content-type': 'text/html'
    'content-encoding': 'mi-sha256',
  }
]
~~~

## Loading a certificate chain ## {#cert-chain-format}

The resource at a signature's `certUrl` MUST have the
`application/tls-cert-chain` content type and MUST contain a TLS 1.3 Certificate
message (Section 4.4.2 of {{!I-D.ietf-tls-tls13}}) containing X.509v3
certificates.

Parsing notes:
1. This resource MUST NOT include the 4-byte header that would appear in a
   Handshake message.
1. Since this fetch is not in response to a CertificateRequest, the
   certificate_request_context MUST be empty, and a non-empty value MUST cause
   the parse to fail.

The client MUST ignore unknown or unexpected extensions.

Loading a `certUrl` takes a `forceFetch` flag. The client MUST:

1. Let `raw-chain` be the result of fetching ({{FETCH}}) `certUrl`. If
   `forceFetch` is *not* set, the fetch can be fulfilled from a cache using
   normal HTTP semantics {{!RFC7234}}. If this fetch fails, return
   "invalid".
1. Let `certificate-chain` be the array of certificates and properties produced
   by parsing `raw-chain` as the TLS Certificate message as described above. If
   any of the requirements above
   aren't satisfied, return "invalid". Note that this validation requirement
   might be impractical to completely achieve due to certificate validation
   implementations that don't enforce DER encoding or other standard
   constraints.
1. Return `certificate-chain`.

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
values (Section 4.8.1 of {{!I-D.ietf-httpbis-header-structure}}) described in
{{signature-header}}. If an error is thrown during this parsing or any of the
requirements described there aren't satisfied, the exchange has no valid
signatures. Otherwise, each member of this list represents a signature with
parameters.

The client MUST use the following algorithm to determine whether each signature
with parameters is invalid or potentially-valid for an `exchange`.
Potentially-valid results include:

* The signed headers of the exchange so that higher-level protocols can avoid
  relying on unsigned headers, and
* Either a certificate chain or a public key so that a higher-level protocol can
  determine whether it's actually valid.

This algorithm accepts a `forceFetch` flag that avoids the cache when fetching
URLs. A client that determines that a potentially-valid certificate chain is
actually invalid due to an expired OCSP response MAY retry with `forceFetch` set
to retrieve an updated OCSP from the original server.
{:#force-fetch}

TODO: Remove OCSP and SCT requirements.

1. Let `payload` be the payload body (Section 3.3 of {{!RFC7230}}) of
   `exchange`. Note that the payload body is the message body with any transfer
   encodings removed.
1. Let:
   * `signature` be the signature (binary content in the parameterised label's
     "sig" parameter).
   * `integrity` be the signature's "integrity" parameter.
   * `validityUrl` be the signature's "validityUrl" parameter.
   * `certUrl` be the signature's "certUrl" parameter, if any.
   * `certSha256` be the signature's "certSha256" parameter, if any.
   * `date` be the signature's "date" parameter, interpreted as a Unix time.
   * `expires` be the signature's "expires" parameter, interpreted as a Unix
     time.
1. If `integrity` names a header field other than `MI`
   ({{!I-D.thomson-http-mice}}) or this header field is not present in
   `exchange`'s response headers or which the client cannot use to check the
   integrity of `payload` (for example, the header field is new and hasn't been
   implemented yet), then return "invalid". Clients MUST be able to check the
   integrity of `payload` using the `MI` ({{!I-D.thomson-http-mice}}) header
   field.
1. Set `publicKey` and `signing-alg` depending on which key fields are present:
   1. Assert: `certUrl` is present.
      1. Let `certificate-chain` be the result of loading the certificate chain
         at `certUrl` passing the `forceFetch` flag ({{cert-chain-format}}). If
         this returns "invalid", return "invalid".
      1. Let `main-certificate` be the first certificate in `certificate-chain`.
      1. Set `publicKey` to `main-certificate`'s public key.
      1. If `publicKey` is not a 2048-bit RSA public key, return "invalid".
      1. The client MUST define a partial function from public key types to
         signing algorithms, and this function must at the minimum include the
         following mappings:

         RSA, 2048 bits:
         : rsa_pss_rsae_sha256 or rsa_pss_pss_sha256, as defined in Section
           4.2.3 of {{!I-D.ietf-tls-tls13}}, depending on which of the
           rsaEncryption OID or RSASSA-PSS OID {{!RFC8017}} is used.

         Set `signing-alg` to the result of applying this function to the type of
         `main-certificate`'s public key. If the function is undefined on this
         input, return "invalid".
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
         1. The text string "certSha256" to the byte string value of
            `certSha256`.
      1. The text string "validityUrl" to the byte string value of
         `validityUrl`.
      1. The text string "date" to the integer value of `date`.
      1. The text string "expires" to the integer value of `expires`.
      1. The text string "headers" to the CBOR representation
         ({{cbor-representation}}) of `exchange`'s headers.
1. If `certUrl` is present and the SHA-256 hash of `main-certificate`'s
   `cert_data` is not equal to `certSha256` (whose presence was checked when the
   `Signature` header field was parsed), return "invalid".

   Note that this intentionally differs from TLS 1.3, which signs the entire
   certificate chain in its Certificate Verify (Section 4.4.3 of
   {{?I-D.ietf-tls-tls13}}), in order to allow updating the stapled OCSP
   response without updating signatures at the same time.
1. If `signature` is a valid signature of `message` by `publicKey` using
   `signing-alg`, return "potentially-valid" with `certificate-chain`.
   Otherwise, return "invalid".

Note that the above algorithm can determine that an exchange's headers are
potentially-valid before the exchange's payload is received. Similarly, if
`integrity` identifies a header field like `MI` ({{?I-D.thomson-http-mice}})
that can incrementally validate the payload, early parts of the payload can be
determined to be potentially-valid before later parts of the payload.
Higher-level protocols MAY process parts of the exchange that have been
determined to be potentially-valid as soon as that determination is made but
MUST NOT process parts of the exchange that are not yet potentially-valid.
Similarly, as the higher-level protocol determines that parts of the exchange
are actually valid, the client MAY process those parts of the exchange and MUST
wait to process other parts of the exchange until they too are determined to be
valid.

### Open Questions ### {#oq-signature-validity}

Should the signed message use the TLS format (with an initial 64 spaces) even
though these certificates can't be used in TLS servers?

## Updating signature validity ## {#updating-validity}

Both OCSP responses and signatures are designed to expire a short
time after they're signed, so that revoked certificates and signed exchanges
with known vulnerabilities are distrusted promptly.

This specification provides no way to update OCSP responses by themselves.
Instead, [clients need to re-fetch the "certUrl"](#force-fetch) to get a chain
including a newer OCSP response.

The ["validityUrl" parameter](#signature-validityurl) of the signatures provides
a way to fetch new signatures or learn where to fetch a complete updated
exchange.

Each version of a signed exchange SHOULD have its own validity URLs, since each
version needs different signatures and becomes obsolete at different times.

The resource at a "validityUrl" is "validity data", a CBOR map matching the
following CDDL ({{!I-D.ietf-cbor-cddl}}):

~~~cddl
validity = {
  ? signatures: [ + bytes ]
  ? update: {
    ? size: uint,
  }
]
~~~

The elements of the `signatures` array are parameterised labels (Section 4.4 of
{{!I-D.ietf-httpbis-header-structure}}) meant to replace the signatures within
the `Signature` header field pointing to this validity data. If the signed
exchange contains a bug severe enough that clients need to stop using the
content, the `signatures` array MUST NOT be present.

If the the `update` map is present, that indicates that a new version of the
signed exchange is available at its effective request URI (Section 5.5 of
{{!RFC7230}}) and can give an estimate of the size of the updated exchange
(`update.size`). If the signed exchange is currently the most recent version,
the `update` SHOULD NOT be present.

If both the `signatures` and `update` fields are present, clients can use the
estimated size to decide whether to update the whole resource or just its
signatures.

### Examples ### {#examples-updating-validity}

For example, say a signed exchange whose URL is `https://example.com/resource`
has the following `Signature` header field (with line breaks included and
irrelevant fields omitted for ease of reading).

~~~http
Signature:
 sig1;
  sig=*MEUCIQ...;
  ...
  validityUrl="https://example.com/resource.validity.1511157180";
  certUrl="https://example.com/oldcerts";
  date=1511128380; expires=1511733180,
 sig2;
  sig=*MEQCIG...;
  ...
  validityUrl="https://example.com/resource.validity.1511157180";
  certUrl="https://example.com/newcerts";
  date=1511128380; expires=1511733180,
 thirdpartysig;
  sig=*MEYCIQ...;
  ...
  validityUrl="https://thirdparty.example.com/resource.validity.1511161860";
  certUrl="https://thirdparty.example.com/certs";
  date=1511478660; expires=1511824260
~~~

At 2017-11-27 11:02 UTC, `sig1` and `sig2` have expired, but `thirdpartysig`
doesn't exipire until 23:11 that night, so the client needs to fetch
`https://example.com/resource.validity.1511157180` (the `validityUrl` of `sig1`
and `sig2`) to update those signatures. This URL might contain:

~~~cbor-diag
{
  "signatures": [
    'sig1; '
    'sig=*MEQCIC/I9Q+7BZFP6cSDsWx43pBAL0ujTbON/+7RwKVk+ba5AiB3FSFLZqpzmDJ0NumNwN04pqgJZE99fcK86UjkPbj4jw; '
    'validityUrl="https://example.com/resource.validity.1511157180"; '
    'integrity="mi"; '
    'certUrl="https://example.com/newcerts"; '
    'certSha256=*J/lEm9kNRODdCmINbvitpvdYKNQ+YgBj99DlYp4fEXw; '
    'date=1511733180; expires=1512337980'
  ],
  "update": {
    "size": 5557452
  }
}
~~~

This indicates that the client could fetch a newer version at
`https://example.com/resource` (the original URL of the exchange), or that the
validity period of the old version can be extended by replacing the first two of
the original signatures (the ones with a validityUrl of
`https://example.com/resource.validity.1511157180`) with the single new
signature provided. (This might happen at the end of a migration to a new root
certificate.) The signatures of the updated signed exchange would be:

~~~http
Signature:
 sig1;
  sig=*MEQCIC...;
  ...
  validityUrl="https://example.com/resource.validity.1511157180";
  certUrl="https://example.com/newcerts";
  date=1511733180; expires=1512337980,
 thirdpartysig;
  sig=*MEYCIQ...;
  ...
  validityUrl="https://thirdparty.example.com/resource.validity.1511161860";
  certUrl="https://thirdparty.example.com/certs";
  date=1511478660; expires=1511824260
~~~

`https://example.com/resource.validity.1511157180` could also expand the set of
signatures if its `signatures` array contained more than 2 elements.

## The Accept-Signature header ## {#accept-signature}

This section isn't implemented.

# Cross-origin trust {#cross-origin-trust}

To determine whether to trust a cross-origin exchange, the client takes a
`Signature` header field ({{signature-header}}) and the `exchange`. The client
MUST parse the `Signature` header into a list of signatures according to the
instructions in {{signature-validity}}, and run the following algorithm for each
signature, stopping at the first one that returns "valid". If any signature
returns "valid", return "valid". Otherwise, return "invalid".

1. If the signature's ["validityUrl" parameter](#signature-validityurl) is not
   [same-origin](https://html.spec.whatwg.org/multipage/origin.html#same-origin)
   with `exchange`'s effective request URI (Section 5.5 of {{!RFC7230}}), return
   "invalid".
1. Use {{signature-validity}} to determine the signature's validity for
   `exchange`, getting `certificate-chain` back. If this returned "invalid" or
   didn't return a certificate chain, return "invalid".
1. If `exchange`'s request method is not safe (Section 4.2.1 of {{!RFC7231}}) or
   not cacheable (Section 4.2.3 of {{!RFC7231}}), return "invalid".
1. If `exchange`'s headers contain a stateful header field, as defined in
   {{stateful-headers}}, return "invalid".
1. Let `authority` be the host component of `exchange`'s effective request URI.
1. Validate the `certificate-chain` using the following substeps. If any of them
   fail, re-run {{signature-validity}} once over the signature with the
   `forceFetch` flag set, and restart from step 2. If a substep fails again,
   return "invalid".
   1. Use `certificate-chain` to validate that its first entry,
      `main-certificate` is trusted as `authority`'s server certificate
      ({{!RFC5280}} and other undocumented conventions). Let `path` be the path
      that was used from the `main-certificate` to a trusted root, including the
      `main-certificate` but excluding the root.
   1. Validate that `main-certificate` has the CanSignHttpExchanges extension
      ({{cross-origin-cert-req}}).
   1. Validate that `main-certificate` has an `ocsp` property
      ({{cert-chain-format}}) with a valid OCSP response whose lifetime
      (`nextUpdate - thisUpdate`) is less than 7 days ({{!RFC6960}}). Note that
      this does not check for revocation of intermediate certificates, and
      clients SHOULD implement another mechanism for that.
   1. Validate that `main-certificate` has an `sct` property
      ({{cert-chain-format}}) containing valid SCTs from trusted logs.
      ({{!RFC6962}})
1. Return "valid".

## Stateful header fields {#stateful-headers}

As described in {{seccons-over-signing}}, a publisher can cause problems if they
sign an exchange that includes private information. There's no way for a client
to be sure an exchange does or does not include private information, but header
fields that store or convey stored state in the client are a good sign.

A stateful request header field informs the server of per-client state. These
include but are not limited to:

* `Authorization`, {{?RFC7235}}
* `Cookie`, {{?RFC6265}}
* `Cookie2`, {{?RFC2965}}
* `Proxy-Authorization`, {{?RFC7235}}
* `Sec-WebSocket-Key`, {{?RFC6455}}

A stateful response header field modifies state, including authentication
status, in the client. The HTTP cache is not considered part of this state.
These include but are not limited to:

* `Authentication-Control`, {{?RFC8053}}
* `Authentication-Info`, {{?RFC7615}}
* `Optional-WWW-Authenticate`, {{?RFC8053}}
* `Proxy-Authenticate`, {{?RFC7235}}
* `Proxy-Authentication-Info`, {{?RFC7615}}
* `Sec-WebSocket-Accept`, {{?RFC6455}}
* `Set-Cookie`, {{?RFC6265}}
* `Set-Cookie2`, {{?RFC2965}}
* `SetProfile`, {{?W3C.NOTE-OPS-OverHTTP}}
* `WWW-Authenticate`, {{?RFC7235}}

## Certificate Requirements {#cross-origin-cert-req}

For this draft, no new X.509 extension is required.

# Transferring a signed exchange {#transfer}

A signed exchange can be transferred in several ways, of which three are
described here.

## Same-origin response {#same-origin-response}

Receiving a Signature header as part of a normal HTTP exchange is not implemented.

## HTTP/2 extension for cross-origin Server Push # {#cross-origin-push}

Cross origin push is not implemented.

## application/signed-exchange format # {#application-signed-exchange}

To parse a resource with content type `application/signed-exchange;v=b0`, the
client MUST run the following algorithm:

Read 3 bytes and interpret them as a big-endian integer `headerLength`.

If `headerLength` is larger than 524288 (512kB), parsing MUST fail.

Read `headerLength` bytes, and parse them as a CBOR item. If this item isn't
canonically encoded ({{canonical-cbor}}) or doesn't match the following CDDL,
parsing MUST fail:

~~~cddl
signed-exchange-header = [
  { ':method': bytes,
    ':url': bytes,
    * bytes => bytes,
  },
  { ':status': bytes,
    'signature': bytes,
    * bytes => bytes,
  },
]
~~~

The first element of the array is interpreted as the exchange's request headers,
with the request method in the ':method' key's value, and the effective request
URI in the ':url' key's value.

The second element of the array is interpreted as the exchange's response
headers, with the 3-digit response status code in the ':status' key's value.

Pass the `Signature` response header and the exchange with that header removed
to the algorithm in {{cross-origin-trust}}. Fail if this returns "invalid".

The remainder of the resource is the exchange's payload, encoded with the
`mi-sha256` content encoding ({{!I-D.thomson-http-mice}}). If the `mi-sha256`
record length (the first 8 bytes of the payload) is greater than 16kB, or if any
of the integrity proofs fail validation, parsing MUST fail.

# Security considerations

## Over-signing ## {#seccons-over-signing}

If a publisher blindly signs all responses as their origin, they can cause at
least two kinds of problems, described below. To avoid this, publishers SHOULD
design their systems to opt particular public content that doesn't depend on
authentication status into signatures instead of signing by default.

Signing systems SHOULD also incorporate the following mitigations to reduce the
risk that private responses are signed:

1. Strip the `Cookie` request header field and other identifying information
   like client authentication and TLS session IDs from requests whose exchange
   is destined to be signed, before forwarding the request to a backend.
1. Only sign exchanges where the response includes a `Cache-Control: public`
   header. Clients are not required to fail signature-checking for exchanges
   that omit this `Cache-Control` response header field to reduce the risk that
   naïve signing systems blindly add it.

### Session fixation ### {#seccons-session-fixation}

Blind signing can sign responses that create session cookies or otherwise change
state on the client to identify a particular session. This breaks certain kinds
of CSRF defense and can allow an attacker to force a user into the attacker's
account, where the user might unintentionally save private information, like
credit card numbers or addresses.

This specification defends against cookie-based attacks by blocking the
`Set-Cookie` response header, but it cannot prevent Javascript or other response
content from changing state.

### Misleading content ### {#seccons-misleading-content}

If a site signs private information, an attacker might set up their own account
to show particular private information, forward that signed information to a
victim, and use that victim's confusion in a more sophisticated attack.

Stripping authentication information from requests before sending them to
backends is likely to prevent the backend from showing attacker-specific
information in the signed response. It does not prevent the attacker from
showing their victim a signed-out page when the victim is actually signed in,
but while this is still misleading, it seems less likely to be useful to the
attacker.

## Off-path attackers ## {#seccons-off-path}

Relaxing the requirement to consult DNS when determining authority for an origin
means that an attacker who possesses a valid certificate no longer needs to be
on-path to redirect traffic to them; instead of modifying DNS, they need only
convince the user to visit another Web site in order to serve responses signed
as the target. This consideration and mitigations for it are shared by the
combination of {{?I-D.ietf-httpbis-origin-frame}} and
{{?I-D.ietf-httpbis-http2-secondary-certs}}.

## Downgrades ## {#seccons-downgrades}

Signing a bad response can affect more users than simply serving a bad response,
since a served response will only affect users who make a request while the bad
version is live, while an attacker can forward a signed response until its
signature expires. Authors should consider shorter signature expiration times
than they use for cache expiration times.

The current implementation does not re-check the
["validityUrl"](#signature-validityurl) to get a TLS guarantee that the exchange
isn't out of date.

## Signing oracles are permanent ## {#seccons-signing-oracles}

An attacker with temporary access to a signing oracle can sign "still valid"
assertions with arbitrary timestamps and expiration times. As a result, when a
signing oracle is removed, the keys it provided access to MUST be revoked so
that, even if the attacker used them to sign future-dated exchange validity
assertions, the key's OCSP assertion will expire, causing the exchange as a
whole to become untrusted.

## Unsigned headers ## {#seccons-unsigned-headers}

The use of a single `Signed-Headers` header field prevents us from signing
aspects of the request other than its effective request URI (Section 5.5 of
{{?RFC7230}}). For example, if an author signs both `Content-Encoding: br` and
`Content-Encoding: gzip` variants of a response, what's the impact if an
attacker serves the brotli one for a request with `Accept-Encoding: gzip`?

The simple form of `Signed-Headers` also prevents us from signing less than the
full request URL. The SRI use case may benefit from being able to
leave the authority less constrained.

{{signature-validity}} can succeed when some delivered headers aren't included
in the signed set. This accommodates current TLS-terminating intermediates and
may be useful for SRI, but is risky for trusting cross-origin
responses. {{cross-origin-push}} requires all headers to be
included in the signature before trusting cross-origin pushed resources, at Ryan
Sleevi's recommendation.

## application/signed-exchange ## {#security-application-signed-exchange}

Clients MUST NOT trust an effective request URI claimed by an
`application/signed-exchange` resource ({{application-signed-exchange}})
without either ensuring the resource was transferred from a server that was
authoritative (Section 9.1 of {{!RFC7230}}) for that URI's origin, or passing
the `Signature` response header field from the exchange stored in the resource,
and that exchange without its `Signature` response header field, to the
procedure in {{cross-origin-trust}}, and getting "valid" back.

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

To prevent network operators other than `o1.com` or `o2.com` from learning which
exchanges were read, clients SHOULD only load exchanges fetched over a transport
that's protected from eavesdroppers. This can be difficult to determine when the
exchange is being loaded from local disk, but when the client itself requested
the exchange over a network it SHOULD require TLS ({{!I-D.ietf-tls-tls13}}) or a
successor transport layer, and MUST NOT accept exchanges transferred over plain
HTTP without TLS.

# IANA considerations

This depends on the following IANA registrations in
{{?I-D.yasskin-http-origin-signed-responses}}:

* The `Signature` header field
* application/signed-exchange;v=0

This document also registers:

## Internet Media Type application/tls-cert-chain

Type name:  application

Subtype name:  tls-cert-chain

Required parameters:  N/A

Optional parameters:  N/A

Encoding considerations:  binary

Security considerations:  N/A

Interoperability considerations:  N/A

Published specification:  This specification (see {{cert-chain-format}}).

Applications that use this media type:  N/A

Fragment identifier considerations:  N/A

Additional information:

  Deprecated alias names for this type:  N/A

  Magic number(s): N/A

  File extension(s): N/A

  Macintosh file type code(s):  N/A

Person and email address to contact for further information: See Authors'
  Addresses section.

Intended usage:  COMMON

Restrictions on usage:  N/A

Author:  See Authors' Addresses section.

Change controller:  IESG

--- back

# Change Log

draft-00

Vs. {{?I-D.yasskin-http-origin-signed-responses}}:

* Removed non-normative sections.

# Acknowledgements
