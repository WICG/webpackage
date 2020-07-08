# URLs and Origins for Resources inside Bundles

Authors:
* Jeffrey Yasskin (Google Chrome)

Participate:
* https://github.com/WICG/webpackage/issues
* https://www.ietf.org/mailman/listinfo/wpack

**Table of Contents**

<!-- TOC -->

- [Introduction](#introduction)
- [Terminology](#terminology)
- [Goals](#goals)
  - [Fully-qualified subresource names](#fully-qualified-subresource-names)
  - [Correctly-identified actors](#correctly-identified-actors)
  - [Correctly-scoped storage](#correctly-scoped-storage)
    - [Distinguish from bundle's server](#distinguish-from-bundles-server)
    - [Distinguish from other bundles](#distinguish-from-other-bundles)
    - [Distinguish from other origins in the same bundle](#distinguish-from-other-origins-in-the-same-bundle)
  - [Don't expose private information](#dont-expose-private-information)
  - [Don't create cross-site messaging channels](#dont-create-cross-site-messaging-channels)
  - [Bundled sites run without modification](#bundled-sites-run-without-modification)
- [Non-goals](#non-goals)
- [Proposal: a `package:` scheme](#proposal-a-package-scheme)
- [Key scenarios](#key-scenarios)
  - [Same-bundle introspection](#same-bundle-introspection)
  - [`Referer` headers](#referer-headers)
    - [Avoiding private information in the referrer](#avoiding-private-information-in-the-referrer)
    - [Anti-tracking in the referrer](#anti-tracking-in-the-referrer)
    - [Avoiding unexpected `package:` referrers within a bundle](#avoiding-unexpected-package-referrers-within-a-bundle)
  - [Origins](#origins)
    - [`Origin` headers](#origin-headers)
    - [`postMessage` target origin](#postmessage-target-origin)
    - [`postMessage` source origin](#postmessage-source-origin)
  - [Rendering the URL bar](#rendering-the-url-bar)
  - [Permissions](#permissions)
- [Detailed design discussion](#detailed-design-discussion)
  - [Exactly how do we compose the package: URL?](#exactly-how-do-we-compose-the-package-url)
- [Considered alternatives](#considered-alternatives)
  - [Rely on storage partitioning](#rely-on-storage-partitioning)
  - [URL encoding variants](#url-encoding-variants)
    - [Fragment-based URL scheme](#fragment-based-url-scheme)
      - [Fragments and MIME types](#fragments-and-mime-types)
    - [Percent-encode "/" instead of replacing it with ","](#percent-encode--instead-of-replacing-it-with-)
    - [Only percent-encode "$"](#only-percent-encode-)
    - [Other Internet-Drafts](#other-internet-drafts)
  - [Referrer computation variants](#referrer-computation-variants)
  - [Origin serialization variants](#origin-serialization-variants)
    - [Allow more origin for CORS and websockets](#allow-more-origin-for-cors-and-websockets)
    - [Omit all paths](#omit-all-paths)
- [Stakeholder Feedback / Support / Opposition](#stakeholder-feedback--support--opposition)
- [References & acknowledgements](#references--acknowledgements)

<!-- /TOC -->

## Introduction

A [web
bundle](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html)
can be fetched from a URL, and it contains a set of HTTP responses identified by
URL. This document explains how to name those component subresources directly
from outside of the bundle and how those names interact with the other parts of
the web platform API.

## Terminology

* A resource's **Claimed URL**: The URL that names the resource within a bundle.
* A resource's **Bundle URL**: The URL that can be fetched to retrieve the bundle
  that the resource appears inside of. As usual, the same sequence of bytes
  representing a bundle can be fetched from more than one URL.
* **Distributor**: the entity that serves a bundle. Identical to the owner of the
  Bundle URL's origin. This entity has the power to change the content of the
  bundle, including subresource named by URLs under other origins.
* **Publisher**: the owner of the Claimed URL's origin. There's no guarantee that
  the publisher had anything to do with the content in the bundle that's
  associated with their name.

## Goals

### Fully-qualified subresource names

When a bundle is used to serve a page's
[subresources](./subresource-loading.md), the simplest way to ensure the bundle
is fetched before the subresource is to name the subresource relative to the
bundle. That requires the subresources to have fully qualified names.

### Correctly-identified actors

When the platform identifies a bundle subresource to some other entity, it must
give the other entity an accurate impression of who's responsible for the
subresource. This can show up in the URL bar, permission dialogs, the `Referer`
and `Origin` headers, the `postMessage()` `targetOrigin` and `event.origin`
fields, etc.

### Correctly-scoped storage

When a bundle is used to serve a [top-level
page](./navigation-to-unsigned-bundles.md) or an iframe, it often needs to
[store some data](#why-store-data) across reloads. This data needs to go in a
[storage shelf](https://storage.spec.whatwg.org/#storage-shelf) that's shared
with the right set of other resources. It's easiest to see how storage needs to
be partitioned with a series of examples.

Imagine that a bundle served at `https://distributor.example/bundle.wbn` contains a subresource named `https://foo.example/page.html`.

#### Distinguish from bundle's server

The bundle should use a different storage shelf from
`https://distributor.example/page.html`.

This provides a way to create
[suborigins](https://w3c.github.io/webappsec-suborigins/), which could be useful
to sites like the Internet Archive. The Internet Archive currently serves copies
of arbitrary websites from inside https://web.archive.org/, which means those
websites can't be allowed to access Javascript storage lest they steal the main
site's login cookies.

#### Distinguish from other bundles

The subresource should use a different storage shelf from
`https://foo.example/page.html` (the same subresource name) inside
`https://distributor.example/otherbundle.wbn`.

* If an archive stores multiple versions of the same website in separate
  bundles, but those versions use storage differently, users couldn't easily try
  more than one version.
* If a user is relying on an application they have saved in one bundle, another
  website shouldn't be able to get access to that application's data just by
  creating a bundle that claims the same URLs.

#### Distinguish from other origins in the same bundle

The subresource should use a different storage shelf from
`https://bar.example/page.html` (a different subresource origin) inside
`https://distributor.example/bundle.wbn` (the same bundle).

This allows a single bundle, for example one created for [El Paquete
Semanal](https://en.wikipedia.org/wiki/El_Paquete_Semanal), to copy in many
applications written by different authors that link to each other and that all
use their own storage without conflicting with each other. This use case is also
achievable by rewriting parts of the applications, so it should be the lowest
priority in this section.

### Don't expose private information

When a bundle is downloaded locally, the file path often contains the user's
name and may contain other sensitive information. This must not leak out onto
the network, either by being exposed inside the bundle which could make a
network request to send it, or by being included in
[`Origin`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Origin) and
other request headers.

Even a network URL for a bundle has the chance of exposing private information
in its path, which we need to be careful to avoid.

### Don't create cross-site messaging channels

In order to eliminate [cross-site
tracking](https://w3cping.github.io/privacy-threat-model/#model-cross-site-recognition),
the web is removing some cases that sent a path from one site to another. The
URLs and origins of resources within packages shouldn't provide a new way to
send paths between different sites.

### Bundled sites run without modification

It should be straightforward to save the resources in a site to a bundle and
then run them from the bundle without changing the resources.

## Non-goals

This document does not discuss how to establish that a response in a bundle is
[authoritative](https://httpwg.org/http-core/draft-ietf-httpbis-semantics-latest.html#establishing.authority)
for its *claimed URL*. Other documents suggest using
[signatures](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html),
an [online adoption
protocol](https://tools.ietf.org/id/draft-thomson-wpack-content-origin-00.html),
[being under the path of the bundle
URL](./navigation-to-unsigned-bundles.md#authoritative), or other techniques to
establish authority, but the scheme here is agnostic to that choice.

## Proposal: a `package:` scheme

We propose to define a new URL scheme, `package:`, that encodes both the *bundle
URL* and the *claimed URL* into a single URL. See below for [other reasonable
ways to do this encoding](#url-encoding-variants).

The `package:` scheme avoids the `//<authority>/` URL syntax to avoid expanding
the set of characters a URL parser needs to expect to find in an authority, but
the first component of the `package:` path acts much like an authority in
picking out a storage shelf.

To put the right parts of the bundle and claimed URLs into that first path
segment, the whole bundle URL and the part of the claimed URL before the path
are percent-encoded except for `/` and `?`, and then those characters are
replaced by `,` and `;`, respectively. See the [details
below](#exactly-how-do-we-compose-the-package-url).

For a bundle URL of `https://distributor.example/package.wbn?q=query` and a claimed URL of `https://claimed.example/path/page.html?q=query`, we get:

```url
package:https:,,distributor.example,package.wbn;q=query$https:,,publisher.example/path/page.html?q=query
```

We define the origin for these URLs to consist of everything before the first `/`:

```url
package:https:,,distributor.example,package.wbn;q=query$https:,,publisher.example
```

These URLs and origins satisfy the goals of letting us [directly address bundle subresouces](#fully-qualified-subresource-names) and [scoping storage correctly](#correctly-scoped-storage).

## Key scenarios

### Same-bundle introspection

Operations within a single bundle should use the *claimed URLs*. This allows a
bundle to act as an archive of a set of sites without breaking their internal
assumptions.

This also helps prevent [exposing private
information](#dont-expose-private-information): if a realm running inside a
downloaded bundle could see the bundle URL and then make network requests, it
could report the path to the bundle.

Inside the `https://claimed.example/page.html` subresource of the
`https://distributor.example/package.wbn` bundle:

```js
expect(location.href).to.be("https://claimed.example/page.html");
expect(document.URL).to.be("https://claimed.example/page.html");
```

### `Referer` headers

A navigation from `package:bundle-url$claimed-url` to a target URL has two
[referrer
policies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy):
the one delivered with the bundle and the one attached to the subresource inside
the bundle. The one delivered with the bundle needs to default to
`strict-origin-when-cross-origin` (or stronger) as proposed in
[whatwg/fetch#952](https://github.com/whatwg/fetch/pull/952), and then we apply
them both to the matching parts of the `package:` URL.

We first apply the bundle's referrer policy to the source *bundle URL*, treating
it as same-origin with the target URL if the target URL is also a `package:` URL
with a same-origin *bundle URL*.

If this drops the source URL's path, the referrer also doesn't include any part
of the *claimed URL*, yielding either `no referrer` or `package:bundle-origin`.
If applying the bundle's referrer policy includes the bundle URL's path, the
subresource's referrer policy is applied to the subresource's name. The target
is considered same-origin only if it's inside the same bundle and its *claimed
URL* is same-origin with the source's *claimed URL*.

If the subresource's computed referrer is `no referrer`, but the bundle's
referrer is present, the `Referer` string has the form
`package:bundle-referrer`, with no `$` marking a separation from the absent
subresource URL.

#### Avoiding private information in the referrer

Generally [a file's path shouldn't be included in the referrer given to a site
that's the target of a navigation from that
file](#dont-expose-private-information). As files have [opaque
origins](https://html.spec.whatwg.org/multipage/origin.html#concept-origin-opaque),
they'll be cross-origin with any navigation target, and because of the referrer
policy of `strict-origin-when-cross-origin` described above, they'll use the
serialization of the file's origin ("null") for navigations out of the same
bundle.

#### Anti-tracking in the referrer

User agents strip some parts of the referrer on cross-site navigations in order
to prevent tracking, and the above algorithm facilitates stripping the right
parts of both the *bundle URL* and the *claimed URL*.

Specifically, if a site is trying to embed identifying information into a URL,
it's equally easy to embed it in the *path* of the *bundle URL* or the *origin*
(or path) of the *claimed URLs*. Since the above algorithm strips both of those
with the same set of referrer policies, it achieves the goal of [not creating
cross-site messaging channels](#dont-create-cross-site-messaging-channels).

#### Avoiding unexpected `package:` referrers within a bundle

Within the same bundle, the bundle's URL isn't exposed, so
[`document.referrer`](https://developer.mozilla.org/en-US/docs/Web/API/Document/referrer),
[`Request.referrer`](https://developer.mozilla.org/en-US/docs/Web/API/Request/referrer),
and the `Referer` header as exposed to a Service Worker's [fetch
event](https://w3c.github.io/ServiceWorker/#fetchevent) have to show only the
filtered *claimed URL* when the referrer's *bundle URL* matches the realm's
*bundle URL*.

> For example, if the true referrer is
`package:https:,,distributor.example,package.wbn$https:,,claimed.example/page.html`,
and the current document is
`package:https:,,distributor.example,package.wbn$https:,,claimed.example/page2.html`,
the `document.referrer` getter will notice the matching *bundle URLs* and return
just the *claimed URL* out of the referrer. In this case, `document.referrer`
is:
>
> ```url
> https://claimed.example/page.html
> ```
>
> Or if the current document is in a different bundle, `document.referrer` is:
>
> ```url
> package:https:,,distributor.example,package.wbn$https:,,claimed.example/page.html
> ```
>
> If the true referrer is `package:https:,,distributor.example,package.wbn` and
the current document is in the same bundle, `document.referrer` is the empty
string.

To avoid adding exceptions to the [above algorithm](#referer-headers) for
same-bundle navigations, there will be some cases where a same-bundle navigation
has no referrer but the same navigation between resources on the web would have
a referrer. Specifically, if the bundle is served with a referrer policy of
`no-referrer`, `origin`, or `strict-origin`, the *claimed URL* is entirely
removed from the referrer, so regardless of the subresource's referrer policy,
`document.referrer` will be the empty string.

### Origins

#### `Origin` headers

When a resource inside a bundle makes a cross-origin network request, what
[`Origin`
header](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Origin) should
be sent?

We propose the simple thing here: Compute the [referrer](#referer-headers) that
would be used for a navigation to the requested resource and send the origin of
that referrer string, computed by dropping the path.

This omits [Fetch's
allowance](https://fetch.spec.whatwg.org/#append-a-request-origin-header) for
cors-mode and websocket requests to get the full origin even if the referrer
policy would disallow it.

As with the referrer, when the `Origin` header is observed within the same
bundle, most likely via service workers, the bundle part needs to be removed.

#### `postMessage` target origin

When sending a message, the sender
sets the
[`targetOrigin`](https://developer.mozilla.org/en-US/docs/Web/API/Window/postMessage#Syntax)
field to determine which recipients can read it.

* When sending from outside the bundle to inside, or from inside one bundle to
  inside a different bundle, the sender uses the full
  `package:bundle-url$claimed-origin`.
* When sending from inside the bundle to inside, the sender uses just the
  claimed origin.
* When sending from inside the bundle to outside, the sender uses the same
  target origin that a sender outside the bundle would use. This means that
  `package:a-bundle$https:,,source.example` uses the same target origin for
  sending to either `https://target.example` or
  `package:a-bundle$https:,,target.example`, which could let either receive
  messages meant for the other. This should be ok: things inside bundles are
  meant to be quotes of things outside, and if the entity who composed a bundle
  is dishonest about this, they can modify the sender as easily as its target.

#### `postMessage` source origin

When receiving a message, the recipient is told the [sender's
origin](https://developer.mozilla.org/en-US/docs/Web/API/Window/postMessage#The_dispatched_event).

* When sending from inside the bundle to outside or from inside one bundle to
  inside another bundle, the exposed value matches the `Origin` header we'd send
  for a request to the target resource.
* When sending from outside the bundle to inside, the origin is the same as if
  the message were received outside a bundle.
* When sending from inside a bundle to inside the same bundle, the target is
  told the claimed origin of the source. This means there's no way to
  distinguish an intra-bundle message from a message that the browser can verify
  comes from its source, which should be ok. As for the target origin, if the
  bundle's composer wants to spoof a message, they could modify its recipient
  inside the bundle.

### Rendering the URL bar

The current [rendering advice in the URL
spec](https://url.spec.whatwg.org/#url-rendering) is not appropriate for the
default display of `package:` URLs, as users won't understand the significance
of its "host" part,
`https:,,distributor.example,otherpackage.wbn;q=query$https:,,publisher.example`.

We suggest that, in places the browser would render just a URL's host, it render
the host of the *bundle URL*, so just `distributor.example` in the above
example. When the browser would render the full URL, it should show just the
bundle's URL with some indication that it's viewing just a piece of that bundle.
To edit the URL, the browser should allow the user to pick from the resources
contained inside the bundle instead of encouraging the user to edit the text of
the `package:` URL.

### Permissions

How should a resource at a `package:` URL be able to ask for permission to use
powerful web APIs? As with the [URL bar](#rendering-the-url-bar), users should
generally see the bundle itself as requesting permission, since it's the
bundle's server that can control its content. However, a bundle server might
host user-provided content and have a better reputation than it wants to lend to
that content. As such, the bundle server needs to be able to choose whether
those permission requests can happen.

[`Permissions-Policy`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Feature_Policy)
may be a good way to express this. Specifically, we could treat bundled content
as cross-origin from its server, and only allow permission requests if the
server sends a `Permissions-Policy: geolocation=(self
"package:https://server.example")` header.

## Detailed design discussion

### Exactly how do we compose the package: URL?

The `package:` URL contains two other URLs, which need to be encoded and
delimited so that they're clearly distinct. Because this involves characters
that would not appear in an `https://` URL's authority, we choose to avoid the
generic URL's authority component, and instead use the `package:<path>` form. To
clearly show the part of the URL that selects a storage shelf, we encode that
into the first path component. This leads to the following algorithm for
combining a *bundle URL* and a *claimed URL* into a `package:` URL:

1. Let the *claimed URL prefix* be the part of the *claimed URL* before its
   path. For example, the prefix of `https://host/path` would be `https://host`,
   while the prefix of `urn:uuid:12345` would be `urn:`.
1. Let the *package percent encode set* be the [C0 control percent-encode
   set](https://url.spec.whatwg.org/#c0-control-percent-encode-set) and `,`,
   `;`, `$`, and `%`.
1. Let the *encoded bundle URL* be the result of
   [UTF-8-percent-encoding](https://url.spec.whatwg.org/#string-utf-8-percent-encode)
   the *bundle URL* with the *package percent encode set*.
1. Let the *encoded claimed URL prefix* be the result of
   [UTF-8-percent-encoding](https://url.spec.whatwg.org/#string-utf-8-percent-encode)
   the *claimed URL prefix* with the *package percent encode set*.
1. In the *encoded bundle URL* and the *encoded claimed URL prefix*, replace `/`
   with  `,` and `?` with `;`.
1. Return the concatenation of:
   1. `package:`
   1. The *encoded bundle URL*
   1. `$`
   1. The *encoded claimed URL prefix*
   1. The path and query of the *claimed URL*.

To parse the `package:` URL into a bundle URL and a claimed URL, we can use the following algorithm:

1. Let *package URL* be the
   [parsed](https://url.spec.whatwg.org/#concept-url-parser) `package:` URL.
1. If *package URL*'s [path](https://url.spec.whatwg.org/#concept-url-path)
   doesn't have exactly 1 component, the URL is malformed. (The URL parser
   doesn't parse out path segments for URLs without `://`.)
1. Split the only component of the *package URL*'s path on the `$` into the
   *encoded bundle URL* and the *encoded claimed URL*. If it doesn't have a `$`,
   it's malformed.
1. Split the *encoded claimed URL* into the *encoded claimed URL prefix* before
   the first `/` if any, and the *claimed URL path and query* including the
   first `/` and after. The *claimed URL path and query* is empty if there is no
   `/`.
1. In both the *encoded bundle URL* and the *encoded claimed URL prefix*,
    replace `,` with `/` and `;` with `?`.
1. Let the *bundle URL* be the [percent decoding](https://url.spec.whatwg.org/#percent-decode) of *encoded bundle URL*.
1. Let the *claimed URL* be the concatenation of
   1. the [percent decoding](https://url.spec.whatwg.org/#percent-decode) of the *encoded claimed URL prefix* with
   1. the *claimed URL path and query*.
1. Return the *bundle URL* and the *claimed URL*.

## Considered alternatives

### Rely on storage partitioning

The above design relies on the URL to both [name a bundle
subresource](#fully-qualified-subresource-names) and [give its storage the right
scope](#correctly-scoped-storage). Instead, we could explicitly define that the
active [storage shelf](https://storage.spec.whatwg.org/#storage-shelf) depends
on the bundle path to the resource (in addition to [the frame path to the
`window`](https://github.com/privacycg/storage-partitioning)).

Environment settings objects loaded from a bundle need to track the bundle
regardless, so that fetches can look in the bundle before going to the network,
so this isn't an extra piece of data to track.

Going this route makes [same-bundle introspection](#same-bundle-introspection)
easier, since the active URL of the document is the *claimed URL*. However, it
makes it harder to ensure [referrers](#referer-headers) and [origins](#origins)
are correct since it avoids defining a syntax that can slot into the existing
headers and fields. That could be positive—existing code won't be expecting a
new scheme here, so one could cause compatibility issues—but also seems likely
to cause confusion about which entity is acting.

### URL encoding variants

There are several reasonable ways to encode the *bundle URL* and *claimed URL* into a
single URL.

This is an open question, with only a small preference for the [encoding
described above](#proposal-a-package-scheme).

#### Fragment-based URL scheme

It's easy to address a subresource of a bundle by putting the *claimed URL* into
the fragment of the *bundle URL*. We can represent recursive fragments with percent-encoding or a key-value format for the fragments:

* `https://distributor.example/package.wbn?q=query#https://claimed.example/path/page.html?q=query%23fragment`
* `https://distributor.example/package.wbn?q=query#url=https://claimed.example/path/page.html?q=query;fragment=fragment`
  (from the [TAG's
  design](https://www.w3.org/TR/2015/WD-web-packaging-20150115/#fragment-identifiers))

This option doesn't immediately satisfy the goal to [scope
storage](#correctly-scoped-storage), but it could if we [define a new
scheme](https://mailarchive.ietf.org/arch/msg/wpack/XkQ8OSlGn18xsVNtSXvoD7s0FIU/):

`pkg+https://distributor.example/package.wbn?q=query#url=https://claimed.example/path/page.html?q=query;fragment=fragment`

The downside here is that the origin computation has to take part of the
fragment into account. This could work, with risks that:

* The fragment might sometimes get dropped by code that was written against the
  currently-correct assumption that the fragment doesn't affect the origin.
* Some code might only be looking at the host, which would fail to [distinguish
  the bundle from its server](#distinguish-from-bundles-server).

##### Fragments and MIME types

The interpretation of a fragment depends on the MIME type of the resource it's
applied to. Because the bundle URL inside a`pkg+https` URL could refer to
several kinds of things with subresources, like ZIP, tar, 7z, etc. files, we
should find some way to provide a consistent fragment format for all of them.
The fact that the other formats only name their contents with paths and not full
URLs only [slightly diminishes the
utility](#distinguish-from-other-origins-in-the-same-bundle).

#### Percent-encode "/" instead of replacing it with ","

The algorithm for encoding `package:` URLs would be simpler if each component URL were uniformly percent-encoded. We'd wind up with something like:

```url
package:https%3A%2F%2Fdistributor.example%2Fpackage.wbn%3Fq%3Dquery$https%3A%2F%2Fclaimed.example/path/page.html?q=query
```

While none of the `package:` URLs are readable by end-users (URLs in general
aren't readable by end-users), there's some benefit to making the URLs as
readable as possible for developers trying to figure out why their links are
broken. Minimizing the amount of percent-encoding helps with that.

#### Only percent-encode "$"

We could reduce percent-encoding even more by only protecting the delimiter,
yielding:

```url
package:https://distributor.example/package.wbn?q=query$https://claimed.example/path/page.html?q=query
```

This is probably the second-most readable option behind [using
fragments](#fragment-based-url-scheme). Like with the fragment, it makes it
unclear which parts of the URL contribute to the storage key, but it doesn't
incur the risk that fragments are dropped before they could contribute to the
origin.

#### Other Internet-Drafts

[draft-soilandreyes-arcp](https://tools.ietf.org/html/draft-soilandreyes-arcp-03)
and
[draft-shur-pack-uri-scheme](https://tools.ietf.org/html/draft-shur-pack-uri-scheme-05)
propose schemes to address path-named components of other packaging formats.
`pack:` proposes an encoding similar to the [`package:` scheme
here](#proposal-a-package-scheme) and could probably be extended to support
URL-named components.

### Referrer computation variants

When the source and target of a navigation aren't the same bundle, it would be
reasonable to omit all information about the bundle subresource from the
`Referer` header, instead of only omitting that information when the referrer
policy omits the bundle's path. This would be simpler, but would strip analytics
information that sites are likely to find as useful as the path information we
allow in other cases.

### Origin serialization variants

#### Allow more origin for CORS and websockets

For non-bundled resources, cors-mode and websocket requests send an Origin
header that's not modified by the referrer policy. It would be possible to send
something like `package:bundle-origin` in those cases even if the bundle's
referrer policy is stricter.

#### Omit all paths

In the other direction, we could cap the information in the `Origin` header to
`package:bundle-origin`, even if the referrer policy would allow more. We would
probably still have to send the whole origin for same-bundle requests.

## Stakeholder Feedback / Support / Opposition

* W3CTAG: No signals
* Browsers:
  * Safari: No signals
  * Firefox: [Concerned about the impact on
    UI](https://github.com/mozilla/standards-positions/issues/264#issuecomment-613976497)
  * Samsung: No signals
  * UC: No signals
  * Opera: No signals
  * Edge: No signals
- Web Developers : No signals

## References & acknowledgements

Many thanks for valuable feedback and advice from:

- Kinuko Yasuda
- Larry Masinter
- Mallory Knodel
- Martin Thomson
- Mike West
- Ryan Sleevi
- The participants in the IETF WPACK Working Group
- Tsuyoshi Horo
