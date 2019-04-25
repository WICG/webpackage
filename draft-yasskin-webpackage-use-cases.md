---
coding: utf-8

title: Use Cases and Requirements for Web Packages
docname: draft-yasskin-webpackage-use-cases-latest
category: info

ipr: trust200902

stand_alone: yes
pi: [comments, sortrefs, strict, symrefs, toc]

author:
 -
    name: Jeffrey Yasskin
    organization: Google
    email: jyasskin@chromium.org

informative:
  ISO28500:
    date: 2017
    target: "https://www.iso.org/standard/68004.html"
    title: "WARC file format"
    seriesinfo:
      ISO: "28500:2017"
  JAR:
    date: 2014
    target: "https://docs.oracle.com/javase/7/docs/technotes/guides/jar/jar.html"
    title: "JAR File Specification"
  MHTML: RFC2557
  ServiceWorkers: W3C.WD-service-workers-1-20161011
  ZIP:
    date: 2014-10-01
    target: "https://pkware.cachefly.net/webdocs/casestudies/APPNOTE.TXT"
    title: "APPNOTE.TXT - .ZIP File Format Specification"

--- abstract

This document lists use cases for signing and/or bundling collections
of web pages, and extracts a set of requirements from them.

--- note_Note_to_Readers

Discussion of this draft takes place on the ART area mailing list
(art@ietf.org), which is archived
at <https://mailarchive.ietf.org/arch/search/?email_list=art>.

The source code and issues list for this draft can be found
in <https://github.com/WICG/webpackage>.

--- middle

# Introduction

People would like to use content offline and in other situations where
there isn’t a direct connection to the server where the content
originates. However, it's difficult to distribute and verify the
authenticity of applications and content without a connection to the
network. The W3C has addressed running applications offline with
Service Workers ({{ServiceWorkers}}), but not the problem of
distribution.

