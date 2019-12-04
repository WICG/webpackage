# Signed Exchange subresource substitution

## Introduction

We want to introduce a new **rel=allowed-alt-sxg** link HTTP header which will be used along with rel=preload link HTTP header. By using this header, content publishers can declare that the UA can load the specific subresources from cached signed exchanges which were prefetched in the referrer page.

## Use Cases

### Privacy-preserving prefetching subresources

A user is browsing a news feed site (https://feed.example/). The user clicks a link to an article in another site (https://publisher.example/article.html). The article is using a large JS library (https://cdn.publisher.example/lib.js) and the user must wait for the JS file to be fetched from the server to read the article.

If the feed site knows that the article is depending on the JS file, the site can insert a link element (`<link rel="prefetch" href="https://cdn.publisher.example/lib.js" as="script">`) to prefetch the JS file while the user is browsing the feed site. But this is problematic from the privacy point of view. And also this doesn’t work if the UA is using [origin isolated HTTPCache mechanism](http://sirdarckcat.blogspot.com/2019/03/http-cache-cross-site-leaks.html).

Signed Exchange has solved this [privacy-preserving prefetching](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#private-prefetch) problem for main resources. If the publisher is providing the article in signed exchange format (article.html.sxg), the UA can prefetch the signed exchange from the feed site’s own server while the user is browsing the feed site. But there is no way to prefetch subresources in a privacy-preserving manner yet.

Our proposal can solve this problem:
1. The publisher provides the script file in signed exchange format (lib.js.sxg) along with its [header integrity value](#header-integrity-of-signed-exchange).
1. The UA prefetches the signed exchange of the script from the feed site’s own server while the user is browsing the feed site.
1. The HTTP response of the article from the publisher's server (https://publisher.example/article.html) has an allowed-alt-sxg link header to declare that the UA can load the script from the prefetched signed exchange.
   ```
   Link: <https://cdn.publisher.example/lib.js>;
         rel="allowed-alt-sxg";
         header-integrity="sha256-..."
   ```
   So, the UA can load the JS file from the prefetched signed exchange. If the publisher is providing the article in signed exchange format (article.html.sxg), the allowed-alt-sxg link header is in the inner response headers of the signed exchange.

### Reading article while offline
If the publisher is providing the main resource and necessary subresources in signed exchange format, the user of the news feed site can read the article even while offline. This can be done using Service Worker.
1. The Service Worker of the news feed site fetches the main resource signed exchange and subresources signed exchanges (maybe using [Background Fetch](https://wicg.github.io/background-fetch/)) while online, and stores them to the CacheStorage of the feed site origin.
1. While the user is browsing the feed site, the site inserts prefetch link elements to prefetch the signed exchanges. The service worker responds to the fetch events of the prefetch with the signed exchanges in the CacheStorage.
1. When the user clicks a link to the article’s signed exchange, the content and all necessary subresources are loaded from the cached signed exchange. So the user can read the article while offline.


# Proposal

1. Introduce SignedExchangeCache and PrefetchedSignedExchangeCache which are attached to a Document. SignedExchangeCache keeps the signed exchanges which are prefetched from the Document. PrefetchedSignedExchangeCache keeps the signed exchanges which were passed from the referrer Document which triggered the navigation to the current Document.
1. While processing "[prefetch](https://html.spec.whatwg.org/multipage/links.html#link-type-prefetch)" link (eg: `<link rel="prefetch" href="https://cdn.feed.example/cdn.publisher.example/lib.js.sxg" as="script">`):
   - If succeeded prefetching the signed exchange (including merkle integrity check of the body), stores the parsedExchange which is the result of [parsing a signed exchange](https://wicg.github.io/webpackage/loading.html#ref-for-parsing-a-signed-exchange) (= inner request URL and inner response) to the SignedExchangeCache.
1. While [navigating across documents](https://html.spec.whatwg.org/multipage/browsing-the-web.html#navigating-across-documents), copy the parsedExchanges in the SignedExchangeCache of the source document to the PrefetchedSignedExchangeCache of the target document. This is intended to provide a way to pass the cached signed exchange across origins even if the UA is using origin isolated HTTPCache mechanism.
1. While processing [preload](https://html.spec.whatwg.org/multipage/links.html#link-type-preload) link HTTP headers (eg: Link: <https://cdn.publisher.example/lib.js>; rel="preload"; as="script"):
   - Check whether matching "allowed-alt-sxg" link HTTP header (`Link: <https://cdn.publisher.example/lib.js>;rel="allowed-alt-sxg";header-integrity="sha256-.."`) exists or not. (Note that if the allowed-alt-sxg link HTTP header has variants and variant-key attributes, the UA must execute the algorithm written in [HTTP Representation Variants spec](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html) to find the matching header.)
   - Check whether all preload links which have matching allowed-alt-sxg link header have matching (url and header-integrity) parsedExchange in PrefetchedSignedExchangeCache. If the check passes, set request's stashed exchange to the parsedExchange so the resource will be loaded from the cached signed exchange.
   - After processing the link HTTP headers, clears the PrefetchedSignedExchangeCache.

# Detailed design discussion
## Header integrity of signed exchange
We use the SHA256 hash value of *signedHeaders* in [application/signed-exchange format](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#application-signed-exchange) for integrity checking. This signedHeaders is *"the canonical serialization of the CBOR representation of the response headers of the exchange represented by the application/signed-exchange resource, excluding the Signature header field"*. So this value doesn’t change even if the publisher signs the content again or changes the signing key. This header-integrity value can verify the integrity of both the response header and the content body, because signed exchange’s response header must have digest header.

The UA needs to check that this value of the prefetched subresource signed exchange is same as the header-integrity attribute of allowed-alt-sxg link header in the response from the publisher. This is intended to prevent the distributor from encoding a tracking ID into the subresource signed exchange.

We can’t use the [SRI’s integrity](https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity) for this purpose, because SRI’s integrity can be used only for verifying the integrity of the content body. So if the UA use the SRI’s integrity value in allowed-alt-sxg link header, we can use the subresource signed exchanges to track the users by changing content-type and detecting the image loading failure.

## Multiple subresource signed exchanges
If there are multiple matching subresource signed exchanges (example: script.js.sxg and image.jpg.sxg), the UA must check that there is no error in the all signed exchanges (eg: sig matching, URL matching, Merkle Integrity error) before processing the content of the signed exchanges. This is intended to prevent the distributor from encoding a tracking ID into the set of subresources it prefetches. this means that the UA can use the subresource signed exchanges only when they are defined in the header.

## Can’t we have a global SignedExchangeCache?
If we can have a global SignedExchangeCache, we can use the all signed exchanges which have previously prefetched, even if they are not prefetched by the referrer page. This can
improve the performance. But this introduces privacy issues such as [Timing Leaks](https://hillbrad.github.io/sri-addressable-caching/sri-addressable-caching.html#cache_timing) and [Deterministic History Leaks](https://hillbrad.github.io/sri-addressable-caching/sri-addressable-caching.html#history_leaks) described in [the document about Subresource Integrity Addressable Caching](https://hillbrad.github.io/sri-addressable-caching/sri-addressable-caching.html).

To avoid these privacy issues, we introduced the limitation that we can only use the signed exchanges which were prefetched by the referrer page.

## Can’t we merge allowed-alt-sxg to preload header?
If we can declare the header-integrity value in the existing preload link HTTP header, we don’t need to introduce a new "allowed-alt-sxg" link HTTP header. But it becomes complicated when supporting the [imagesrcset attribute of preload link](https://github.com/w3c/preload/issues/120).

For Example:
```
Link: <https://publisher.example/wide.jpg>;
      rel="preload";
      as="image";
      imagesrcset="https://publisher.example/wide.jpg 640w,
                   https://publisher.example/narrow.jpg 320w";
      imagesizes="(max-width: 640px) 100vw, 640px"
```
In this case, we want to declare that both wide.jpg and narrow.jpg can be loaded from signed exchanges and the header-integrity is sha256-XXX and sha256-YYY. But how to express in this preload link header?

In our proposal, we can have two allowed-alt-sxg link headers.
```
Link: <https://publisher.example/wide.jpg>;
      rel="allowed-alt-sxg";
      header-integrity="sha256-XXX"
Link: <https://publisher.example/narrow.jpg>;
      rel="allowed-alt-sxg";
      header-integrity="sha256-YYY"
```

## Lifetime of the entry in SignedExchangeCache
The UA must check both the [signature expire time](https://wicg.github.io/webpackage/loading.html#exchange-signature-expiration-time) of the signed exchange and Cache-Control header of the outer response. The UA may discard the entry if it is expired.

## Content negotiation using Variants and Variant-Key
The **allowed-alt-sxg** link headers can have [variants and variant-key](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html) attributes to support content negotiation (eg: [WebP support](https://developers.google.com/speed/webp/faq#server-side_content_negotiation_via_accept_headers)).
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
If a UA supports WebP format, the UA can use the signed exchange which header-integrity is sha256-BBB if available in the cache. Otherwise the UA can use the signed exchange which header-integrity is sha256-AAA if available in the cache.

If the sha256-AAA signed exchange exists in the cache but the sha256-BBB signed exchange doesn’t exists, the UA which supports WebP format MUST ignore the sha256-AAA signed exchange and fetch the original URL. Otherwise this can be used for sending tracking ID.

## Security and Privacy Considerations
- The publishers can know whether the referrer page has prefetched the signed exchange subresources or not by checking the resource timing information. But this only exposes 1 bit information (= yes or no) because UAs can use the cached signed exchange only if the required signed exchanges are all available.
- The UA can use the prefetched signed exchange subresources,  only when they were prefetched in the referrer page. This is intended to avoid leaking the prefetching state to succeeding pages.
- The UA check the header integrity value, so the distributor of the subresource signed exchange can’t inject arbitrary resources to the publisher’s page. This prevents distributors from sending tracking IDs to the publisher’s page.
- The UA need to check both [the request URL](https://wicg.github.io/webpackage/loading.html#exchange-request-url) and the header integrity value of the signed exchange to avoid [Origin Laundering](https://hillbrad.github.io/sri-addressable-caching/sri-addressable-caching.html#origin_laundering) attack. The URL is not in signedHeaders, so the header integrity value can’t check the URL is correct.
- The UA must fetch the signed exchange subresource (https://cdn.feed.example/cdn.publisher.example/lib.js.sxg) while processing the prefetch link element  (`<link rel="prefetch" href="https://cdn.feed.example/cdn.publisher.example/lib.js.sxg" as="script">`) even if there is the original subresource (http://cdn.publisher.example/lib.js) in the HTTPCache. Otherwise it leaks the state of  publisher’s site to the distributor of the signed exchange.
- The UA must ignore the not-yet-completed subresource signed exchange prefetch to prevent distributors from interfering the publisher’s page. (Eg. Intentionally block or delay the subresource loading.) So if the user clicks the link to the article while the UA is still prefetching the subresource signed exchange, the UA must fetch the subresource from the original URL after navigation.
- The UA can use the prefetched signed exchange subresources only if the preload HTTP link headers for the subresources exist in the response headers. It is because the UA need to check the availability of all required signed exchanges before start loading subresources.

## [Self-Review Questionnaire: Security and Privacy](https://www.w3.org/TR/security-privacy-questionnaire/)
1. What information might this feature expose to Web sites or other parties, and for what purposes is that exposure necessary?
   - This feature exposes the 1 bit information "the referrer page has prefetched the signed exchange subresources or not" to the publisher.
1. Is this specification exposing the minimum amount of information necessary to power the feature?
   - Yes. This proposal has limitations such as "all subresource SXG must be finished prefetching, otherwise ignored", "subresource SXG must be prefetched even if the original subresource exists in HTTPCache".
1. How does this specification deal with personal information or personally-identifiable information or information derived thereof?
   - Signed Exchange should not include personal information.
1. How does this specification deal with sensitive information?
   - Signed Exchange should not include sensitive information.
1. Does this specification introduce new state for an origin that persists across browsing sessions?
   - No. The prefetched signed exchange subresources can be used only from the pages which are navigated from the document which prefetched them.
1. What information from the underlying platform, e.g. configuration data, is exposed by this specification to an origin?
   - This exposes whether the UA support this feature or not.
1. Does this specification allow an origin access to sensors on a user’s device
   - No
1. What data does this specification expose to an origin? Please also document what data is identical to data exposed by other features, in the same or different contexts.
   - This feature exposes the 1 bit information "the referrer page has prefetched the signed exchange subresources or not" to the publisher.
   - Sending 1 bit information from the distributor to the publisher is already easily possible just by changing the URL.
1. Does this specification enable new script execution/loading mechanisms?
   - This specification introduces a new script loading path, from prefetched signed exchange. The existing security checks such as CSP/CORP must be applied as if the script is loaded from the original URL.
1. Does this specification allow an origin to access other devices?
   - No
1. Does this specification allow an origin some measure of control over a user agent’s native UI?
   - No
1. What temporary identifiers might this this specification create or expose to the web?
   - No
1. How does this specification distinguish between behavior in first-party and third-party contexts?
   - This feature treats all entities (the distributor of signed exchange, the publisher site, the origin of subresource URL) as third-party origins.
   - To avoid leaking user-specific data in the distributor of signed exchange, the prefetch request must not contain credentials. This is covered by the  "Prefetch and double-key caching" issue.
   - The origin of subresource URL could be different from the origin of publisher site. The cross origin security checks (CORS/CORB/CORP/..) must be executed while reading the response from the cached signed exchanges.
1. How does this specification work in the context of a user agent’s Private \ Browsing or "incognito" mode?
   - No difference while the user is browsing sites in Private mode.
   - If the user opens a link in private mode while browsing in normal mode (eg: "Open link in incognito window"), the prefetched signed exchanges must be ignored.
1. Does this specification have a "Security Considerations" and "Privacy Considerations" section?
   - Yes
1. Does this specification allow downgrading default security characteristics?
   - No

