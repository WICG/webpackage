---
coding: utf-8

title: Bundled HTTP Exchanges
docname: draft-yasskin-dispatch-bundled-exchanges-latest
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
  appmanifest: W3C.WD-appmanifest-20171129
  SRI: W3C.REC-SRI-20160623

--- abstract

Bundled exchanges provide a way to bundle up groups of HTTP request+response
pairs to transmit or store them together. The component exchanges can be signed
using {{?I-D.yasskin-http-origin-signed-responses}} to establish their
authenticity.

--- note_Note_to_Readers

Discussion of this draft takes place on the ART area mailing list
(art@ietf.org), which is archived
at <https://mailarchive.ietf.org/arch/search/?email_list=art>.

The source code and issues list for this draft can be found
in <https://github.com/WICG/webpackage>.

--- middle

# Introduction

To satisfy the use cases in {{?I-D.yasskin-webpackage-use-cases}}, this document
proposes a new bundling format to group HTTP resources. Several of the use cases
require the resources to be signed: that's provided by bundling signed exchanges
({{?I-D.yasskin-http-origin-signed-responses}}) rather than natively in this
format.

## Terminology

Exchange (noun)
: An HTTP request/response pair. This can either be a request from a client and
the matching response from a server or the request in a PUSH_PROMISE and its
matching response stream. Defined by Section 8 of {{!RFC7540}}.

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL
NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED",
"MAY", and "OPTIONAL" in this document are to be interpreted as
described in BCP 14 {{!RFC2119}} {{!RFC8174}} when, and only when, they
appear in all capitals, as shown here.

# Semantics

A bundle is logically a set of HTTP exchanges, with one identified as the App
Manifest ({{appmanifest}}) of the bundle itself. That App Manifest then
identifies other significant exchanges within the bundle, for example, a start
page.

While the order of the exchanges is not semantically meaningful, it can
significantly affect performance when the bundle is loaded from a network
stream.

Bundle parsers support two operations, each of which can fail:

## Load a bundle's metadata ## {#semantics-load-metadata}

This takes a sequence of bytes representing the bundle and returns the set of
HTTP requests for the exchanges in the bundle, metadata for each request to be
passed to {{semantics-load-response}}, and identifies the request for the
bundle's App Manifest ({{appmanifest}}).

This sequence of bytes can be embedded in a longer sequence and identified by
only its start or end point.

When the bundle is identified by its start point, for example when it's being
downloaded over a network stream, this operation only inspects a prefix of the
bytes that, if the bundle is encoded with the "responses" section last, ends
before the first response.

This operation's implementation is in {{load-metadata}}.

## Load a response from a bundle ## {#semantics-load-response}

This takes the sequence of bytes representing the bundle, and the metadata
returned for one request from {{semantics-load-metadata}}, and returns the
response matching that request.

This operation can be completed without inspecting bytes other than those that
make up the loaded response, although higher-level operations like proving that
an exchange is correctly signed ({{I-D.yasskin-http-origin-signed-responses}})
may need to load other responses.

This operation's implementation is in {{load-response}}.

# Format   {#format}

## Mode of specification  {#mode}

This specification defines how conformant bundle parsers convert a sequence of
bytes into the semantics of a bundle. It does not constrain how encoders produce
such a bundle: although there are some guidelines in {{authoring-guidelines}},
encoders MAY produce any sequence of bytes that a conformant parser would parse
into the intended semantics.

In places, this specification says the parser "MUST fail". The parser MAY report
these failures to its caller in any way, but MUST NOT return any data it has
parsed so far.

This specification creates local variables with the phrase "Let *variable-name*
be ...". Use of a variable before it's created is a defect in this
specification.

## Canonical encoding ## {#canonical-encoding}

A bundle parser MUST fail to parse CBOR items that are not encoded canonically, using the rules in this section. These rules are based on Section 3.9 of {{?RFC7049}} with erratum 4964 applied.

* Integers MUST be as small as possible.
  * 0 to 23 and -1 to -24 MUST be expressed in the same byte as the major type;
  * 24 to 255 and -25 to -256 MUST be expressed only with an additional uint8_t;
  * 256 to 65535 and -257 to -65536 MUST be expressed only with an additional
    uint16_t;
  * 65536 to 4294967295 and -65537 to -4294967296 MUST be expressed only with an
    additional uint32_t.
* The expression of lengths in major types 2 through 5 MUST be as short as
  possible.  The rules for these lengths follow the above rule for integers.
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
* Indefinite-length items MUST NOT appear.

This format does not use floating point values or tags, so there's no need to
canonicalize those.

