# Explainer: Subresource loading with Web Bundles

Last updated: April 2020

We propose a new approach to load a large number of resources efficiently using
a format that allows multiple resources to be bundled, e.g.
[Web Bundles](https://web.dev/web-bundles/).

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

## Proposal

We propose a new `link rel=”webbundle”` notion [**1**] that lets the browser
fetch bundled resources and attach them to the page. Such as:

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  scope="https://example.com/scope/"
/>
```

If the main document has such a `link` element, a browser must do:

- A browser must fetch the specified Web Bundle, `subresources.wbn`.

- All the subresources whose URLs are under the scope
  `https://example.com/scope/` must [**2**] be loaded from the specified Web
  Bundle.

[**1**] The syntax is tentative. The API surface is not so important for us yet.
Any alternative ideas are welcome. This is basically about mapping a set of
resources to a particular bundled resource. Something like
[Fetch maps proposal](https://discourse.wicg.io/t/proposal-fetch-maps/4259)
might be able to extend to this case.

[**2**] The exact behavior is TBD. There are several possible behaviors when the
bundle does not contain the subresource; e.g. just fail, fallback to the
network, or allow users to specify its behavior with a `link` element's
attribute.

The primary requirement to avoid fetching the same bytes twice is that "If a
subresource under that scope is needed later in the document, that later fetch
should block until at least the index of the bundle has downloaded to see if
it's there."

It seems secondary to then say that if the subresource within that scope isn't
in the bundle, its fetch should fail or otherwise notify the developer: that
just prevents delays in starting the subresource fetch.

## Example

### The bundle

Suppose that the bundle, `subresources.wbn`, includes the following resources:

```
- https://example.com/scope/a.js (which depends on ./b.js)
- https://example.com/scope/b.js
- https://example.com/scope/c.png
- … (omitted)
```

### The main document

```html
<link rel="webbundle"
    href="https://example.com/dir/subresources.wbn"
    scope="https://example.com/scope/"
/>

<script type=”module” src=”https://example.com/scope/a.js”></script>
<img src=https://example.com/scope/c.png>
```

Then, a browser must fetch the bundle, `subresources.wbn`, and load
subresources, `a.js`, `b.js`, and `c.png`, from the bundle.

## Subsequent loading and Caching

On the subsequent navigations for the same page/origin, it’s desirable that the
resources that are already fetched in the bundle shouldn't be refetched. We're
thinking about two possibilities here:

1. Embed the details of version / cached resources in the URL
2. Use a cache digest or something (and also embed it in the request)

Then the server should be able to dynamically subset the Bundle and serve it.
Note that here we assume this cache should be same-origin only therefore should
not leak the cross-origin information.
