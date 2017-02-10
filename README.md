# Web Packaging Format Explainer
This document intends to list use cases motivating the improvements of the current [Web Packaging Draft](https://w3ctag.github.io/packaging-on-the-web/). It serves similar role as typical "Introduction" or "Using" and other non-normative sections of specs.

## Background
Some new use cases for Web technology have motivated thinking about a multi-resource packaging format. Those new opportunity include:

### Local Sharing

Local sharing is quite popular, especially in Emerging Markets countries, due to cost and limitations on cellular data and relatively spotty WiFi availability. It is typically done over local Bluetooth/WiFi, by either built-in OS features like [Android Beam](https://en.wikipedia.org/wiki/Android_Beam) or with popular 3-rd party apps, such as [ShareIt](https://play.google.com/store/apps/details?id=com.lenovo.anyshare.gps) or [Xender](https://play.google.com/store/apps/details?id=cn.xender)). Typically, the locally stored media files and apps (APK files for Android for example) are shared this way. Extending sharing to bundles of content and web apps (Progressive Web Apps in particular) opens up new possibilities, especially if combined with a form of signing the content. Cryptographic signing could make it possible to afford the shared content the treatment normally reserved for the origin that content is claiming to be in - by verifying that the content indeed was produced by the holder of the corresponding certificate.

### Physical Web
[Beacons](https://google.github.io/physical-web/) and other physical web devices often want to 'broadcast' various content locally. Today, they broadcast a URL and make the user's device go to a web site. This delivers the trusted content to the user's browser (user can observe the address bar to verify) and allow web apps to talk back to their services. It can be useful to be able to broadcast a package containing several pages or even a simple web app, even without a need to immediately have a Web connection - for example, via Bluetooth. If combined with signature from the publisher, the loaded pages may be treated as if they were loaded via TLS connection with a valid certificate, in terms of the [origin-based security model](https://tools.ietf.org/html/rfc6454). For example, they can use [`fetch()`](https://fetch.spec.whatwg.org/#fetch-api) against its service or use "Add To Homescreen" for the convenience of the user.

### Content Distribution Networks and Web caches.
The CDNs can provide service of hosting web content that should be delivered at scale. This includes both hosting subresources (JS libraries, images) as well as entire content ([Google AMP](https://developers.google.com/amp/cache/overview)) on network of servers, often provided as a service by 3rd party. Unfortunately, origin-based security model of the Web limits the ways a 3rd-party caches/servers can be used. Indeed, for example in case of hosting JS subresources, the original document must explicitly trust the CDN origin to serve the trusted script. The user agent must use protocol-based means to verify the subresource is coming from the trusted CDN. Another example is a CDN that caches the whole content. Because the origin of CDN is different from the origin of the site, the browser normally can't afford the origin treatment of the site to the loaded content. Look at how an article from USA Today is represented:

<img align="center" width=350 src="buick.png">

Note the address bar indicating google.com. Also, since the content of USA Today is hosted in an iframe, it can't use all the functionality typically afforded to a top-level document:
- Can't request permissions
- Can't be added to homescreen

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

The package is logically structured as a flat sequence of resources represented
as HTTP responses, combined with an optional index and an optional tree of
signed manifests. The top-level manifest verifies the authenticity of resources
from the overall package's origin and contains a list of sub-manifests which
each verify a sub-package's resources.

The physical structure follows. All "offsets" are 8-byte little-endian
integers representing a byte offset from the start of the initial magic number.

1. A magic number: `F0 9F 8C 90 F0 9F 93 A6` (🌐📦 in UTF-8).
2. The length of the package (all bytes from the first of the initial magic
   number to the last of the final magic number) as a little-endian 8-byte
   integer.
3. A 1-byte count of metadata sections followed by that many of the following
   entries. Repeated entry IDs and non-monotonically-increasing offsets cause
   the package to fail to parse.
   * The byte `01` indicating an [index](#index), followed by the offset of the
     start of the index.
   * The byte `02` indicating a block of [certificates](#certificates), followed
     by the offset of the start of the certificates.
   * The byte `03` indicating a [manifest](#manifest), followed by the offset of
     the start of the manifest.

   Unknown entry IDs are assumed to be followed by an offset, and the pair must
   be ignored.
4. The offset of the [main content](#main-content) of the package.
5. The metadata sections promised above.
6. A sequence of HTTP responses constituting the main content of the package.
   Mismatches with the manifest usually cause the package to fail to parse, as
   described in the [manifest](#manifest) section, but mismatches with the index
   might not.
7. The length of the package, repeated from earlier. This allows tools to
   operate on the package even if it's been prepended with a self-extracting
   executable.
8. The same magic number: `F0 9F 8C 90 F0 9F 93 A6` (🌐📦 in UTF-8).

All metadata sections and each resource in the main content is prefixed with its
8-byte little-endian length.

The MIME type of this format is `application/package`. TODO: Is that right?

Note that this top-level information is *not signed*, and so can't be trusted.
Only information in the manifest and below can be trusted.

The manifest and certificates are used to sign the contents of the package. The
index is used when the package is stored on disk, to find resources quickly.

### Index

The index is optional, and can be added by a client after the package is
downloaded.

It consists of its own 8-byte length, followed by a series of lines (terminated
by `\r\n`) of the form

```
URL offset
```

Where an offset is a 64-bit integer (written out as digits), and the URLs are
absolute. Offsets are interpreted relative to the first byte of the package.
There's no need to include a resource's length in the index because the offset
points directly to the length at the start of the resource. All resources in the
package should be included in the index, even if they come from a different
origin.

TODO: Provide base URLs, maybe as INI-format sections.

```
https://example.org/index.html 7000 500
https://example.org/img.png 7500 1000
...
```

### Certificates

The certificates block contains an 8-byte length followed by
an
[application/pkcs7-mime; smime-type=certs-only](https://tools.ietf.org/html/rfc5751) resource
holding enough [X.509](https://tools.ietf.org/html/rfc5280) certificates that
chains can be built from the certificates that sign the top-level
and [sub-package](#sub-packages) manifests to root certificates recognized by
the client.

Requirements on the
certificates' [Key Usage](https://tools.ietf.org/html/rfc5280#section-4.2.1.3)
and [Extended Key Usage](https://tools.ietf.org/html/rfc5280#section-4.2.1.12)
are TBD. It may or may not be important to prevent TLS serving certificates from
being used to sign packages, in order to prevent cross-protocol attacks.

### Manifest

A package's manifest lists all of the resources included in that package, and
any sub-packages the package depends on. It consists of an 8-byte length
followed by an HTTP response including at least the `Content-Type`,
`Content-Location`, and `Date` headers, and one or
more [signatures](#signatures).

At least one of a manifest's signatures must be from a certificate that has a
valid chain to a known root, and that is trusted to sign content from the
[origin](https://html.spec.whatwg.org/multipage/browsers.html#concept-origin) of
the manifest's `Content-Location`.

If no signature gives the manifest authority to speak for its origin, or if any
resource or sub-package doesn't match the description in the manifest, the whole
package must fail to parse. TODO: Should there be a way to express that
particular resources don't need to verify, in order to let the UA load the beginning of a package before the whole package has transferred?

As a special case, if the package is being transferred from the manifest's
origin under TLS, the UA may load it before checking that the resources match
the manifest.

TODO: Should we just extend
the [Web App Manifest](https://www.w3.org/TR/appmanifest/), or does the fact
that we're going to sign this rule out JSON?

Resources are described by a line consisting of:

```
relative/url hashalgorithm-base64digest hashalgorithm2-base64digest2 ...
```

The URL is relative to the manifest's `Content-Location`.

The hashes are in the format defined for the
[`integrity` HTML attribute](https://w3c.github.io/webappsec-subresource-integrity/#the-integrity-attribute).
This allows multiple hashes in order to provide agility in the face of future
cryptographic discoveries. See also https://tools.ietf.org/html/rfc7696.

The hashes are computed over the entire resource, including its length, HTTP
headers, and body of the named resource. This differs
from [SRI](https://w3c.github.io/webappsec-subresource-integrity), which only
hashes the body. Note: This will usually prevent a package from relying on some
of its contents being transferred as normal network responses, unless its author
can guarantee the network won't change or reorder the headers.

TODO: Do we need the length to be shown in the manifest, or is its presence at
the beginning of the resource, included in the hash, enough?


#### Sub-packages

There are three possible forms of dependencies on sub-packages, of which we
allow two. Note that a sub-package is or can be protected by its
own [signature](#signatures), so if the main package trusts the sub-package's
server, it could avoid specifying a version of the sub-package at all. However,
this opens the main package up to downgrade attacks, where the sub-package is
replaced by an older, vulnerable version, so we don't allow this option.

If the main package wants to load either the sub-package it was built with or
any upgrade, it can specify the date of the original sub-package:

```
https://package/manifest/url Date: Tue, 07 Feb 2017 01:05:54 GMT
```

Constraining packages with their date makes it possible to link together
sub-packages with common dependencies, even if the sub-packages were built at
different times.

If the main package wants to be certain it's loading the exact version of a
sub-package that it was built with, it can constrain sub-package with a hash of its manifest:

```
https://package/manifest/url hashalgorithm-base64digest hashalgorithm2-base64digest2 ...
```

Note that because the sub-package may include sub-sub-packages by date, the top
package may need to explicitly list those sub-sub-packages' hashes in order to
be completely constrained.

It should be possible to additionally constrain packages using semver, for the
benefit of NPM and other packaging systems, but I haven't explored that
extension here.


#### Signatures

A [manifest](#manifest) is signed by one or more `Signature` headers. The
message to sign consists of the length, followed by the concatenation of all
headers except any `Signature` headers, followed by the body of the manifest. A
`Signature` header has the form:

``` http
Signature: keyId="cert1",algorithm="ed25519",signature="Base64(ed25519(privatekey(cert1), message))"
```

This is similar to the `Signature` header specified by
https://tools.ietf.org/html/draft-cavage-http-signatures-06, except that we sign
all headers instead of a specified subset, and we include the body instead of
requiring an extra Digest header.

TODO: We also need to define the format of the `keyId`, which must identify one
of the certificates sent with the package. I don't know the best format for this
ID.

The algorithm must be one of the signature algorithms defined
by [TLS1.3](https://tlswg.github.io/tls13-spec/#rfc.section.4.2.3), except for
the ones marked "SHOULD NOT be offered". TODO: Limit to a smaller set? TODO:
Identify algorithms by their enum value or some other identifier?

Multiple `Signature` headers are allowed in order to
allow [cryptographic agility](https://tools.ietf.org/html/rfc7696) and to allow
multiple entities to sign a package, for example to express that the package was
checked for malicious behavior by some authority in addition to its author.

### Main content

The main content of a package is a sequence of 8-byte-length-prefixed HTTP
responses holding the manifests of sub-packages and the resources in the package
and all of its sub-packages. These resources can appear in any order.

## Use cases

Following are some example usages that correspond to these additions:

### Use Case: a couple of web pages with resources in a package.
The example web site contains two HTML pages and an image. This is straightforward case, demonstrating the following:

1. See the [Package Header](https://w3ctag.github.io/packaging-on-the-web/#package-header) section at the beginning. It contains a Content-Location of the package, which also serves as base URL to resolve the relative URLs of the [parts](https://w3ctag.github.io/packaging-on-the-web/#parts). So far, this is straight example of the package per existing spec draft.
2. The Package Header section also contains Date/Expires headers that specify when the package can be used by UA, similar to HTTP 1.1 [Expiration Model](https://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html#sec13.2). The actual expiration model is TBD and to be reflected in the spec.
3. Note the "main resource" of the package specified by Link: header with **rel=describedby** in the Package Header section.

```html
Content-Type: application/package
Content-Location: http://example.org/examplePack.pack
Date: Wed, 15 Nov 2016 06:25:24 GMT
Expires: Thu, 01 Jan 2017 16:00:00 GMT
Link: </index.html>; rel=describedby

--j38n02qryf9n0eqny8cq0
Content-Location: /index.html
Content-Type: text/html

<body>
  <a href="otherPage.html">Other page</a>
</body>
--j38n02qryf9n0eqny8cq0
Content-Location: /otherPage.html
Content-Type: text/html

<body>
  Hello World! <img src="images/world.png">
</body>
--j38n02qryf9n0eqny8cq0
Content-Location: /images/world.png
Content-Type: image/png
Transfer-Encoding: binary

... binary png image ...
--j38n02qryf9n0eqny8cq0--
```


### Use Case: a web page with a resources from the other origin.
The example web site contains an HTML page and pulls a script from the well-known location (different origin). Note the usage of the nested package to contain a resource (JS library) from a separate origin, as well as the "forward declaration" of the package via **Link:** header. We propose the nested packages to be the only way to keep resources from the origins different from the Content-Location origin of the main package itself.

There are a few things shown here:

1. Note the nested package uses its own separate boundary string.
2. The **scope=** attribute of the Link: header is used to resolve the "https://ajax.googleapis.com/ajax/libs/jquery/3.1.0/jquery.min.js" URL in the page to the package that contains it. The simple text match is used.
3. The nested package from https://ajax.googleapis.com is used that contains the js file referenced by the main page. Note the package may contain other resources as well and can be signed by googleapis.com.
4. Note the parts of this package is not signed (see below for optional signing) so when such package is opened by a browser, and index.html page is loaded, there is no way to validate the stated origin of https://example.com and the package should be treated as if it was local (file:) resource. This can be useful in many cases, but browsers should be careful to assign a unique origin to resources of such packages, as [discussed here](https://tools.ietf.org/html/rfc6454#section-4) for file: URLs.
5. While the resources in such package may not be trusted, there can be many interesting use cases addressed by such easy-to-produce packages - peer-to-peer sharing of web pages' snapshots for example, which is mostly done today by capturing screenshot images and then sharing them via WhatsApp or similar.

```html
Content-Type: application/package
Content-Location: https://example.org/examplePack.pack
Link: </index.html>; rel=describedby
Link: <https://ajax.googleapis.com/packs/jquery_3.1.0.pack>; rel=package; scope=/ajax/libs/jquery/3.1.0

--j38n02qryf9n0eqny8cq0
Content-Location: /index.html
Content-Type: text/html
<head>
<script src="https://ajax.googleapis.com/ajax/libs/jquery/3.1.0/jquery.min.js"></script>
<body>
...
</body>
--j38n02qryf9n0eqny8cq0
Content-Location: https://ajax.googleapis.com/packs/jquery_3.1.0.pack
Content-Type: application/package

--klhfdlifhhiorefioeri1
Content-Location: /ajax/libs/jquery/3.1.0/jquery.min.js
Content-Type" application/javascript

... some JS code ...
--klhfdlifhhiorefioeri1--
--j38n02qryf9n0eqny8cq0--
```


### Use case: Optional Content Index
The package in this case contains a lot of pages with resources ("Encyclopedia in a file") or multiple sites (in subpackages). The proposed structure facilitates efficient access, assuming the whole package is available locally. Several important notes:

1. The Link: header with **rel=index** declares a specified part to be a Content Index of the package. The **offset=12014** attribute specifies the octet offset/size from the beginning of the Package Header of the package to the Content Index part. That can be used in file-seek APIs to quickly read the part without a need to parse the potentially huge package itself.
2. Content Index part is typically generated during package creation, it doesn't have a natural URL. We propose to use [`urn:uuid:`](https://tools.ietf.org/html/rfc4122) generated URLs (UUID-based, 128-bit) for such generated parts. The visibility scope of those URLs is limited similar to package boundaries, and is for the current package only.
3. Content-type of the Content Index is application/package.index.
4. The Content Index consists of the Content Index Entries (see below for the discussion of what they are).
5. Content Index part may be compressed (as specified by Transfer-Encoding header).


```html
Content-Type: application/package
Content-Location: http://example.org/examplePack.pack
Link: </index.html>; rel=describedby
Link: <urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479>; rel=index; offset=12014/2048

--j38n02qryf9n0eqny8cq0
Content-Location: /index.html
Content-Type: text/html

<body>
  <a href="otherPage.html">Other page</a>
</body>
--j38n02qryf9n0eqny8cq0
Content-Location: /otherPage.html
Content-Type: text/html

<body>
  Hello World! <img src="images/world.png">
</body>
--j38n02qryf9n0eqny8cq0
Content-Location: /images/world.png
Content-Type: image/png
Transfer-Encoding: binary

... binary png image ...
--j38n02qryf9n0eqny8cq0
Content-Location: urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/package.index

/index.html     sha384-Li9vy3DqF8tnTXuiaAJuML3ky+er10rcgNR/VqsVpcw+ThHmYcwiB1pbOxEbzJr7 153 215
/otherPage.html sha384-8tnTXuiaAJuMLi9vy3DqFL3ky+er10rcgN1pbOxEbzJr7R/VqsVpcw+ThHmYcwiB 368 180
/images/world.png     sha384-vy3DqFLi98t3ky+er10nTXuiaAJuMLrczJr7gNR/VqsVpcw+ThHmYcwiB1pbOxEb 548 1024
--j38n02qryf9n0eqny8cq0--
```

#### Content Index Entry
Each Content Index entry is a line that looks like following:
> /index.html sha384-Li9vy3DqF8tnTXuiaAJuML3ky+er10rcgNR/VqsVpcw+ThHmYcwiB1pbOxEbzJr7 153 215

Where:

```
content-index-entry = part-id SP part-hash SP part-offset SP part-size CRLF
  part-id = part-url [":" <headers that are mentioned in Vary: header of the part>]
  part-hash = <hash-algorithm> "-" <hash of the part>
  part-location = <octet offset of the part from the beginning of the current package>
  part-size = <octet size of the part>
```

The **part-hash** is used in signing (see below) and can be optional (filled with 0x0 value) if signing is not used.

### Use Case: Signed package, one origin.
The example contains an HTML page and an image. The package is signed by the example.com publisher, using the same private key that example.com uses for HTTPS. The signed package ensures the verification of the origin even if the package is stored in a local file or obtained via other insecure ways like HTTP, or hosted on another origin's server.

Important notes:

1. The very first header in Package Header section of the package is **Package-Signature**, a new header that contains a signed hash of the Package Header section (not including Package-Signature header) and Content Index. It also contains a reference (via urn:uuid: UUID-based URL) to the part that contains the public key certificate (or if needed, a chain of certificates to the root CA).
2. The **certificate algorithm** must be encoded within the certificate that signed the package. The **algorithm** in **Package-Signature** is the hash algorithm used to sign the **Content Index** and produce the Package-Signature.
3. The Content Index contains hashes of all parts of the package, so it is enough to validate the index to trust its hashes, then compute the hash of each part upon using it to validate each part. Hashes have the hash algorithm specified in front.
4. Content Index Entry `part-location` and `part-size` must not refer to locations outside of the package which contains the entry or to locations within nested packages. They may refer to the boundaries of a nested package.
5. The inclusion of certificate makes it possible to validate the package offline (certificate revocation aside, this can be done out-of-band when device is actually online).
6. Certificate is included as a standard DER-encoded resource (with proper Content-type) corresponding to a [X.509 certificate](https://tools.ietf.org/html/rfc5280). The policies for verifying the validity of the certificate are left up to the host environment. There are no requirements on any fields in the certificate such as [Key Usage](https://tools.ietf.org/html/rfc5280#section-4.2.1.3); this also is left up to the host environment.

```html
Package-Signature: Li9vy3DqF8tnTXuiaAJuML3ky+er10rcgNR/VqsVpcw+ThHmYcwiB1pbOxEbzJr7; algorithm: sha384; certificate=urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/package
Content-Location: http://example.org/examplePack.pack
Link: </index.html>; rel=describedby
Link: <urn:uuid:d479c10b-58cc-4243-97a5-0e02b2c3f47a>; rel=index; offset=12014/2048

--j38n02qryf9n0eqny8cq0
Content-Location: /index.html
Content-Type: text/html

<body>
  Hello World!
</body>
--j38n02qryf9n0eqny8cq0
Content-Location: urn:uuid:d479c10b-58cc-4243-97a5-0e02b2c3f47a
Content-Type: application/package.index

/index.html sha384-WeF0h3dEjGnea4ANejO7+5/xtGPkQ1TDVTvNucZm+pASWjx5+QOXvfX2oT3oKGhP 153 215
--j38n02qryf9n0eqny8cq0
Content-Location: urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/pkcs7-mime

... certificate (or a chain) in any of the
--j38n02qryf9n0eqny8cq0--
```

The process of validation:

1. Validate the signature provided by Package-Signature header, using provided public key cert and content of application/package.index part of the package.
2. That establishes authenticity and integrity of the Content Index that contains hashes of all the parts int he package.
2. When a part is loaded, compute its hash and compare it with the hash in the Content Index. If they match, the part is deemed validated.


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

##FAQ

> Why signing but not encryption? HTTPS provides both...

The signing part of the proposal addresses *integrity* and *authenticity* aspects of the security. It is enough for the resource to be signed to validate it belongs to the origin corresponding to the certificate used. This, in turn allows the browsers and other user agents to afford the 'origin treatment' to the resources in the package, because there is a guarantee that those resources were not tampered with.

>What about certificate revocation? Many use cases assume package is validated while offline.

Indeed, as is the case with web browsers as well, certificate revocation is not instant on the web. In case of packages that are consumed while device is offline (maybe for a long period of time), the revocation of the certificate may not reach device promptly. But then again, if the web resources were stored in a browser cache, or if pages were Saved As, and used when device is offline, there would be no way to receive the CRL or use OCSP for real-time certificate validation as well. Once the device is online, the certificate should be validated using best practices of the user agent and access revoked if needed.

>Is that Package-Signature a MAC or HMAC of the package?

No, we don't use what commonly is called [MAC](https://en.wikipedia.org/wiki/Message_authentication_code) here because the packages are not encrypted (and there is no strong use case motivating such encryption) so there is no symmetrical key and therefore the traditional concept of MAC is not applicable. However, the Package-Signature contains a [Digital Signature](https://en.wikipedia.org/wiki/Digital_signature) which is a hash of the (Content Index + Package Header) signed with a private key of the publisher. The Content Index contains hashes for each resource part included in the package, so the Package-Signature validates each resource part as well.

>Does the Package-Signature cover all the bits of the package?

Yes, the Package Header and Content Index are hashed and this hash, signed, is provided in the Package-Signature header. The Content Index, in turn, has hashes for all resources (Header+Body) so all bits of package are covered.

>Are subpackages signed as well?

No. If a package contains subpackages, those subpackages are not covered by the package's signature or hashes and have to have their own Package-Signature header if they need to be signed. This reflects the fact that subpackages typically group resources from a different origin, with their own certificate. The [sub]packages are the units that are typically package resources from their respective origins and are therefore separately signed.

>What happens if urn:uuid: URLs collide?

Since they are Version 4 UUIDs, the chances of them colliding are vanishingly small.

>What if a publisher signed a package with a JS library, and later discovered a vulnerability in it. On the Web, they would just replace the JS file with an updated one. What is the story in case of packages?

The expiration headers in Package Headers section prescribe the 'useful lifetime' of the package, with UA optionally indicating the 'stale' state to the user and asking to upgrade, or automatically fetching a new one. While offline, the expiration may be ignored (not unlike Cache-Control: no-cache) but once user is online, the UA should verify both the certificate and if the package Content-Location contains an updated package (per Package Headers section) - and replace the package if necessary. In general, if the device is online and the package is expired, and the original location has updated package, the UA should obtain a new one (details TBD).

>Why is there a Content Index that specifies where each part is, and also MIME-like 'boundaries' that separate parts in the package?

This is due to two main use cases of package loading:

1. Loading the package from Web as part of the page or some other resource. In this case, the package is streamed from the server and **boundaries** allow to parse the package as it comes in and start using subresources as fast as possible. If the package has to be signed though, the package in its entirety has to be loaded first.
2. Loading a (potentially large) package offline. In that case, it is important to provide a fast access to subresources as they are requested, w/o unpacking the package (it takes double the storage at least to unpack and significant time). Using direct byte-offset Content Index allows to directly access resources in a potentially large package.

>Is the certificate supplied by package a full chain to the known CA root?

Not necessarily. To quote one of the Chrome security engineers, "because different devices have different sets of roots in their trust stores, it's not always the case that there is a single "correct" set of certificates to send that will work best for all clients. Instead, for compatibility, sites will sometimes send a set of certificates and rely on clients to dynamically fetch additional intermediates when needed". We suspect that to be an interesting issue since offline validation would need a full chain. Browsers (at least Chrome) are implementing automatic fetching of intermediate certificates but for this to work they have to be online. This will probably become a matter of best practices when creating the packaging format, possibly with validation by the packaging tool.


