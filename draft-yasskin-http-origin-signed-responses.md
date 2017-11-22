---
coding: utf-8

title: Origin-signed HTTP Responses
docname: draft-yasskin-http-origin-signed-responses-latest
category: std

ipr: trust200902
area: gen
workgroup: http
keyword: Internet-Draft

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
that origin. For now, it focuses on the constraints covering any such mechanism.

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
: An HTTP request from a client and the matching response from a server. These
make up an entire HTTP/2 stream or a contiguous block of an HTTP/1 connection.
Defined by {{!RFC7540}} section 8.

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
path, `example.com` might PUSH that resource ({{?RFC7540}}, section 8.2), signed
by `jquery.com`.

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
for example, a `<meta name="dist-base">` tag elsewhere in the page.

This could be used for some of the same purposes as SRI ({{uc-sri}}).

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

# Requirements and open questions

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

If these risks are too high, we could define a new Extended Key Usage
({{?RFC5280}}, section 4.2.1.12) that requires CAs to issue new keys for this
purpose or a new certificate extension to do the same. A new EKU would probably
require CAs to also issue new intermediate certificates because of how browsers
trust EKUs. Both an EKU and a new extension take a long time to deploy and allow
CAs to charge package-signers more than normal server operators, which will
reduce adoption.

The rest of this document will assume we can re-use existing TLS server
certificates.

### Signature constraints

In order to prevent an attacker who can convince the server to sign some
resource from causing those signed bytes to be interpreted as something else,
signatures here need to:

1. Avoid key types that are used for non-TLS protocols whose output could be
   confused with a signature. That may be just the `rsaEncryption` OID from
   {{?RFC2437}}.
