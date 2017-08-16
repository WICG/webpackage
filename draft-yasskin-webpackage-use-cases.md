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
  ServiceWorkers: W3C.WD-service-workers-1-20161011

--- abstract

This document lists use cases for signing and/or bundling collections
of web pages, and extracts a set of requirements from them.

--- middle

# Introduction

People would like to use content offline and in other situations where
there isn’t a direct connection to the server where the content
originates. However, it's difficult to distribute and verify the
authenticity of applications and content without a connection to the
network. The W3C has addressed running applications offline with
Service Workers ({{ServiceWorkers}}), but not the problem of
distribution.


# Use cases

These use cases are in rough descending priority order. If use cases
have conflicting requirements, the design should enable more important
use cases.

## Offline installation {#offline-installation}

Alex can download a file containing a website
(a [PWA](https://developers.google.com/web/progressive-web-apps/checklist))
including a Service Worker from origin `O`, and transmit it to their peer Bailey,
and then Bailey can install the Service Worker with a proof that it came from `O`.
This saves Bailey the bandwidth costs of transferring the website.

Associated requirements:

* {{urls}}{:format="title"}
* {{request-headers}}{:format="title"}
* {{response-headers}}{:format="title"}
* {{signing}}{:format="title"}
* {{existing-certs}}{:format="title"}
* {{multiple-origins}}{:format="title"}
* {{crypto-agility}}{:format="title"}
* {{revocation}}{:format="title"}
* {{no-downgrades}}{:format="title"}

### Online use

Bailey may have an internet connection through which they can, in real time,
fetch updates to the package they received from Alex.

### Fully offline use

Or Bailey may not have any internet connection a significant fraction of the
time, either because they have no internet at all, because they turn off
internet except when intentionally downloading content, or because they use up
their plan partway through each month.

Associated requirements beyond {{offline-installation}}{:format="title"}:

* {{packaged-validity}}{:format="title"}

## Offline browsing {#offline-browsing}

Alex can download a file containing a large website (e.g. Wikipedia) from its
origin, save it to transferrable storage (e.g. an SD card), and hand it to their
peer Bailey. Then Bailey can browse the website with a proof that it came from
`O`. Bailey may not have the storage space to copy the website before browsing
it.

This use case may also cover the
W3C's [Packaged Web Publications](https://www.w3.org/TR/pwp-ucr/#pwp) use case.

Associated requirements beyond {{offline-installation}}{:format="title"}:

* {{random-access}}{:format="title"}
* {{stored-compression}}{:format="title"}

## Save and share a web page {#snapshot}

Casey is viewing a web page and wants to save it either for offline use or to
show it to their friend Dakota. Since Casey isn't the web page's author, they
don't have the private key needed to sign the page. Browsers currently allow
their users to save pages, but each browser uses a different format (MHTML, Web
Archive, or files in a directory), so Dakota and Casey would need to be using
the same browser. Casey could also take a screenshot, at the cost of losing
links and accessibility.

Associated requirements:

* {{unsigned-content}}{:format="title"}
* {{multiple-origins}}{:format="title"}
* {{urls}}{:format="title"}
* {{response-headers}}{:format="title"}

## Third-party security review {#security-review}

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

## Building packages from multiple libraries {#libraries}

Large programs are built from smaller components. In the case of the web,
components can be included either as Javascript files or as `<iframe>`d
subresources. In the first case, the packager could copy the JS files to their
own origin; but in the second, it may be important for the `<iframe>`d resources
to be able to make
[same-origin](https://html.spec.whatwg.org/multipage/origin.html#same-origin)
requests back to their own origin, for example to implement federated sign-in.

Associated requirements:

* {{multiple-origins}}{:format="title"}
* {{deduplication}}{:format="title"}

### Shared libraries

In ecosystems like [Electron](https://electron.atom.io/)
and [Node](https://nodejs.org/en/), many packages may share some common
dependencies. The cost of downloading each package can be greatly reduced if the
package can merely point at other dependencies to download instead of including
them all inline.

Associated requirements:

* {{external-dependencies}}{:format="title"}

## CDNs {#cdns}

CDNs want to re-publish other origins' content so readers can access
it more quickly or more privately. Currently, to attribute that
content to the original origin, they need the full ability to publish
arbitrary content under that origin's name. There should be a way to
let them attribute only the exact content that the original origin
published.

Web Packages would allow CDNs to publish content as another site as
long as the user visited a URL explicitly mentioning the CDN.

CDNs want to serve only the bytes that most optimally represent the
content the current user needs, even though the origin needs to
provide representations for all users. Think PNG vs WebP and small vs
large resolutions.

Associated requirements:

* {{streamed-loading}}{:format="title"}
* {{signing}}{:format="title"}
* {{subsetting}}{:format="title"}
* {{transfer-compression}}{:format="title"}

## Installation from a self-extracting executable {#self-extracting}

The Node and Electron communities would like to install packages using
self-extracting executables. The traditional way to design a
self-extracting executable is to concatenate the package to the end of
the executable, have the executable look for a length at its own end,
and seek backwards from there for the start of the package.

Associated requirements:

* {{trailing-length}}{:format="title"}

## Ergonomic replacement for HTTP/2 PUSH {#push-replacement}

HTTP/2 PUSH ({{?RFC7540}}, section 8.2) is hard for developers to configure, and
an explicit package format might be easier. That said, the HTTPWG is doing a lot
of work to let servers optimize the PUSHed data, and packaging would have to
re-do that.

Associated requirements:

* {{streamed-loading}}{:format="title"}
* {{transfer-compression}}{:format="title"}
* {{urls}}{:format="title"}
* {{response-headers}}{:format="title"}

## Packages in version control {#version-control}

Once packages are generated, they should be stored in version control. Many
popular VC systems auto-detect text files in order to "fix" their line endings.
If the first bytes of a package look like text, while later bytes store binary
data, VC may break the package.

Associated requirements:

* {{binary}}{:format="title"}

# Requirements {#requirements}

## Indexed by URL {#urls}

Resources should be keyed by URLs, matching how browsers look
resources up over HTTP.

## Request headers {#request-headers}

Resource keys should include request headers like `accept` and
`accept-language`, which allows content-negotiated resources to be
represented.

## Response headers {#response-headers}

Resources should include their HTTP response headers, like
`content-type`, `content-encoding`, `expires`,
`content-security-policy`, etc.

## Signing {#signing}

Resources within a package are provably from an entity with the
ability to serve HTTPS requests for those resources' origin.

## Random access {#random-access}

When a package is stored on disk, the browser can access
arbitrary resources without a linear scan.

## Streamed loading {#streamed-loading}

The browser can load a package as it downloads.

## Resources from multiple origins in a package {#multiple-origins}

A package from origin `A` can contain resources from origin `B`
authenticated at the same level as those from `A`.

## Cryptographic agility {#crypto-agility}

Obsolete cryptographic algorithms can be replaced.

## Cross-signatures {#cross-signatures}

Third-parties can vouch for packages by signing them.

## Unsigned content {#unsigned-content}

Alex can create their own package without a CA-signed
certificate, and Bailey can view the content of the package.

## Certificate revocation {#revocation}

When a package is signed by a revoked certificate, online browsers
can detect this reasonably quickly.

## Downgrade prevention {#no-downgrades}

Attackers can't cause a browser to trust an older, vulnerable
version of a package after the browser has seen a newer version.

# Nice to have

## Binary {#binary}

The format is identified as binary by tools that might try to "fix"
line endings.

## Deduplication of diamond dependencies {#deduplication}

Nested packages that have multiple dependency routes to the same
sub-package, can be transmitted and stored with only one copy of that
sub-package.

## Old crypto can be removed {#crypto-removal}

The ecosystem can identify when an obsolete cryptographic algorithm is
no longer needed and can be removed.

## Compress transfers {#transfer-compression}

Transferring a package over the network takes as few bytes as
possible.

## Compress stored packages {#stored-compression}

Storing a package on disk takes as few bytes as possible.

## Subsetting and reordering {#subsetting}

Resources can be removed from and reordered within a package, without
breaking [signatures](#signing).

## Packaged validity information {#packaged-validity}

{{revocation}}{:format="title"} and {{no-downgrades}}{:format="title"}
information can itself be packaged or included in other packages.

## Signing uses existing TLS certificates {#existing-certs}

A "normal" TLS certificate can be used for signing packages. Avoiding
extra requirements like "code signing" certificates makes packaging
more accessible to all sites.

## External dependencies {#external-dependencies}

Sub-packages can be "external" to the main package, meaning the browser
will need to either fetch them separately or already have them.
([#35, App Installer Story](https://github.com/WICG/webpackage/issues/35))

## Trailing length {#trailing-length}

The package's length in bytes appears a fixed offset from the end of
the package.

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

--- back

# Acknowledgements

