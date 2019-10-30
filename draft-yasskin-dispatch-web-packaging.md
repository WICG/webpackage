---
coding: utf-8

title: Web Packaging
docname: draft-yasskin-dispatch-web-packaging-latest
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
  CBOR: RFC7049
  CDDL: I-D.greevenbosch-appsawg-cbor-cddl
  appmanifest: W3C.WD-appmanifest-20170608
  SRI: W3C.REC-SRI-20160623

informative:
  ServiceWorkers: W3C.WD-service-workers-1-20161011

--- abstract

Web Packages provide a way to bundle up groups of web resources to
transmit them together. These bundles can be signed to establish their
authenticity.

--- middle

Introduction        {#problems}
============

People would like to use content offline and in other situations where
there isn’t a direct connection to the server where the content
originates. However, it's difficult to distribute and verify the
authenticity of applications and content without a connection to the
network. The W3C has addressed running applications offline with
Service Workers ({{ServiceWorkers}}), but not
the problem of distribution.

We've started work on this problem
in [](https://github.com/WICG/webpackage), but we suspect that the
IETF may be the right place to standardize the overall format. More
details can be found in that repository.

Use Cases    {#use-cases}
---------

See {{?I-D.yasskin-wpack-use-cases}}.

Why not ZIP?   {#not-zip}
------------

[WICG/webpackage#45](https://github.com/WICG/webpackage/issues/45)

The Need for Standardization   {#need}
----------------------------

Publishers and readers should be able to generate a package once, and have it
usable by all browsers.


Terminology          {#Terminology}
-----------

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL
NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED",
"MAY", and "OPTIONAL" in this document are to be interpreted as
described in BCP 14 {{!RFC2119}} {{!RFC8174}} when, and only when, they
appear in all capitals, as shown here.


# Format   {#format}

## Mode of specification  {#mode}

This specification defines how conformant web package parsers convert a sequence
of bytes into the semantics of a web package. It does not constrain how web
package encoders produce such a package: although there are some guidelines in
{{authoring-guidelines}}, encoders MAY produce any sequence of bytes that a
conformant parser would parse into the intended semantics.

In places, this specification says the parser "MAY return" some data. This
indicates that the described data is complete enough that later parsing failures
do not need to discard it.

In places, this specification says the parser "MUST fail". The parser MAY report
these failures to its caller in any way, but MUST NOT return any data it has
parsed so far that wasn't mentioned in a "MAY return" statement.

This specification creates local variables with the phrase "Let *variable-name*
be ...". Use of a variable before it's created is a defect in this
specification.

## Top-level structure  {#top-level}

The package is roughly a CBOR item with the following CDDL schema, but package
parsers are required to successfully parse some byte strings that aren't valid
CBOR. For example, sections may have padding between them, or even overlap, as
long as the embedded relative offsets cause the parsing algorithm in this
specification to return data.

~~~~~ cddl
webpackage = [
  ; 🌐📦 in UTF-8.
  magic1: h'F0 9F 8C 90 F0 9F 93 A6',
  section-offsets: { * (($section-name .within tstr) => uint) },
  sections: [ *$section ],
  length: uint,  ; Total number of bytes in the package.
  ; 🌐📦 in UTF-8.
  magic2: h'F0 9F 8C 90 F0 9F 93 A6',
]

$section-name /= "indexed-content" / "manifest"

$section /= indexed-content / signed-manifest

indexed-content = [index, [*response]]
~~~~~

The parser MAY begin parsing at either the [beginning](#from-start)
or [end](#from-end) of the byte string representing the package. Parsing from
the end is useful when the package is embedded in another format such as a
self-extracting executable, while parsing from the beginning is useful when
loading from a stream.

### From the end {#from-end}

To parse from the end, the parser MUST load the last 18 bytes as the following
{{CDDL}} group in array context: [^ednote-loading-cddl]{: source="jyasskin"}

[^ednote-loading-cddl]: CDDL doesn't actually define how to use it as a schema
    to load CBOR data.

~~~~~ cddl
tail = (
  ; Total number of bytes in the package.
  length: uint,
  ; 🌐📦 in UTF-8.
  magic2: h'F0 9F 8C 90 F0 9F 93 A6',
)
~~~~~

If the bytes don't match this group or these two CBOR items don't occupy exactly
18 bytes, parsing MUST fail.

Otherwise, continue as if the byte `length` bytes before the end of the string
were the beginning of the package, and the parser were
a [from the beginning](#from-start) parser.

### From the beginning {#from-start}

If the first 10 bytes of the package are not "85 48 F0 9F 8C 90 F0 9F 93 A6"
(the CBOR encoding of the 5-item array header and 8-byte bytestring header,
followed by 🌐📦 in UTF-8), parsing MUST fail.

Parse one CBOR item starting at the 11th byte of the package. If this does not
match the CDDL

~~~~~ cddl
section-offsets = { * tstr => uint },
~~~~~

or it is not a Canonical CBOR item (Section 3.9 of {{CBOR}}), parsing MUST fail.

Let *sections-start* be the offset of the byte after the `section-offsets` item.
For example, if `section-offsets` were 52 bytes long, *sections-start* would be
63.

This specification defines two section names: "indexed-content" and "manifest".

If `section-offsets`\["indexed-content"] is not present, parsing MUST fail.

The parser MUST ignore unknown keys in the `section-offsets` map because new
sections may be defined in future specifications.
[^ednote-critical-sections]{:source="jyasskin"}

[^ednote-critical-sections]: Do we need to mark critical section names?

Let *index* be the result of parsing the bytes starting at offset
*sections-start* + `section-offsets`\["indexed-content"] using the instructions in
{{index}}.

If `section-offsets`\["manifest"] is present, let *manifest* be the
result of parsing the bytes starting at offset *sections-start* +
`section-offsets`\["manifest"] using the instructions in {{manifest}}.

The parser MAY return a semantic package consisting of *index*, and,
if initialized, *manifest*.

To parse each resource described within *index*, the parser MUST follow the
instructions in {{resource}}.

## Parsing the index {#index}

The main content of a package is an index of HTTP requests pointing to HTTP
responses. These request/response pairs hold the manifests of sub-packages and
the resources in the package and all of its sub-packages. Both the requests and
responses can appear in any order, usually chosen to optimize loading while the
package is streamed.

To parse the index, starting at offset *index-start*, the parser MUST do the
following:

If the byte at *index-start* is not 0x82 (the {{CBOR}} header for a 2-element
array), the parser MUST fail.

Load a CBOR item starting at *index-start* + 1 as the `index` array in the following CDDL:

~~~~~ cddl
index = [* [resource-key: http-headers,
            offset: uint,
            ? length: uint] ]

; http-headers is a byte string in HPACK format (RFC7541).
http-headers = bstr
~~~~~

If the item doesn't match this CDDL, or it is not a Canonical CBOR item (Section
3.9 of {{CBOR}}), the parser MUST fail.

Let *resources-start* be the offset immediately after the `index` item. For
example, if *index-start* were 75 and the `index` item were 105 bytes long,
*resources-start* would be 75+1+105=181. (1 for the 0x82 array header.)

Decode all of the `resource-key`s using {{!HPACK=RFC7541}}, with an
initially-empty dynamic table for each one. [^ednote-compression]{:
source="jyasskin"} The decoded `resource-key`s are header lists
({{HPACK}}, Section 1.3), ordered lists of name-value pairs.

The parser MUST fail if any of the following is true:

1. HPACK decoding encountered an error.
1. Any `resource-key`'s first three headers are not named ":scheme",
   ":authority", and ":path", in that order. Note that ":method" is
   intentionally omitted because only the GET method is meaningful.
1. Any of the pseudo-headers' values violates a requirement in Section
   8.1.2.3 of {{!HTTP2=RFC7540}}.
1. Any `resource-key` has a non-pseudo-header name that includes the
   ":" character or is not lower-case ascii ({{HTTP2}}, Section
   8.1.2).
1. Any two decoded `resource-key`s are the same. Note that header
   lists with the same header fields in a different order are not the
   same.

Increment all `offset`s by *resources-start*.

Return the resulting `index`, an array of decoded-resource-key, adjusted-offset,
and optional-length triples.

[^ednote-compression]: This spec has different security constraints from the
    ones that drove HPACK, so we may be able to do better with another
    compression format.

The optional `length` field in the index entries is redundant with the length
prefixes on the `response-headers` and `body` in the content, but it can be used
to issue Range requests {{?RFC7233}} for responses that appear late in the
content.


## Parsing the manifest {#manifest}

A package's manifest contains some metadata for the package; hashes, used in
{{hashing-resources}}, for all resources included in that package; and
validity information for any [sub-packages](#subpackages) the package depends
on. The manifest is signed, so that UAs can trust that it comes from its claimed
origin. [^ednote-manifest-name]{: source="jyasskin"}

[^ednote-manifest-name]: This section doesn't describe a manifest
    (https://www.merriam-webster.com/dictionary/manifest#h3), so consider
    renaming it to something like "authenticity".

To parse a manifest starting at *manifest-start*, a parser MUST do the following:

Load one CBOR item starting at *manifest-start* as a `signed-manifest` from the
following CDDL:

~~~~~ cddl
signed-manifest = {
  manifest: manifest,
  certificates: [+ certificate],
  signatures: [+ signature]
}

manifest = {
  metadata: manifest-metadata,
  resource-hashes: {* hash-algorithm => [hash-value]},
  ? subpackages: [* subpackage],
}

manifest-metadata = {
  date: time,
  origin: uri,
  * tstr => any,
}

; From https://www.w3.org/TR/CSP3/#grammardef-hash-algorithm.
hash-algorithm /= "sha256" / "sha384" / "sha512"
; Note that a hash value is not base64-encoded, unlike in CSP.
hash-value = bstr

; X.509 format; see https://tools.ietf.org/html/rfc5280
certificate = bstr

signature = {
  ; This is the index of the certificate within the
  ; certificates array to use to validate the signature.
  keyIndex: uint,
  signature: bstr,
}
~~~~~

If the item doesn't match the CDDL or it's not a Canonical CBOR item (Section
3.9 of {{CBOR}}), parsing MUST fail.

Parse the elements of `certificates` as X.509 certificates within the
{{!RFC5280}} profile. If any certificate fails to parse, parsing MUST fail.

Let *message* be the concatenation of the following byte strings. This matches
the {{?TLS1.3=I-D.ietf-tls-tls13}} format to avoid cross-protocol attacks when
TLS certificates are used to sign manifests.

1. A string that consists of octet 32 (0x20) repeated 64 times.
1. A context string: the ASCII encoding of "Web Package Manifest".
1. A single 0 byte which serves as a separator.
1. The bytes of the `manifest` CBOR item.

Let *signing-certificates* be an empty array.

For each element *signature* of `signatures`:

1. Let *certificate* be `certificates`\[*signature*\["keyIndex"]].

1. The parser MUST define a partial function from public key types to signing
   algorithms, and this function must at the minimum include the following
   mappings:

   RSA, 2048 bits:
   : rsa_pss_sha256 as defined in Section 4.2.3 of {{!TLS1.3}}

   EC, with the secp256r1 curve:
   : ecdsa_secp256r1_sha256 as defined in Section 4.2.3 of {{!TLS1.3}}

   EC, with the secp384r1 curve:
   : ecdsa_secp384r1_sha384 as defined in Section 4.2.3 of {{!TLS1.3}}

   Let *signing-alg* be the result of applying this function to the key type in
   *certificate*'s Subject Public Key Info. If the function is undefined on this
   input, the parser MUST continue to the next *signature*.

1. Use *signing-alg* to verify that *signature*\["signature"] is *message*'s
   signature by *certificate*'s public key. If it's not, the parser MUST
   continue to the next *signature*.

1. Append *certificate* to *signing-certificates*. Note that failed signatures
   simply cause their certificate to be ignored, so that packagers can give new
   signature types to parsers that understand them.


Let *origin* be `manifest`\["metadata"]\["origin"].

Iterate through *signing-certificates* until one is found that has an identity
({{!RFC2818}}, Section 3.1) matching *origin*'s hostname, and that is trusted
for serverAuth ({{!RFC5280}}, Section 4.2.1.12) using paths built from elements
of `certificates` or any other certificates the parser is aware of. If no such
certificate is found, and the package is not already trusted as received from
*origin*'s hostname, for example because it was received over a TLS connection
to that host, then parsing MUST fail.

**TODO:** Process the `subpackages` item by fetching those manifests via the index,
and checking their signatures and dates/hashes, recursively.

The parsed manifest consists of the set of *signing-certificates* and the
`manifest` CBOR item. The items in `manifest`\["metadata"] SHOULD be interpreted
as described in the {{appmanifest}} specification.


### Sub-packages {#subpackages}

A sub-package is represented by a [manifest](#manifest) file looked up as
a [resource](#resource) within the `indexed-content` section. The sub-package's
resources are not otherwise distinguished from the rest of the resources in the
package. Sub-packages can form an arbitrarily-deep tree.

~~~~~ cddl
subpackage = [
  resource: resource-key,
  validation: {
    ? hash: {+ hash-algorithm => hash-value},
    ? notbefore: time,
  }
]
~~~~~

There are three possible forms of dependencies on sub-packages, of which we
allow two. Because a sub-package's manifest is protected by its own signature,
if the main package trusts the sub-package's server, it could avoid specifying a
version of the sub-package at all. However, this opens the main package up to
downgrade attacks, where the sub-package is replaced by an older, vulnerable
version, so we don't allow this option.

If the main package wants to load either the sub-package it was built with or
any upgrade, it can specify the date of the original sub-package:

~~~~~ cbor-diag
[32("https://example.com/loginsdk.package"),
 {"notbefore": 1(1486429554)}]
~~~~~

Constraining packages with their date makes it possible to link together
sub-packages with common dependencies, even if the sub-packages were built at
different times.

If the main package wants to be certain it's loading the exact version
of a sub-package that it was built with, it can constrain the sub-package
with a hash of its manifest:

~~~~~ cbor-diag
[32("https://example.com/loginsdk.package"),
 {"hash": {"sha256":
     b64'9qg0NGDuhsjeGwrcbaxMKZAvfzAHJ2d8L7NkDzXhgHk='}}]
~~~~~

Note that because the sub-package may include sub-sub-packages by date, the top
package may need to explicitly list those sub-sub-packages' hashes in order to
be completely constrained. For example, if `loginsdk.package` has subpackages
of the form:

~~~~~ cbor-diag
[
  [32("https://other.example.com/helper.package"),
   {"notbefore": 1(1486429554)}]
]
~~~~~

the top-level package needs to specify:

~~~~~ cbor-diag
[
  [32("https://example.com/loginsdk.package"),
   {"hash": {"sha256":
        b64'9qg0NGDuhsjeGwrcbaxMKZAvfzAHJ2d8L7NkDzXhgHk='}}],
  [32("https://other.example.com/helper.package"),
   {"hash": {"sha256":
        b64'SG2GjbrpfVCh21HPLMIXD17fHNCst1Gz/MbQOqihG68='}}]
]
~~~~~

in order to completely constrain all the versions of its dependencies.

## Parsing a resource {#resource}

To parse the resource from a *package* corresponding to a *header-list*, a
parser MUST do the following:

Find the (*resource-key*, *offset*, *length*) triple in *package*'s index where
*resource-key* is the same as *header-list*. If no such triple exists, the
parser MUST fail.

Parse one CBOR item starting at *offset* as the following CDDL:

~~~~~ cddl
response = [response-headers: http-headers, body: bstr]
~~~~~

If the item doesn't match the CDDL or it's not a Canonical CBOR item (Section
3.9 of {{CBOR}}), parsing MUST fail.

Decode the `response-headers` field using {{HPACK}}, with an
initially-empty dynamic table. The decoded `response-headers` is a
header list ({{HPACK}}, Section 1.3), an ordered list of name-value
pairs.

The parser MUST fail if any of the following is true:

1. HPACK decoding encountered an error.
1. The first header name within `response-headers` is not ":status",
   or this pseudo-header's value violates a requirement in Section
   8.1.2.3 of {{!HTTP2=RFC7540}}.
1. Any other header name includes the ":" character or is not
   lower-case ascii ({{HTTP2}}, Section 8.1.2).
1. The *header-list* contains any header names other than ":scheme",
   ":authority", ":path", and either `response-headers` has no "vary"
   header (Section 7.1.4 of {{!RFC7231}}) or these header names aren't
   listed in it.

Let *origin* be the Web Origin {{!RFC6454}} of *header-list*'s ":scheme" and
":authority" headers.

{:anchor="hashing-resources"}
Let *resource-bytes* be the result of encoding the array of
\[*header-list*, `response-headers`, `body`] as Canonical CBOR in the
following CDDL schema: [^ednote-figure-in-list]{:source="jyasskin"}

[^ednote-figure-in-list]: This step would be inside the manifest-only block, but
    then the code block is rendered out-of-order.

~~~~~ cddl
resource-bytes = [
  request: [
    *(header-name: bstr, header-value: bstr)
  ],
  response-headers: [
    *(header-name: bstr, header-value: bstr)
  ],
  response-body: bstr
]
~~~~~

Note that this uses the decoded header fields, not the bytes originally included
in the package.

The hashed data differs from {{SRI}}, which only hashes the body. Including the
headers will usually prevent a package from relying on some of its contents
being transferred as normal network responses, unless its author can guarantee
the network won't change or reorder the headers.

If the *package* contains a *manifest*:

1. **TODO:** Let *origin-manifest* be the signed manifest for *origin*, found
   by searching through *manifest*'s subpackages for a matching origin.

1. Let *alg* be one of the `hash-algorithm`s within *origin-manifest*. The
   parser SHOULD select the most collision-resistant hash algorithm. If the
   parser also implements {{SRI}}, it SHOULD use the same order as its
   `getPrioritizedHashFunction()` implementation.

1. If the digest of *resource-bytes* using *alg* does not appear in the
   *origin-manifest*'s `resource-hashes`\[*alg*] array, the parser MUST fail.


Return the (decoded `response-headers`, `body`) pair.


# Guidelines for package authors {#authoring-guidelines}

Packages SHOULD consist of a single Canonical CBOR item matching the
`webpackage` CDDL rule in {{top-level}}.

Every resource's hash SHOULD appear in every array within
`resource-hashes`: otherwise the set of valid resources will depend on
the parser's choice of preferred hash algorithm.

Security Considerations  {#security}
=======================

Signature validation is difficult.

Packages with a valid signature need to be invalidated when either

* the private key for any certificate in the signature's validation
  chain is leaked, or
* a vulnerability is discovered in the package's contents.

Because packages are intended to be used offline, it's impossible to
inject a revocation check into the critical path of using the package,
and even in online scenarios,
such
[revocation checks don't actually work](https://www.imperialviolet.org/2012/02/05/crlsets.html).
Instead, package consumers must check for a sufficiently recent set of
validation files, consisting of OCSP responses {{!RFC6960}} and signed
package version constraints, for example within the last 7-30 days.
**TODO:** These version constraints aren't designed yet.

Relaxing the requirement to consult DNS when determining authority for an origin
means that an attacker who possesses a valid certificate no longer needs to be
on-path to redirect traffic to them; instead of modifying DNS, they need only
convince the user to visit another Web site, in order to serve packages signed
as the target.

All subpackages that mention a particular origin need to be validated
before loading resources from that origin. Otherwise, package A could
include package B and an old, vulnerable version of package C that B
also depends on. If B's dependency isn't checked before loading
resources from C, A could compromise B.

# IANA considerations

## Internet Media Type Registration

IANA maintains the registry of Internet Media Types {{?RFC6838}}
at <https://www.iana.org/assignments/media-types>.

* Type name: application

* Subtype name: package+cbor [^ednote-mime-naming]{: source="jyasskin"}

* Required parameters: N/A

* Optional parameters: N/A

* Encoding considerations: binary

* Security considerations: See {{security}} of this document.

* Interoperability considerations: N/A

* Published specification: This document

* Applications that use this media type: None yet, but it is expected that web
     browsers will use this format.

* Fragment identifier considerations: N/A

* Additional information:

  * Deprecated alias names for this type: N/A
  * Magic number(s): 85 48 F0 9F 8C 90 F0 9F 93 A6
  * File extension(s): .wpk
  * Macintosh file type code(s): N/A

* Person & email address to contact for further information:
  See the Author's Address section of this specification.

* Intended usage: COMMON

* Restrictions on usage: N/A

* Author:
  See the Author's Address section of this specification.

* Change controller:
  The IESG <iesg@ietf.org>

* Provisional registration? (standards tree only): Not yet.

[^ednote-mime-naming]: I suspect the mime type will need to be a bit longer:
    application/webpackage+cbor or similar.

--- back

# Acknowledgements

Thanks to Adam Langley and Ryan Sleevi for in-depth feedback about the security
impact of this proposal.
