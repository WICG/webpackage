# Explainer: Navigation to Unsigned Web Bundles<br>(a.k.a. Bundled HTTP Exchanges)

Last updated: Feb 04, 2020

Participate at https://github.com/WICG/webpackage and
https://datatracker.ietf.org/wg/wpack/about/.

Users want to be able to share content with their friends, even when neither has
an internet connection. Some websites, for example some books, encyclopedias,
and games, want to help their users share them in this way. For individual
images or videos, users have native apps like
[SHAREit](https://www.ushareit.com/), [Xender](http://www.xender.com/), or
[Google Files](https://files.google.com/) that can share files.

<!-- TOC -->

- [Proposal](#proposal)
  - [Relevant structure of a bundle](#relevant-structure-of-a-bundle)
  - [Process of loading a bundle](#process-of-loading-a-bundle)
  - [Loading a trusted unsigned bundle.](#loading-a-trusted-unsigned-bundle)
  - [Loading an untrusted bundle](#loading-an-untrusted-bundle)
  - [Subresource loads with an attached bundle](#subresource-loads-with-an-attached-bundle)
  - [Service Workers](#service-workers)
  - [URLs for bundle components](#urls-for-bundle-components)
- [Open design questions](#open-design-questions)
  - [Network access](#network-access)
- [Security and privacy considerations](#security-and-privacy-considerations)
- [Considered alternatives](#considered-alternatives)
  - [Alternate formats considered](#alternate-formats-considered)
    - [Save as directory tree](#save-as-directory-tree)
    - [Save as MHTML](#save-as-mhtml)
    - [Save as Web Archive](#save-as-web-archive)
    - [WARC](#warc)
    - [Mozilla Archive Format](#mozilla-archive-format)
    - [A bespoke ZIP format](#a-bespoke-zip-format)
  - [Alternate URL schemes considered](#alternate-url-schemes-considered)
    - [arcp](#arcp)
    - [pack](#pack)
- [Stakeholder feedback](#stakeholder-feedback)
- [Acknowledgements](#acknowledgements)

<!-- /TOC -->

## Proposal

We propose to use the [Web Bundle
format](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html),
without signatures, to give users a format they can create from any website
they're visiting, share it using existing peer-to-peer sharing apps,
and then have a peer load it in their browser. The IETF has
[approved](https://ietf.org/blog/ietf106-highlights/) but not yet created a
WPACK working group to continue developing this format as of February 2020.

Here's a [demo video](https://youtu.be/xAujz66la3Y) of this use-case for Web
Bundles (a.k.a. Bundled HTTP Exchanges):

<a href="https://www.youtube.com/watch?v=xAujz66la3Y">
<img src="https://img.youtube.com/vi/xAujz66la3Y/0.jpg">
</a>

This explainer doesn't currently propose a standard way to identify all the
resources from a loaded web page that need to be included in a bundle, or to
identify a pre-packaged bundle that contains the current page. Instead it
focuses on how to safely load and navigate within a bundle that's already
created.

### Relevant structure of a bundle

A bundle is loaded from a "*bundle URL*", and it contains a set of HTTP *items*,
each with a "*claimed URL*". The items are serialized in a particular order
within the bundle, and this will affect the performance of loading the bundle
when the bundle is loaded from a non-random-access medium (like the network),
but it doesn't affect the semantics of the bundle.

The bundle URL might be a `file://` URL containing private information like the
user's name, which they might not want exposed to a website they saved for
offline use.

### Process of loading a bundle

When the browser navigates to an `application/webbundle;v=___` resource, or
opens a file that the filesystem identifies as that MIME type, it first parses
the resource enough to pull out the bundle's [primary
URL](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#name-load-a-bundles-metadata)
and a set of flags to control loading (not yet defined in the format). One of
these flags defines whether the claimed URLs in the bundle are expected to be
trusted, which for unsigned bundles means the bundle is served from the origin
that claims it.

### Loading a trusted unsigned bundle.

The browser selects an **unsigned bundle scope** that's based on the bundle's
URL in the same way the [service worker's scope
restriction](https://w3c.github.io/ServiceWorker/#path-restriction) is based on
the URL of the service worker script. If the bundle is served with the
[`Service-Worker-Allowed`
header](https://w3c.github.io/ServiceWorker/#service-worker-allowed), that sets
the unsigned bundle scope to the value of that header. (We are also considering
using a differently-named header that applies only to bundles.)

Bundles can also contain a
[manifest](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#name-parsing-the-manifest-sectio)
whose [`scope` key](https://www.w3.org/TR/appmanifest/#scope-member) can narrow
the scope but not broaden it.

> Browsers might also provide a way for device manufacturers to define an area
  of the local filesystem that can serve trusted unsigned bundles without any
  scope restriction. This would make it easier to pre-install applications in a way that has a chance of working across different browsers.

The browser redirects to the bundle's primary URL. If that primary URL is within
the unsigned bundle scope, it also attaches the bundle itself and its unsigned
bundle scope to the redirect and the subsequent environment settings object.
That is, the bundle is treated as just a very expensive redirect if it tries to
claim URL space it's not allowed to define.

Subsequent subresource requests will also only be served from the bundle if
they're inside its scope.

### Loading an untrusted bundle

The browser redirects to [a URL that refers to the bundle's primary URL *inside
its bundle*](#urls-for-bundle-components), with the bundle itself attached.

Because the user might save a bundle to a file whose path includes private
information like their username, APIs like
[`Window.location`](https://developer.mozilla.org/en-US/docs/Web/API/Window/location)
and
[`Document.URL`](https://developer.mozilla.org/en-US/docs/Web/API/Document/URL)
have to return the claimed URL and not the full encoded URL. Avoiding the new
scheme in these APIs will also probably improve compatibility.

### Subresource loads with an attached bundle

The bundle that's attached to the request above will eventually get attached to
the environment settings object created for the response, from where subsequent
fetches can use it.

For trusted bundles, any fetch, both navigations and subresources, within the
bundle's scope checks the bundle for a response before going to the network. If a navigation fetch finds a response inside the bundle,

For untrusted bundles, all fetches should check inside the bundle before going
to the network. [It's not clear whether subresource requests that aren't inside
the bundle should be able to touch the network at all.](#network-access)

### Service Workers

We plan to, but haven't yet, defined an API to expose trusted bundles to that
origin's service worker. This API should allow the service worker to fill its
cache with the contents of the bundle.

### URLs for bundle components

Each item of the bundle is addressible using a URL, with the new scheme
`package:`. (See [below](#alternate-URL-schemes-considered) for some details of
this choice.) This scheme identifies both the URL of the bundle itself (e.g.
`https://distributor.example/package.wbn?q=query`) and the claimed URL inside
the bundle (e.g. `https://publisher.example/page.html?q=query`) These are
encoded so that the [normal algorithm for computing an origin from a
URL](https://url.spec.whatwg.org/#concept-url-origin) works, by replacing `:`
with `!`, `/` with `,`, and `?` with `;`, and separating the 2 URLs with `$`.
Any instance of `!`, `,`, `;`, or `$` in the URLs themselves is %-encoded:

```url
package://https!,,distributor.example,package.wbn;q=query$https!,,publisher.example/page.html?q=query
```

The origin for that URL is:

```url
package://https!,,distributor.example,package.wbn;q=query$https!,,publisher.example
```

These URLs and origins allow the components of bundles to save data in a way
that's shared within the same claimed origin within a single bundle, but which
can't conflict with or attack:

* sites served verifiably from that claimed origin (e.g.
  `https://publisher.example/page.html?q=query`),
* other bundles (e.g. `package://https!,,distributor.example,otherpackage.wbn;q=query$https!,,publisher.example/page.html?q=query`), or
* other origins within the same bundle (e.g. `package://https!,,distributor.example,package.wbn;q=query$https!,,otherpublisher.example/page.html?q=query`).

## Open design questions

* Do we need an "internal redirect" notion for the bundle->primary-URL redirect?
  See also [whatwg/fetch#576](https://github.com/whatwg/fetch/issues/576).

### Network access

Should untrusted bundles be able to go to the network at all? Should users be
able to explicitly grant permission?

Under repressive governments, untrusted bundles may be a good way for dissidents
to share information, but if the government can add their own tracking code to
the bundles, they may be able to catch people using the banned information.

However, there are also use cases like for sharing book previews, where a
publisher might want to provide both a free preview and the encrypted rest of
the book in the same unsigned bundle, while letting users buy a small decryption
key with an online flow. On the other hand, perhaps that use case should use the
signed bundles that aren't being proposed in this explainer.

## Security and privacy considerations

* Unsigned bundles served by one origin can't be trusted to provide content for
  another origin. The [URL design](#urls-for-bundle-components) and the division
  between trusted and untrusted bundles are designed to keep separate origins
  safely separate, but this is more than a trivial amount of complexity, and
  browsers will need to be careful to actually enforce the boundaries.

* It's straightforward for someone serving an unsigned bundle to include a
  unique ID in the resources within that bundle. If the bundle can then make
  [network requests](#network-access), especially credentialed network requests,
  the author can determine a set of people who are connected to eachother in
  some way. This is detectable if users have a way to inspect two bundles that
  are supposed to hold the same content, but since the whole point of sharing
  bundles is to reduce redundant transfers, it's unlikely many users  will
  actually check this.

  Sites currently gather information on this kind of link sharing by annotating
  their URLs with unique or semi-unique IDs. These can usually be removed by
  removing query parameters, but if a significant number of users cleaned their
  URLs, the tracking could move to path segments.

## Considered alternatives

### Alternate formats considered

There are several existing ways to serialize a web page or website, but all of
them have problems:

#### Save as directory tree

This is currently supported by Chromium and Firefox using the `Save As...` |
`Webpage, Complete` dialog. The result is a whole directory, which doesn't match
the sharing models of the existing apps.

#### Save as MHTML

This is currently supported by Chromium using the `Save As...` | `Webpage,
Single File` dialog.

MHTML isn't a random-access format, which means that the entire prefix of the
file needs to be parsed to get to any particular resource. This probably
wouldn't be an issue for single web pages, but for full websites or bundles of
multiple websites, it can be slow.

#### Save as Web Archive

This is currently supported by Safari using the `Save As...` | `Web Archive` dialog.

A Web Archive is in the semi-documented binary plist format and isn't random
access either. Web Archives also currently allow
[UXSS](https://blog.rapid7.com/2013/04/25/abusing-safaris-webarchive-file-format/)
if a user opens one from an untrustworthy source.

#### WARC

[WARC](https://en.wikipedia.org/wiki/Web_ARChive) was designed for archiving
websites and is in active use by the [Internet Archive](https://archive.org/)
and many other web archives. It includes more information about the archival
process itself than MHTML does. Most of this extra information would probably be
irrelevant to the use cases in this document, although the archival timestamp
could be useful.

Like MHTML, WARC is not random access.

#### Mozilla Archive Format

"The [MAFF](http://maf.mozdev.org/maff-specification.html) format is designed to
work in the same way as when a saved page is opened directly from a local file
system", which means that response headers can't be encoded at all. MAFF also
doesn't pick one of its contained pages as the default one, so there's nothing
to navigate to if a user just opens the MAFF as a whole.

As a ZIP file, each of the subresources of one of the contained pages can be
accessed randomly, but MAFF doesn't appear to define any way for the pages to
link to each other.

#### A bespoke ZIP format

ZIP is nicely random access if we encode the resource URL into the path.
However, URLs aren't quite paths, so we would probably need to encode them.

ZIP holds files, so we would need to encode any content negotiation information and response headers into the stored files.

The ZIP format has some ambiguities in parsing it that have led to
[vulnerabilities in
Android](https://nakedsecurity.sophos.com/2013/07/10/anatomy-of-a-security-hole-googles-android-master-key-debacle-explained/).

ZIP files place their index at the end, which allows them to be generated
incrementally or updated by simply appending new data. However, the trailing
authoritative index prevents a consumer from processing anything until the
complete file is received. Streaming isn't particularly useful for the
peer-to-peer transfer use case, but it allows the same format to be used for
subresource bundles (which will have a separate explainer).

### Alternate URL schemes considered

A Google doc, [Origins for Resources in Unsigned
Packages](https://docs.google.com/document/d/1BYQEi8xkXDAg9lxm3PaoMzEutuQAZi1r8Y0pLaFJQoo/edit),
goes into more detail about why we need a new scheme at all, and discusses the
2-URL scheme described above and a hash+URL scheme similar to [`arcp://ni,`
URIs](https://tools.ietf.org/html/draft-soilandreyes-arcp-03#section-4.1).

#### arcp

[draft-soilandreyes-arcp](https://tools.ietf.org/html/draft-soilandreyes-arcp-03),
from 2018, proposes an `arcp:` scheme that encodes a UUID, hash, or name for an
archive/package, plus a path within that package. It's designed to be usable
with several different kinds of archive, listing "application/zip",
"application/x-7z-compressed", LDP Containers, installed Web Apps, and a BagIt
folder structure as possible kinds.

By using just a path to identify the thing within the package, `arcp:` isn't
compatible with the way bundles include resources from several different
origins. On the other hand, the [`package:` scheme](#urls-for-bundle-components)
discussed above can work for path-based archives, by either using a `file:` URL
to identify the element of the package, or perhaps by omitting the part of the
authority after `$`. Adding a second URL may also be a compatible addition to
`arcp:`.

`arcp:` doesn't define a way to locate the archive, which prevents an external
link from pointing directly to one element of the archive, but an external link
can still point to the archive as a whole. Once a client has seen an archive, it
can store a mapping from any of the `arcp:` URI forms to the archive itself,
which allows navigation within the archive. The `arcp://ni,` form, in
particular, has the benefit of allowing users to move an archive around without
losing its storage; and the drawback of preventing the user from upgrading the
archive without losing its storage.

`arcp:`'s flexibility in naming the archive, between random UUIDs, UUIDs based
on the archive's location, hashes of the archive content, or system-dependent
archive names, could be useful for varying system requirements, but it also
introduces implementation complexity. We'd probably want to pick just one of those options for bundles.

#### pack

[draft-shur-pack-uri-scheme](https://tools.ietf.org/html/draft-shur-pack-uri-scheme-05),
from 2009, defines a `pack:` scheme meant for use with OOXML packages. It
identifies a package with an encoded URL, which partially inspired the
[encoding](#urls-for-bundle-components) proposed in this document. Like `arcp:`
it only expects elements of a package to have paths, which doesn't allow the
full URLs needed by web bundles.

## Stakeholder feedback

* W3CTAG: ?
* Browsers:
  * Safari: ?
  * Firefox: ?
  * Samsung: ?
  * UC: ?
  * Opera: ?
  * Edge: ?
* Web developers: ?

## Acknowledgements

Many people have helped in the design of web packaging in general. The design of
the URL scheme was helped in particular by:

* Anne van Kesteren
* Graham Klyne and the uri-review@ietf.org list