Previous attempts at packaging web resources
(e.g.
[Resource Packages](https://www.mnot.net/blog/2010/02/18/resource_packages) and
the
[W3C TAG's packaging proposal](https://w3ctag.github.io/packaging-on-the-web/))
were motivated by speeding up the download of resources from a single server,
which is probably better achieved through other mechanisms like HTTP/2 PUSH,
possibly augmented with
a
[simple manifest of URLs a page plans to use](https://lists.w3.org/Archives/Public/public-web-perf/2015Jan/0038.html).
This attempt is instead motivated by avoiding a connection to the origin server
at all. It may still be useful for the earlier use cases, so they're still
listed, but they're not primary.

# Use cases

These use cases are in rough descending priority order. If use cases
have conflicting requirements, the design should enable more important
use cases.

## Essential {#essential-use-cases}

### Offline installation {#offline-installation}

Alex can download a file containing a website
(a [PWA](https://developers.google.com/web/progressive-web-apps/checklist))
including a Service Worker from origin `O`, and transmit it to their peer Bailey,
and then Bailey can install the Service Worker with a proof that it came from `O`.
This saves Bailey the bandwidth costs of transferring the website.

There are roughly two ways to accomplish this:

1. Package just the Service Worker Javascript and any other Javascript that it
   [importScripts()](https://w3c.github.io/ServiceWorker/#importscripts), with
   their URLs and enough metadata to synthesize a
   [navigator.serviceWorker.register(scriptURL, options)
   call](https://w3c.github.io/ServiceWorker/#navigator-service-worker-register),
   along with an uninterpreted but signature-checked blob of data that the
   Service Worker can interpret to fill in its caches.
1. Package the resources so that the Service Worker can fetch() them to populate
   its cache.

Associated requirements for just the Service Worker:

* {{urls}}{:format="title"}: The `register()` and `importScripts()` calls have
  semantics that depend on the URL.
* {{signing}}{:format="title"}: To prove that the file came from `O`.
* {{existing-certs}}{:format="title"}: So `O` doesn't have to spend lots of
  money buying a specialized certificate.
* {{crypto-agility}}{:format="title"}: Today's algorithms will eventually be
  obsolete and will need to be replaced.
* {{revocation}}{:format="title"}: `O`'s certificate might be compromised or
  mis-issued, and the attacker shouldn't then get an infinite ability to mint
  packages.
* {{no-downgrades}}{:format="title"}: `O`'s site might have an XSS
  vulnerability, and attackers with an old signed package shouldn't be able to
  take advantage of the XSS forever.
* {{metadata}}{:format="title"}: Just enough to generate the `register()` call,
  which is less than a full W3C Application Manifest.

Additional associated requirements for packaged resources:

* {{urls}}{:format="title"}: Resources on the web are addressed by URL.
* {{request-headers}}{:format="title"}: If Bailey's running a different browser
  from Alex or has a different language configured, the `accept*` headers are
  important for selecting which resource to use at each URL.
* {{response-headers}}{:format="title"}: The meaning of a resource is heavily
  influenced by its HTTP response headers.
* {{multiple-origins}}{:format="title"}: So the site can
  be [built from multiple components](#libraries).
* {{metadata}}{:format="title"}: The browser needs to know which resource within
  a package file to treat as its Service Worker and/or initial HTML page.

#### Online use

Bailey may have an internet connection through which they can, in real time,
fetch updates to the package they received from Alex.

#### Fully offline use

Or Bailey may not have any internet connection a significant fraction of the
time, either because they have no internet at all, because they turn off
internet except when intentionally downloading content, or because they use up
their plan partway through each month.

Associated requirements beyond {{offline-installation}}{:format="title"}:

* {{packaged-validity}}{:format="title"}: Even without a direct internet
  connection, Bailey should be able to check that their package is still valid.

### Offline browsing {#offline-browsing}

Alex can download a file containing a large website (e.g. Wikipedia) from its
origin, save it to transferrable storage (e.g. an SD card), and hand it to their
peer Bailey. Then Bailey can browse the website with a proof that it came from
`O`. Bailey may not have the storage space to copy the website before browsing
it.

This use case is harder for publishers to support if we specialize
{{offline-installation}} for Service Workers since it requires the publisher to
adopt Service Workers before they can sign their site.

Associated requirements beyond {{offline-installation}}{:format="title"}:

* {{random-access}}{:format="title"}: To avoid needing a long linear scan before
  using the content.
* {{stored-compression}}{:format="title"}: So that more content can fit on the
  same storage device.

### Save and share a web page {#snapshot}

Casey is viewing a web page and wants to save it either for offline use or to
show it to their friend Dakota. Since Casey isn't the web page's publisher, they
don't have the private key needed to sign the page. Browsers currently allow
their users to save pages, but each browser uses a different format (MHTML, Web
Archive, or files in a directory), so Dakota and Casey would need to be using
the same browser. Casey could also take a screenshot, at the cost of losing
links and accessibility.

Associated requirements:

* {{unsigned-content}}{:format="title"}: A client can't sign content as another
  origin.
* {{multiple-origins}}{:format="title"}: General web pages include resources
  from multiple origins.
* {{urls}}{:format="title"}: Resources on the web are addressed by URL.
* {{response-headers}}{:format="title"}: The meaning of a resource is heavily
  influenced by its HTTP response headers.

### Privacy-preserving prefetch {#private-prefetch}

Lots of websites link to other websites. Many of these source sites would like
the targets of these links to load quickly. The source could use `<link
rel="prefetch">` to prefetch the target of a link, but if the user doesn't
actually click that link, that leaks the fact that the user saw a page that
linked to the target. This can be true even if the prefetch is made without
browser credentials because of mechanisms like TLS session IDs.

Because clients have limited data budgets to prefetch link targets, this use
case is probably limited to sites that can accurately predict which link their
users are most likely to click. For example, search engines can predict that
their users will click one of the first couple results, and news aggreggation
sites like Reddit or Slashdot can hope that users will read the article if
they've navigated to its discussion.

Two search engines have built systems to do this with today's technology:
Google's [AMP](https://www.ampproject.org/) and Baidu's
[MIP](https://www.mipengine.org/) formats and caches allow them to prefetch
search results while preserving privacy, at the cost of showing the wrong URLs
for the results once the user has clicked. A good solution to this problem would
show the right URLs but still avoid a request to the publishing origin until
after the user clicks.

Associated requirements:

* {{signing}}{:format="title"}: To prove the content came from the original
  origin.
* {{streamed-loading}}{:format="title"}: If the user clicks before the target
  page is fully transferred, the browser should be able to start loading early
  parts before the source site finishes sending the whole page.
* {{transfer-compression}}{:format="title"}
* {{subsetting}}{:format="title"}: If a prefetched page includes subresources,
  its publisher might want to provide and sign both WebP and PNG versions of an
  image, but the source site should be able to transfer only best one for each
  client.

## Nice-to-have {#nice-to-have-use-cases}

### Packaged Web Publications {#uc-web-pub}

The
[W3C's Publishing Working Group](https://www.w3.org/publishing/groups/publ-wg/),
merged from the International Digital Publishing Forum (IDPF) and in charge of
EPUB maintenance, wants to be able to create publications on the web and then
let them be copied to different servers or to other users via arbitrary
protocols. See
their [Packaged Web Publications use cases](https://www.w3.org/TR/pwp-ucr/#pwp)
for more details.

Associated requirements:

* {{urls}}{:format="title"}: Resources on the web are addressed by URL.
* {{signing}}{:format="title"}: So that readers can be sure their copy is
  authentic and so that copying the package preserves the URLs of the content
  inside it.
* {{no-downgrades}}{:format="title"}: An early version of a publication might
  contain incorrect content, and a publisher should be able to update that
  without worrying that an attacker can still show the old content to users.
* {{metadata}}{:format="title"}: A publication can have copyright and licensing
  concerns; a title, author, and cover image; an ISBN or DOI name; etc.; which
  should be included when that publication is packaged.

Other requirements are similar to those from
{{offline-installation}}{:format="title"}:

* {{random-access}}{:format="title"}: To avoid needing a long linear scan before
  using the content.
* {{stored-compression}}{:format="title"}: So that more content can fit on the
  same storage device.
* {{request-headers}}{:format="title"}: If different users' browsers have
  different capabilities or preferences, the `accept*` headers are important for
  selecting which resource to use at each URL.
* {{response-headers}}{:format="title"}: The meaning of a resource is heavily
  influenced by its HTTP response headers.
* {{existing-certs}}{:format="title"}: So a publisher doesn't have to spend lots
  of money buying a specialized certificate.
* {{crypto-agility}}{:format="title"}: Today's algorithms will eventually be
  obsolete and will need to be replaced.
* {{revocation}}{:format="title"}: The publisher's certificate might be
  compromised or mis-issued, and an attacker shouldn't then get an infinite
  ability to mint packages.

### Avoiding Censorship {#anti-censorship}

Some users want to retrieve resources that their governments or network
providers don't want them to see. Right now, it's straightforward for someone in
a privileged network position to block access to particular hosts, but TLS makes
it difficult to block access to particular resources on those hosts.

Today it's straightforward to retrieve blocked content from a third party, but
there's no guarantee that the third-party has sent the user an accurate
representation of the content: the user has to trust the third party.

With signed web packages, the user can re-gain assurance that the content is
authentic, while still bypassing the censorship. Packages don't do anything to
help discover this content.

Systems that make censorship more difficult can also make legitimate content
filtering more difficult. Because the client that processes a web package always
knows the true URL, this forces content filtering to happen on the client
instead of on the network.

Associated requirements:

* {{urls}}{:format="title"}: So the user can see that they're getting the
  content they expected.
* {{signing}}{:format="title"}: So that readers can be sure their copy is
  authentic and so that copying the package preserves the URLs of the content
  inside it.

### Third-party security review {#security-review}

Some users may want to grant certain permissions only to applications that have
been reviewed for security by a trusted third party. These third parties could
provide guarantees similar to those provided by the iOS, Android, or Chrome OS
app stores, which might allow browsers to offer more powerful capabilities than
have been deemed safe for unaudited websites.

Binary transparency for websites is similar: like with Certificate Transparency
{{?RFC6962}}, the transparency logs would sign the content of the package to
provide assurance that experts had a chance to audit the exact package a client
received.

Associated requirements:

* {{additional-signatures}}{:format="title"}

### Building packages from multiple libraries {#libraries}

Large programs are built from smaller components. In the case of the web,
components can be included either as Javascript files or as `<iframe>`d
subresources. In the first case, the packager could copy the JS files to their
own origin; but in the second, it may be important for the `<iframe>`d resources
to be able to make
[same-origin](https://html.spec.whatwg.org/multipage/origin.html#same-origin)
requests back to their own origin, for example to implement federated sign-in.

Associated requirements:

* {{multiple-origins}}{:format="title"}: Each component may come from its own origin.
* {{deduplication}}{:format="title"}: If we have dependencies A->B->D and
  A->C->D, it's important that a request for a D resource resolves to a single
  resource that both B and C can handle.

#### Shared libraries

In ecosystems like [Electron](https://electron.atom.io/)
and [Node](https://nodejs.org/en/), many packages may share some common
dependencies. The cost of downloading each package can be greatly reduced if the
package can merely point at other dependencies to download instead of including
them all inline.

Associated requirements:

* {{external-dependencies}}{:format="title"}

### Cross-CDN Serving {#cross-cdn-serving}

When a web page has subresources from a different origin, retrieval of those
subresources can be optimized if they're transferred over the same connection as
the main resource. If both origins are distributed by the same CDN, in-progress
mechanisms like {{?I-D.ietf-httpbis-http2-secondary-certs}} allow the server to
use a single connection to send both resources, but if the resource and
subresource don't share a CDN or don't use a CDN at all, existing mechanisms
don't help.

If the subresource is signed by its publisher, the main resource's server can
forward it to the client.

There are some yet-to-be-solved privacy problems if the client and server want
to avoid transferring subresources that are already in the client's cache:
naively telling the server that a resource is already present is a privacy leak.

Associated requirements:

* {{streamed-loading}}{:format="title"}: To get optimal performance, the browser
  should be able to start loading early parts of a resource before the
  distributor finishes sending the whole resource.
* {{signing}}{:format="title"}: To prove the content came from the original
  origin.
* {{transfer-compression}}{:format="title"}

### Least Authority CDN Serving {#least-authority-cdn-serving}

Currently, a CDN trusted via TLS to serve content for an FQDN may also serve
arbitrary content for that FQDN. Given the distributed deployment of points of
presence (POPs) -- often spanning many datacenters and jurisdictions -- this
creates a security risk for the manipulation of content delivered from those
POPs. Yet, in many CDN deployments, POPs merely distribute content generated by
the origin without alteration. For these deployments, it would be useful to
remove the authority of the CDN to serve arbitrary content.

SXG offers a means to reducing CDN authority via HTTP clients configured to
expect SXG for all responses, regardless of other trust. Applicable clients
might include package managers, native mobile applications, desktop
applications, and server applications.

SXG would also offer a foundation for dynamically imposing such client
behavior using a mechanism similar to HSTS. This would allow supporting clients
to be hardened against POP compromise without any effect on non-supporting
clients.

A related IETF proposal is Delegated Credentials for TLS
(draft-ietf-tls-subcerts-03), but that proposal solves a different use case
where a CDN or server needs the authority to serve abitrary or altered content.
Given that need, the proposal merely limits the intervals and ciphers used by
the deputized CDN/servers. In contrast, SXG allows content distribution without
fully deputizing the servers performing distribution (at the cost of not
supporting the same extent of use cases).

SXG can provide stronger integrity guarantees -- compared to today or a future
with Delegated Credential for TLS -- for popular content distribution patterns.

Associated requirements:

* {{streamed-loading}}{:format="title"}: To get optimal performance, the browser
  should be able to start loading early parts of a resource before the
  distributor finishes sending the whole resource.
* {{signing}}{:format="title"}: To prove the content came from the original
  origin.

### Installation from a self-extracting executable {#self-extracting}

The Node and Electron communities would like to install packages using
self-extracting executables. The traditional way to design a
self-extracting executable is to concatenate the package to the end of
the executable, have the executable look for a length at its own end,
and seek backwards from there for the start of the package.

Associated requirements:

* {{trailing-length}}{:format="title"}

### Packages in version control {#version-control}

Once packages are generated, they should be stored in version control. Many
popular VC systems auto-detect text files in order to "fix" their line endings.
If the first bytes of a package look like text, while later bytes store binary
data, VC may break the package.

Associated requirements:

* {{binary}}{:format="title"}

### Subresource bundling {#bundling}

Text based subresources often benefit from improved compression ratios when
bundled together.

At the same time, the current practice of JS and CSS bundling, by compiling
everything into a single JS file, also has negative side-effects:

1. Dependent execution - in order to start executing *any* of the bundled
   resources, it is required to download, parse and execute *all* of them.
1. Loss of caching granularity - Modification of *any* of the resources results
   in caching invalidation of *all* of them.
1. Loss of module semantics - ES6 modules must be delivered as independent
   resources. Therefore, current bundling methods, which deliver them with other
   resources under a common URL, require transpilation to ES5 and result in loss
   of ES6 module semantics.

An on-the-fly readable packaging format, that will enable resources to maintain
their own URLs while being physically delivered with other resources, can
resolve the above downsides while keeping the upsides of improved compression
ratios.

To improve cache granularity, the client needs to tell the server which versions
of which resources are already cached, which it could do with a Service Worker
or perhaps with {{?I-D.ietf-httpbis-cache-digest}}.

Associated requirements:

* {{urls}}{:format="title"}
* {{streamed-loading}}{:format="title"}: To solve downside 1.
* {{transfer-compression}}{:format="title"}: To keep the upside.
* {{response-headers}}{:format="title"}: At least the Content-Type is needed to
  load JS and CSS.
* {{unsigned-content}}{:format="title"}: Signing same-origin content wastes
  space.

### Archival {#archival}

Existing formats like WARC ({{ISO28500}}) do a good job of accurately
representing the state of a web server at a particular time, but a browser can't
currently use them to give a person the experience of that website at the time
it was archived. It's not obvious to the author of this draft that a new
packaging format is likely to improve on WARC, compared to, for example,
implementing support for WARC in browsers, but folks who know about archiving
seem interested, e.g.:
<https://twitter.com/anjacks0n/status/950861384266416134>.

Because of the time scales involved in archival, any signatures from the
original host would likely not be trusted anymore by the time the archive is
viewed, so implementations would need to sandbox the content instead of running
it on the original origin.

Associated requirements:

* {{urls}}{:format="title"}
* {{response-headers}}{:format="title"}: To accurately record the server's
  response.
* {{unsigned-content}}{:format="title"}: To deal with expired signatures.
* {{timeshifting}}{:format="title"}

# Requirements {#requirements}

## Essential {#essential-reqs}

### Indexed by URL {#urls}

Resources should be keyed by URLs, matching how browsers look
resources up over HTTP.

### Request headers {#request-headers}

Resource keys should include request headers like `accept` and
`accept-language`, which allows content-negotiated resources to be
represented.

This would require an extension to {{MHTML}}, which uses the `content-location`
response header to encode the requested URL, but has no way to encode other
request headers. MHTML also has no instructions for handling multiple resources
with the same `content-location`.

This also requires an extension to {{ZIP}}: we'd need to encode the request
headers into ZIP's filename fields.

### Response headers {#response-headers}

Resources should include their HTTP response headers, like
`content-type`, `content-encoding`, `expires`,
`content-security-policy`, etc.

This requires an extension to {{ZIP}}: we'd need something like {{JAR}}'s
`META-INF` directory to hold extra metadata beyond the resource's body.

### Signing as an origin {#signing}

Resources within a package are provably from an entity with the ability to serve
HTTPS requests for those resources' origin {{?RFC6454}}.

Resources within a package are provably from an entity with the ability to serve
HTTPS requests for those resources' origin {{?RFC6454}}.

Note that previous attempts to sign HTTP messages
({{?I-D.thomson-http-content-signature}}, {{?I-D.burke-content-signature}}, and
{{?I-D.cavage-http-signatures}}) omit a description of how a client should use a
signature to prove that a resource comes from a particular origin, and they're
probably not usable for that purpose.

This would require an extension to the {{ZIP}} format, similar to {{JAR}}'s
signatures.

In any cryptographic system, the specification is responsible to make correct
implementations easier to deploy than incorrect implementations
({{easy-implementation}}).

### Random access {#random-access}

When a package is stored on disk, the browser can access
arbitrary resources without a linear scan.

{{MHTML}} would need to be extended with an index of the byte offsets of each
contained resource.

### Resources from multiple origins in a package {#multiple-origins}

A package from origin `A` can contain resources from origin `B`
authenticated at the same level as those from `A`.

### Cryptographic agility {#crypto-agility}

Obsolete cryptographic algorithms can be replaced.

Planning to upgrade the cryptography also means we should include some way to
know when it's safe to remove old cryptography ({{crypto-removal}}).

### Unsigned content {#unsigned-content}

Alex can create their own package without a CA-signed
certificate, and Bailey can view the content of the package.

### Certificate revocation {#revocation}

When a package is signed by a revoked certificate, online browsers
can detect this reasonably quickly.

### Downgrade prevention {#no-downgrades}

Attackers can't cause a browser to trust an older, vulnerable
version of a package after the browser has seen a newer version.

### Metadata {#metadata}

Metadata like that found in the W3C's Application Manifest
{{?W3C.WD-appmanifest-20170828}} can help a client know how to load and display
a package.

### Implementations are hard to get wrong {#easy-implementation}

The design should incorporate aspects that tend to cause incorrect
implementations to get noticed quickly, and avoid aspects that are easy to
implement incorrectly. For example:

* Explicitly specifying a cryptographic algorithm identifier in {{?RFC7515}}
  made it easy for implementations to trust that algorithm,
  which
  [caused vulnerabilities](https://paragonie.com/blog/2017/03/jwt-json-web-tokens-is-bad-standard-that-everyone-should-avoid).
* {{ZIP}}'s duplicate specification of filenames makes it easy for
  implementations to
  [check the signature of one copy but use the other](https://nakedsecurity.sophos.com/2013/07/10/anatomy-of-a-security-hole-googles-android-master-key-debacle-explained/).
* Following
  [Langley's Law](https://blog.gerv.net/2016/09/introducing-deliberate-protocol-errors-langleys-law/) when
  possible makes it hard to deploy incorrect implementations.


## Nice to have {#nice-to-have-reqs}

### Streamed loading {#streamed-loading}

The browser can load a package as it downloads.

This conflicts with ZIP, since ZIP's index is at the end.

### Additional signatures {#additional-signatures}

Third-parties can vouch for packages by signing them.

### Binary {#binary}

The format is identified as binary by tools that might try to "fix"
line endings.

This conflicts with using an {{MHTML}}-based format.

### Deduplication of diamond dependencies {#deduplication}

Nested packages that have multiple dependency routes to the same
sub-package, can be transmitted and stored with only one copy of that
sub-package.

### Old crypto can be removed {#crypto-removal}

The ecosystem can identify when an obsolete cryptographic algorithm is
no longer needed and can be removed.

### Compress transfers {#transfer-compression}

Transferring a package over the network takes as few bytes as possible. This is
an easier problem than {{stored-compression}}{:format="title"} since it doesn't
have to preserve {{random-access}}{:format="title"}.

### Compress stored packages {#stored-compression}

Storing a package on disk takes as few bytes as possible.

### Subsetting and reordering {#subsetting}

Resources can be removed from and reordered within a package, without
breaking [signatures](#signing).

### Packaged validity information {#packaged-validity}

{{revocation}}{:format="title"} and {{no-downgrades}}{:format="title"}
information can itself be packaged or included in other packages.

### Signing uses existing TLS certificates {#existing-certs}

A "normal" TLS certificate can be used for signing packages. Avoiding
extra requirements like "code signing" certificates makes packaging
more accessible to all sites.

### External dependencies {#external-dependencies}

Sub-packages can be "external" to the main package, meaning the browser
will need to either fetch them separately or already have them.
([#35, App Installer Story](https://github.com/WICG/webpackage/issues/35))

### Trailing length {#trailing-length}

The package's length in bytes appears a fixed offset from the end of
the package.

This conflicts with {{MHTML}}.

### Time-shifting execution {#timeshifting}

In some unsigned packages, Javascript time-telling functions should return the
timestamp of the package, rather than the true current time.

We should explore if this has security implications.

# Non-goals

Some features often come along with packaging and signing, and it's
important to explicitly note that they don't appear in the list of
{{requirements}}{:format="title"}.

## Store confidential data {#confidential}

Packages are designed to hold public information and to be shared to
people with whom the original publisher never has an interactive
connection. In that situation, there's no way to keep the contents
confidential: even if they were encrypted, to make the data public,
anyone would have to be able to get the decryption key.

It's possible to maintain something similar to confidentiality for
non-public packaged data, but doing so complicates the format design
and can give users a false sense of security.

We believe we'll cause fewer privacy breaches if we omit any mechanism
for encrypting data, than if we include something and try to teach
people when it's unsafe to use.

## Generate packages on the fly {#streamed-generation}

See discussion at [WICG/webpackage#6](https://github.com/WICG/webpackage/issues/6#issuecomment-275746125).

## Non-origin identity {#non-origin-identity}

A package can be primarily identified as coming from something other
than a
[Web Origin](https://html.spec.whatwg.org/multipage/browsers.html#concept-origin).

## DRM {#drm}

Special support for blocking access to downloaded content based on
licensing. Note that DRM systems can be shipped inside the package
even if the packaging format doesn't specifically support them.

## Ergonomic replacement for HTTP/2 PUSH {#push-replacement}

HTTP/2 PUSH ({{?RFC7540}}, section 8.2) is hard for developers to configure, and
an explicit package format might be easier. However, experts in this area
believe we should focus on improving PUSH directly instead of routing around it
with a bundling format.

Trying to bundle resources in order to speed up page loads has a long history,
including
[Resource Packages](https://www.mnot.net/blog/2010/02/18/resource_packages) from
2010 and
the
[W3C TAG's packaging proposal](https://w3ctag.github.io/packaging-on-the-web/)
from 2015.

However, the HTTPWG is doing a lot of work to let servers optimize the PUSHed
data, and packaging would either have to re-do that or accept lower performance.
For example:

* {{?I-D.vkrasnov-h2-compression-dictionaries}} should allow individual small
  resources to be compressed as well as they would be in a bundle.
* {{?I-D.ietf-httpbis-cache-digest}} tells the server which resources it doesn't
  need to PUSH.

Associated requirements:

* {{streamed-loading}}{:format="title"}: If the whole package has to be
  downloaded before the browser can load a piece, this will definitely be slower
  than PUSH.
* {{transfer-compression}}{:format="title"}: Keep up with
  {{?I-D.vkrasnov-h2-compression-dictionaries}}.
* {{urls}}{:format="title"}: Resources on the web are addressed by URL.
* {{request-headers}}{:format="title"}:
  [PUSH_PROMISE](http://httpwg.org/specs/rfc7540.html#PUSH_PROMISE)
  ({{?RFC7540}}, section 6.6) includes request headers.
* {{response-headers}}{:format="title"}: PUSHed resources include their response
  headers.

# Security Considerations

The security considerations will depend on the solution designed to satisfy the
above requirements. See {{?I-D.yasskin-dispatch-web-packaging}} for one possible
set of security considerations.

# IANA Considerations

This document has no actions for IANA.

--- back

# Acknowledgements

Thanks to Yoav Weiss for the Subresource bundling use case and discussions about
content distributors.