## Top-level structure  {#top-level}

The bundle is roughly a CBOR item ({{?RFC7049}}) with the following CDDL
({{?I-D.ietf-cbor-cddl}}) schema, but bundle parsers are required
to successfully parse some byte strings that aren't valid CBOR. For example,
sections may have padding between them, or even overlap, as long as the embedded
relative offsets cause the parsing algorithms in this specification to return
data.

~~~~~ cddl
bundle = [
  ; 🌐📦 in UTF-8.
  magic: h'F0 9F 8C 90 F0 9F 93 A6',
  section-offsets: {* (($section-name .within tstr) => uint) },
  sections: [* $section ],
  length: bytes .size 8,  ; Big-endian number of bytes in the bundle.
]

$section-name /= "index" / "manifest" / "critical" / "responses"

$section /= index / manifest / critical / responses

responses = [*response]

~~~~~

## Load a bundle's metadata ## {#load-metadata}

This operation takes as input a sequence of bytes representing the bundle,
identified by either its first or last byte.

If the sequence is identified by its last byte, use the procedure in
{{from-end}} to identify the first byte.

If the first 10 bytes of the bundle are not "84 48 F0 9F 8C 90 F0 9F 93 A6" (the
CBOR encoding of the 4-item array initial byte and 8-byte bytestring initial
byte, followed by 🌐📦 in UTF-8), parsing MUST fail.

Parse one CBOR item ({{!RFC7049}}) starting at the 11th byte of the bundle. If
this does not match the CDDL ({{!I-D.ietf-cbor-cddl}}):

~~~~~ cddl
section-offsets = {* tstr => uint },
~~~~~

or it is not encoded canonically ({{canonical-encoding}}), parsing MUST fail.

Let *sections-start* be the offset of the byte after the `section-offsets`
item. For example, if `section-offsets` were 52 bytes long, *sections-start*
would be 63.

Data in a bundle is expressed in named sections. The `section-offsets` item maps
the name of a section to the byte at which that section's data starts. Sections
generally consist of CBOR items, so their length is discoverable by parsing
them.

