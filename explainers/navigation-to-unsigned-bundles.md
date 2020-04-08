# Explainer: Navigation to Unsigned Web Bundles<br>(a.k.a. Bundled HTTP Exchanges)

Last updated: Apr 07, 2020

Participate at https://github.com/WICG/webpackage and
https://datatracker.ietf.org/wg/wpack/about/.

<!-- TOC depthTo:3 -->

- [Goals](#goals)
- [Proposal](#proposal)
  - [Relevant structure of a bundle](#relevant-structure-of-a-bundle)
  - [Process of loading a bundle](#process-of-loading-a-bundle)
  - [Loading a trusted unsigned bundle.](#loading-a-trusted-unsigned-bundle)
  - [Loading an untrusted bundle](#loading-an-untrusted-bundle)
  - [Subsequent loads with an attached bundle](#subsequent-loads-with-an-attached-bundle)
  - [Service Workers](#service-workers)
  - [URLs for bundle components](#urls-for-bundle-components)
- [Open design questions](#open-design-questions)
  - [Network access](#network-access)
  - [Non-origin-trusted signatures](#non-origin-trusted-signatures)
- [Security and privacy considerations](#security-and-privacy-considerations)
  - [Security/Privacy Questionaire](#securityprivacy-questionaire)
- [Considered alternatives](#considered-alternatives)
  - [Alternate formats considered](#alternate-formats-considered)
  - [Alternate URL schemes considered](#alternate-url-schemes-considered)
- [Stakeholder feedback](#stakeholder-feedback)
- [Acknowledgements](#acknowledgements)

<!-- /TOC -->

## Goals

1. Provide a way for users to share web content to other users who may be
   offline when they try to visit the content. This should work with their
   existing file-sharing apps, like [SHAREit](https://www.ushareit.com/),
   [Xender](http://www.xender.com/), or [Google
   Files](https://files.google.com/), which present the unit of sharing as a
   single file.

   <details>
   <summary>

   ![A drawing of a person downloading a site, bundling its content, and passing it to a friend without their own connection to the internet.](https://www.plantuml.com/plantuml/svg/NOonQiD044Jx_Oh91zWVO8nm4KAgj6dIdAILlN2tA_RsS17oxpbZ156wOURDkzH87grachAr6PyyLWd6Dm6BPCQQhdoyHSbR8UNHh7gb7r2QmXolKiDbR8zyFh-tadGOQ5CT3iEEE47DIyeOtUvLkYXvoD9Lk3ylnx7fd9d-lhfaltRFCrJwDtJqpOLr1ga58so5BLjtmeTXCbMUGao_D0nnOuW6ktAyqALZhUHV)

   </summary>

   ```plantuml
   @startuml
   cloud Website {
     file page.html
     file image.png
   }
   actor Distributor <<Human>>
   Website --> Distributor : normal browsing
   artifact website.bundle
   Distributor -> website.bundle : bundles
   website.bundle -> Friend
   note top of Friend : No connection\nto the internet!
   @enduml
   ```

   </details>

1. Provide a way for sites like https://www.isocfoundation.org/ to package their
   content so communities can fetch it once over a perhaps-expensive, slow, or
   otherwise sometimes-unavailable link and then share it internally without
   each recipient needing to be online when they get it.

   <details>
   <summary>

   ![A drawing of a website bundling its own content, which it provides over an expensive, slow, or not-always-available link to one person, who passes it to their community over cheaper or more-reliable links.](https://www.plantuml.com/plantuml/svg/ZOv1QiCm44NtFSLSm0bjToMOB4hf3RhfHfP_aY7Io4YZkb1wzsemc1GIo4Q8_zz_pBweorfZU20Y7r8TwGD3OGNzM4Hqu02Qt16RangsPXmjdEIuP4t31-ULvcM_6QgC0Kkvxgdh-gl4Qhj1_DhJz2dJAnVDF5JxxtRlDJhfUwl_hwXvBj4NmlS4AVo5RGbftfOKeHnHkY4aVyRuAIoAB53oIGHUEOc9BpLstbjcoFZObFu4Dv50vvJFjz6d-z7dQ-Y-5JM6Fm00)

   </summary>

   ```plantuml
   @startuml
   cloud Website {
     file page.html
     file image.png
     artifact website.bundle
     page.html --> website.bundle
     image.png --> website.bundle
   }
   actor Distributor <<Human>>
   website.bundle -> Distributor : expensive/slow/sometimes-blocked\ninternet connection
   Distributor --> Friend1 : cheap network
   Distributor --> Friend2 : cheap network
   Distributor --> Friend3 : cheap network
   @enduml
   ```

   </details>

1. Provide a way for sites like https://archive.org/ to distribute archived
   content without having to rewrite all its internal links.

   <details>
   <summary>

   ![A drawing of an archive bundling the contents of several other websites, and then passing those on to several users.](https://www.plantuml.com/plantuml/svg/bP1BZeCm38RtSmfV02Ji_OWvnAXh7WOYXOT2qYwgths1jeh00QbBF_nz-Wq0-MmBOrslVtnHwT7LSE5oLfOpk2yzW4PfXgbeEKixwnT3K_LhTnhQfVaG21G8Z2Bm6442GL44HH1_fkhKbJy4drCrHMNXzWwObcweDR-c8I0aoMzy9-Gzt14M51OK5fGMt5lmr4B2Gi92qaB18dVMJth7QE1_PfDjIzoMvClzGnPmE0qvlXZYmL0uwUoIefSv3xNhzHC0)

   </summary>

   ```plantuml
   @startuml
   cloud Website1 {
     file page1.html
     file image1.png
   }
   cloud Website2 {
     file page2.html
     file image2.png
   }
   cloud Website3 {
     file page3.html
     file image3.png
   }
   cloud Archive {
     artifact website1.bundle
     page1.html --> website1.bundle
     image1.png --> website1.bundle
     artifact website2.bundle
     page2.html --> website2.bundle
     image2.png --> website2.bundle
     artifact website3.bundle
     page3.html --> website3.bundle
     image3.png --> website3.bundle
   }
   actor User1
   actor User2
   actor User3
   website1.bundle --> User1
   website2.bundle --> User1
   website2.bundle --> User2
   website2.bundle --> User3
   website3.bundle --> User2
   website3.bundle --> User3
   @enduml
   ```

   </details>


## Proposal

We propose to use the [Web Bundle
format](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html),
without origin-trusted signatures, to give users a format they can create from
any website they're visiting, share it using existing peer-to-peer sharing apps,
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

> An origin could use an "untrusted" bundle with same-origin resources to create
> a kind of [suborigin](https://w3c.github.io/webappsec-suborigins/).

The diagram below shows how signatures, flags, and primary URL of a WebBundle determine the mode in which the bundle is loaded.

![@startuml
:Navigation to an application/webbundle resource;
:Parse primary URL and control flags;
if (Signed?) then (yes)
  :Load as a signed bundle
  (Out of the scope of
  this document);
  stop
else (no)
  if (The trusted flag is set?) then (yes)
    if (The primary URL is\nwithin the unsigned\nbundle scope?) then (yes)
      :Load as a trusted bundle;
      stop
    else (no)
      :Redirect to the primary URL;
      stop
    endif
  else (no)
    :Load as an
    untrusted bundle;
    stop
  endif
endif
@enduml](http://www.plantuml.com/plantuml/svg/POyzRWCX48LxJl7ATPNUMyG7i9B8IJet66OT97POCKCitpuOP2KRDy2Wz_FDJjHcBNCqPljYlyFPQaWCJR0CkomnkFRpTA7JgR2FX4oIIdOqcksRpK9OSfXjlkBpiAyk3vTOSugOeZtBQCA4uJsScVpp1lf5ZE5AiZ70Tf-iXnLOI1EWLnXWU2sADDtq49SMgeD17OF09rTcOjsC1X1DYw4eX87JBVHMzr5Tceie-KQ1wXBI__rtyNg584U-XDh4hRrmPpjoX-iuZr6hTUxbtJ8sGMTjpp-ytNaW7p8vXIReckVHp3vCPXMoAkSs5tvW-0tf4VqqktgNEVu0)

The following sections describe how bundles are loaded in each mode.

### Loading a trusted unsigned bundle.

The browser selects an **unsigned bundle scope** that's based on the bundle's
URL in the same way the [service worker's scope
restriction](https://w3c.github.io/ServiceWorker/#path-restriction) is based on
the URL of the service worker script. If the bundle is served with the
[`Service-Worker-Allowed`
header](https://w3c.github.io/ServiceWorker/#service-worker-allowed), that sets
the unsigned bundle scope to the value of that header. (We are also considering
using a differently-named header that applies only to bundles.)

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

### Subsequent loads with an attached bundle

The bundle that's attached to the request above will eventually get attached to
the environment settings object created for the response, from where subsequent
fetches can use it.

For trusted bundles, any fetch, both navigations and subresources, within the
bundle's scope checks the bundle for a response before going to the network. If
a navigation fetch finds a response inside the bundle, the bundle is propagated
to the environment settings object created for that navigation.

For untrusted bundles, all fetches should check inside the bundle before going
to the network. [We need more discussion about whether subresource requests that
aren't inside the bundle should be able to touch the network at
all.](#network-access)

### Service Workers

We plan to, but haven't yet, defined an API to expose trusted bundles to that
origin's service worker. This API should allow the service worker to fill its
cache with the contents of the bundle.

### URLs for bundle components

Each item of the bundle is addressible using a URL, with the new scheme
`package:`. (See [below](#alternate-URL-schemes-considered) for some details of
this choice.) This scheme identifies both the URL of the bundle itself (e.g.
`https://distributor.example/package.wbn?q=query`) and the claimed URL inside
the bundle (e.g. `https://publisher.example/page.html?q=query`). These are
encoded to minimize the changes needed to the [algorithm for computing an origin
from a URL](https://url.spec.whatwg.org/#concept-url-origin), by replacing `:`
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

### Non-origin-trusted signatures

While this explainer doesn't propose any particular ways to use any signatures
in the bundle, there are several options:

* Support [signature-based SRI](https://github.com/mikewest/signature-based-sri).
* Display a real-world publisher for the content, which is especially useful for
  books.
* Allow the application to [pin updates to a particular signing
  key](https://wicg.github.io/webpackage/draft-yasskin-wpack-use-cases.html#name-protecting-users-from-a-com).
  (Which has risks.)
* Grant access to some particularly-powerful API because the signature vouches
  that a trusted auditor checked the package for malicious behavior.

## Security and privacy considerations

* Unsigned bundles served by one origin can't be trusted to provide content for
  another origin. The [URL design](#urls-for-bundle-components) and the division
  between trusted and untrusted bundles are designed to keep separate origins
  safely separate, but this is more than a trivial amount of complexity, and
  browsers will need to be careful to actually enforce the boundaries.

* It's straightforward for someone serving an unsigned bundle to include a
  unique ID in the resources within that bundle. If the bundle can then make
  [network requests](#network-access), the author can determine a rough number
  and set of IP addresses who received a copy of the same download. If the
  author can additionally convince the user to log in or enter other identifying
  information, they can identify the set of users who are connected. This is
  detectable if users have a way to inspect two bundles that are supposed to
  hold the same content, but since the whole point of sharing bundles is to
  reduce redundant transfers, it's unlikely many users  will actually check
  this.

  Sites currently gather information on this kind of link sharing by annotating
  their URLs with unique or semi-unique IDs. These can usually be removed by
  removing query parameters, but if a significant number of users cleaned their
  URLs, the tracking could move to path segments.

* The path of a locally-saved bundle may include private information like a
  username. As [Loading an untrusted bundle](#loading-an-untrusted-bundle)
  suggests, the URL of the bundle itself needs to be hidden from web APIs to
  avoid exposing this.

### Security/Privacy Questionaire

This section contains answers to the [W3C TAG Security and Privacy
Questionnaire](https://w3ctag.github.io/security-questionnaire/).

#### 1. What information might this feature expose to Web sites or other parties, and for what purposes is that exposure necessary?

If we allow [network access](#network-access) from untrusted bundles, they could
be abused to [identify the set of people who have copies of the same
download](#security-and-privacy-considerations).

We're blocking access to the `package:` URL because for local bundles that would
include the path of the bundle.

#### 2. Is this specification exposing the minimum amount of information necessary to power the feature?

Yes.

#### 3. How does this specification deal with personal information or personally-identifiable information or information derived thereof?

This feature blocks access to local paths, which might otherwise be exposed by the `package:` scheme.

#### 4. How does this specification deal with sensitive information?

See #3.

#### 5. Does this specification introduce a new state for an origin that persists across browsing sessions?

This proposal introduces a new kind of origin with state that persists across
browsing sessions. Specifically, the [`package:`
scheme](#urls-for-bundle-components) defines an origin based on the location or
the URL of the bundle itself, and the claimed URL inside the bundle.

#### 6. What information from the underlying platform, e.g. configuration data, is exposed by this specification to an origin?

None.

#### 7. Does this specification allow an origin access to sensors on a user’s device

No.

#### 8. What data does this specification expose to an origin? Please also document what data is identical to data exposed by other features, in the same or different contexts.

Network requests (e.g. fetch or iframe) from the unsigned bundle could expose IP
address of the user, in the same way as regular navigation does.  See also this
[section](#network-access) about how network requests from untrusted bundles
should be performed or not.

Navigation and subresource requests within the [unsigned bundle
scope](#loading-a-trusted-unsigned-bundle) of a "trusted" bundle should expose
strictly less information than loading each contained resource directly from the
server, since it stops exposing the time that resource was needed.

#### 9. Does this specification enable new script execution/loading mechanisms?

No.

#### 10. Does this specification allow an origin to access other devices?

No.

#### 11. Does this specification allow an origin some measure of control over a user agent’s native UI?

No.

#### 12. What temporary identifiers might this specification create or expose to the web?

This explainer avoids exposing the [new URL scheme for untrusted
bundles](#urls-for-bundle-components) to the web, as the bundle URL piece of the
authority can include private information including identifiers.

#### 13. How does this specification distinguish between behavior in first-party and third-party contexts?

For navigation, this specification itself doesn’t distinguish between behavior
in first-party (i.e. top-level navigation) and third-party (i.e. iframe or
nested navigation) and doesn't affect other constraints on navigations.

A first-party bundle (that is, one whose claimed URLs are same-origin with the
bundle itself) can be trusted, while a third-party one must be untrusted (as
this explainer doesn't cover signed bundles).

For subsequent subresource loading with an attached bundle, again there is no
particular difference from regular subresource loading, and origins between
different claimed URLs for each resource is distinguished as different
third-party origins.

#### 14. How does this specification work in the context of a user agent’s Private Browsing or "incognito" mode?

This specification doesn't interact with Private Browsing, although it would be
plausible to make Private Browsing affect the default state of the [Network
access](#network-access) open design question.

#### 15. Does this specification have a "Security Considerations" and "Privacy Considerations" section?

See the [security considerations in the format
specification](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#security),
[security and privacy considerations in this
explainer](#security-and-privacy-considerations), and the [open design questions
in this explainer](#open-design-questions).

#### 16. Does this specification allow downgrading default security characteristics?

Similarly to Service Workers, this specification allows a resource fetched from
one path to provide responses for another path within the same origin. For
Service Workers, that's the Service Worker script itself; here it's the Web
Bundle. Both are [constrained by default](#loading-a-trusted-unsigned-bundle) to
only override the subtree of paths rooted at their own directory, and someone
with control of the server's response headers can loosen the constraint using
the `Service-Worker-Allowed` response header (or possibly a differently-named
header for bundles).

#### 17. What should this questionnaire have asked?

## Considered alternatives

### Alternate formats considered

There are several existing ways to serialize a web page or website, but all of
them have shortcomings for the use cases we're focusing on:

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

MHTML would need an extension to support content negotiation between multiple
representations at the same URL.

Although this wouldn't be a good reason to avoid MHTML on its own, we can do
better nowadays than a text format with boundary strings.

#### Save as Web Archive

This is currently supported by Safari using the `Save As...` | `Web Archive` dialog.

A Web Archive is in the semi-documented binary plist format and doesn't support
random access or content negotiation either. Web Archives also currently allow
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
  * Firefox: [In progress](https://github.com/mozilla/standards-positions/issues/264)
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
