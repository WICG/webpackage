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

### Bundled exchanges

We're defining a new Zip-like archive format to serve the Web's needs. It
currently only exists in a [Pull
Request](https://github.com/WICG/webpackage/pull/98).

This format will map HTTP requests, not just filenames, to HTTP responses, not
just file contents. It will probably incorporate compression to shrink headers,
and even in compressed bundles will allow random access to resources without
uncompressing other resources, like Zip rather than Gzipped Tar. It will be able
to include [signed exchanges](#signed-http-exchanges) from multiple origins and
will probably incorporate an optimization to reduce the number of public key
operations for bundles containing many such signatures.

Bundles will probably also include a way to depend on other bundles by reference
without directly including their content.

<a id="loading"></a>
### Loading specification

We'll need to specify how browsers load both signed exchanges and bundles of exchanges.

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
certificate. When the author needs to replace a vulnerable resource, the signature's
[`validityUrl`](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#updating-validity) would omit an updated signature, and the client would need to re-download the resource from its original location.

While a client is continuously offline, like with OCSP checks, it might choose
to continue using the vulnerable resource somewhat past its expiration, to
bridge to the start of a new data plan cycle.
