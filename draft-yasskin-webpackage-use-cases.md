---
coding: utf-8

title: Use Cases and Requirements for Web Packages
docname: draft-yasskin-webpackage-use-cases-latest
category: info

ipr: trust200902
area: gen
workgroup: dispatch
keyword: Internet-Draft

stand_alone: yes
pi: [comments, sortrefs, strict, symrefs, toc]

author:
 -
    name: Jeffrey Yasskin
    organization: Google
    email: jyasskin@chromium.org

informative:
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

Associated requirements:

* {{urls}}{:format="title"}: Resources on the web are addressed by URL.
* {{request-headers}}{:format="title"}: If Bailey's running a different browser
  from Alex or has a different language configured, the `accept*` headers are
  important for selecting which resource to use at each URL.
* {{response-headers}}{:format="title"}: The meaning of a resource is heavily
  influenced by its HTTP response headers.
* {{signing}}{:format="title"}: To prove that the file came from `O`.
* {{existing-certs}}{:format="title"}: So `O` doesn't have to spend lots of
  money buying a specialized certificate.
* {{multiple-origins}}{:format="title"}: So the site can
  be [built from multiple components](#libraries).
* {{crypto-agility}}{:format="title"}: Today's algorithms will eventually be
  obsolete and will need to be replaced.
* {{revocation}}{:format="title"}: `O`'s certificate might be compromised or
  mis-issued, and the attacker shouldn't then get an infinite ability to mint
  packages.
* {{no-downgrades}}{:format="title"}: `O`'s site might have an XSS
  vulnerability, and attackers with an old signed package shouldn't be able to
  take advantage of the XSS forever.
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

Associated requirements beyond {{offline-installation}}{:format="title"}:

* {{random-access}}{:format="title"}: To avoid needing a long linear scan before
  using the content.
* {{stored-compression}}{:format="title"}: So that more content can fit on the
  same storage device.

### Save and share a web page {#snapshot}

Casey is viewing a web page and wants to save it either for offline use or to
show it to their friend Dakota. Since Casey isn't the web page's author, they
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

### Third-party security review {#security-review}

Some users may want to grant certain permissions only to applications that have
been reviewed for security by a trusted third party. These third parties could
provide guarantees similar to those provided by the iOS, Android, or ChromeOS
app stores, which might allow browsers to offer more powerful capabilities than
have been deemed safe for unaudited websites.

Binary transparency for websites is similar: like with Certificate Transparency
{{?RFC6962}}, the transparency logs would sign the content of the package to
provide assurance that experts had a chance to audit the exact package a client
received.

Associated requirements:

* {{cross-signatures}}{:format="title"}

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

### Content Distributors {#content-distributors}

Content distributors want to re-publish other origins' content so readers can
access it more quickly or more privately. Currently, to attribute that content
to the original origin, they need to be full CDNs with the ability to publish
arbitrary content under that origin's name. There should be a way to let them
attribute only the exact content that the original origin published.

Web Packages would allow distributors to publish another site's signed content
as long as the user visited a URL explicitly mentioning the distributor.

Content distributors want to serve only the bytes that most optimally represent
the content the current user needs, even though the origin needs to provide
representations for all users. Think PNG vs WebP and small vs large resolutions.

This use case is also addressed by
{{?I-D.yasskin-http-origin-signed-responses}}.

Associated requirements:

* {{streamed-loading}}{:format="title"}: To get optimal performance, the browser
  should be able to start loading early resources before the distributor
  finishes sending the whole package.
* {{signing}}{:format="title"}: To prove the content came from the original
  origin.
* {{subsetting}}{:format="title"}: If a package includes both WebP and PNG
  versions of an image, the distributor should be able to select the best one to
  send to each client.
* {{transfer-compression}}{:format="title"}

### Installation from a self-extracting executable {#self-extracting}

The Node and Electron communities would like to install packages using
self-extracting executables. The traditional way to design a
self-extracting executable is to concatenate the package to the end of
the executable, have the executable look for a length at its own end,
and seek backwards from there for the start of the package.

Associated requirements:

* {{trailing-length}}{:format="title"}

### Ergonomic replacement for HTTP/2 PUSH {#push-replacement}

HTTP/2 PUSH ({{?RFC7540}}, section 8.2) is hard for developers to configure, and
an explicit package format might be easier.

Trying to bundle resources in order to speed up page loads has a long history,
including
[Resource Packages](https://www.mnot.net/blog/2010/02/18/resource_packages) from
2010 and
the
[W3C TAG's packaging proposal](https://w3ctag.github.io/packaging-on-the-web/)
from 2015.

However, the HTTPWG is doing a lot of work to let servers optimize the PUSHed
data, and packaging would either have to re-do that or accept lower performance.
Accepting lower performance might be worthwhile if it allows more developers to
adopt the smaller optimization.

Associated requirements:

* {{streamed-loading}}{:format="title"}: If the whole package has to be
  downloaded before the browser can load a piece, this will definitely be slower
  than PUSH.
* {{transfer-compression}}{:format="title"}: I believe PUSHed resources cannot
  keep a single compression state across resource boundaries, so this might be
  an advantage for packaging.
* {{urls}}{:format="title"}: Resources on the web are addressed by URL.
* {{request-headers}}{:format="title"}:
  [PUSH_PROMISE](http://httpwg.org/specs/rfc7540.html#PUSH_PROMISE)
  ({{?RFC7540}}, section 6.6) includes request headers.
* {{response-headers}}{:format="title"}: PUSHed resources include their response
  headers.

### Packages in version control {#version-control}

Once packages are generated, they should be stored in version control. Many
popular VC systems auto-detect text files in order to "fix" their line endings.
If the first bytes of a package look like text, while later bytes store binary
data, VC may break the package.

Associated requirements:

* {{binary}}{:format="title"}

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

### Cross-signatures {#cross-signatures}

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

# Non-goals

Some features often come along with packaging and signing, and it's
important to explicitly note that they don't appear in the list of
{{requirements}}{:format="title"}.

## Store confidential data {#confidential}

Packages are designed to hold public information and to be shared to
people with whom the original author never has an interactive
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

# Security Considerations

The security considerations will depend on the solution designed to satisfy the
above requirements. See {{?I-D.yasskin-dispatch-web-packaging}} for one possible
set of security considerations.

# IANA Considerations

This document has no actions for IANA.

--- back

# Acknowledgements

