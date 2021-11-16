# Explainer: Subresource loading with Web Bundles

Last updated: Apr 2021

We propose a new approach to load a large number of resources efficiently using
a format that allows multiple resources to be bundled, e.g.
[Web Bundles](https://web.dev/web-bundles/).

<!-- TOC -->

- [Backgrounds](#backgrounds)
- [Requirements](#requirements)
- [`<script>`-based API](#script-based-api)
- [Example](#example)
  - [The bundle](#the-bundle)
  - [The main document](#the-main-document)
- [Request's mode and credentials mode](#requests-mode-and-credentials-mode)
- [Request's destination](#requests-destination)
- [CORS and CORP for subresource requests](#cors-and-corp-for-subresource-requests)
- [Content Security Policy (CSP)](#content-security-policy-csp)
- [Extensions](#extensions)
- [Subsequent loading and Caching](#subsequent-loading-and-caching)
- [Compressed list of resources](#compressed-list-of-resources)
- [Alternate designs](#alternate-designs)
  - [`<link>`-based API](#link-based-api)
  - [Resource Bundles](#resource-bundles)
  - [Summarizing the contents of the bundle](#summarizing-the-contents-of-the-bundle)
    - [Defining the scope](#defining-the-scope)
    - [Approximate Membership Query datastructure](#approximate-membership-query-datastructure)
    - [No declarative scope](#no-declarative-scope)
  - [Naming](#naming)

<!-- /TOC -->

## Backgrounds

- Loading many unbundled resources is still slower in 2020. We concluded that
  [bundling was necessary in 2018](https://v8.dev/features/modules#bundle), and
  our latest local measurement still suggests that.

- The output of JS bundlers (e.g. webpack) doesn't interact well with the HTTP
  cache. They are pretty good tools but configuring them to work in an optimal
  way is tough, and sometimes they'are also incompatible with new requirements
  like
  [dynamic bundling](https://github.com/azukaru/progressive-fetching/blob/master/docs/dynamic-bundling/index.md)
  (e.g. small edit with tree shaking could invalidate everything).

- With JS bundlers, execution needs to wait for the full bytes to come. Ideally
  loading multiple subresources should be able to utilize full streaming and
  parallelization, but that's not possible if all resources are bundled as one
  javascript. (For JS modules execution still needs to be waited for the entire
  tree due to the current
  [deterministic execution model](https://docs.google.com/document/d/1MJK0zigKbH4WFCKcHsWwFAzpU_DZppEAOpYJlIW7M7E/edit#heading=h.5652gd5ks5id))

- Related issues: [#411](https://github.com/WICG/webpackage/issues/411),
  [#526](https://github.com/WICG/webpackage/issues/526)

## Requirements

Web pages will declare that some of their subresources are provided by the
[Web
Bundle](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html)
at a particular URL.

It's likely that the HTML parser will encounter some of the bundle's
subresources before it receives the bundle's index. The declaration needs to
somehow prevent the parser from double-fetching those bytes, which it can
accomplish in a couple ways.

We don't see an initial need for an associated Javascript API to pull
information out of the bundle.

We also don't address a way for Service Workers to use bundles to fill a Cache.
Service Workers can technically unpack a bundle into
[`cache.put()`](https://developer.mozilla.org/en-US/docs/Web/API/Cache/put)
calls themselves, and, while the result may take an inefficient amount of
browser-internal communication, letting some sites experiment with this will
give us a better chance of designing the right API.

This feature is a powerful feature that can replace any subresources in the
page. So we limit the use of this feature only in [secure contexts](https://www.w3.org/TR/powerful-features/).

This feature is NOT related to [Signed
Exchanges](https://web.dev/signed-exchanges/), that is a common
misunderstanding. The bundle doesn't have to be signed.

## `<script>`-based API

Note that this syntax is still tentative.

Developers will write

```html
<script type="webbundle">
{
   source: "https://example.com/dir/subresources.wbn",
   resources: ["https://example.com/dir/a.js", "https://example.com/dir/b.js", "https://example.com/dir/c.png"]
}
</script>
```

to tell the browser that subresources specified in `resources` can
be found within the `https://example.com/dir/subresources.wbn` bundle.

When the browser parses such a `script` element, it:

1. Fetches the specified Web Bundle, `https://example.com/dir/subresources.wbn`.

2. Records the `resources` and _delays_ fetching a subresource specified there if
   a subresource's origin is the [same origin](https://html.spec.whatwg.org/#same-origin)
   as the bundle's origin and its [path](https://url.spec.whatwg.org/#concept-url-path)
   contains the bundle's path as a prefix.

3. As the bundle arrives, the browser fulfills those pending subresource
   fetches from the bundle's contents.

4. If a fetch isn't actually contained inside the bundle, it's
   probably better to fail that fetch than to go to the network, since
   it's easier for developers to fix a deterministic network error
   than a performance problem.

   The primary requirement to avoid fetching the same bytes twice is that "If a
   specified subresource is needed later in the document, that later fetch
   should block until at least the index of the bundle has downloaded to see if
   it's there."

   It seems secondary to then say that if a specified subresource isn't
   in the bundle, its fetch should fail or otherwise notify the developer: that
   just prevents delays in starting the subresource fetch.

## Example

### The bundle

Suppose that the bundle, `subresources.wbn`, includes the following resources:

```
- https://example.com/dir/a.js (which depends on ./b.js)
- https://example.com/dir/b.js
- https://example.com/dir/c.png
- … (omitted)
```

A URL of the resource in the bundle can be a [relative
URL](https://url.spec.whatwg.org/#syntax-url-relative) to the bundle.
A browser must [parse a URL](https://html.spec.whatwg.org/#parse-a-url)
using bundle's URL.

### The main document

```html
<script type="webbundle">
{
  source: "https://example.com/dir/subresources.wbn",
  resources: ["https://example.com/dir/a.js", "https://example.com/dir/b.js", "https://example.com/dir/c.png"]
}
</script>

<script type=”module” src=”https://example.com/dir/a.js”></script>
<img src=https://example.com/dir/c.png>
```

Then, a browser must fetch the bundle, `subresources.wbn`, and load
subresources, `a.js`, `b.js`, and `c.png`, from the bundle.

If a URL is available from an attached bundle, the browser must
retrieve it from the bundle, instead of using any [registered custom
protocol
handler](https://html.spec.whatwg.org/multipage/system-state.html#dom-navigator-registerprotocolhandler)
for its scheme.

A URL in `source` can be a [relative
URL](https://url.spec.whatwg.org/#syntax-url-relative) and must be resolved on
document's [base URL](https://html.spec.whatwg.org/#document-base-url).

A URL in `resources` and `scopes` can be a [relative
URL](https://url.spec.whatwg.org/#syntax-url-relative) and must be resolved on
the bundle's URL.

`<script type="webbundle">` doesn't support `src=` attribute. The rule must be inline.

`<script type="webbundle">` doesn't fire a `load` event nor an `error` event.

Note: We could fire `load` and `error` events if we wanted to. See [issue
#697](https://github.com/WICG/webpackage/issues/697)

Note that neither `<script type=importmaps>` nor `<script type=speculationrules>`
fires a `load` event for an inline rule.

## Request's mode and credentials mode

A [request](https://fetch.spec.whatwg.org/#concept-request) for a bundle
will have its [mode][request mode] set to "`cors`" and its
[credentials mode][credentials mode] set to "`same-origin`" unless a
`credentials` is specified in its JSON as follows:

``` html
<script type="webbundle">
{
  source: "https://example.com/dir/subresources.wbn",
  credentials: "omit",
  resources: ["https://example.com/dir/a.js", "https://example.com/dir/b.js", "https://example.com/dir/c.png"]
}
</script>
```

A possible value is "`omit`", "`same-origin`", or "`include"`. See [the fetch spec][credentials mode] for details.
If other values are specified, a [credentials mode][credentials mode] is set to "`same-origin`" .

Note: `<script>` element's [crossorigin][crossorigin attribute] attribute is not used.

[crossorigin attribute]: https://html.spec.whatwg.org/multipage/semantics.html#attr-link-crossorigin
[request mode]: https://fetch.spec.whatwg.org/#concept-request-mode
[credentials mode]: https://fetch.spec.whatwg.org/#concept-request-credentials-mode

## Request's destination

With the `<script>`-based API, a
[request](https://fetch.spec.whatwg.org/#concept-request) for a bundle
will have its
[destination](https://fetch.spec.whatwg.org/#concept-request-destination)
set to "`webbundle`"
([whatwg/fetch#1120](https://github.com/whatwg/fetch/issues/1120)).

## CORS and CORP for subresource requests
[CORS](https://fetch.spec.whatwg.org/#http-cors-protocol) and
[CORP](https://fetch.spec.whatwg.org/#cross-origin-resource-policy-header)
checks on subresources in bundles are based on the URL and response headers
of requested subresource.

For example, if a cors request is made to a cross-origin subresource in a
bundle, and the subresource does not have an `Access-Control-Allow-Origin:`
header, the request will fail.

Similarly, if a no-cors request is made to a cross-origin subresource in a
bundle, and the subresource has `Cross-Origin-Resource-Policy: same-origin`
header, the request will fail.

## Content Security Policy (CSP)

For resources loaded from bundles, URL matching of CSP is done based on the URL
of the resource, not the URL of the bundle. For example, given this CSP header:
```
Content-Security-Policy: script-src https://example.com/script/
```

In the following, `a.js` will be loaded, but `b.js` will be blocked:

```
<script type="webbundle">
{
  source: "https://example.com/subresources.wbn",
  resources: ["https://example.com/script/a.js",
              "https://example.com/b.js"]
}
</script>

<script src=”https://example.com/script/a.js”></script>
<script src=”https://example.com/b.js”></script>
```

#### Defining the scopes

Instead of including a list of resources, the `<script>` defines a `scopes`.

```html
<script type="webbundle">
{
  source: "https://example.com/dir/subresources.wbn",
  scopes: ["https://example.com/dir/js/",
           "https://example.com/dir/img/",
           "https://example.com/dir/css/"]
}
</script>
```

Any subresource under the `scopes` will be fetched from the bundle.


## Extensions

There are several extensions to this explainer, aiming to support
various use cases which this explainer doesn't support:

- [Subresource loading with Web Bundles: Support opaque origin iframes](./subresource-loading-opaque-origin-iframes.md)

See [issue #641](https://github.com/WICG/webpackage/issues/641) for
the motivation of splitting the explainer into the core part, this
explainer, and the extension parts.

## Subsequent loading and Caching

[Dynamic bundle serving with
WebBundles](https://docs.google.com/document/d/11t4Ix2bvF1_ZCV9HKfafGfWu82zbOD7aUhZ_FyDAgmA/edit)
is a detailed exploration of how to efficiently retrieve only updated resources
on the second load. The key property is that the client's request for a bundle
embeds something like a [cache
digest](https://httpwg.org/http-extensions/cache-digest.html) of the resources
it already has, and the server sends down the subset of the bundle that the
client doesn't already have.

## Compressed list of resources

As discussed in [Dynamic bundle serving with
WebBundles](https://docs.google.com/document/d/11t4Ix2bvF1_ZCV9HKfafGfWu82zbOD7aUhZ_FyDAgmA/edit),
simply including a list of resources in the HTML [may cost as little as 5 bytes
per URL on average after the HTML is
compressed](https://github.com/yoavweiss/url_compression_experiments).

## Alternate designs

### `<link>`-based API

This explainer had used `<link>`-based API before adopting `<script>`-based API:

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  resources="https://example.com/dir/a.js https://example.com/dir/b.js https://example.com/dir/c.png"
/>
```

However, we abandoned `<link>`-based API, in favor of `<script>`-based
API. See [issue #580](https://github.com/WICG/webpackage/issues/580)
for the motivation. Note that some of the following alternate designs
were proposed at the era of `<link>`-based API. This explainer doesn't
rewrite them with `<script>`-based API yet.

Note that Chromium's experimental implementation currently supports
only `<link>`-based API as of M95.

### Resource Bundles

A [resource bundle] is the same effort, with a particular scope. A
[resource bundle] has a good
[FAQ](https://github.com/WICG/resource-bundles/blob/main/faq.md#q-how-does-this-proposal-relate-to-the-web-packageweb-packagingweb-bundlesbundled-exchange-effort-repo)
which explains how this proposal and a [resource bundle] are related.

We have been collaborating closely to gather more feedback to draw a shared conclusion.

[resource bundle]: https://github.com/WICG/resource-bundles

### Summarizing the contents of the bundle

Several other mechanisms are available to give the bundler more flexibility or to compress the resource list.

#### Approximate Membership Query datastructure

A page still executes correctly, albeit slower than optimal, if a resource
that's in a bundle is fetched an extra time, or a resource that's not in a
bundle waits for the bundle to arrive before its fetch starts. That raises the
possibility of putting a Bloom filter or other _approximate membership query_
datastructure, like a cuckoo filter or quotient filter, in the scoping
attribute.

In this case, it must not be an error if a resource matches the filter but turns
out not to be in the bundle, since that's an expected property of this
datastructure.

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  digest="cuckoo-CwAAAAOztbwAAAM2AAAAAFeafVZwIPgAAAAA"
/>
```

#### No declarative scope

In some cases, the page might be able to control when it issues fetches for all
of the resources contained in a bundle. In that case, it doesn't need to
describe the bundle's scope in the `<link>` element but can instead listen for
its `load` event:

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  onload="startUsingTheSubresources()"
/>
```

Since the web bundles format includes an index before the content, we can
optimize this by firing an event after the index is received (which expresses
the bundle's exact scope) but before the content arrives:

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  onscopereceived="startUsingTheSubresources()"
/>
```

### Naming

We might be able to use a link type as general as `"bundle"`, especially if it
also uses the MIME type of the bundle resource to determine how to process it.

We'll need to disambiguate between a bundle meant for preloading subresources
and a bundle meant as an alternative form of the current page. The second can
use `<link rel="alternate" type="application/web-bundle">`.

# Acknowledgements

Thanks to https://github.com/yoavweiss/cache-digests-cuckoo and
https://github.com/google/brotli for the software used to generate sample
attribute values.