For each "*key*"/*value* pair in `section-offsets`, parse the "*key*" section
using the instructions in the section of this document identified below, passing
*sections-start* + *value* as the start byte of the section. So, for example,
the "index" section would be parsed starting at *sections-start* +
`section-offsets`\["index"] using the instructions in {{index-section}}.

The parser MUST ignore unknown keys in the `section-offsets` map, except as
specified in {{critical-section}}, to be compatible with future versions of this
specification.

This specification defines four sections:

"index"
: {{index-section}}; this section also uses the value of *sections-start* +
  `section-offsets`\["responses"].

"manifest"
: {{manifest-section}}

"critical"
: {{critical-section}}

"responses"
: This section is not parsed while loading a bundle's metadata and is instead
  used to load individual exchanges from the bundle ({{load-response}}).

If the "responses" section is not present, parsing MUST fail. If the "index"
section or another section with equivalent information (for example, a
compressed index defined by a future specification) is not present, parsing MUST
fail.

Return the values returned by parsing each section.

### Parsing the index section {#index-section}

To parse the index, starting at offset *index-start*, with a "responses" section
starting at *responses-start*, the parser MUST do the following:

Load a CBOR item starting at *index-start*. Parsing MUST fail if any of the
following is true:

* This item doesn't match the `index` rule in the following CDDL
  ({{!I-D.ietf-cbor-cddl}}).
* This item isn't encoded canonically ({{canonical-encoding}}).
* Any key is not a valid request item ({{request-items}}).

~~~ cddl
index = {* http-request => [ offset: uint,
                             length: uint] }
~~~

Adjust all of the offsets by adding *responses-start*, so they point at the
start of the response relative to the start of the whole bundle. Note that
offsets are stored relative to the responses section instead of the start of the
whole bundle so that they don't need to incorporate their own size, which would
make it more difficult to encode them canonically.

Interpret the keys as requests as described in {{request-items}}, and return
this set of requests with each request's adjusted offset and length attached.

### Parsing the manifest section {#manifest-section}

To parse the manifest section, starting at offset *manifest-start*, the parser
MUST load a CBOR item starting at *manifest-start*. If this item isn't a valid
request item, as defined by {{request-items}}, the parser MUST fail.

~~~ cddl
manifest = http-request
~~~

Return the request represented by this parsed item, interpreted as defined by
{{request-items}}.

### Parsing the critical section {#critical-section}

To parse the critical section, starting at offset *critical-start*, the parser
MUST load a CBOR item starting at *critical-start*. If this item doesn't match
the `critical` rule in the following CDDL ({{!I-D.ietf-cbor-cddl}}), the parser
MUST fail.

~~~ cddl
critical = [*tstr]
~~~

For each value *section-name* in the `critical` list, if the parser does not
support sections named *section-name*, the parser MUST fail.

This section returns no information.

### Interpreting request CBOR items {#request-items}

HTTP requests are represented by CBOR items matching the following CDDL
({{!I-D.ietf-cbor-cddl}}).

~~~ cddl
http-request = {
  * bstr => bstr
}
~~~

A valid request item *R*:

* Matches the above CDDL.
* Has only lower-case ASCII keys, matching the requirement in Section 8.1.2 of
  {{?RFC7540}}.
* Has exactly two keys starting with a ':' character, ':method' and ':url'.
* *R*\[':method'] is an HTTP method defined as cacheable (Section 4.2.3 of
  {{!RFC7231}}) and safe (Section 4.2.1 of {{!RFC7231}}), as required for
  PUSH_PROMISEd requests (Section 8.2 of {{?RFC7540}}). This currently consists
  of only the `GET` and `HEAD` methods.
* *R*\[':url'] is an absolute URI (Section 4.3 of {{!RFC3986}}).

A valid request item *R* is interpreted as an HTTP request by interpreting
*R*\[':method'] as the request's method (Section 4 of {{!RFC7231}}),
*R*\[':url'] as the request's effective request URI (Section 5.5 of
{{!RFC7230}}), and the remaining key/value pairs as the request's header fields.

### Starting from the end {#from-end}

If the 8th byte before the last isn't 0x48 (the CBOR ({{?RFC7049}}) initial byte
for an 8-byte byte string), the parser MUST fail. Otherwise, load the last 8
bytes as a big-endian integer `length`. The first byte of the bundle is `length
- 1` bytes before the end byte.

~~~ ascii-art

 +------------+-----+----+----+----+----+----+----+----+----+----+
 | first byte | ... | 48 | 00 | 00 | 00 | 00 | 00 | BC | 61 | 4E |
 +------------+-----+----+----+----+----+----+----+----+----+----+
             /       \
       0xBC614E-10=12345668 omitted bytes
~~~
{: title="Example trailing bytes" anchor="example-trailing-bytes"}

Parsing from the end allows the bundle to be appended to another format such as
a self-extracting executable.

## Load a response from a bundle ## {#load-response}

To parse the response from a *bundle* for a *request*'s *offset* and *length*,
the parser MUST do the following:

Parse one CBOR item *R* starting at *offset*. Parsing MUST fail if any of the
following is true:

* This item does not take exactly *length* bytes.
* This item isn't encoded canonically ({{canonical-encoding}}).
* This item doesn't match the `response` rule in the following CDDL
  ({{!I-D.ietf-cbor-cddl}}).
* The `headers` map doesn't have a ':status' key.
* The `headers` map has any other key starting with a ':' character.

~~~~~ cddl
response = [headers: {* bstr => bstr}, payload: bstr]
~~~~~

Return an HTTP response with a status code (Section 3.1.2 of {{!RFC7230}}) of
*R*.headers\[':status'], header fields (Section 3.2 of {{!RFC7230}}) consisting
of the other key/value pairs in *R*.headers, and a payload body (Section 3.3 of
{{!RFC7230}}) consisting of *R*.payload.

# Guidelines for bundle authors {#authoring-guidelines}

Bundles SHOULD consist of a single canonical CBOR item ({{canonical-encoding}})
matching the `webbundle` CDDL rule in {{top-level}}.

# Security Considerations # {#security}

Bundles currently have no mechanism for ensuring that they signed exchanges they
contain constitute a consistent version of those resources. Even if a website
never has a security vulnerability when resources are fetched at a single time,
an attacker might be able to combine a set of resources pulled from different
versions of the website to build a vulnerable site. While the vulnerable site
could have occurred by chance on a client's machine due to normal HTTP caching,
bundling allows an attacker to guarantee that it happens. Future work in the
{{appmanifest}} specification might allow a bundle to constrain its resources to
come from a consistent version.

# IANA considerations

## Internet Media Type Registration

IANA maintains the registry of Internet Media Types {{?RFC6838}}
at <https://www.iana.org/assignments/media-types>.

* Type name: application

* Subtype name: webbundle+cbor

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
  * Magic number(s): 84 48 F0 9F 8C 90 F0 9F 93 A6
  * File extension(s): .wbn
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

--- back

# Acknowledgements
