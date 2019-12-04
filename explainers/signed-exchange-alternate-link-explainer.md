# Signed Exchange alternate link
## Introduction
We want to extend the usage of the existing [rel=alternate](https://html.spec.whatwg.org/multipage/links.html#rel-alternate) link header for signed exchange. Using this link header, the content publishers can declare that the resource is available in signed exchange format. This can be used both by the crawlers of aggregator sites  (SNS, News site, search engine..) and by the UAs. The crawlers can cache and serve the signed exchange of the content in their own server. The UAs can provide the users with a way to save the page in signe exchange format. And also signed exchange alternate links can be used to recursively prefetch appropriate subresource signed exchanges while prefetching the main resource signed exchange.

## Use Cases
### Signed Exchange discovery of main resource
1. Content publishers can declare the URL of signed exchange format of the content using an alternate link HTTP header or using an alternate link HTML element.
   - Example of signed exchange alternate link HTTP header in the HTTP response headers of the main resource of the content (https://publisher.example/article.html):
      ```
      Link: <https://sxg.publisher.example/article.html.sxg>;
              rel="alternate";
              type="application/signed-exchange;v=b3";
              anchor="https://publisher.example/article.html"
      ```
   - Example of signed exchange alternate link HTML element in the main resource of the content (https://publisher.example/article.html):
      ```
      <link href="https://sxg.publisher.example/article.html.sxg"
            rel="alternate"
            type="application/signed-exchange;v=b3"
            anchor="https://publisher.example/article.html">
      ```
1. This signed exchange alternate link of main resource can be used both by the crawlers and the UAs.
   - When the crawlers detects the signed exchange alternate link, the crawlers can fetch the signed exchange.  And when a user is browsing the aggregator site (https://feed.example) and it has a link to the article, the signed exchange can be used for [privacy-preserving prefetching](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#private-prefetch) using a prefetch `<link>` element and `<a>` link to the signed exchange .
      ```
      <link href="https://feed.example/article.html.sxg"
            rel="prefetch">
      <a href="https://feed.example/article.html.sxg">feed.example</a>
      ```
   - While the user of the UA is browsing an article (article.html), and if there is a signed exchange alternate link, the UA can provide the user with a way to save the page in signe exchange format. The saved file can be used to share with other users.

### Recursive subresource signed exchange prefetch
1. A crawler of an aggregator site fetches an article (https://publisher.example/article.html).
1. There is an alternate link of signed exchange of the article in the response header:
    ```
    Link: <https://sxg.publisher.example/article.html.sxg>;
            rel="alternate";
            type="application/signed-exchange;v=b3";
            anchor="https://publisher.example/article.html"
    ```
1. The crawler fetches and verifies the signed exchange (article.html.sxg).

   1. The response from the server has an alternate link of subresource signed exchange:
      ```
      Link: <https://sxg.publisher.example/lib.js.sxg>;
              rel="alternate";
              type="application/signed-exchange;v=b3";
              anchor="https://cdn.publisher.example/lib.js"
      ```
   1. And the inner response of the main resource signed exchange (article.html.sxg) has a preload header and an [allowed-alt-sxg header](./signed-exchange-subresource-subtitution-explainer.md):
      ```
      Link: <https://cdn.publisher.example/lib.js>;
              rel="preload";
              as="script"
      Link: <https://cdn.publisher.example/lib.js>;
              rel="allowed-alt-sxg";
              header-integrity="sha256-XXXXXX"
      ```
1. The crawler fetches and verifies the subresource signed exchange (lib.js.sxg).
1. The aggregator site will serve the signed exchanges (article.html.sxg and lib.js.sxg) from their own server.

1. While a user is browsing the aggregator site (https://feed.example), a prefetch link element is inserted because the use is likely to want to read the article.
   ```
   <link rel="prefetch" href="https://feed.example/sxg.publisher.example/article.html.sxg">
   ```
1. The UA prefetches the signed exchange (article.html.sxg).
   1. The response from the server has an alternate link of subresource signed exchange:
      ```
      Link: <https://feed.example/sxg.publisher.example/lib.js.sxg>;
              rel="alternate";
              type="application/signed-exchange;v=b3";
              anchor="https://cdn.publisher.example/lib.js"
      ```
   1. And the inner response of the main resource signed exchange (article.html.sxg) has a preload header and an allowed-alt-sxg header:
      ```
      Link: <https://cdn.publisher.example/lib.js>;
              rel="preload";
              as="script"
      Link: <https://cdn.publisher.example/lib.js>;
              rel="allowed-alt-sxg";
              header-integrity="sha256-XXXXXX"
      ```
1. The UA recursively prefetches lib.js.sxg.
1. If the user clicks a link the article’s signed exchange, both the main resource of the article and the script are loaded from the prefetched signed exchanges.

## Proposal
While processing preload link HTTP headers in prefetched main resource signed exchange’s inner response:
1. Check whether matching “allowed-alt-sxg” link HTTP header in the inner response exists or not. (Note that if the allowed-alt-sxg link HTTP header has variants and variant-key attributes, the UA must execute the algorithm written in [HTTP Representation Variants](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html) spec to find the matching header.)
1. If exists, check whether matching “alternate” link HTTP header in the outer response exists or not.
1. If exists, prefetches the matching signed exchange instead of prefetching the original resource URL.
1. The prefetched signed exchange will be stored to the SignedExchangeCache of the Document. And it will be passed to the next Document and used while processing the preload link header. This behavior is written in [Signed Exchange subresource substitution explainer](./signed-exchange-subresource-subtitution-explainer.md).

## Detailed design discussion
### Content negotiation using Variants and Variant-Key
The **alternate** link header and **allowed-alt-sxg** link headers can have [variants and variant-key](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html) attributes to support content negotiation (eg: [WebP support](https://developers.google.com/speed/webp/faq#server-side_content_negotiation_via_accept_headers)).

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
If a UA supports WebP, the UA should prefetch **image_webp.sxg** which content is WebP format. Otherwise the UA should prefetch **image_jpeg.sxg** which content is JPEG format.

## Security and Privacy Considerations
The UA must fetch the alternate signed exchange subresource (lib.js.sxg) even if there is the original subresource (lib.js) in the HTTPCache. Otherwise it leaks the state of publisher’s site to the distributor of the signed exchange.

## [Self-Review Questionnaire: Security and Privacy](https://www.w3.org/TR/security-privacy-questionnaire/)
1. What information might this feature expose to Web sites or other parties, and for what purposes is that exposure necessary?
   - The existence of the alternative signed exchange in HTTP Cache is exposed.
1. Is this specification exposing the minimum amount of information necessary to power the feature?
   - Yes.
1. How does this specification deal with personal information or personally-identifiable information or information derived thereof?
   - Signed Exchange should not include personal information.
1. How does this specification deal with sensitive information?
   - Signed Exchange should not include sensitive information.
1. Does this specification introduce new state for an origin that persists across browsing sessions?
   - No. The prefetched signed exchange is stored to HTTPCache. But this is the existing behavior when directly prefetching the signed exchange using `<link rel=prelfetch>`.
1. What information from the underlying platform, e.g. configuration data, is exposed by this specification to an origin?
   - This exposes whether the UA support this feature or not.
1. Does this specification allow an origin access to sensors on a user’s device
   - No
1. What data does this specification expose to an origin? Please also document what data is identical to data exposed by other features, in the same or different contexts.
   - The existence of the alternative signed exchange in HTTP Cache is exposed.
   - But this is the same as the existing behavior when directly prefetching the signed exchange using `<link rel=prelfetch>`.
1. Does this specification enable new script execution/loading mechanisms?
   - No
1. Does this specification allow an origin to access other devices?
   - No
1. Does this specification allow an origin some measure of control over a user agent’s native UI?
   - No
1. What temporary identifiers might this this specification create or expose to the web?
   - No
1. How does this specification distinguish between behavior in first-party and third-party contexts?
   - This feature should work well with third-party signed exchange.
   - To avoid leaking the state of publisher’s site to the distributor of the signed exchange, the UA must fetch the alternate signed exchange subresource even if there is the original subresource in the HTTPCache.
1. How does this specification work in the context of a user agent’s Private \ Browsing or "incognito" mode?
   - No difference while the user is browsing sites in Private mode.
1. Does this specification have a "Security Considerations" and "Privacy Considerations" section?
   - Yes
1. Does this specification allow downgrading default security characteristics?
   - No
