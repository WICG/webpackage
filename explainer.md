# Web Packaging Format Explainer

This document describes use cases for a new package format for web sites and
applications and outlines such a format. It replaces the
~~[W3C TAG's Web Packaging Draft](https://w3ctag.github.io/packaging-on-the-web/)~~.
It serves similar role as typical "Introduction" or "Using" and other
non-normative sections of specs.

## Background
Some new use cases for Web technology have motivated thinking about a multi-resource packaging format. Those new opportunities include:

### Local Sharing

Local sharing is quite popular, especially in Emerging Markets countries, due to
cost and limitations on cellular data and relatively spotty WiFi availability.
It is typically done over local Bluetooth/WiFi, by either built-in OS features
like [Android Beam](https://en.wikipedia.org/wiki/Android_Beam) or with popular
3-rd party apps, such as
[ShareIt](https://play.google.com/store/apps/details?id=com.lenovo.anyshare.gps)
or [Xender](https://play.google.com/store/apps/details?id=cn.xender)).
Typically, the locally stored media files and apps (APK files for Android for
example) are shared this way. Extending sharing to bundles of content and web
apps (Progressive Web Apps in particular) opens up new possibilities:

#### Unsigned snapshots

Any client can snapshot the page they're currently reading. This is currently
done mostly by capturing a screenshot or saving into a browser-specific format
like [MHTML](https://tools.ietf.org/html/rfc2557)
or [Web Archive](https://en.wikipedia.org/wiki/Webarchive), but these only
support at most a single page per archive, and the second
set [aren't supported by other browsers](https://xkcd.com/927/).

#### Signed applications

If the original origin signs a package, the contents can get full access to
browser state and network requests for that origin. This lets people share full
PWAs peer-to-peer.

Local sharing tends to be popular where connectivity to the web is expensive:
each byte costs money. This means the client may be stuck with an outdated
version of a package for a significant amount of time, including that package's
vulnerabilities. It may be feasible to periodically check for OCSP notification
that a package's certificate has been revoked. We also need to design a cheap
notification that the package is critically vulnerable and needs to be disabled
until it can be updated.

### Physical Web
[Beacons](https://google.github.io/physical-web/) and other physical web devices often want to 'broadcast' various content locally. Today, they broadcast a URL and make the user's device go to a web site. This delivers the trusted content to the user's browser (user can observe the address bar to verify) and allow web apps to talk back to their services. It can be useful to be able to broadcast a package containing several pages or even a simple web app, even without a need to immediately have a Web connection - for example, via Bluetooth. If combined with signature from the publisher, the loaded pages may be treated as if they were loaded via TLS connection with a valid certificate, in terms of the [origin-based security model](https://tools.ietf.org/html/rfc6454). For example, they can use [`fetch()`](https://fetch.spec.whatwg.org/#fetch-api) against its service or use "Add To Homescreen" for the convenience of the user.

Physical web beacons may be located in a place that has no connectivity, and
their owner may only visit to update them as often as their battery needs
replacement: annually or less often. This leaves a large window during which a
certificate could be compromised or a package could get out of date. We think
there's no way to prevent a client from trusting its initial download of a
package signed by a compromised certificate. When the client gets back to a
public network, it should attempt to validate both the certificate and the
package using the mechanisms alluded to under [Local Sharing](#local-sharing).

### Content Distribution Networks and Web caches.
The CDNs can provide service of hosting web content that should be delivered at scale. This includes both hosting subresources (JS libraries, images) as well as entire content ([Google AMP](https://developers.google.com/amp/cache/overview)) on network of servers, often provided as a service by 3rd party. Unfortunately, origin-based security model of the Web limits the ways a 3rd-party caches/servers can be used. Indeed, for example in case of hosting JS subresources, the original document must explicitly trust the CDN origin to serve the trusted script. The user agent must use protocol-based means to verify the subresource is coming from the trusted CDN. Another example is a CDN that caches the whole content. Because the origin of CDN is different from the origin of the site, the browser normally can't afford the origin treatment of the site to the loaded content. Look at how an article from USA Today is represented:

<img align="center" width=350 src="buick.png">

Note the address bar indicating google.com. Also, since the content of USA Today is hosted in an iframe, it can't use all the functionality typically afforded to a top-level document:
- Can't request permissions
- Can't be added to homescreen

Packages served to CDNs can staple an OCSP response and have a short expiration
time, avoiding the above problems with outdated packages.


## Goals and non-goals

See
https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#requirements

## Proposal

We propose to introduce a packaging format for the Web that would be able to contain multiple resources (HTML, CSS, Images, Media files, JS files etc) in a "bundle". That bundle can be distributed as regular Web resource (over HTTP[S]) and also by non-Web means, which includes local storage, local sharing, non-HTTP protocols like Bluetooth, etc. Being a single "bundle", it facilitates various modes of transfer. The **packages may be nested**, providing natural way to represent the bundles of resources corresponding to different origins and sites.

In addition, that format would include optional **signing** of the resources, which can be used to verify authenticity and integrity of the content. Once and if verified (this may or may not require network connection), the content can be afforded the treatment of the claimed origin - for example showing a "green lock" with URL in a browser, or being able to send network request to the origin's server. This disconnects the verification of the origin from actual network connection and enables many new scenarios for the web content to be consumed, including time-shifted delivery (when content is delivered by an opportunistic restartable download for example), peer-to-peer sharing or caching on local file servers.

Since the packaged "bundle" can be quite large (a game with a lot of resources or content of multiple web sites), efficient access to that content becomes important. For example, it would be often prohibitively expensive to "unpack" or somehow else pre-process such a large resource on the client device. Unpacking, for example, may require twice the space to be occupied in device's storage, which can be a problem, especially on low-end devices. We propose a optional **Content Index** structure that allows the bundle to be consumed (browsed) efficiently as is, without unpacking - by adding an index-like structure which provides direct offsets into the package.

This is roughly based on the existing [Packaging on the Web](https://w3ctag.github.io/packaging-on-the-web/) W3C TAG proposal, but we've made significant changes:

1. Index of content to facilitate local resource fetching from the package.
2. Package manifests that can refer to sub-package manifests to allow resources
   from multiple origins.
3. Signature block, to cryptographically sign the content of the package.
4. Removed the
   [fragment-based URL schema](https://w3ctag.github.io/packaging-on-the-web/#fragment-identifiers)
   from the spec as it's complex with limited use cases.

Note that this is just an explainer, **not a specification**. We'll add more
precision when we translate it to a spec.

### Overall format

The package is a [CBOR-encoded data item](https://tools.ietf.org/html/rfc7049)
with MIME type `application/package+cbor`. It logically contains a flat sequence
of resources represented as HTTP responses. The package also includes metadata
such as a manifest and an index to let consumers validate the resources and
access them directly.

The overall structure of the item is described by the following
[CDDL](https://tools.ietf.org/html/draft-greevenbosch-appsawg-cbor-cddl):

```cddl
webpackage = [
  magic1: h'F0 9F 8C 90 F0 9F 93 A6',  ; 🌐📦 in UTF-8.
  section-offsets: { * (($section-name .within tstr) => offset) },
  sections: ({ * $$section }) .within ({ * $section-name => any }),
  length: uint,                        ; Total number of bytes in the package.
  magic2: h'F0 9F 8C 90 F0 9F 93 A6',  ; 🌐📦 in UTF-8.
]

; Offsets are measured from the first byte of the webpackage item to the first
; byte of the target item.
offset = uint
```

Each section-offset points to a section with the same key, by holding the byte
offset from the start of the webpackage item to the start of the section's name
item.

The length holds the total length in bytes of the `webpackage` item and must be
encoded in the uint64_t format, which makes it possible to build self-extracting
executables by appending a normal web package to the extractor executable.

The defined section types are:

* [`"indexed-content"`](#main-content): The only required section.
  Maps resource keys (URLs possibly extended with HTTP headers) to
  HTTP2 responses. The mapping uses byte offsets to allow random
  access to any resource.
* [`"manifest"`](#manifest): Validates that resources came from the expected
  source. May refer to other manifests among the responses. If this section
  isn't provided, the resources are un-signed and can be loaded as untrusted
  data.

More sections may be defined later. If an unexpected section is encountered, it
is ignored.

Note that this top-level information is *not signed*, and so can't be trusted.
Only information in the manifest and below can be trusted.

### Main content

The main content of a package is an index of HTTP requests pointing to HTTP
responses. These request/response pairs hold the manifests of sub-packages and
the resources in the package and all of its sub-packages. Both the requests and
responses can appear in any order, usually chosen to optimize loading while the
package is streamed.

```cddl
$section-name /= "indexed-content"
$$section //= ("indexed-content" => [
  index: [* [resource-key, offset, ? length: uint] ],
  responses: [* [response-headers: http-headers, body: bstr]],
])

resource-key = uri / http-headers

; http-headers is a byte string in HPACK format (RFC7541).
; The dynamic table begins empty for each instance of http-headers.
http-headers = bstr
```

A `uri` `resource-key` is equivalent to an `http-headers` block with ":method"
set to "GET" and with ":scheme", ":authority", and ":path" headers set from the
URI as described in
[RFC7540 section 8.1.2.3](https://tools.ietf.org/html/rfc7540#section-8.1.2.3).

As an optimization, the `resource-key`s in the index store relative instead of
absolute URLs. Each entry is resolved relative to the resolved version of the
previous entry.

TODO: Consider random access into large indices.

In addition to the CDDL constraints:

* All byte strings must use a definite-length encoding so that package consumers
  can parse the content directly instead of concatenating the indefinite-length
  chunks first. The definite lengths here may also help a package consumer to
  quickly send resources to other threads for parsing.
* The index must not contain two resolved `resource-key`s with the
  same [header list](http://httpwg.org/specs/rfc7541.html#rfc.section.1.3) after
  HPACK decoding.
* The `resource-key` must not contain any headers that aren't either ":method",
  ":scheme", ":authority", ":path", or listed in the
  `response-headers`'
  ["Vary" header](https://tools.ietf.org/html/rfc7231#section-7.1.4).
* The `resource-key` must contain at most one of each ":method", ":scheme",
  ":authority", ":path" header, in that order, before any other headers.
  Resolving the `resource-key` fills in any missing pseudo-headers from that
  set, ensuring that all resolved keys have exactly one of each.

The optional `length` field in the index entries is redundant with the length
prefixes on the `response-headers` and `body` in the content, but it can be used
to issue [Range requests](https://tools.ietf.org/html/rfc7233) for responses
that appear late in the `content`.


### Manifest

TODO: Now that this no longer contains
a [manifest](https://www.merriam-webster.com/dictionary/manifest#h3),
consider renaming it to something like "authenticity".

A package's manifest contains some metadata for the
package, [hashes](#validating-resources) for all resources included in that
package, and validity information for any [sub-packages](#sub-packages) the
package depends on. The manifest is signed, so that UAs can trust that it comes
from its claimed origin.

```cddl
$section-name /= "manifest"
$$section //= ("manifest" => signed-manifest)

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
  ; RFC5280 says certificates can be identified by either the
  ; issuer-name-and-serial-number or by the subject key identifier. However,
  ; issuer names are complicated, and the subject key identifier only identifies
  ; the public key, not the certificate, so we identify certificates by their
  ; index in the certificates array instead.
  keyIndex: uint,
  ; Encoded as described in TLS 1.3,
  ; https://tlswg.github.io/tls13-spec/#signature-algorithms.
  signature: bstr,
}
```

The metadata must include an absolute URL identifying
the
[origin](https://html.spec.whatwg.org/multipage/browsers.html#concept-origin)
vouching for the package and the date the package was created. It may contain
more keys defined in https://www.w3.org/TR/appmanifest/.

#### Manifest signatures

The manifest is signed by a set of certificates, including at least one that is
trusted to sign content from the
manifest's
[origin](https://html.spec.whatwg.org/multipage/browsers.html#concept-origin).
Other certificates can sign to vouch for the package along other dimensions, for
example that it was checked for malicious behavior by some authority.

The signed sequence of bytes is the concatenation of the following byte strings.
This matches the TLS1.3 format to avoid cross-protocol attacks when TLS
certificates are used to sign manifests.
1. A string that consists of octet 32 (0x20) repeated 64 times.
1. A context string: the ASCII encoding of "Web Package Manifest".
1. A single 0 byte which serves as a separator.
1. The bytes of the `manifest` CBOR item.

Each signature uses the `keyIndex` field to identify the certificate used to
generate it.
The [TLS 1.3 signing algorithm](https://tlswg.github.io/tls13-spec/#rfc.section.4.2.3)
is determined from the certificate's public key type:

* RSA, 2048 bits: rsa_pss_sha256
* secp256r1: ecdsa_secp256r1_sha256
* secp384r1: ecdsa_secp384r1_sha384

The following key types are not supported, for the mentioned reason:

* secp521r1: [Chrome doesn't support this curve](https://crbug.com/477623), so
  certificates aren't using it on the web.
* ed25519 and ed448:
  The [RFC](https://tools.ietf.org/html/draft-josefsson-tls-ed25519) for using
  these in certificates isn't yet final.
* RSA != 2048: Only 4096 has measurable usage, and it's very low.

As a special case, if the package is being transferred from the manifest's
origin under TLS, the UA may load it without checking that its own resources match
the manifest. The UA still needs to validate resources provided by sub-manifests.


#### Certificates

The `signed-manifest.certificates` array should contain enough
X.509 certificates to chain from the signing certificates, using the rules
in [RFC5280](https://tools.ietf.org/html/rfc5280), to roots trusted by all
expected consumers of the package.

[Sub-packages](#sub-packages') manifests can contain their own certificates or
can rely on certificates in their parent packages.

Requirements on the
certificates' [Key Usage](https://tools.ietf.org/html/rfc5280#section-4.2.1.3)
and [Extended Key Usage](https://tools.ietf.org/html/rfc5280#section-4.2.1.12)
are TBD. It may or may not be important to prevent TLS serving certificates from
being used to sign packages, in order to prevent cross-protocol attacks.


#### Validating resources

For a resource to be valid, then for each `hash-algorithm => [hash-value]` in
`resource-hashes`, the resource's hash using that algorithm needs to appear in
that list of `hash-value`s. Like
in [Subresource Integrity](https://www.w3.org/TR/SRI/#agility), the UA will only
check one of these, but it's up to the UA which one.

The hash of a resource is the hash of its Canonical CBOR encoding using the
following CDDL. Headers are decompressed before being encoded and hashed.

``` cddl
resource = [
  request: [
    ':method', bstr,
    ':scheme', bstr,
    ':authority', bstr,
    ':path', bstr,
    * (header-name, header-value: bstr)
  ],
  response-headers: [
    ':status', bstr,
    * (header-name, header-value: bstr)
  ],
  response-body: bstr
]

# Headers must be lower-case ascii per
# http://httpwg.org/specs/rfc7540.html#rfc.section.8.1.2, and only
# pseudo-headers can include ":".
header-name = bstr .regexp "[\x21-\x39\x3b-\x40\x5b-\x7e]+"
```

This differs from [SRI](https://w3c.github.io/webappsec-subresource-integrity),
which only hashes the body. Note: This will usually prevent a package from
relying on some of its contents being transferred as normal network responses,
unless its author can guarantee the network won't change or reorder the headers.


#### Sub-packages

A sub-package is represented by a [manifest](#manifest) file in
the [`"content"`](#main-content) section, which contains hashes of resources
from another origin. The sub-package's resources are not otherwise distinguished
from the rest of the resources in the package. Sub-packages can form an
arbitrarily-deep tree.

There are three possible forms of dependencies on sub-packages, of which we
allow two. Because a sub-package is protected by its
own [signature](#signatures), if the main package trusts the sub-package's
server, it could avoid specifying a version of the sub-package at all. However,
this opens the main package up to downgrade attacks, where the sub-package is
replaced by an older, vulnerable version, so we don't allow this option.

```cddl
subpackage = [
  resource: resource-key,
  validation: {
    ? hash: hashes,
    ? notbefore: time,
  }
]
```

If the main package wants to load either the sub-package it was built with or
any upgrade, it can specify the date of the original sub-package:

```cbor-diag
[32("https://example.com/loginsdk.package"), {"notbefore": 1(1486429554)}]
```

Constraining packages with their date makes it possible to link together
sub-packages with common dependencies, even if the sub-packages were built at
different times.

If the main package wants to be certain it's loading the exact version of a
sub-package that it was built with, it can constrain sub-package with a hash of its manifest:

```cbor-diag
[32("https://example.com/loginsdk.package"),
 {"hash": {"sha256": 22(b64'9qg0NGDuhsjeGwrcbaxMKZAvfzAHJ2d8L7NkDzXhgHk=')}}]
```

Note that because the sub-package may include sub-sub-packages by date, the top
package may need to explicitly list those sub-sub-packages' hashes in order to
be completely constrained.


## Examples

Following are some example usages that correspond to these additions.
The packages are written in
[CBOR's extended diagnostic notation](https://tools.ietf.org/html/draft-greevenbosch-appsawg-cbor-cddl-10#appendix-G),
with the extensions that:
1. `hpack({key:value,...})` is an [hpack](https://tools.ietf.org/html/rfc7541)
   encoding of the described headers.
2. `DER(...)` is the DER encoding of a certificate described partially by the
   contents of the `...`.

All examples are available in the [examples](examples) directory.

### Single site: a couple of web pages with resources in a package.
The example web site contains two HTML pages and an image. This is straightforward case, demonstrating the following:

1. The `section-offsets` section declares one main section starting 1 byte into
   the `sections` item. (The 1 byte is the map header for the `sections` item.)
2. The `index` maps [hpack](http://httpwg.org/specs/rfc7541.html)-encoded
   headers for each resource to the start of that resource, measured relative to
   the start of the `responses` item.
3. Each resource contains `date`/`expires` headers that specify when the
   resource can be used by UA, similar to HTTP 1.1
   [Expiration Model](https://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html#sec13.2).
   The actual expiration model is TBD and to be reflected in the spec. Note that
   we haven't yet described a way to set an `expires` value for the whole
   package at once.
4. The length of the whole package always appears from the 10th to 18th bytes
   before the end of the package, in big-endian format.

```cbor-diag
['🌐📦',
    {"indexed-content": 1},
    {"indexed-content":
        [
            [ # Index.
                [hpack({
                    :method: GET,
                    :scheme: https
                    :authority: example.com
                    :path: /index.html
                }), 1],
                [hpack({
                    :method: GET
                    :scheme: https
                    :authority: example.com
                    :path: /otherPage.html
                }), 121],
                [hpack({
                    :method: GET
                    :scheme: https
                    :authority: example.com
                    :path: /images/world.png
                }), 243]
            ],
            [ # Resources.
                [
                    hpack({
                        :status: 200
                        content-type: text/html
                        date: Wed, 15 Nov 2016 06:25:24 GMT
                        expires: Thu, 01 Jan 2017 16:00:00 GMT
                    }),
                    '<body>\n  <a href=\"otherPage.html\">Other page</a>\n</body>\n'
                ],
                [
                    hpack({
                        :status: 200
                        content-type: text/html
                        date: Wed, 15 Nov 2016 06:25:24 GMT
                        expires: Thu, 01 Jan 2017 16:00:00 GMT
                    }),
                    '<body>\n  Hello World! <img src=\"images/world.png\">\n</body>\n'
                ], [
                    hpack({
                        :status: 200
                        content-type: image/png
                        date: Wed, 15 Nov 2016 06:25:24 GMT
                        expires: Thu, 01 Jan 2017 16:00:00 GMT
                    }),
                    '... binary png image ...'
                ]
            ]
        ]
    },
    473,  # Always 8 bytes long.
    '🌐📦'
]
```

### Multiple Origins: a web page with a resources from the other origin.

The example web site contains an HTML page and pulls a script from the
well-known location (different origin). Note that there's no need to distinguish
the resources from other origins vs the ones from the main origin. Since none of
them are signed, the browser won't treat any as
[same-origin](https://html.spec.whatwg.org/multipage/browsers.html#same-origin)
with their claimed origin.

```cbor-diag
['🌐📦',
    {"indexed-content": 1},
    {"indexed-content":
        [
            [
                [hpack({
                    :method: GET
                    :scheme: https
                    :authority: example.com
                    :path: /index.html
                }), 1],
                [hpack({
                    :method: GET
                    :scheme: https
                    :authority: ajax.googleapis.com
                    :path: /ajax/libs/jquery/3.1.0/jquery.min.js
                }), 179]
            ],
            [
                [
                    hpack({
                        :status: 200
                        content-type: text/html
                        date: Wed, 15 Nov 2016 06:25:24 GMT
                        expires: Thu, 01 Jan 2017 16:00:00 GMT
                    }),
                    '<head>\n<script src=\"https://ajax.googleapis.com/ajax/libs/jquery/3.1.0/jquery.min.js\"></script>\n<body>\n...\n</body>\n'
                ],
                [
                    hpack({
                        :status: 200
                        content-type: text/html
                        date: Wed, 15 Nov 2016 06:25:24 GMT
                        expires: Thu, 01 Jan 2017 16:00:00 GMT
                    }),
                    '... some JS code ...\n'
                ]
            ]
        ]
    },
    396,
    '🌐📦'
]
```

### Use Case: Signed package, one origin.
The example contains example.com/index.html. The package is signed by the
example.com publisher, using the same private key that example.com uses for
HTTPS. The signed package ensures the verification of the origin even if the
package is stored in a local file or obtained via other insecure ways like HTTP,
or hosted on another origin's server.

Some interesting things to notice in this package:

1. The `"manifest"` map contains `"certificates"` and `"signatures"` arrays
   describing how the manifest is signed.
2. The signature identifies the first element of `"certificates"` as the signing
   certificate.
3. The elements of `"certificates"` are
   DER-encoded [X.509 certificates](https://tools.ietf.org/html/rfc5280).
   The [signing certificate](go/webpack/testdata/pki/example.com.cert) is
   trusted for `example.com`, and that certificate chains,
   using [other elements](go/webpack/testdata/pki/intermediate1.cert) of
   `"certificates"`, to
   a [trusted root certificate](go/webpack/testdata/pki/root1.cert). The chain
   is built and trusted in the same way as TLS chains during normal web
   browsing.
4. The signature algorithm is determined by the signing certificate's public key
   type, `prime256v1`, and isn't encoded separately in the signature block.
5. The manifest contains a `"resource-hashes"` block, which contains the hashes,
   using the SHA384 algorithm in this case, of all resources in the package.
   Unlike in
   [Subresource Integrity](https://w3c.github.io/webappsec-subresource-integrity/),
   the hashes include the request and response headers.
6. The inclusion of a certificate chain makes it possible to validate the
   package offline. Browsers detect revoked certificates and packages with known
   vulnerabilities by looking for separately signed files containing OCSP and
   recency information, but this package does not demonstrate how to attach
   those.

```cbor-diag
[
  '🌐📦',
  {
    "manifest": 1,
    "indexed-content": 2057
  },
  {
    "manifest": {
      "manifest": {
        "metadata": {
          "date": 1(1494583200),
          "origin": 32("https://example.com")
        },
        "resource-hashes": {
          "sha384": [
            h'3C3A03F7C3FC99494F6AAA25C3D11DA3C0D7097ABBF5A9476FB64741A769984E8B6801E71BB085E25D7134287B99BAAB',
            h'5AA8B83EE331F5F7D1EF2DF9B5AFC8B3A36AEC953F2715CE33ECCECD58627664D53241759778A8DC27BCAAE20F542F9F',
            h'D5B2A3EA8FE401F214DA8E3794BE97DE9666BAF012A4B515B8B67C85AAB141F8349C4CD4EE788C2B7A6D66177BC68171'
          ]
        }
      },
      "signatures": [
        {
          "keyIndex": 0,
          "signature": h'3044022015B1C8D46E4C6588F73D9D894D05377F382C4BC56E7CDE41ACEC1D81BF1EBF7E02204B812DACD001E0FD4AF968CF28EC6152299483D6D14D5DBE23FC1284ABB7A359'
        }
      ],
      "certificates": [
        DER(
          Certificate:
              ...
              Signature Algorithm: ecdsa-with-SHA256
                  Issuer: C=US, O=Honest Achmed's, CN=Honest Achmed's Test Intermediate CA
                  Validity
                      Not Before: May 10 00:00:00 2017 GMT
                      Not After : May 18 00:10:36 2018 GMT
                  Subject: C=US, O=Test Example, CN=example.com
                  Subject Public Key Info:
                      Public Key Algorithm: id-ecPublicKey
                          Public-Key: (256 bit)
                          pub:
                              ...
                          ASN1 OID: prime256v1
                  ...
        ),
        DER(
          Certificate:
              ...
              Signature Algorithm: sha256WithRSAEncryption
                  Issuer: C=US, O=Honest Achmed's, CN=Honest Achmed's Test Root CA
                  Validity
                      Not Before: May 10 00:00:00 2017 GMT
                      Not After : May 18 00:10:36 2018 GMT
                  Subject: C=US, O=Honest Achmed's, CN=Honest Achmed's Test Intermediate CA
                  Subject Public Key Info:
                      Public Key Algorithm: id-ecPublicKey
                          Public-Key: (521 bit)
                          pub:
                              ...
                          ASN1 OID: secp521r1
                  ...
        )
      ]
    },
    "indexed-content": [
      [
        [ hpack({
            :method: GET
            :scheme: https
            :authority: example.com
            :path: /index.html
          }), 1]
        [ hpack({
            :method: GET
            :scheme: https
            :authority: example.com
            :path: /otherPage.html
          }), 121],
        [ hpack({
            :method: GET
            :scheme: https
            :authority: example.com
            :path: /images/world.png
          }), 243]
        ],
      ],
      [
        [ hpack({
            :status: 200
            content-type: text/html
            date: Wed, 15 Nov 2016 06:25:24 GMT
            expires: Thu, 01 Jan 2017 16:00:00 GMT
          }),
          '<body>\n  <a href=\"otherPage.html\">Other page</a>\n</body>\n'
        ]
        [ hpack({
            :status: 200
            content-type: text/html
            date: Wed, 15 Nov 2016 06:25:24 GMT
            expires: Thu, 01 Jan 2017 16:00:00 GMT
          }),
          '<body>\n  Hello World! <img src=\"images/world.png\">\n</body>\n'
        ],
        [ hpack({
            :status: 200
            content-type: image/png
            date: Wed, 15 Nov 2016 06:25:24 GMT
            expires: Thu, 01 Jan 2017 16:00:00 GMT
          }),
          '... binary png image ...'
        ]
      ]
    ]
  },
  2541,
  '🌐📦'
]
```

The process of validation:

1. Verify that certificates identified by signature elements chain to trusted roots.
2. Find the subset of the signatures that correctly sign the manifest's bytes
   using their identified certificates' public keys.
3. Parse the manifest and find its claimed origin.
4. Verify that at least one correct signature identifies a certificate that's
   trusted for use by that origin.
5. When loading a resource, pick the strongest hash function in the
   `"resource-hashes"` map, and use that to hash the Canonical CBOR
   representation of its request headers, response headers, and body. Verify
   that the resulting digest appears in that array in the `"resource-hashes"`
   map.

**_Examples below here are out of date_**

### Use Case: Signed package, 2 origins

Lets add signing to the example mentioned above where a page uses a cross-origin JS library, hosted on https://ajax.googleapis.com. Since this package includes resources from 2 origins, this means there are 2 packages, one of them nested. Both of them should be signed by their respective publisher, since for the main page to be validated as secure (green lock, origin access) all resources that comprise it must be signed/validated - equivalent of them being loaded via HTTPS.

Important notes:

1. Nested package with the JS library, obtained from googleapis.com, is separately signed by googleapis.com
2. Nested packages may have their own signatures and Content Index.  They can be included verbatim as a [part](https://w3ctag.github.io/packaging-on-the-web/#parts) of the outer package. Therefore, their index entries will be relative to the inner package.  This does mean that accessing a part of a nested package will require multiple index lookups depending on how deeply nested a package is, as it will be necessary to locate the inner package using the outer package's Content Index.
3. Alternative for example.com would be to include the JS library into its own package and sign it as part of example.com, but this is useful example on how the nested signed package looks like.
4. The nested package has been indented for illustration purposes but would not be in an actual package.


```html
Package-Signature: NNejtdEjGnea4VTvO7A/x+5ucZm+pGPkQ1TD32oT3oKGhPWeF0hASWjxQOXvfX5+; algorithm=sha384; certificate=urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/package
Content-Location: https://example.org/examplePack.pack
Link: </index.html>; rel=describedby
Link: <https://ajax.googleapis.com/packs/jquery_3.1.0.pack>; rel=package; scope=/ajax/libs/jquery/3.1.0
Link: <urn:uuid:d479c10b-58cc-4243-97a5-0e02b2c3f47a>; rel=index; offset=12014/2048

--j38n02qryf9n0eqny8cq0
Content-Location: /index.html
Content-Type: text/html
<head>
<script src="https://ajax.googleapis.com/ajax/libs/jquery/3.1.0/jquery.min.js"></script>
<body>
...
</body>
--j38n02qryf9n0eqny8cq0
	Package-Signature: A/xtdEjGnea4VTvNNejO7+5ucZm+pGPkQ1TD32oT3oKGhPWeF0hASWjx5+QOXvfX; algorithm=sha384; certificate=urn:uuid:7af4c10b-58cc-4372-8567-0e02b2c3dabc
	Content-Location: https://ajax.googleapis.com/packs/jquery_3.1.0.pack
	Content-Type: application/package
	Link: <urn:uuid:aaf4c10b-58cc-4372-8567-0e02b2c3daaa>; rel=index; offset=12014/2048

	--klhfdlifhhiorefioeri1
	Content-Location: /ajax/libs/jquery/3.1.0/jquery.min.js
	Content-Type: application/javascript

	... some JS code ...
	--klhfdlifhhiorefioeri1   (This is Content Index for ajax.googleapis.com subpackage)
	Content-Location: urn:uuid:aaf4c10b-58cc-4372-8567-0e02b2c3daaa
	Content-Type: application/package.index

	/ajax/libs/jquery/3.1.0/jquery.min.js sha384-3dEjGnea4A/xtGPkQ1TDVTvNNejO7+5ucZm+pASWjx5+QOXvfX2oT3oKGhPWeF0h 102 3876
	... other entries ...
	--klhfdlifhhiorefioeri1
	Content-Location: urn:uuid:7af4c10b-58cc-4372-8567-0e02b2c3dabc
	Content-Type: application/pkix-cert

	... certificate for ajax.googleapi.com ...
	--klhfdlifhhiorefioeri1--
--j38n02qryf9n0eqny8cq0   (This is Content Index for example.com package)
Content-Location: urn:uuid:d479c10b-58cc-4243-97a5-0e02b2c3f47a
Content-Type: application/package.index

/index.html sha384-WeF0h3dEjGnea4ANejO7+5/xtGPkQ1TDVTvNucZm+pASWjx5+QOXvfX2oT3oKGhP 153 215
--j38n02qryf9n0eqny8cq0
Content-Location: urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/pkcs7-mime

... certificate for example.com ...
--j38n02qryf9n0eqny8cq0--

```

## FAQ

### Why signing but not encryption? HTTPS provides both...

The signing part of the proposal addresses *integrity* and *authenticity* aspects of the security. It is enough for the resource to be signed to validate it belongs to the origin corresponding to the certificate used. This, in turn allows the browsers and other user agents to afford the 'origin treatment' to the resources in the package, because there is a guarantee that those resources were not tampered with.

### What about certificate revocation? Many use cases assume package is validated while offline.

Indeed, as is the case with web browsers as well, certificate revocation is not instant on the web. In case of packages that are consumed while device is offline (maybe for a long period of time), the revocation of the certificate may not reach device promptly. But then again, if the web resources were stored in a browser cache, or if pages were Saved As, and used when device is offline, there would be no way to receive the CRL or use OCSP for real-time certificate validation as well. Once the device is online, the certificate should be validated using best practices of the user agent and access revoked if needed.

### Is that Package-Signature a MAC or HMAC of the package?

No, we don't use what commonly is called [MAC](https://en.wikipedia.org/wiki/Message_authentication_code) here because the packages are not encrypted (and there is no strong use case motivating such encryption) so there is no symmetrical key and therefore the traditional concept of MAC is not applicable. However, the Package-Signature contains a [Digital Signature](https://en.wikipedia.org/wiki/Digital_signature) which is a hash of the (Content Index + Package Header) signed with a private key of the publisher. The Content Index contains hashes for each resource part included in the package, so the Package-Signature validates each resource part as well.

### Does the Package-Signature cover all the bits of the package?

Yes, the Package Header and Content Index are hashed and this hash, signed, is provided in the Package-Signature header. The Content Index, in turn, has hashes for all resources (Header+Body) so all bits of package are covered.

### Are subpackages signed as well?

No. If a package contains subpackages, those subpackages are not covered by the package's signature or hashes and have to have their own Package-Signature header if they need to be signed. This reflects the fact that subpackages typically group resources from a different origin, with their own certificate. The [sub]packages are the units that are typically package resources from their respective origins and are therefore separately signed.

### What happens if urn:uuid: URLs collide?

Since they are Version 4 UUIDs, the chances of them colliding are vanishingly small.

### What if a publisher signed a package with a JS library, and later discovered a vulnerability in it. On the Web, they would just replace the JS file with an updated one. What is the story in case of packages?

The expiration headers in Package Headers section prescribe the 'useful lifetime' of the package, with UA optionally indicating the 'stale' state to the user and asking to upgrade, or automatically fetching a new one. While offline, the expiration may be ignored (not unlike Cache-Control: no-cache) but once user is online, the UA should verify both the certificate and if the package Content-Location contains an updated package (per Package Headers section) - and replace the package if necessary. In general, if the device is online and the package is expired, and the original location has updated package, the UA should obtain a new one (details TBD).

### Why is there a Content Index that specifies where each part is, and also MIME-like 'boundaries' that separate parts in the package?

This is due to two main use cases of package loading:

1. Loading the package from Web as part of the page or some other resource. In this case, the package is streamed from the server and **boundaries** allow to parse the package as it comes in and start using subresources as fast as possible. If the package has to be signed though, the package in its entirety has to be loaded first.
2. Loading a (potentially large) package offline. In that case, it is important to provide a fast access to subresources as they are requested, w/o unpacking the package (it takes double the storage at least to unpack and significant time). Using direct byte-offset Content Index allows to directly access resources in a potentially large package.

### Does the package supply a full chain of certificates to a known CA root?

Not necessarily. Different devices have different sets of roots in their trust
stores, so there is not a single "correct" set of certificates to send that will
work best for all clients. Instead, for compatibility, packages may need to
include a set of certificates from which chains can be built to multiple roots,
or rely on clients to dynamically fetch additional intermediates when needed.

This becomes a tradeoff between the package size vs the set of clients that can
validate the signature offline. We expect that packaging tools will allow their
users to configure this tradeoff in appropriate ways.

