# Signed Exchange subresource substitution

## Introduction

Users want to see the result of their clicks as fast as possible. This goal
benefits from letting a site tell the UA to
[`prenavigate`](https://github.com/w3c/resource-hints/issues/82#issuecomment-532072736)
to the particular outbound link(s) it thinks the user is most likely to click
on. However, naively prenavigating to a cross-origin link
[leaks that user visited](https://wicg.github.io/webpackage/draft-yasskin-wpack-use-cases.html#name-privacy-preserving-prefetch)
the referring page, which the referrer shouldn't do before the user has clicked.
The referrer can safely prenavigate to a referrer-origin signed exchange for the
top-level HTML of that link, but the UA still can't prefetch that link's
subresources without leaking the same information about the user.

We want the referrer to be able to identify that a particular subresource of a
prenavigated link is available as a signed exchange served by their own
organization. The
[`Link: <subresource.sxg>; rel="alternate", type="application/signed-exchange"; anchor="subresource"` header](https://html.spec.whatwg.org/multipage/links.html#rel-alternate)
is already defined to identify such alternate forms, where the
[`anchor` parameter](https://tools.ietf.org/html/rfc8288#section-3.2) states
that the alternate form is for a resource other than the one the `Link` header
is attached to.

Arbitrarily replacing a target link's subresources is unsafe for several
reasons, so we propose that the target link opt into particular replacements by
including a link with `rel=allowed-alt-sxg`.

## Use Cases

While a user is browsing an aggregator site (https://feed.example), the
aggregator guesses that the user is likely to want to read a particular article
(https://publisher.example/article.html) and so inserts a prefetch link pointing
to a signed exchange version of that article.

```
<link rel="prefetch" href="https://feed.example/sxg.publisher.example/article.html.sxg">
```

When the UA prefetches the signed exchange (article.html.sxg), the aggregator
server includes a declaration that one of `article.html`'s subresources
(https://cdn.publisher.example/lib.js) is also available from the same
aggregator. The aggregator server expresses this by serving `article.html.sxg`
with a `Link` header identifying the subresource's alternate form:

```
Link: <https://feed.example/sxg.publisher.example/lib.js.sxg>;
        rel="alternate";
        type="application/signed-exchange;v=b3";
        anchor="https://cdn.publisher.example/lib.js"
```

To prevent an attacker from loading an incompatible version of the subresource,
the resource _inside_ the signed exchange has to identify the exact version of
the replacement signed exchange using a `Link: ... rel="allowed-alt-sxg"` with
the hash of the signed headers (which themselves include a hash of the content).

To prevent a tracker from conveying a user ID in their choice of which
subresources to prefetch, the inner resource has to preload the same
subresources that the aggregator prefetches.
And the inner response of the main resource signed exchange (article.html.sxg)
has a preload header and an allowed-alt-sxg header:

```
Link: <https://cdn.publisher.example/lib.js>;
        rel="preload";
        as="script"
Link: <https://cdn.publisher.example/lib.js>;
        rel="allowed-alt-sxg";
        header-integrity="sha256-XXXXXX"
```

The UA recursively prefetches lib.js.sxg.

If the user navigates to the expected article, both the main resource of the
article and the script subresource are loaded from the prefetched signed
exchanges.

# Proposal

- While prenavigating an HTML resource in signed exchange format:
    1. When the UA detects a "preload" link HTTP header in the inner response,
    check whether a matching “allowed-alt-sxg” link HTTP header in the inner
    response exists or not. (Note that multiple `allowed-alt-sxg` links can be
    present for the same preload if they include `variants` and `variant-key`
    attributes. In that case, the UA uses the algorithm written in
    [HTTP Representation Variants](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html)
    spec to find the matching header.)
    1. If an `allowed-alt-sxg` link exists, check whether the signed exchange
    was served with a matching “alternate” link HTTP header.
    1. If the outer signed exchange did identify an alternate version of the
    subresource, prefetch the subresource signed exchange instead of prefetching
    the original resource URL.
    1. If the UA successfully prefetches the signed exchange (including the
    merkle integrity check of the body), it stores the parsedExchange which is
    the result of
    [parsing a signed exchange](https://wicg.github.io/webpackage/loading.html#parsing-a-signed-exchange)
    (= inner request URL and inner response) to a new cache mechanism which is
    attached to the Document.
- While
[navigating across documents](https://html.spec.whatwg.org/multipage/browsing-the-web.html#navigating-across-documents),
the UA copies the parsedExchanges in the cache of the source document to the
target document except for the one that serves the navigation itself.
    - This is intended to provide a way to transfer the cached signed exchange
    across origins even if the UA is using origin isolated HTTPCache mechanism.
    (Note that [header integrity check](#header-integrity-of-signed-exchange)
    prohibits the distributor from passing a tracking ID to the publisher.)
- The navigated-to document has a set of preloads for which it uses the
allowed-alt-sxg link relation to declare that they can be served by signed
exchanges. The UA either serves all of them from SXGs prefetched by the previous
page, or none of them. So while processing
[preload](https://html.spec.whatwg.org/multipage/links.html#link-type-preload)
link HTTP headers (eg: Link: <https://cdn.publisher.example/lib.js>;
rel="preload"; as="script"):
    1. The UA checks whether matching "allowed-alt-sxg" link HTTP header
    (`Link: <https://cdn.publisher.example/lib.js>;rel="allowed-alt-sxg";header-integrity="sha256-.."`)
    exists or not. (Note that if the allowed-alt-sxg link HTTP header has
    variants and variant-key attributes, the UA must execute the algorithm
    written in
    [HTTP Representation Variants spec](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html)
    to find the matching header.)
    1. The UA checks whether all preload links which have matching
    allowed-alt-sxg link header have matching (url and header-integrity)
    parsedExchange which was copied from the referrer page.
    1. If the check passes, in "linked resource fetch setup steps" for all
    preload links which have matching allowed-alt-sxg link header the UA sets
    request's stashed exchange to the matching parsedExchange so the resource
    will be loaded from the cached signed exchange.

# Detailed design discussion
## Header integrity of signed exchange
We propose a new `rel="allowed-alt-sxg"` link header with a `header-integrity`
parameter. Publishers can declare that the subresource can be served by a signed
exchange, using this link header in the inner response of the main resource
signed exchange.

```
Link: <https://cdn.publisher.example/lib.js>;
        rel="allowed-alt-sxg";
        header-integrity="sha256-XXXXXX"
```

This `header-integrity` parameter is the SHA256 hash value of the
*signedHeaders* value from the
[application/signed-exchange format](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#application-signed-exchange)
for integrity checking. This signedHeaders is *"the canonical serialization of
the CBOR representation of the response headers of the exchange represented by
the application/signed-exchange resource, excluding the Signature header
field"*. So this value doesn’t change even if the publisher signs the content
again or changes the signing key. This header-integrity value also guarantees
the content body hasn't changed, because the signed headers are required to
include a `Digest` header with a hash of the content body.

The UA needs to check that this value of the prefetched subresource signed
exchange is same as the header-integrity attribute of allowed-alt-sxg link
header in the response from the publisher. This is intended to prevent the
distributor from encoding a tracking ID into the subresource signed exchange.

We can’t use the
[SRI’s integrity](https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity)
for this purpose, because SRI’s integrity only covers the content body and not
any of the headers. So if the UA use the SRI’s integrity value in
allowed-alt-sxg link header, we can use the subresource signed exchanges to
track the users by changing content-type and detecting the image loading
failure.

## Multiple subresource signed exchanges
If there are multiple subresource preloads that are provided by prefetches on
the previous page in signed exchange format (example: script.js.sxg and
image.jpg.sxg), the UA must check that there is no error in the any of the
signed exchanges (eg: sig matching, URL matching, Merkle Integrity error) before
processing the content of the signed exchanges. This is intended to prevent the
distributor from encoding a tracking ID into the set of subresources it
prefetches. And to prevent the distributor from selecting a version of the
subresource that isn't compatible with the selected version of the main
resource, which might introduce a security hole.


## Can’t we have a global cache of parsedExchanges?
If we can have a global cache of parsedExchanges, we can use the all signed
exchanges which have previously prefetched, even if they are not prefetched by
the referrer page. This can improve the performance. But this introduces privacy
issues such as
[Timing Leaks](https://hillbrad.github.io/sri-addressable-caching/sri-addressable-caching.html#cache_timing)
and [Deterministic History Leaks](https://hillbrad.github.io/sri-addressable-caching/sri-addressable-caching.html#history_leaks)
described in [the document about Subresource Integrity Addressable Caching](https://hillbrad.github.io/sri-addressable-caching/sri-addressable-caching.html).

To avoid these privacy issues, we introduced the limitation that we can only use
the signed exchanges which were prefetched by the referrer page.

## Can’t we merge allowed-alt-sxg to preload header?
If we can declare the header-integrity value in the existing preload link HTTP
header, we don’t need to introduce a new "allowed-alt-sxg" link HTTP header.
However, the [imagesrcset attribute](https://github.com/w3c/preload/issues/120)
allows a single preload link to declare multiple different target URLs, and it's
difficult to embed a `header-integrity` value for each of those URLs into the
existing syntax. Instead, we use a separate link that gives a hash of the
expected content for each of the possible URLs, while the preload tag continues
to select which of the URLs is actually used.

For Example:
```
Link: <https://publisher.example/wide.jpg>;
      rel="preload";
      as="image";
      imagesrcset="https://publisher.example/wide.jpg 640w,
                   https://publisher.example/narrow.jpg 320w";
      imagesizes="(max-width: 640px) 100vw, 640px"
Link: <https://publisher.example/wide.jpg>;
      rel="allowed-alt-sxg";
      header-integrity="sha256-XXX"
Link: <https://publisher.example/narrow.jpg>;
      rel="allowed-alt-sxg";
      header-integrity="sha256-YYY"
```

## Lifetime of the entry in SignedExchangeCache
The UA must check both the
[signature expire time](https://wicg.github.io/webpackage/loading.html#exchange-signature-expiration-time)
of the signed exchange and Cache-Control header of the outer response. The UA
may discard the entry if it is expired.

## Content negotiation using Variants and Variant-Key

The **allowed-alt-sxg** link headers can have
[variants and variant-key](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html)
attributes to support content negotiation (eg:
[WebP support](https://developers.google.com/speed/webp/faq#server-side_content_negotiation_via_accept_headers)).
```
Link: <https://publisher.example/image>;
      rel="allowed-alt-sxg";
      variants-05="accept;image/jpeg;image/webp";
      variant-key-05="image/jpeg";
      header-integrity="sha256-AAA"
Link: <https://publisher.example/image>;
      rel="allowed-alt-sxg";
      variants-05="accept;image/jpeg;image/webp";
      variant-key-05="image/webp";
      header-integrity="sha256-BBB"
Link: <https://publisher.example/image>; rel=preload; as=image;
```
If a UA supports WebP format, the UA can use the signed exchange which
header-integrity is sha256-BBB if available in the cache. Otherwise the UA can
use the signed exchange which header-integrity is sha256-AAA if available in the
cache.

When the sha256-AAA signed exchange exists in the cache but the sha256-BBB
signed exchange doesn’t exists, the UA which supports WebP format MUST ignore
the sha256-AAA signed exchange and fetch the original URL. Otherwise this can be
used for sending tracking ID.

The **alternate** link also can have
[variants and variant-key](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html)
attributes to support content negotiation for recursive prefetch.

- In **outer** HTTP response of article.html.sxg from feed.example:
    ```
    Link: <https://feed.example/publisher.example/image_jpeg.sxg>;
          rel="alternate";
          type="application/signed-exchange;v=b3";
          variants-05="accept;image/jpeg;image/webp";
          variant-key-05="image/jpeg";
          anchor="https://publisher.example/image";
    Link: <https://feed.example/publisher.example/image_webp.sxg>;
          rel="alternate";
          type="application/signed-exchange;v=b3";
          variants-05="accept;image/jpeg;image/webp";
          variant-key-05="image/webp";
          anchor="https://publisher.example/image";
    ```
- In **inner** response header of article.html.sxg:
    ```
    Link: <https://publisher.example/image>;
          rel="allowed-alt-sxg";
          variants-05="accept;image/jpeg;image/webp";
          variant-key-05="image/jpeg";
          header-integrity="sha256-AAA"
    Link: <https://publisher.example/image>;
          rel="allowed-alt-sxg";
          variants-05="accept;image/jpeg;image/webp";
          variant-key-05="image/webp";
          header-integrity="sha256-BBB"
    Link: <https://publisher.example/image>; rel=preload; as=image;
    ```
If a UA supports WebP, the UA must prefetch **image_webp.sxg** holding a WebP
image. Otherwise the UA must prefetch **image_jpeg.sxg** holding a JPEG image.

## Security and Privacy Considerations
- The publishers can know whether the referrer page has prefetched the signed
exchange subresources or not by checking the resource timing information. But
this only exposes 1 bit information (= yes or no) because UAs can use the cached
signed exchange only if the required signed exchanges are all available.
- To prevent tracking (user ID transfer), if the aggregator failed to prefetch a
subresource that the main resource preloads, the UA must drop all of the
subresource prefetches. If the aggregator prefetches a superset of the preloaded
subresources, the UA must drop the ones that weren't preloaded.
- The UA can use the prefetched signed exchange subresources, only when they
were prefetched in the rimmediate eferrer page. This is intended to avoid
leaking the prefetching state to succeeding pages.
- The UA check the header integrity value, so the distributor of the subresource
signed exchange can’t inject arbitrary resources to the publisher’s page. This
prevents distributors from sending tracking IDs to the publisher’s page.
- The UA needs to check that the signed
[request URL](https://wicg.github.io/webpackage/loading.html#exchange-request-url)
matches the preload link and not just that the header-integrity value matches,
since the header-integrity hash doesn't cover the request URL.
- The UA must fetch the signed exchange subresource
(https://cdn.feed.example/cdn.publisher.example/lib.js.sxg) while processing the
prefetch link element  (`<link rel="prefetch"
href="https://cdn.feed.example/cdn.publisher.example/lib.js.sxg" as="script">`)
even if the HTTP cache already contains the original subresource
(http://cdn.publisher.example/lib.js). Otherwise it leaks the state of
publisher’s site to the distributor of the signed exchange.
- If a replaced subresource prefetch hasn't completed by the time the UA would
start fetching it in the course of loading the next page, the UA must cancel
that prefetch and fetch the resource from its original URL. This prevents the
distributor from interfering the publisher’s page. (Eg. intentionally block or
delay the subresource loading.)

## [Self-Review Questionnaire: Security and Privacy](https://www.w3.org/TR/security-privacy-questionnaire/)
1. What information might this feature expose to Web sites or other parties, and
for what purposes is that exposure necessary?
    - This feature exposes the 1 bit information "the referrer page has
    prefetched the signed exchange subresources or not" to the publisher.
1. Is this specification exposing the minimum amount of information necessary to
power the feature?
    - Yes. This proposal has limitations such as "all subresource SXG must be
    finished prefetching, otherwise ignored", "subresource SXG must be prefetched
    even if the original subresource exists in HTTPCache". Thanks to these
    limitations, this feature exposes only 1 bit information to the publisher.
1. How does this specification deal with personal information or
personally-identifiable information or information derived thereof?
    - Signed Exchange should not include personal information.
    - Any personal information that was incorrectly included in a signed
    exchange would be the information of the aggregator that fetched the SXG,
    and not the end user.
    - The use of `<link rel=alternate>` to identify the SXG for the current page
    could inform the UA to omit credentials in fetching that SXG, which would
    prevent any personal information from being accidentally included.
1. How does this specification deal with sensitive information?
    - Signed Exchange should not include sensitive information.
    - The state of the cache for another origin is potentially sensitive, and
    this specification avoids exposing it by making the decision to fetch an
    alternative not depend on the presence or absence of the subresource in its
    cache.
1. Does this specification introduce new state for an origin that persists
across browsing sessions?
    - Prefetched resources, including signed exchanges, are stored to the HTTP
    cache as normal, but the association of a signed exchange with its contained
    resource is not persisted. Right now, the contained resource is not
    independently stored in the HTTP cache, although that decision may be
    revisited.
1. What information from the underlying platform, e.g. configuration data, is
exposed by this specification to an origin?
    - The use of Variants exposes the UA's content negotiation preferences to
    the aggregator's origin, but that's already exposed by the UA's Accept
    headers.
1. Does this specification allow an origin access to sensors on a user’s device
    - No
1. What data does this specification expose to an origin? Please also document
what data is identical to data exposed by other features, in the same or
different contexts.
    - The existence of the alternative signed exchange in HTTP Cache is exposed.
    - But this is the same as the existing behavior when directly prefetching the
    signed exchange using `<link rel=prelfetch>`.
1. Does this specification enable new script execution/loading mechanisms?
    - This specification introduces a new script loading path, from prefetched
    signed exchange. The existing security checks such as CSP/CORP must be
    applied as if the script is loaded from the original URL.
1. Does this specification allow an origin to access other devices?
    - No
1. Does this specification allow an origin some measure of control over a user
agent’s native UI?
    - No
1. What temporary identifiers might this this specification create or expose to
the web?
    - None
1. How does this specification distinguish between behavior in first-party and
third-party contexts?
    - This feature treats all entities (the distributor of signed exchange, the
    publisher site, the origin of subresource URL) as third-party origins.
    - To avoid leaking the state of publisher’s site to the distributor of the
    signed exchange, the UA must fetch the alternate signed exchange subresource
    even if there is the original subresource in the HTTPCache.
    - To avoid leaking user-specific data in the distributor of signed exchange,
    the prefetch request must not contain credentials. This is covered by the
    "Prefetch and double-key caching"
    [issue](https://github.com/w3c/resource-hints/issues/82).
    - The origin of subresource URL could be different from the origin of
    publisher site. The cross origin security checks (CORS/CORB/CORP/..) must be
    executed while reading the response from the cached signed exchanges.
1. How does this specification work in the context of a user agent’s Private
Browsing or "incognito" mode?
    - No difference while the user is browsing sites in Private mode.
    - If the user opens a link in private mode while browsing in normal mode
    (eg: "Open link in incognito window"), the prefetched signed exchanges must
    be ignored.
1. Does this specification have a "Security Considerations" and "Privacy
Considerations" section?
    - Yes
1. Does this specification allow downgrading default security characteristics?
    - There's active discussion about how signed exchanges are a downgrade
    compared to TLS, and this particular specification allows recursive use of
    signed exchanges.