2. Use the same format as TLS's signatures, specified in {{?I-D.ietf-tls-tls13}}
   section 4.4.3, with a context string that's specific to this use.

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
be [bundled](#uc-offline-websites) with the signed content
or [PUSHed](#uc-pushed-subresources) with it. The risks from the
`client_certificate_url` extension ({{RFC6066}} section 11.3) don't seem to
apply here, since an attacker who can get a client to load a package and fetch
the certificates it references, can also get the client to perform those fetches
by loading other HTML.

To avoid using an unintended certificate with the same public key as the
intended one, the content of the certificate chain should be included in the
signed data, like TLS does ({{?I-D.ietf-tls-tls13}}, section 4.4.3).

## How much to sign

The previous {{?I-D.thomson-http-content-signature}} and
{{?I-D.burke-content-signature}} schemes signed just the content, while
({{?I-D.cavage-http-signatures}} could also sign the response headers and the
request method and path. However, the same path, response headers, and content
may mean something very different when retrieved from a different server, so
this document expects to include the whole URL in the signed data as well.

The question of whether to include other request headers---primarily the
`accept*` family---is still open. These headers need to be represented so that
clients wanting a different language, say, can avoid using the wrong-language
response, but it's not obvious that there's a security vulnerability if an
attacker can spoof them. That said, it's always safer to include everything in
the signature.

In order to allow multiple clients to consume the same signed response, the
response shouldn't include the exact headers that any particular client sends.
For example, a Japanese resource wouldn't include

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

The rest of this document will assume that all request headers are included in
the signature.

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
2. A replacement for the whole exchange's signature. This requires the author to
   separately re-sign each valid version and requires each version to include a
   strong validator {{?RFC7232}}, but allows intermediates to serve less data.
3. A replacement for the exchange's signature and an update for the embedded
   `expires` and related cache-control HTTP headers {{?RFC7234}}. This naturally
   extends authors' intuitions about cache expiration and the existing cache
   revalidation behavior to signed exchanges. However, it also requires that the
   update procedure for the response headers on the client must produce
   something that's bytewise identical to the updated headers on the server.

The signature also needs to include instructions to intermediates for how to
fetch updated validity assertions.

## Conveying the signed headers

HTTP headers are traditionally munged by proxies, making it impossible to
guarantee that the client will see the same sequence of bytes as the author
wrote. In the HTTPS world, we have more end-to-end header integrity, but it's
still likely that there are enough TLS-terminating proxies that the author's
signatures would tend to break before getting to the client.

Since proxies don't modify unknown content types, I expect to propose an
`application/http2` format to serialize the request headers, response headers,
and body so they can be signed. This could be as simple as a series of HTTP/2
frames, or could

1. Allow longer contiguous bodies than
   [HTTP/2's 16MB frame limit](https://tools.ietf.org/html/rfc7540#section-4.2), and
2. Use better compression than {{?RFC7541}} for the non-confidential headers.
   Note that header compression can probably share a compression state across a
   single signed exchange, but probably cannot use any compression state from
   other responses.

To help the PUSHed subresources use case ({{uc-pushed-subresources}}), we might
also want to extend the `PUSH_PROMISE` frame type to include a signature, and
then there might be some way to include the request headers directly in that
frame.

# Straw proposal # {#proposal}

As a response to an HTTP request (the "physical request"), or as a Server Push
({{!RFC7540}}, section 8.2) the server MAY send a "logical request" and its
response, with significant headers identified by `Signed-Request-Headers` and
`Signed-Response-Headers` headers ({{signed-r-headers}}), and with signatures
included in one or more `Signature` headers ({{signature-header}}).

The client reconstructs the "significant" parts of the exchange
({{significant-parts}}) either by taking the sent `application/http2`
({{application-http2}}) content directly or by building an `application/http2`
item from the significant request and response headers and the response payload.
Then it categorizes each `Signature` header as "valid" or "invalid" by
validating that header's certificate, metadata, and signature against the
significant parts of the pair ({{signature-validity}}). This validity then
informs higher-level protocols.

The `Signature` header includes information to let a client fetch assurance a
signed exchange is still valid, in the face of revoked certificates and
newly-discovered vulnerabilities. This assurance can be bundled back into the
signed exchange and forwarded to another client, which won't have to re-fetch
this validity information for some period of time.

## The Signed-Request-Headers and Signed-Response-Headers Headers ## {#signed-r-headers}

Each of the `Signed-Request-Headers` and `Signed-Response-Headers` headers
identifies an ordered list of request and response, respectively, header field
names to include in a signature. These lists of header field names need to be
consistent across all signatures of an exchange, to avoid forcing higher-level
protocols to merge the header field lists of valid signatures, so they're
provided by headers instead of being included in the `Signature` header field.

`Signed-Request-Headers` and `Signed-Response-Headers` are Structured Headers as
defined by {{!I-D.nottingham-structured-headers}}. Their values MUST be lists
({{!I-D.nottingham-structured-headers}}, section 4.7) of lowercase strings
({{!I-D.nottingham-structured-headers}}, section 4.2) naming HTTP headers,
following the ordering and lowercasing rules from {{!RFC7540}}, section 8.1.2.

## The Signature Header ## {#signature-header}

The `Signature` header conveys a signature for an exchange and information about
how to determine the authority of and refresh that signature.

The `Signature` header is a Structured Header as defined by
{{!I-D.nottingham-structured-headers}}. Its value MUST be a dictionary
({{!I-D.nottingham-structured-headers}}, section 4.7).

The dictionary MUST contain members named "validityUrl", "date", "expires", and
"sig", and either "certUrl" and "certSha256" members or an "ed25519Key" member.
The present members MUST have the following values:

"certUrl"

: A string ({{!I-D.nottingham-structured-headers}}, section 4.2) containing a
  [valid URL string](https://url.spec.whatwg.org/#valid-url-string).

"certSha256"

: Binary content ({{!I-D.nottingham-structured-headers}}, section 4.5) holding
  the SHA-256 hash of the first certificate found at "certUrl".

"ed25519Key"

: Binary content ({{!I-D.nottingham-structured-headers}}, section 4.5) holding
  an Ed25519 public key ({{!RFC8032}}).

{:#signature-validityurl} "validityUrl"

: A string ({{!I-D.nottingham-structured-headers}}, section 4.2) containing a
  [valid URL string](https://url.spec.whatwg.org/#valid-url-string).

"date" and "expires

: An unsigned integer ({{!I-D.nottingham-structured-headers}}, section 4.1)
  representing a Unix time.

"sig"

: Binary content ({{!I-D.nottingham-structured-headers}}, section 4.5) holding
  the signature of most of this header and the significant parts of the exchange
  {{significant-parts}}.

The "certUrl" and "validityUrl" members are *not* signed, so intermediates can
update them with pointers to cached versions.

### Open Questions
Should the certUrl and validityUrl be lists so that intermediates can offer a
cache without losing the original URLs? Putting lists in dictionary fields is
more complex than {{?I-D.nottingham-structured-headers}} allows, but Mark might
be willing to update that.

## Content of a signed exchange

The payload of a signed exchange can appear in one of two forms:

1. `application/http2` content describing the exact request and response headers
   for the logical request ({{application-http2}}).
1. The payload of the logical response.

In the first case, `Signed-Request-Headers` and `Signed-Response-Headers` header
fields MUST NOT appear. In the second case, they MUST appear and identify which
headers are significant and in which order.

## application/http2 content type ## {#application-http2}

The `application/http2` content type encodes an HTTP2 ({{!RFC7540}})
exchange, including an ordered list of pseudo and normal request
header fields, optionally a request body, an ordered list of pseudo and normal
response header fields, a payload body, and optionally an ordered list of
trailer header fields.

This content type is **not** simply the concatenation of a sequence of HTTP/2
frames ({{?RFC7540}}, section 6). Using this concatenation would have the two downsides that 1) there's no canonical form

This content type consists of a canonical CBOR ({{!RFC7049}}, section 3.9) array
containing alternating member names encoded as text strings, {{!RFC7049}},
section 2.1), and member values, each consisting of a single CBOR item with a
type and meaning determined by the member name.

Note that this format does not use CBOR dictionaries, so that it can specify a
natural order for the members.

This specification defines the following member names with their associated
values:

"request"

: An array of alternating request header field names and header field values,
  encoded as byte strings ({{!RFC7049}}, section 2.1), following the ordering
  and lowercasing rules from {{!RFC7540}}, section 8.1.2.

"request payload"

: A byte string ({{!RFC7049}}, section 2.1) containing the request payload body.

"response"

: An array of alternating response header field names and header field values,
  encoded as byte strings ({{!RFC7049}}, section 2.1), following the ordering
  and lowercasing rules from {{!RFC7540}} section 8.1.2.

"payload"

: A byte string ({{!RFC7049}}, section 2.1) containing the response payload body
  ({{!RFC7230}}, section 3.3).

"trailer"

: An array of alternating trailer header field names and header field values,
  encoded as byte strings ({{!RFC7049}}, section 2.1), following the ordering
  and lowercasing rules from {{!RFC7540}} section 8.1.2.

A parser MAY return incremental information while parsing `application/http2`
content.

Members "request", "response", and "payload" MUST be present. If one is missing,
the parser MUST stop and report an error.

The member names MUST appear in the order:

1. "request"
1. "request payload"
1. "response"
1. "payload"
1. "trailer"

If a member name is not a text string, appears out of order, or is followed by a
value not matching its description above, the parser MUST stop and report an
error.

If the parser encounters an unknown member name, it MUST skip the following item
and resume parsing at the next member name.

### Example ### {#example-application-http2}

An example `application/http2` file representing a possible exchange with
<https://example.com/> follows, in the extended diagnostic format defined in
{{?I-D.ietf-cbor-cddl}} appendix G:

~~~cbor-diag
[
  "request",
  [
    ':method', 'GET',
    ':scheme', 'https',
    ':authority', 'example.com',
    ':path', '/',
    'accept', '*/*'
  ],
  "response",
  [
    ':status', '200',
    'content-type', 'text/html'
  ],
  "payload",
  '<!doctype html>\r\n<html>...'
]
~~~

## Significant parts of an exchange ## {#significant-parts}

The significant parts of an exchange are either an `application/http2` item
({{application-http2}}) or a failure.

If the `Content-Type` ({{!RFC7231}}) of the response is `application/http2` and
neither the `Signed-Request-Headers` nor the `Signed-Response-Headers` header
field is present, the significant parts are the payload itself. If either of
those header fields is present, computing the significant parts fails.

Otherwise (the `Content-Type` is not `application/http2`), if either of the
`Signed-Request-Headers` or the `Signed-Response-Headers` header field is
missing, or if either throws an error while parsing as the structured header
defined in {{signed-r-headers}}, computing the significant parts fails.

If the client is building the significant parts of an exchange for an HTTP
Request/Response Exchange ({{RFC7540}}, section 8.1), the "input request" is the
request the client sent. If the client is doing so for a Server Push
({{RFC7540}}, section 8.2), the input request is the request represented in the
PUSH_PROMISE and its CONTINUATIONs.

The significant parts of the exhange are the `application/http2` CBOR item
({{application-http2}}) with the following content:

1. The text string "request".
1. The array consisting of the appended items from, for each element `name` of
   the `Signed-Request-Headers` header field value,
   1. Append `name` as a byte string.
   1. If there is a header field named `name` in the input request's header
      fields, append that header field's value, as a byte string.
   1. Otherwise, computing the significant parts fails.
1. The text string "response".
1. The array consisting of the appended items from, for each element `name` of
   the `Signed-Response-Headers` header field value,
   1. Append `name` as a byte string.
   1. If there is a header field named `name` in the response's header fields,
      append that header field's value, as a byte string.
   1. Otherwise, computing the significant parts fails.
1. The text string "payload".
1. The byte string containing the response's payload body ({{!RFC7230}}, section
   3.3). Note that the payload body is the message body with any transfer
   encodings removed.

## Signature validity ## {#signature-validity}

The client MUST use the following algorithm to determine whether each
`Signature` header field ({{signature-header}}) is invalid or potentially-valid.
Potentially-valid results include:

* The signed parts of the exchange so that higher-level protocols can avoid
  relying on unsigned headers, and
* Either a certificate chain or a public key so that a higher-level protocol can
  determine whether it's actually valid.

This algorithm accepts a `forceFetch` flag that avoids the cache when fetching
URLs. A client that determines that a potentially-valid certificate chain is
actually invalid due to expired OCSP responses MAY retry with `forceFetch` set
to retrieve updated OCSPs from the original server.
{:#force-fetch}

1. Let `exchange` be the significant parts ({{significant-parts}}) of the
   `Signature` header's exchange. If `exchange` is a failure, then return
   "invalid".
1. If an error is thrown while parsing the `Signature` header as the dictionary
   described in {{signature-header}}  ({{!I-D.nottingham-structured-headers}},
   section 4.7.1), return "invalid".
1. Let:
   * `certUrl` be the header's "certUrl" member, if any
   * `certSha256` be the header's "certSha256" member, if any
   * `ed25519Key` be the header's "ed25519Key" member, if any
   * `date` be the header's "date" member, interpreted as a Unix time
   * `expires` be the header's "expires" member, interpreted as a Unix time
   * `signature` be the header's "sig" member
1. Set `publicKey` and `signing-alg` depending on which key fields are present:
   1. If `certUrl` is present:
      1. Let `certificate-chain` be the result of fetching ({{FETCH}}) `certUrl`
         and parsing it as a TLS 1.3 Certificate message
         ({{!I-D.ietf-tls-tls13}}, section 4.4.2) containing X.509v3
         certificates. If `forceFetch` is *not* set, the fetch can be fulfilled
         from a cache using normal HTTP semantics {{!RFC7234}}. If this fetch or
         parse fails, return "invalid".
      1. Let `main-certificate` be the first certificate in `certificate-chain`.
      1. If the SHA-256 hash of `main-certificate`'s `cert_data` is not equal to
        `certSha256`, return "invalid". See the [open questions](#hash-whole-chain).
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
   1. The bytes of the canonical CBOR encoding of a CBOR array consisting of:
      1. The text string "certSha256".
      1. The byte string `certSha256`.
      1. The text string "date".
      1. The integer value of `date`.
      1. The text string "expires".
      1. The integer value of `expires`.
      1. The text string "exchange".
      1. The CBOR item `exchange`. See the [open questions](#incremental-validity).
1. If `signature` is `message`'s signature by `main-certificate`'s public key
   using `signing-alg`, return "potentially-valid" with `exchange` and whichever
   is present of `certificate-chain` or `ed25519Key`. Otherwise, return
   "invalid".

### Validating a certificate chain for an authority

When the physical request is different from the logical request, clients will often want to verify that a potentially-valid certificate chain is trusted for the logical request. They can use the following algorithm to do so:

1. Run {{signature-validity}} over the `Signature` header, getting `exchange`
   and `certificate-chain` back. If this returned "invalid", return "invalid".
1. Let `authority` be the ":authority" request header from `exchange`. If
   `exchange` doesn't include its ":authority" header, return "invalid".
1. Validate the `certificate-chain` using the following substeps. If any of them
   fail, re-run {{signature-validity}} once over the `Signature` header with the
   `forceFetch` flag set, and restart from step 2.
   1. Use `certificate-chain` to validate that its first entry,
      `main-certificate` is trusted as `authority`'s server certificate
      ({{!RFC5280}} and other undocumented conventions). Let `path` be the path
      that was used from the `main-certificate` to a trusted root, including the
      `main-certificate` but excluding the root.
   1. Validate that all certificates in `path` include "status_request"
      extensions with valid OCSP responses. ({{!RFC6960}})
   1. Validate that all certificates in `path` include
      "signed_certificate_timestamp" extensions containing valid SCTs from
      trusted logs. ({{!RFC6962}})
1. Return "valid".

### Open Questions

TLS 1.3 signs the entire certificate chain, but doing that here would preclude
updating the OCSP signatures without replacing all `Signature` header fields at
the same time. What attack do I allow by hashing only the end-entity
certificate?
{:#hash-whole-chain}

Including the entire exchange in the signed data forces a client to download the
whole thing before trusting any of it. {{?I-D.thomson-http-mice}} is designed to
let us check the validity of just the `MI` header up front and then
incrementally check blocks of the payload as they arrive. What's the best way to
integrate that? Maybe make the {{significant-parts}} omit the actual payload
when there's an `MI` header with a matching `Content-Encoding` and warn clients
to enforce the `MI` header when using the body?
{:#incremental-validity}

## Updating signature validity ## {#updating-validity}

Both OCSP responses and `Signature` header fields are designed to expire a short
time after they're signed, so that revoked certificates and signed exchanges
with known vulnerabilities are distrusted promptly.

This specification provides no way to update OCSP responses by themselves.
Instead, [clients need to re-fetch the "certUrl"](#force-fetch) to get a chain
including newer OCSPs.

The ["validityUrl" member](#signature-validityurl) of the `Signature` header
provides a way to fetch new signatures or learn where to fetch a complete
updated package.

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
the `Signature` header fields pointing to this validity data. If the signed
exchange contains a bug severe enough that clients need to stop using the
content, the `signatures` array MUST NOT be present.

The `update` map gives a location to update the entire signed exchange and an
estimate of the size of the resource at that URL. If the signed exchange is
currently the most recent version, the `update` SHOULD NOT be present.

If both the `signatures` and `update` fields are present, clients can use the
estimated size to decide whether to update the whole resource or just its
signatures.

### Examples ### {#examples-updating-validity}

For example, if a signed exchange has the following `Signature` header fields:

~~~http
Signature: validityUrl="https://example.com/resource.validity",
  certUrl="https://example.com/certs",
  certSha256=*W7uB969dFW3Mb5ZefPS9Tq5ZbH5iSmOILpjv2qEArmI,
  date=1511128380, expires=1511560380,
  sig=*MEUCIQDXlI2gN3RNBlgFiuRNFpZXcDIaUpX6HIEwcZEc0cZYLAIga9DsVOMM+g5YpwEBdGW3sS+bvnmAJJiSMwhuBdqp5UY
Signature: validityUrl="https://example.com/resource.validity",
  certUrl="https://example.com/certs",
  certSha256=*kQAA8u33cZRTy7RHMO4+dv57baZL48SYA2PqmYvPPbg,
  date=1511301183, expires=1511905983,
  sig=*MEQCIGjZRqTRf9iKNkGFyzRMTFgwf/BrY2ZNIP/dykhUV0aYAiBTXg+8wujoT4n/W+cNgb7pGqQvIUGYZ8u8HZJ5YH26Qg
Signature: validityUrl="https://thirdparty.example.com/resource.validity",
  certUrl="https://thirdparty.example.com/certs",
  certSha256=*UeOwUPkvxlGRTyvHcsMUN0A2oNsZbU8EUvg8A9ZAnNc,
  date=1511301183, expires=1511905983,
  sig=*MEYCIQCNxJzn6Rh2fNxsobktir8TkiaJYQFhWTuWI1i4PewQaQIhAMs2TVjc4rTshDtXbgQEOwgj2mRXALhfXPztXgPupii+
~~~

https://example.com/resource.validity might contain:

~~~cbor-diag
{
  "signatures": [
    'validityUrl="https://example.com/resource.validity", '
    'certUrl="https://example.com/certs", '
    'certSha256=*W7uB969dFW3Mb5ZefPS9Tq5ZbH5iSmOILpjv2qEArmI, '
    'date=1511467200, expires=1511985600, '
    'sig=*MEQCIC/I9Q+7BZFP6cSDsWx43pBAL0ujTbON/+7RwKVk+ba5AiB3FSFLZqpzmDJ0NumNwN04pqgJZE99fcK86UjkPbj4jw'
  ],
  "update": {
    "url": "https://example.com/resource",
    "size": 5557452
  }
}
~~~

This indicates that the first two of the original `Signature` header fields (the
ones with a validityUrl of "https://example.com/resource.validity") can be
replaced with a single new header field. The `Signature` header fields of the
updated signed exchange would be:

~~~http
Signature: validityUrl="https://example.com/resource.validity",
  certUrl="https://example.com/certs",
  certSha256=*W7uB969dFW3Mb5ZefPS9Tq5ZbH5iSmOILpjv2qEArmI,
  date=1511467200, expires=1511985600,
  sig=*MEQCIC/I9Q+7BZFP6cSDsWx43pBAL0ujTbON/+7RwKVk+ba5AiB3FSFLZqpzmDJ0NumNwN04pqgJZE99fcK86UjkPbj4jw
Signature: validityUrl="https://thirdparty.example.com/resource.validity",
  certUrl="https://thirdparty.example.com/certs",
  certSha256=*UeOwUPkvxlGRTyvHcsMUN0A2oNsZbU8EUvg8A9ZAnNc,
  date=1511301183, expires=1511905983,
  sig=*MEYCIQCNxJzn6Rh2fNxsobktir8TkiaJYQFhWTuWI1i4PewQaQIhAMs2TVjc4rTshDtXbgQEOwgj2mRXALhfXPztXgPupii+
~~~

https://example.com/resource.validity could also expand the set of `Signature`
header fields if its `signatures` array contained more than 2 elements.

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
assertions, the key's OCSP assertions will expire, causing the package as a
whole to become untrusted.

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

TODO: register application/http2 and possibly the validityUrl format.

--- back

# Acknowledgements

Thanks to Ilari Liusvaara for comments that improved this draft.
