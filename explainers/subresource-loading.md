# Explainer: Subresource loading with Web Bundles

Last updated: April 2020

We propose a new approach to load a large number of resources efficiently using
a format that allows multiple resources to be bundled, e.g.
[Web Bundles](https://web.dev/web-bundles/).

<!-- TOC -->

- [Backgrounds](#backgrounds)
- [Requirements](#requirements)
- [`<link>`-based API](#link-based-api)
- [Example](#example)
  - [The bundle](#the-bundle)
  - [The main document](#the-main-document)
- [Subsequent loading and Caching](#subsequent-loading-and-caching)
- [Authority](#authority)
- [Alternate designs](#alternate-designs)
  - [Defining the scope](#defining-the-scope)
    - [Compressed list of resources](#compressed-list-of-resources)
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

Web pages will declare that some of their their subresources are provided by the
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

## `<link>`-based API

Note that this syntax is still tentative.

Developers will write

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  scope="https://example.com/dir/scope/"
/>
```

to tell the browser that subresources under `https://example.com/dir/scope/` can
be found within the `https://example.com/dir/subresources.wbn` scope.

When the browser parses such a `link` element, it:

1. Fetches the specified Web Bundle, `/dir/subresources.wbn`.

2. Records the scope and _delays_ fetching any other subresource within that
   scope.

3. As the bundle arrives, the browser fulfills those other pending subresource
   fetches from the bundle's contents.

4. If a fetch within the bundle's declared scope isn't actually contained inside
   the bundle, it's probably better to fail that fetch than to go to the
   network, since it's easier for developers to fix a deterministic network
   error than a performance problem.

As mentioned above, several aspects of this design are tentative. [Alternate
Designs](#alternate-designs) suggests several ways we're considering changing
it.

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
    href="https://example.com/subresources.wbn"
    scope="https://example.com/scope/"
/>

<script type=”module” src=”https://example.com/scope/a.js”></script>
<img src=https://example.com/scope/c.png>
```

Then, a browser must fetch the bundle, `subresources.wbn`, and load
subresources, `a.js`, `b.js`, and `c.png`, from the bundle.

## Subsequent loading and Caching

[Dynamic bundle serving with
WebBundles](https://docs.google.com/document/d/11t4Ix2bvF1_ZCV9HKfafGfWu82zbOD7aUhZ_FyDAgmA/edit)
is a detailed exploration of how to efficiently retrieve only updated resources
on the second load. The key property is that the client's request for a bundle
embeds something like a [cache
digest](https://httpwg.org/http-extensions/cache-digest.html) of the resources
it already has within the bundle's scope, and the server sends down the subset
of the bundle that the client doesn't already have.

## Authority

As with [navigation to unsigned
bundles](https://github.com/WICG/webpackage/blob/master/explainers/navigation-to-unsigned-bundles.md#loading-an-authoritative-unsigned-bundle),
a bundle's responses aren't always
[authoritative](https://httpwg.org/http-core/draft-ietf-httpbis-semantics-latest.html#establishing.authority)
for their claimed URLs. Responses on the bundle's origin and (by default) under
the bundle's path are authoritative. Responses with a claimed URL that is a hash
of the response itself, like the [`ni:///`
scheme](https://tools.ietf.org/html/rfc6920), could also be authoritative.

Unlike in the navigation case, but like with Service Workers, it may be
acceptable to let a page lie to itself and use non-authoritative responses from
bundles it has selected. However, we do not intend to allow this for now.

## Alternate designs

### Defining the scope

The approach above of just defining a scope directory doesn't allow much
flexiblity in how resources are grouped into bundles. Several other scoping
mechanisms are available to give the bundler more flexibility, at the cost of
being bigger and less readable.

#### Compressed list of resources

As discussed in [Dynamic bundle serving with
WebBundles](https://docs.google.com/document/d/11t4Ix2bvF1_ZCV9HKfafGfWu82zbOD7aUhZ_FyDAgmA/edit),
simply including a base64-encoded compressed list of resources may cost as
little as 5 bytes per URL on average. (That document doesn't show the details of
that measurement.) The compressed list of resources could look like:

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  resources="br-ofgFAO9IsGNWvPgiTk1dFeb2JqXM6FKBaQ5qcg7PdtubJNbKHsM8xoZYw5RxpG0udZvcCxS5jf6eWVHIDLLp+1Bo"
/>
```

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
