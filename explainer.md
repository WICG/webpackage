# Web Packaging Format Explainer

This document describes use cases for packaging websites and explains how to use
the cluster of specifications in this repository to accomplish those use cases.
It serves similar role as typical "Introduction" or "Using" and other
non-normative sections of specs. The specifications here replace the ~~[W3C
TAG's Web Packaging Draft](https://w3ctag.github.io/packaging-on-the-web/)~~.

<!-- TOC -->

- [Basic terminology](#basic-terminology)
- [Component documents](#component-documents)
  - [List of use cases](#list-of-use-cases)
  - [Signed HTTP exchanges](#signed-http-exchanges)
  - [Bundled exchanges](#bundled-exchanges)
  - [Loading specification](#loading-specification)
- [Use cases](#use-cases)
  - [Offline installation](#offline-installation)
  - [Save and share a web page](#save-and-share-a-web-page)
  - [Privacy-preserving prefetch](#privacy-preserving-prefetch)
  - [Packaged Web Publications](#packaged-web-publications)
  - [Third-party security review](#third-party-security-review)
- [Signed Exchange Loading Sketch](#signed-exchange-loading-sketch)
  - [Fetch the distributing URL](#fetch-the-distributing-url)
  - [Fetch the certificate chain](#fetch-the-certificate-chain)
  - [Signature verification](#signature-verification)
  - [Prefetching stops here](#prefetching-stops-here)
  - [No nested signed exchanges](#no-nested-signed-exchanges)
  - [Navigations and subresources redirect](#navigations-and-subresources-redirect)
  - [Matching prefetches with subresources](#matching-prefetches-with-subresources)
  - [To consider: Cache the inner exchange](#to-consider-cache-the-inner-exchange)
- [FAQ](#faq)
  - [Why signing but not encryption? HTTPS provides both...](#why-signing-but-not-encryption-https-provides-both)
  - [What if a publisher accidentally signs a non-public resource?](#what-if-a-publisher-accidentally-signs-a-non-public-resource)
  - [What about certificate revocation, especially while offline?](#what-about-certificate-revocation-especially-while-offline)
  - [What if a publisher signed a package with a JS library, and later discovered a vulnerability in it. On the Web, they would just replace the JS file with an updated one. What is the story in case of packages?](#what-if-a-publisher-signed-a-package-with-a-js-library-and-later-discovered-a-vulnerability-in-it-on-the-web-they-would-just-replace-the-js-file-with-an-updated-one-what-is-the-story-in-case-of-packages)

<!-- /TOC -->

## Basic terminology

An **HTTP exchange** consists of an HTTP request and its response.

A **publisher** (like https://theestablishment.co/) writes (or has an **author**
write) some content and owns the domain where it's published. A **client** (like
Firefox) downloads content and uses it. An **intermediate** (like Fastly, the
AMP cache, or old HTTP proxies) downloads content from its author (or another
intermediate) and forwards it to a client (or another intermediate).

When an HTTP exchange is encoded into a resource, the resource can be fetched
from a **distributing URL** that is different from the **publishing URL** of the
encoded exchange. We talk about the **inner** exchange and its inner request and
response, the **outer** resource it's encoded into, and sometimes the outer
exchange whose response contains the outer resource.

## Component documents

The [original web packaging IETF draft,
draft-yasskin-dispatch-web-packaging](https://tools.ietf.org/html/draft-yasskin-dispatch-web-packaging)
has been superseded by the following documents and specifications:

### List of use cases

[draft-yasskin-webpackage-use-cases](https://wicg.github.io/webpackage/#go.draft-yasskin-webpackage-use-cases.html)
([IETF
draft](https://tools.ietf.org/html/draft-yasskin-webpackage-use-cases)) contains
a full list of use cases and resulting requirements.

### Signed HTTP exchanges

The [Signed HTTP exchanges
draft](https://wicg.github.io/webpackage/#go.draft-yasskin-http-origin-signed-responses.html)
([IETF
draft](https://tools.ietf.org/html/draft-yasskin-http-origin-signed-responses))
allows a publisher to sign their HTTP exchanges and intermediates to forward
those exchanges without breaking the signatures.

These signatures can be used in three related ways:

1. When signed by a certificate that meets roughly the TLS server certificate
   requirements, clients can trust that the exchange is authoritative for its
   request URL, if that URL's host matches one of the certificate's domains.
2. When signed by another kind of certificate, the signature proves some other
   kind of statement about the exchange. For example, the signature might act as
   a proof that the exchange appears in a transparency log or that some sort of
   static analysis has been run.
3. When signed by a raw public key, the signatures enable [signature-based
   subresource integrity](https://github.com/mikewest/signature-based-sri).

Signed exchanges can also be sent to a client in three ways:

1. Using a `Signature` header in a normal HTTP response. This way is used for
   non-origin signatures and to provide an origin-trusted signature to
   intermediates.
1. Enveloped into the `application/signed-exchange` content type. In this case,
   the signed exchange has both the publishing URL of its inner request, and the
   distributing URL of the outer envelope.
1. In an HTTP/2-Pushed exchange.

We publish [periodic snapshots of this
draft](https://tools.ietf.org/html/draft-yasskin-httpbis-origin-signed-exchanges-impl)
so that test implementations can interoperate.

### Bundled exchanges

We're defining a [new Zip-like archive
format](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html)
to serve the Web's needs. ([IETF
draft](https://tools.ietf.org/html/draft-yasskin-wpack-bundled-exchanges))

This format will map HTTP requests, not just filenames, to HTTP responses, not
just file contents. It will probably incorporate compression to shrink headers,
and even in compressed bundles will allow random access to resources without
uncompressing other resources, like Zip rather than Gzipped Tar. It will be able
to include [signed exchanges](#signed-http-exchanges) from multiple origins and
will allow signers to sign a group of contained exchanges as a unit, both to
optimize the number of public key operations and to prevent attackers from
mixing versions of resources.

Bundles will probably also include a way to depend on other bundles by reference
without directly including their content.

Like enveloped signed exchanges, bundles have a distributing URL in addition to
the publishing URLs of their contained exchanges.

<a id="loading"></a>
### Loading specification

The [Web Package Loading specification](https://wicg.github.io/webpackage/loading.html) specifies how browsers load signed exchanges, and will eventually also describe how to load bundled exchanges.

## Use cases

This section describes how to achieve several of the [use cases](https://wicg.github.io/webpackage/#go.draft-yasskin-webpackage-use-cases.html) using the above specifications.

### Offline installation

[Use case](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#offline-installation)

Peer-to-peer sharing is quite popular, especially in emerging Markets, due to
cost and limitations on cellular data and relatively spotty WiFi availability.
It is typically done over local Bluetooth/WiFi, by either built-in OS features
like [Android Beam](https://en.wikipedia.org/wiki/Android_Beam) or with popular
third-party apps, such as
[ShareIt](https://play.google.com/store/apps/details?id=com.lenovo.anyshare.gps)
or [Xender](https://play.google.com/store/apps/details?id=cn.xender)). People
currently share locally-stored media files and apps (APK files for Android for
example) this way. This collection of specifications extends that ability to
bundles of content and Progressive Web Apps.

To install a website while offline, the client needs all of the HTTP exchanges
that make up the website, signed to prove they're authentic, probably in a
bundle so the collection is easy to keep track of.

To make sure the site keeps working while offline, the publisher should include
a Service Worker, and identify it in the bundle's metadata. When the browser
[loads](#loading) the bundle, it'll register the identified Service Worker and
pass it the contents of the bundle to add to its cache.

The publisher doesn't need to include everything the bundle might possibly refer
to in the bundle itself. In a bundled version of Twitter, for example, the
application would appear in the bundle, but the client would need to go online
to retrieve tweets. A bundled video player, by contrast, might include the
player itself in the initial bundle but sign videos separately. People could
transfer the player from one peer and then share videos individually with other
peers.

### Save and share a web page

[Use case](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#snapshot)

Any client can snapshot the page or website they're currently reading by
building a bundle of exchanges without signing them. A client could do part of
this today by taking screenshots or building
[MHTML](https://tools.ietf.org/html/rfc2557) or [Web
Archive](https://en.wikipedia.org/wiki/Webarchive) files from the content, but
each of these options has significant downsides:

* Screenshots only represent part of a single page, and links can't be clicked.
* MHTML files can only represent a single top-level resource, incur base64
  overhead for binary resources, and aren't random access.
* Users MUST NOT open Web Archive files they didn't themselves create, on pain
  of UXSS:
  https://blog.rapid7.com/2013/04/25/abusing-safaris-webarchive-file-format/

### Privacy-preserving prefetch

[Use case](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#private-prefetch)

1. A publisher signs a page that they'd like a source website to be able to
   prefetch and serves it with an appropriate [`Signature`
   header](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#signature-header).
1. When the source website decides to link to the target page, it fetches the
   page with the [`Accept-Signature` request
   header](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#accept-signature)
   to get the response to include the `Signature` header, and serializes that
   exchange into the [`application/http-exchange+cbor`
   format](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#application-http-exchange).
   For example, `https://source.example.com/` might copy
   `https://publisher.example.com/page.html` to
   `https://cache.source.example.com/publisher.example.com/page.html`
1. On pages that link to a signed target, the source website includes a `<link
   rel="prefetch"
   href="https://cache.source.example.com/publisher.example.com/page.html">` tag
   or the equivalent `Link` HTTP header.
1. A client loading this source page would fetch
   https://cache.source.example.com/publisher.example.com/page.html, discover
   that it has a valid signature for https://publisher.example.com/page.html,
   and use it for subsequent uses of that URL.

To work in general, this requires some changes to the [`prefetch` link relation
type](https://w3c.github.io/resource-hints/#prefetch), which currently doesn't
guarantee that it won't make further cross-origin requests. However, certain
target resources (especially AMP and MIP resources) can be statically analyzed
to guarantee that they preserve privacy before the server decides to prefetch
them.

### Packaged Web Publications

[Use case](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#uc-web-pub)

A [packaged web publication](https://w3c.github.io/pwpub/) will probably use the
[Bundling](#bundled-exchanges) specification to represent a collection of web
resources with the bundle's manifest either being or including the [web
publication's manifest](https://www.w3.org/TR/wpub/#manifest).

Publications may be signed in a couple ways:

* Individual resources can be signed as their origin to prove to the browser
  that they're authentic. However, the limited expiration time of these
  signatures limits their utility for books that should remain usable for years
  to centuries.
* A whole publication might be signed by its author and/or its publisher, using
  a certificate type that's recognized by ebook readers rather than general web
  browsers.

### Third-party security review

[Use case](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#security-review)

If a third-party reviews a package for security in some way, which could be as
complex as the static analysis used to guard the Apple App Store, or as simple
as inclusion in a [transparency
log](https://www.certificate-transparency.org/general-transparency). The
third-party then signs either the exchanges in a package or the package as a
whole using a certificate whose metadata reflects whichever property was
reviewed for.

## Signed Exchange Loading Sketch

Signed exchanges fit into the loading stack between the prefetch cache and
Service Workers, leading to a stack with the following layers:

Network → HTTP/2 Push cache → HTTP Cache → prefetch cache → **signed exchange
handling** → Service Workers → preload cache → memory/image cache → actual
rendering

When an **embedder** prefetches or embeds an application/signed-exchange
resource, or a client navigates from the embedder to an
application/signed-exchange, the client goes through several steps to open the
outer envelope and load the inner exchange.

### Fetch the distributing URL

The client won't know that a URL holds a signed exchange until it receives the
`Content-Type` in the response, so the initial request is identical to any other
request in the same context. It follows redirects, is constrained by the
embedder's and any parent frame's Content Security Policy, and goes through the
distributing URL's Service Worker for navigations or the embedder's otherwise.

Once the response comes back, it's cached in the HTTP layer like any other
response, but the `Content-Type: application/signed-exchange` header tells the
client that it's the outer resource of a signed exchange, which causes the
signed-exchange handler to return either an annotated redirect or a network
error to higher layers.

### Fetch the certificate chain

The client parses the beginning of the `application/signed-exchange` resource to
extract the [`Signature`
header](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#signature-header).
Each signature in the `Signature` header has a `cert-url` field that identifies
a certificate chain to use to validate the signature, and each certificate chain
is fetched and validated.

The request for this certificate chain is made without credentials and skips
Service Workers, due to the layering of signed-exchange handling. The request
may be fulfilled from the HTTP cache, and its response is cached as a normal
HTTP response. We might also define an additional content-addressed cache using
the `cert-sha256` field, if we can show that this avoids the [privacy problems
of the SRI
cache](https://hillbrad.github.io/sri-addressable-caching/sri-addressable-caching.html).

The client checks the leaf certificate's hash against the `cert-sha256` field of
the `Signature` header and then validates the certificate chain. Certificate
validation is a call to an OS-controlled library and may make further URL
requests that aren't under the browser's control and therefore use a separate,
probably ephemeral, cookie jar, cache, and set of service workers.

Each signature with a valid certificate chain is passed on to the next step.

### Signature verification

Once the certificates are validated and enough of the outer resource is received
to parse the claimed inner headers, the client extracts the publishing URL from
those headers and then tries to find a valid signature over the headers that is
trusted for the publishing URL's origin. If none of the signatures are valid and
trusted, it either

1. redirects to the publishing URL as if the outer response were a 303 redirect,
   or
2. fails with a network error.

We're not yet certain which behavior is best. The first is slightly more
resilient to clock skew, while the second encourages intermediates to get their
implementations right.

If the client does find a valid signature, it "releases" the headers for further
processing and starts validating
[mi-sha256](https://tools.ietf.org/html/draft-thomson-http-mice-02#section-2)
records into a response stream as they arrive.

### Prefetching stops here

At this point, the client's behavior depends on whether the outer exchange was
requested as a [prefetch](https://w3c.github.io/resource-hints/#prefetch). To
satisfy the [privacy-preserving prefetch](#privacy-preserving-prefetch) use
case, prefetches have to be careful not to store the inner response in the HTTP
cache or anywhere else that would be visible to the publishing URL's server.

Prefetches can and should process any `Link: <>; rel=preload` headers they find,
as prefetches. If those point at signed exchanges, this process repeats.

### No nested signed exchanges

To limit the complexity of the implementation, we're currently planning to
disallow signed exchanges that contain either signed exchanges or redirects.
This may change if use cases come up or if the implementation turns out to be
simpler than expected.

### Navigations and subresources redirect

At this point the client redirects to the signed exchange's publishing URL, with
[the inner request and response stream associated with the outer
request](https://wicg.github.io/webpackage/loading.html#mp-request-stashed-exchange).
For navigations, this request goes through the publishing URL's Service Worker,
if any. For subresources, like other redirects, it doesn't go through the
Service Worker again.

If there's no Service Worker handling this `fetch` event or the Service Worker's
handler fetches the original request or its clone from the network, either by
returning without calling `e.respondWith()` or by calling `fetch(e.request)`,
this tries to return the response stream that was attached to the redirect.
However, if either of the following conditions is met, the fetch bypasses the
attached exchange and continues down to the lower caches and the network:

* The inner request doesn't
  [match](https://wicg.github.io/webpackage/loading.html#request-matching) the
  `Request` the Service Worker sent. This prevents a malicious intermediate from
  causing the client to use the wrong content-negotiated resource. If we later
  put inner responses in the HTTP cache (TBD), this also prevents the
  intermediate from putting the wrong resource there.
* There's a response in a lower cache with a newer Date header than the inner
  response's Date header. This prevents some downgrade attacks.
   * Content-negotiated responses may need to allow separate
     monotonically-increasing `Date` sequences for each
     [variant](https://tools.ietf.org/html/draft-nottingham-variants-02). We're
     not addressing this for V1 because it seems rare to expect a single client
     to see multiple different variants for the same URL.

The Service Worker can also explicitly return something else by calling
`e.respondWith(somethingElse)`.

For now, there's no explicit notification to the Service Worker that this
request is part of handling a signed exchange. Because the inner exchange is
attached, as described above, the Service Worker can check whether it's
immediately available by doing a fetch of the *same request* or its clone with
[`.cache`](https://fetch.spec.whatwg.org/#dom-request-cache) set to
`"only-if-cached"`, or it can turn on
[`navigationPreload`](https://developers.google.com/web/updates/2017/02/navigation-preload)
and check
[`e.preloadResponse`](https://w3c.github.io/ServiceWorker/#dom-fetchevent-preloadresponse).
It might make sense to attach some metadata to that response describing the
situation, but we don't currently propose that metadata. The Service Worker also
currently gets no description of any differences between the signed inner
request and the browser's request in `e.request`.

### Matching prefetches with subresources

If page `A` prefetches two signed exchanges `B.sxg` and `C.sxg` containing
publishing URLs `B` and `C`, respectively, and the user then navigates to
`B.sxg`, we'd like as many of `B`'s subresource fetches as possible to be
fulfilled from the prefetched content. However, the fact that we don't put the
inner exchange into the HTTP cache limits which ones we can wire up.

1. B can use `C.sxg` as a subresource and have that fulfilled by `A`'s prefetch
   of `C.sxg`.
1. B can include a `Link: <C.sxg>; rel=preload` header and then have a `C`
   subresource fulfilled by `A`'s prefetch. This works because the preload cache
   is populated from the Service Worker's response, so it sees the result of the
   redirect. TODO: Check that this doesn't violate the security requirements
   around exposing redirects.

However,

1. B *cannot* use `C` as a subresource and have that fulfilled by `A`'s prefetch of
   `C.sxg`.
1. B *cannot* include a `Link: <C>; rel=preload` header and then have a `C`
   subresource fulfilled by `A`'s prefetch.

This means that a page with subresources that wants its referring sites to be
able to prefetch it in a privacy-preserving way, has to use distributing URLs
for internal links, which means it must be re-signed for each distributing
cache.

We consider the centralizing properties of that restriction to be a bug. When
[bundles](#bundled-exchanges) are more fully specified, we expect sites to be
able to use publishing URLs in their internal links by including both `B` and
`C` in a single bundle, and then a single signed bundle can be used for multiple
distributing caches.

### To consider: Cache the inner exchange

If the signed exchange was requested as a navigation or subresource (i.e. *not*
prefetches), we may want to add the inner exchange to the HTTP cache, or to a
layer above the HTTP cache that's roughly equivalent.

We aren't specifying or implementing this yet in order to provide time for
security folks to decide whether it's safe. However, we're considering the
following:

This entry needs to have the same lifetime-extension semantics as the request
that retrieved the outer response. For example, if the outer response was
[prefetched](https://w3c.github.io/resource-hints/), the inner one's lifetime in
the cache needs to be [at least 5
minutes](https://github.com/w3c/ServiceWorker/issues/1302#issuecomment-382043843).
If the outer response was preloaded, the inner one needs to live in the [preload
cache](https://github.com/whatwg/fetch/issues/590) for the fetch group.

It's probably unsafe for the inner exchange to stay in the cache longer than the
outer signature is valid. This makes it more difficult for an attacker to get
persistent access to an XSS vulnerability by sending a signed response with a
long HTTP cache lifetime, since the Signature will expire more quickly. Some
concerns have been raised about allowing a Service Worker to add signed
responses to the long-lived SW Cache, but this should be safe since the SW
script itself will expire, and newer SW scripts can explicitly reject particular
vulnerable resources. (TODO: Figure out whether developers can be that careful
in practice.)

This increases the number of ways a resource can be "stale":

1. The HTTP caching information is fresh, but the `Signature` header's
   certificate expiration, OCSP response's `nextUpdate`, or signature expiration
   has passed.
1. The `Signature` header isn't expired, but the HTTP cache entry is stale.
1. Both.

For *later* loads of the publishing URL (in particular, not the load that's
happening through the signed exchange, since it's fulfilled using the
above-mentioned prefetch cache), a stale entry can be revalidated in the
following ways:

1. If only the `Signature` is expired, the client can fetch the `validity-url
   `to update just the signature. It *must not* send an `If-None-Match` or
   `If-Modified-Since` request to update the cache, because the `Signature`
   expiring means we don't trust the claimed ETag or date anymore.
1. If only the HTTP caching information is stale, the client can send the
   publishing URL an `If-None-Match` or `If-Modified-Since` request hoping for a
   304. Note that the client has to keep the original response headers if it
   intends to use the `validity-url` to update the signature in the future.
1. If both are stale, the client could first update the signature and then check
   for a 304 (or even do both concurrently), but it may be easier to just do an
   unconditional request.

## FAQ

### Why signing but not encryption? HTTPS provides both...

The use cases concern distributing public resources in new ways. Clients need to
know that the content is authentic, so that statements like "I trust
nytimes.com" have meaning, and signing provides that. However, since the
resources are public and need to be forwardable to any client who wants them,
their content isn't secret, and any encryption winds up being misleading at
best.

### What if a publisher accidentally signs a non-public resource?

There are three variants of this mistake:

1. The publisher signs private information that only you should see. For
   example, a bank might sign the page showing your balance. If we assume that
   the publisher already checks that it only serves you your own balance, then
   just adding a signature doesn't leak any of your information. You could then
   forward the information to someone else whether or not it has a signature.

   However, an attacker might take advantage of this by showing their victim a
   signed bank statement that was sent to the attacker. Since the URL would show
   the victim's actual bank, it might be easier to get them to believe the wrong
   balance as a step in a layer 8 attack.
1. The publisher signs a `Set-Cookie` response header, or other state-changing
   HTTP header. This could be used in a session fixation attack or to bypass
   certain kinds of CSRF protection. To prevent this, browsers must not trust
   signatures that cover these response headers.
1. The publisher might sign a web page with Javascript that modifies state in a
   way that depends on request metadata. A simple example might be a "you're
   logged in" page that sets `document.cookie` instead of using the `Set-Cookie`
   header. More sophisticated vulnerabilities are probably also possible. The
   client can't detect this, so instead we need guidelines to make it less
   likely that server will sign non-public resources.

### What about certificate revocation, especially while offline?

The certificate(s) used to sign a package must be accompanied by a
short-lived OCSP response, guaranteeing that it hasn't been revoked for more
than about 7 days. This is the same time-limit that browsers use.

The OCSP response can be updated without needing to update the whole package,
and this update can be stored as a file and forwarded peer-to-peer.

It may also be safe to extend trust in OCSP responses for some number of days
past their expiration, as long as the client is continuously offline. This would
be useful to bridge the offline resource to the beginning of the next month,
when the client's data plan renews. This question is tracked in
[WICG/webpackage#117](https://github.com/WICG/webpackage/issues/117).

### What if a publisher signed a package with a JS library, and later discovered a vulnerability in it. On the Web, they would just replace the JS file with an updated one. What is the story in case of packages?

The `expires` timestamp in the `Signature` header limits the lifetime of the
resource in the same way as the OCSP response limits the lifetime of a
certificate. When the author needs to replace a vulnerable resource, the
signature's
[`validity-url`](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#updating-validity)
would omit an updated signature, and the client would need to re-download the
resource from its original location.

While a client is continuously offline, like with OCSP checks, it might choose
to continue using the vulnerable resource somewhat past its expiration, to
bridge to the start of a new data plan cycle.
