# Signed Exchange alternate link
## Introduction
We want to allow the publisher of a resource to declare that a signed exchange
is available holding the content of either that resource or one of its
subresources. We expect aggregator sites (social networks, News site, search
engine..) to use this to cache the signed version of a resource in order to
serve it to their users. We expect UAs to use this to allow users to save the
page in signed exchange format. When the publisher identifies a same-origin
signed exchange for a cross-origin subresource, the UA can use that information
to recursively prefetch the subresource without exposing its speculative
activity across origins.

[`<link rel="alternate" type="application/signed-exchange" href=...>`](https://html.spec.whatwg.org/multipage/links.html#rel-alternate)
and the equivalent `Link` header are already defined to declare that the
referenced document is a reformulation of the current document as a signed
exchange. To offer signed exchanges for subresources, we propose to use the
[`anchor` parameter](https://tools.ietf.org/html/rfc8288#section-3.2) to
identify the replaced subresource. This may be the first use of the `anchor`
parameter in the web platform.


## Use Cases
### Recursive subresource signed exchange prefetch
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
aggregator.

The response from the server has an alternate link of subresource signed
exchange:

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

## Proposal
While prefetching a HTML resource in signed exchange format:

1. Whent the UA detects a "preload" link HTTP header in the inner response,
check whether matching “allowed-alt-sxg” link HTTP header in the inner response
exists or not. (Note that if the allowed-alt-sxg link HTTP header has variants
and variant-key attributes, the UA must execute the algorithm written in
[HTTP Representation Variants](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html)
spec to find the matching header.)
1. If exists, check whether matching “alternate” link HTTP header in the outer
response exists or not.
1. If exists, prefetches the matching signed exchange instead of prefetching the
original resource URL.
1. The prefetched signed exchange will be stored to the SignedExchangeCache of
the Document. And it will be passed to the next Document and used while
processing the preload link header. This behavior is written in
[Signed Exchange subresource substitution explainer](./signed-exchange-subresource-subtitution-explainer.md).

## Content negotiation using Variants and Variant-Key
The **alternate** link header and **allowed-alt-sxg** link headers can have
[variants and variant-key](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html)
attributes to support content negotiation (eg:
[WebP support](https://developers.google.com/speed/webp/faq#server-side_content_negotiation_via_accept_headers)).

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
If a UA supports WebP, the UA should prefetch **image_webp.sxg** which content
is WebP format. Otherwise the UA should prefetch **image_jpeg.sxg** which
content is JPEG format.

## Security and Privacy Considerations
The UA must fetch the alternate signed exchange subresource (lib.js.sxg) even if
there is the original subresource (lib.js) in the HTTPCache. Otherwise it leaks
the state of publisher’s site to the distributor of the signed exchange.
Also, to prevent tracking (user ID transfer), if the aggregator failed to
prefetch a subresource that the main resource preloads, the UA must drop all of
the subresource prefetches. If the aggregator prefetches a superset of the
preloaded subresources, the UA must drop the ones that weren't preloaded.

## [Self-Review Questionnaire: Security and Privacy](https://www.w3.org/TR/security-privacy-questionnaire/)
1. What information might this feature expose to Web sites or other parties, and
for what purposes is that exposure necessary?
   - The existence of the alternative signed exchange in HTTP Cache is exposed.
1. Is this specification exposing the minimum amount of information necessary to
power the feature?
   - Yes.
1. How does this specification deal with personal information or
personally-identifiable information or information derived thereof?
   - Signed Exchange should not include personal information.
1. How does this specification deal with sensitive information?
   - Signed Exchange should not include sensitive information.
1. Does this specification introduce new state for an origin that persists
across browsing sessions?
   - No. The prefetched signed exchange is stored to HTTPCache. But this is the
   existing behavior when directly prefetching the signed exchange using
   `<link rel=prelfetch>`.
1. What information from the underlying platform, e.g. configuration data, is
exposed by this specification to an origin?
   - This exposes whether the UA support this feature or not.
1. Does this specification allow an origin access to sensors on a user’s device
   - No
1. What data does this specification expose to an origin? Please also document
what data is identical to data exposed by other features, in the same or
different contexts.
   - The existence of the alternative signed exchange in HTTP Cache is exposed.
   - But this is the same as the existing behavior when directly prefetching the
   signed exchange using `<link rel=prelfetch>`.
1. Does this specification enable new script execution/loading mechanisms?
   - No
1. Does this specification allow an origin to access other devices?
   - No
1. Does this specification allow an origin some measure of control over a user
agent’s native UI?
   - No
1. What temporary identifiers might this this specification create or expose to
the web?
   - No
1. How does this specification distinguish between behavior in first-party and
third-party contexts?
   - This feature should work well with third-party signed exchange.
   - To avoid leaking the state of publisher’s site to the distributor of the
   signed exchange, the UA must fetch the alternate signed exchange subresource
   even if there is the original subresource in the HTTPCache.
1. How does this specification work in the context of a user agent’s Private
Browsing or "incognito" mode?
   - No difference while the user is browsing sites in Private mode.
1. Does this specification have a "Security Considerations" and "Privacy
Considerations" section?
   - Yes
1. Does this specification allow downgrading default security characteristics?
   - No
