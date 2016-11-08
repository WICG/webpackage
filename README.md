# Web Packaging Format Explainer
This document intends to list use cases motivating the improvements of the current [Web Packaging Draft](https://w3ctag.github.io/packaging-on-the-web/). It serves similar role as typical "Introduction" or "Using" and other non-normative sections of specs.

## Background
Some new use cases for Web technology have motivated thinking about a multi-resource packaging format. Those new opportunity include:

### Local Sharing

Local sharing is quite popular, especially in Emerging Markets countries, due to cost and limitations on cellular data and relatively spotty WiFi availability. It is typically done over local Bluetooth/WiFi, by either built-in OS features like [Android Beam](https://en.wikipedia.org/wiki/Android_Beam) or with popular 3-rd party apps, such as [ShareIt](https://play.google.com/store/apps/details?id=com.lenovo.anyshare.gps) or [Xender](https://play.google.com/store/apps/details?id=cn.xender)). Typically, the locally stored media files and apps (APK files for Android for example) are shared this way. Extending sharing to bundles of content and web apps (Progressive Web Apps in particular) opens up new possibilities, especially if combined with a form of signing the content. Cryptographic signing could make it possible to afford the shared content the treatment normally reserved for the origin that content is claiming to be in - by verifying that the content indeed was produced by the holder of the corresponding certificate.

### Physical Web.
[Beacons](https://google.github.io/physical-web/) and other physical web devices often want to 'broadcast' various content locally. Today, they broadcast a URL and make the user's device go to a web site. This delivers the trusted content to the user's browser (user cna observe the address bar to verify) and allow web apps to talk back to their services. It can be useful to be able to broadcast a package containing several pages or even a simple web app, even without a need to immediately have a Web connection - for example, via Bluetooth. If combined with signature form the publisher, the loaded pages may be treated as if they were laoded via TLS connection with a valid certificate, in terms of [origin-based security model](https://tools.ietf.org/html/rfc6454). For example, they can use XMLHttpRequest against its service or use "Add To Homescreen" for the convenience of the user.

### Content Distribution Networks and Web caches.
The CDNs can provide service of hosting web content that should be delivered at scale. This includes both hosting subresources (JS libraries, images) as well as entire content ([Google AMP](https://developers.google.com/amp/cache/overview)) on network of servers, often provided as a service by 3rd party. Unfortunately, origin-based security model of the Web limits the ways a 3rd-party caches/servers can be used. Indeed, for example in case of hosting JS subresourcves, the original document must explicitly trust the CDN origin to serve the trusted script. The user agent must use protocol-based means to verify the subresource is coming from the trusted CDN. Another example is a CDN that caches the whole content. Because the origin of CDN is different from the origin of the site, the browser normally can't afford the origin treatment of the site tot he loaded content. Look at how an article from USA Today is represented:

<img align="center" width=350 src="buick.png">

Note the address bar indicating google.com. Also, since the content of USA Today is hosted in an iframe, it can't use all the functionality typically afforded to a top-level document:
- Can't request permissions
- Can't be added to homescreen
- Can't install a Service Worker (true???).

## Proposal

We propose to introduce a packaging format for the Web that would be able to contain multiple resources (HTML, CSS, Images, Media files, JS files etc) in a "bundle". That bundle can be distributed as regular Web resource (over HTTP[S]) and also by non-Web means, which includes local storage, local sharing, non-HTTP protocols like Bluetooth, etc. Being a single "bundle", it facilitates various modes of transfer. The **packages may be nested**, providing natural way to represent the bundles of resources corresponding to different origins and sites.

In addition, that format would include optional **signing** of the resources, which can be used to verify authenticity and integrity of the content. Once and if verified (this may or may not require network connection), the content can be afforded the treatment of the claimed origin - for example showing a "green lock" with URL in a browser, or being able to send network request to the origin's server. This disconnects the verification of the origin from actual network connection and enables many new scenarios for the web content to be consumed, including time-shifted delivery (when content is delivered by an opportunistic restartable download for example), peer-to-peer sharing or caching on local file servers.

Since the packaged "bundle" can be quite large (a game with a lot of resources or content of multiple web sites), the efficient access to that content becomes important. For example, it would be often prohibitively expensive to "unpack" or somehow else pre-process such a large resource on the client device. Unpacking, for example, may require a double size occupied in device's storage, which can be a problem, especially on low-end devices. We propose a optional **Content Index** structure that allows the bundle to be consumed (browsed) efficiently as is, without unpacking - by adding an index-like structure which provides direct offsets into the package.

There is already a [packaging format proposal](https://w3ctag.github.io/packaging-on-the-web/) which we will base upon.
We are proposing to improve on the spec, in particular by introducing 3 major additions:

1. Hierarchical structure of the sub-packages to allow resources from multiple origins.
2. Index of content to facilitate local resource fetching from the package.
3. Signature block, to cryptographically sign the content of the package.

We also propose to remove the [fragment-based URL schema](https://w3ctag.github.io/packaging-on-the-web/#fragment-identifiers) from the spec as it is not clear what would be the use case supporting it.

Following are some example usages that correspond to these additions:

### Use Case: a couple of web pages with resources in a package.
The example web site contains two HTML pages and an image. This is straightforward case, demonstrating the following:

1. [Package Header](https://w3ctag.github.io/packaging-on-the-web/#package-header) section at the beginning. It contains a Content-Location of the package, which also serves as base URL to resolve the relative URLs of the [parts](https://w3ctag.github.io/packaging-on-the-web/#parts). So far, this is straight example of the package per existing spec draft.
2. Note the "main resource" of the package specified by Link: header with **rel=describedby** in th ePackage Header section.

```html
Content-Type: application/package
Content-Location: http://example.org/examplePack.pack
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
4. Note the parts of this package is not signed (see below for optional signing) so when such package is opened by a browser, and index.html page is loaded, there is no way to validate the stated origin of https://example.com and the package should be treated as if it was local (file:) resource. This can be useful in many cases, but browsers should be careful to assign an unique origin to resources of such packages, as [discussed here](https://tools.ietf.org/html/rfc6454#section-4) for file: URLs.
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
2. Content Index part is typically generated during package creation, it doesn't have a natural URL. We propose to use cid: generated URLs (UUID-based, 128-bit) for such generated parts. The visibility scope of those URLs is limited similar to package boundaries, and is for the current package only.
3. Content-type of the Content Index is application/index.
4. The Content Index consists of the Content Index Entries (see below for the discussion of what they are).
5. Content Index part may be compressed (as specified by Transfer-Encoding header).


```html
Content-Type: application/package
Content-Location: http://example.org/examplePack.pack
Link: </index.html>; rel=describedby
Link: <cid:f47ac10b-58cc-4372-a567-0e02b2c3d479>; rel=index; offset=12014/2048

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
Content-Location: cid:f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/index

/index.html     0xde7c9b85b8b78aa6bc8a7a36f70a90701c9db4d9 153 215
/otherPage.html 0xabc126434d7fed989ca0e3d88379acef897ffc98 368 180
/mages/world.png     0x4d7fed989ca0eef897ffc996f70a90701c9989ca 548 1024
--j38n02qryf9n0eqny8cq0--
```

#### Content Index Entry
Each Content Index entry is a line that looks like following:
> /index.html 0xde7c9b85b8b78aa6bc8a7a36f70a90701c9db4d9 153 215

Where:

```
content-index-entry = part-id SP part-hash SP part-offset SP part-size CRLF
  part-id = part-url [":" <headers that are mentioned in Vary: header of the part>]
  part-hash = <cryptographic hash of the part>
  part-location = <octet offset of the part from the beginning of the package>
  part-size = <octet size of the part>
```

The **part-hash** is used in signing (see below) and can be optional (filled with 0x0 value) if signing is not used.

### Use Case: Signed package, one origin.
The example contains an HTML page and an image. The package is signed by the example.com publisher, using the same private key that example.com uses for HTTPS. The signed package ensures the verification of the origin even if the package is stored in a local file or obtained via other insecure ways like HTTP, or hosted on another origin's server.

Important notes:

1. The very first header in Package Header section of the package is **Package-Signature**, a new header that contains an encrypted hash of the Package Header section (not including Package-Signature header) and Content Index. It also contains a reference (via cid: UUID-based URL) to the part that contains the public key certificate (or if needed, a chain of certificates to the root CA).
2. The Content Index contains hashes of all parts of the package, so it is enough to validate the index to trust its hashes, then compute the hash of the each part upon using it to validate each part.
3. The inclusion of certificate makes it possible to validate the package offline (certificate revocation aside, this can be done out-of-band when device is actually online).
4. Certificate is included as one of standard the DER-encoded resource (with proper Content-type).

```html
Package-Signature: 0xabc126434d7fed989ca0e3d88379acef897ffc98; certificate=cid:f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/package
Content-Location: http://example.org/examplePack.pack
Link: </index.html>; rel=describedby
Link: <cid:d479c10b-58cc-4243-97a5-0e02b2c3f47a>; rel=index; offset=12014/2048

--j38n02qryf9n0eqny8cq0
Content-Location: /index.html
Content-Type: text/html

<body>
  Hello World!
</body>
--j38n02qryf9n0eqny8cq0
Content-Location: cid:d479c10b-58cc-4243-97a5-0e02b2c3f47a
Content-Type: application/index

/index.html 0xde7c9b85b8b78aa6bc8a7a36f70a90701c9db4d9 153 215
--j38n02qryf9n0eqny8cq0
Content-Location: cid:f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/pkcs7-mime

... certificate (or a chain) in any of the
--j38n02qryf9n0eqny8cq0--
```

The process of validation:

1. Decrypt the hash provided by Package-Signature header (lets call result of decryption HPackage) using the provided public key cert.
2. Compute the hash of the application/index part. Lets call it HIndex.
3. If HPackage == HIndex, the index of the archive is deemed validated.
4. When a part is loaded, compute its hash and compare it with the hash in the Content Index. If they match, the part is deemed validated.


### Use Case: Signed package, 2 origins

Lets add signing to example mentioned above where a page used cross-origin JS library, hosted on https://ajax.googleapis.com. Since this package includes resources fmor 2 origins, this means there are 2 packages, one of them nested. Both of them should be signed by their respective publisher, since for the main page to be validated as secure (green lock, origin access) all resources that conprise it must be signed/validated - equivalent of them being loaded via HTTPS.

Important notes:

1. Nested package with the JS library, obtained from googleapis.com, is separately signed by googleapis.com
2. Alternative for example.com would be to include the JS library into its own package and sign it as part of example.com, but this is useful example on how the nested signed package looks like.
3. The nested package indented for illustration purposes.

```html
Package-Signature: 0xd83he34d7fed989ca0e3d88379acef897ffc11; certificate=cid:f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/package
Content-Location: https://example.org/examplePack.pack
Link: </index.html>; rel=describedby
Link: <https://ajax.googleapis.com/packs/jquery_3.1.0.pack>; rel=package; scope=/ajax/libs/jquery/3.1.0
Link: <cid:d479c10b-58cc-4243-97a5-0e02b2c3f47a>; rel=index; offset=12014/2048

--j38n02qryf9n0eqny8cq0
Content-Location: /index.html
Content-Type: text/html
<head>
<script src="https://ajax.googleapis.com/ajax/libs/jquery/3.1.0/jquery.min.js"></script>
<body>
...
</body>
--j38n02qryf9n0eqny8cq0
	Package-Signature: 0xabc126434d7fed989ca0e3d88379acef897ffc98; certificate=cid:7af4c10b-58cc-4372-8567-0e02b2c3dabc
	Content-Location: https://ajax.googleapis.com/packs/jquery_3.1.0.pack
	Content-Type: application/package
	Link: <cid:aaf4c10b-58cc-4372-8567-0e02b2c3daaa>; rel=index; offset=12014/2048

	--klhfdlifhhiorefioeri1
	Content-Location: /ajax/libs/jquery/3.1.0/jquery.min.js
	Content-Type" application/javascript

	... some JS code ...
	--klhfdlifhhiorefioeri1   (This is Content Index for ajax.googleapis.com subpackage)
	Content-Location: cid:aaf4c10b-58cc-4372-8567-0e02b2c3daaa
	Content-Type: application/index

	/ajax/libs/jquery/3.1.0/jquery.min.js 0x5b8b78de7c9b8aa6bc8a7a36f70a9db4d990701c 102 3876
	... other entries ...
	--klhfdlifhhiorefioeri1
	Content-Location: cid:7af4c10b-58cc-4372-8567-0e02b2c3dabc
	Content-Type: application/pkix-cert

	... certificate for ajax.googleapi.com ...
	--klhfdlifhhiorefioeri1--
--j38n02qryf9n0eqny8cq0   (This is Content Index for example.com package)
Content-Location: cid:d479c10b-58cc-4243-97a5-0e02b2c3f47a
Content-Type: application/index

/index.html 0xde7c9b85b8b78aa6bc8a7a36f70a90701c9db4d9 153 215
--j38n02qryf9n0eqny8cq0
Content-Location: cid:f47ac10b-58cc-4372-a567-0e02b2c3d479
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

>Is Package-Signature covering all the bits of the package?

Yes, the Package Header and Content Index are hashed and this hash, signed, is provided in the Package-Signature header. The Content Index, in turn, has hashes for all resources (Header+Body) so all bits of package are covered.

>Are subpackages signed as well?

No. If a package contains subpackages, those subpackages are not covered by the package's signature or hashes and have to have their own Package-Signature header if they need to be signed. This reflects the fact that subpackages typically group resources from a different origin, with their own certificate. The [sub]packages are the units that are typically package resources from their respective origins and are therefore separately signed.

>What exactly are cid: URLs mentioned in this doc? Are they the same cid: URLs described in [RFC-2392](https://tools.ietf.org/html/rfc2392)?

No, they are a bit different. They are simply a random-generated (as [Version 4 UUIDs](https://tools.ietf.org/html/rfc4122)) content IDs used exclusively inside the package to refer to the generated, non-content parts of the package - namely Content Index and Certificate used for signature.

>What happens if cid: URLs collide?

Since they are Version 4 UUIDs, the chances of them colliding are disappearingly small.

