# Explainer: Resource loading with Content Addressable Bundles

Last update: Mar 2021

This is a strawperson proposal at very early stage, aiming to load multiple
resources efficiently, using a Content-Addressable Bundle, which can link to
other bundles.

## Authors

- Hayato Ito (hayato@google.com)

## Participate

- [WICG/webpackage](https://github.com/WICG/webpackage/)
  ([#638](https://github.com/WICG/webpackage/issues/638))

<!-- TOC -->

## Table of Contents

- [Introduction](#introduction)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Web Bundle format](#web-bundle-format)
- [Web APIs](#web-apis)
  - [Declarative form](#declarative-form)
    - [Example](#example)
  - [How to Load](#how-to-load)
  - [Navigate to a bundle](#navigate-to-a-bundle)
- [Cache strategy](#cache-strategy)
- [Key scenarios](#key-scenarios)
- [Considered alternatives](#considered-alternatives)
  - [Resource Bundles](#resource-bundles)
- [FAQ](#faq)
- [Stakeholder Feedback / Opposition](#stakeholder-feedback--opposition)
- [References & acknowledgements](#references--acknowledgements)

<!-- /TOC -->

## Introduction

- Loading many unbundled resources is still slower in 2021. We concluded that
  [bundling was necessary in 2018](https://v8.dev/features/modules#bundle), and
  our latest local measurement still suggests that.

- With JS bundlers (e.g. [webpack]), execution needs to wait for the full bytes
  to come. Ideally loading multiple subresources should be able to utilize full
  streaming and parallelization, but that's not possible if all resources are
  bundled as one javascript. (For JS modules execution still needs to be waited
  for the entire tree due to the current
  [deterministic execution model](https://docs.google.com/document/d/1MJK0zigKbH4WFCKcHsWwFAzpU_DZppEAOpYJlIW7M7E/edit#heading=h.5652gd5ks5id))

There is a proposal which aims to address the above issues, called [Subresource
loading with Web Bundles], a new way to load a large number of resources
efficiently using a format that allows multiple resources to be bundled, e.g.
[Web Bundles](https://web.dev/web-bundles/).

With the [Subresource loading with Web Bundles] proposal, developers will write

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  resources="https://example.com/dir/a.js https://example.com/dir/b.js https://example.com/dir/c.png"
/>
```

to tell the browser that subresources specified in `resources` attribute can be
found within the `https://example.com/dir/subresources.wbn` bundle. The browsers
will fetch a bundle, and load resources from the bundle.

However, currently, there is no mechanism to fetch the partial content of the
bundle. Browsers will fetch the entire bundle even if only one of the resources
in the bundle is updated in a server side.

This explainer proposes a new approach on the top of [Subresource loading with
Web Bundles], aiming more flexibility of how resources are grouped together as
bundles. This allows web developers to split code into various bundles which can
then be loaded on demand or in parallel, and also provide a new capability to
express a dependency between bundles.

## Goals

- This proposal aims to support [Code Splitting] use cases, as bundlers like
  [Webpack](https://webpack.js.org/guides/code-splitting/) or [Browserify]
  already support as a user-land solution.

  As an application grows in complexity or is maintained, CSS and JavaScripts
  files or bundles grow in byte size, especially as the number and size of
  included third-party libraries increases.

  Smaller bundles, if used correctly, can have a major impact on load time.
  Features required at page load can be downloaded immediately with additional
  bundles being lazy loaded after the page or application is interactive, thus
  improving performance.

- This proposal aims to add a new capability to declare dependencies between
  bundles inside of a _bundle itself_. This self-descriptiveness makes an
  on-demand lazy-loading possible without any further additional configuration
  outside of a bundle.

  A good analogy would be dynamic shared libraries, like `*.so` files. They have
  a dependencies section to declare dependencies to other libraries.

- Non-opinionated about bundle granularity. There are trade-offs how a site
  composes their resources into bundles in order to balance various factors like
  total bytes transferred, loading latency, or cache granularity. Instead of a
  _all-or-nothing_ bundle, this proposal aims to provide a way to express a
  dependency graph of bundles. The use of bundlers is an established practice in
  Web development. Bundlers would know much about which resources should be
  grouped as a bundle, and might want to express their intent as a dependency
  graph of bundles, considering various trade-offs.

- Bundles can be served from a static content server. The proposal doesn't
  require any smart server, such as dynamically assembling resources into a
  bundle. Bundles should be statistically generated by a bundler and can be
  copied to a static content server. This proposals assumes that this is
  important for a wide adoption.

- This proposal aims to achieve Content-Addressability to a bundle. If a
  bundle's URL doesn't change, we can assume the bundle's contents are _exactly_
  same. This is not an effort by a convention. The proposal aims to _force_
  immutability by introducing a Content-Addressable Hash, which is conceptually
  similar to a [Git]'s commit ID you might be familiar with.

  Content-Addressability gives web developers fearless reproducible builds, and
  prevents some kinds of attacks, like unexpected manipulation of fetching
  resources. Brave
  [raised the concern](https://brave.com/webbundles-harmful-to-content-blocking-security-tools-and-the-open-web/)
  that bundling systems could be used in a way where URLs are "rotated" between
  different requests, making URLs less meaningful/stable. Content-Addressability
  prevents this kind of undesired behaviors on a server side.

  Content-Addressability also encourages fearless sharing bundles, instead of
  embedding-all-for-safety strategy.

## Non-Goals

- Although this proposal aims to give a browser an opportunity to improve cache
  efficiency by utilizing immutability and a dependency graph of bundles, this
  proposal doesn't define any normative procedure how a browser should cache
  resources. That's up-to a browser, as of now.

- This proposal's current scope is to serve static contents which don't contain
  any personal information. Although nothing prevents us from including such
  personal information, this proposal currently assumes every resources are
  publicly viewable ones.

- [Dynamic bundling](https://github.com/azukaru/progressive-fetching/blob/master/docs/dynamic-bundling/index.md)
  use case. This use case requires a server side support. That's out of scope in
  this proposal.

- A fine-grained cache control for each individual resource in a bundle. That's
  out of scope in this proposal. This proposal assumes that a granularity of a
  cache can be controlled by a granularity of bundles, if used correctly, at
  some level. If you want a fine-grained cache control for an individual
  resource in a bundle, this proposal doesn't meet your requirements.

## Web Bundle format

TODO(hayato): Define a format using [CDDL]. For a while, this section explains a
non-normative _conceptual_ format.

[cddl]: https://datatracker.ietf.org/doc/html/rfc8610

Bundle's URL should be `<main-resource-url>.<hash>.wbn`.

- Example: `https://example.com/app/index.js.abcd.wbn`
- We'll explain how a hash is calculated later.

This proposal extends the [WebBundle Format][web bundles] with a new version
number. The bundle should have the following fields:

1. bundle's hash: `abcd`

   Note: This proposal has not decided which hash functions we should use. This
   section writes down a hash as just 4 characters, however, a hash would be
   much longer in real cases, like 40 characters used in a Git commit ID.

2. bundle's main-resource URL: `https://example.com/app/index.js`

   Note: The current WebBundle format also defines a main-resource URL. Now, a
   main resource URL must be a part of the bundle's URL,
   "`<main-resource-url>.<hash>.wbn`".

3. Resources section, which is conceptually as follows:

| URL                                           | type     | response headers            | body     | hash |
| --------------------------------------------- | -------- | --------------------------- | -------- | ---- |
| ./index.js                                    | inline   | \<encoded response header\> | \<body\> | b0b0 |
| ./foo.js                                      | inline   | \<encoded response header\> | \<body\> | c0c0 |
| ./bar.js                                      | inline   | \<encoded response header\> | \<body\> | d0d0 |
| https://cdn.example.com/date-util.js.f1f2.wbn | external | NA                          | NA       | f1f2 |

- A resource's URL is either relative or absolute. If a URL is relative, we call
  it an _inline resource_. If a URL is absolute, we call it an _external
  resource_.
- A relative URL must be resolved with base of `<main-resource-url>`. e.g.

  `./foo.js` is resolved as `https://example.com/app/foo.js`.

  Note: The motivation of using a relative URL for inline resources is to make
  it clear that
  [path restriction](https://w3c.github.io/ServiceWorker/#path-restriction) is
  always met.

- Inline resources must have _response headers_ and _body_.
- External resources must not have _response headers_ nor _body_ (Marked as `NA`
  in the table).
- Each resource must have a _hash_.
- Bundle's main resource must be there and must be an inline resource.
- External resource's URL must be formatted as
  `<external-main-resource>.<hash>.wbn`, which is pointing to an external
  bundle.

  Note that an external resource entry tells a browser that an external resource
  can be found in a linked external bundle. In the example,
  `https://cdn.example.com/date-util.js` can be loaded from
  `https://cdn.example.com/date-util.js.f1f2.wbn`.

- An external resource's URL can be cross-origin to the containing bundle.

- A bundle can have any number of inline resources and any number of external
  resource entries.

A _hash_ must be calculated as follows:

- For an inline resource:

  ```
  resource's hash := hash(hash(`<resource-url>`) + hash(`<canonicalized response headers>`) + hash(`<body>`))
  ```

- For an external resource:

  ```
  resource's hash := external bundle's hash
  ```

- For a bundle:

  ```
  bundle's hash := hash(main-resource-hash + resource0-hash + resource1-hash + ... (for every resources in a bundle))
  ```

  (This implies we calculate a hash recursively if a bundle contains an external
  resource)

  Note: A hash should be changed if a main resource is changed.

We call a bundle which satisfies the above requirements, **Content Addressable
Bundles** ; In short, **CAB**.

These requirements imply that a dependency graph of bundles forms a DAG
(directed acyclic graph). If it has a _cycle_, a hash can't be calculated.

To distinguish CAB and other existing Web Bundles which are not CAB, we may call
the former _immutable_ bundles, and the latter _mutable_ bundles in order to
make the difference clear in some contexts. Unless otherwise noted, a bundle
means a CAB, an _immutable_ bundle, in this proposal.

## Web APIs

### Declarative form

Note: Declarative syntax is tentative. We borrow
[\<link\>-based API](https://github.com/WICG/webpackage/blob/main/explainers/subresource-loading.md#link-based-api)
from [Subresource loading with web bundles] proposal for the purpose of the
explanation.

#### Example

The page (`https://example.com/app/index.html`):

<!-- prettier-ignore -->
```html
<link rel=webbundle
      href=https://example.com/app/index.js.abcd.wbn
      resources="index.js" >

<script type="module" src="index.js" />
```

The bundle (`https://example.com/app/index.js.abcd.wbn`):

| URL                                           | type     | response headers            | body     | hash |
| --------------------------------------------- | -------- | --------------------------- | -------- | ---- |
| ./index.js                                    | inline   | \<encoded response header\> | \<body\> | b0b0 |
| ./foo.js                                      | inline   | \<encoded response header\> | \<body\> | c0c0 |
| ./bar.js                                      | inline   | \<encoded response header\> | \<body\> | d0d0 |
| https://cdn.example.com/date-util.js.f1f2.wbn | external | NA                          | NA       | f1f2 |

Another bundle (`https://cdn.example.com/date-util.js.f1f2.wbn`) (which is
linked from the `index.js.abcd.wbn`):

| URL                                             | type     | response headers            | body     | hash |
| ----------------------------------------------- | -------- | --------------------------- | -------- | ---- |
| ./date-util.js                                  | inline   | \<encoded response header\> | \<body\> | 0808 |
| ./string-util.js                                | inline   | \<encoded response header\> | \<body\> | 0909 |
| https://cdn.example.com/common-util.js.3b3b.wbn | external | NA                          | NA       | 3b3b |

For illustration purposes, the dependency graph of resources is:

```
toppage
└── index.js
    ├── foo.js
    │   └── https://cdn.example.com/date-util.js
    │       ├── string-util.js
    │       └── https://cdn.example.com/common-util.js
    │           └── ...
    └── bar.js

```

These resources are bundled into three bundles, as follows:

```
toppage
└── https://example.com/app/index.js.abcd.wbn
    ├── index.js (main-resurce)
    ├── foo.js
    ├── bar.js
    └── https://cdn.example.com/date-util.js.f1f2.wbn
        ├── date-util.js (main-resurce)
        ├── string-util.js
        └── https://cdn.example.com/common-util.3b3b.js
            ├── commons-util.js (main-resource)
            └── ...
```

### How to Load

1. When the HTML page is parsed, a browser records that `index.js` can be loaded
   from the bundle, `https://example.com/app/index.js.abcd.wbn`.

2. The browser starts to fetch `https://example.com/app/index.js.abcd.wbn`.

   1. (This is asynchronously done) The browser must parse the index section of
      the fetched bundle, and record a list of resource's URLs, `foo.js`,
      `bar.js`, and `https://cdn.example.com/date-util.js`, as resources which
      must be loaded from the bundle.

   Note a relative URL is resolved based on the bundle's URL. For an external
   resource, we use a filename convention. If the URL ends with "`.<hash>.wbn`",
   the browser removes this suffix and then record the URL.

3. The browser sees `<script type="module" src="index.js" />` tag. Since
   `index.js` is already noted, the browser must load it from the bundle.

4. Suppose that `index.js`, which is an inline resource in the bundle, has the
   following content:

   ```js
   import * as foo from "./foo.js";
   import * as bar from "./bar.js";
   ...
   ```

   The browser must load `./foo.js` and `./bar.js` from the bundle since these
   resources are already recorded.

5. Suppose that `foo.js`, which is an inline resource in the bundle, has the
   following content:

   ```js
   import * as date from "https://cdn.example.com/date-util.js";
   ...
   ```

   The browser knows `https://cdn.example.com/date-util.js` is an external
   resource and it should be loaded from
   `https://cdn.example.com/date-util.js.f1f2.wbn`. The browser must start to
   fetch `https://cdn.example.com/date-util.js.f1f2.wbn`.

6. The browser must parse the index section of the bundle,
   `date-util.js.f1f2.wbn`, and record a list of resource's URLs, as the browser
   did for `index.js.abcd.wbn`.

7. After recorded, the browser loads the main resource, `date-util.js`, from the
   bundle. If `date-util.js` depends on `./string-util.js`, `string-util.js` is
   loaded from the bundle.
8. Continue processing...

Notes:

- The browser must validate a fetched bundle's hash integrity, and let a
  resource loading fail if the bundle is invalid.
- The browser can cache a bundle by its Content-Addressable hash, and free to
  load a bundle from the cache, instead of from the network.
- The browser may prefetch an external resource bundle as long as a semantics
  doesn't change. The browser should be careful not to record resource entries
  listed in a prefetched bundle too early.

### Navigate to a bundle

In the previous example, we use `<link>`-based API to declare a starting _node_
and its index information, as a _bootstrap_, however, if a browser supports
_Navigate to a bundle_ feature, we don't need such a declarative form in HTML.

For example, given that a bundle, `https://example.com/app/index.html.0101.wbn`,
whose main resource is HTML file:

| URL                                           | type     | response headers            | body     | hash |
| --------------------------------------------- | -------- | --------------------------- | -------- | ---- |
| ./index.html (main-resource)                  | inline   | \<encoded response header\> | \<body\> | 0101 |
| ./index.js                                    | inline   | \<encoded response header\> | \<body\> | b0b0 |
| ./foo.js                                      | inline   | \<encoded response header\> | \<body\> | c0c0 |
| ./bar.js                                      | inline   | \<encoded response header\> | \<body\> | d0d0 |
| https://cdn.example.com/date-util.js.f1f2.wbn | external | NA                          | NA       | f1f2 |

If a browser supports entering `https://example.com/app/index.html.0101.wbn` URL
_directly_ in its address bar, `./index.html` doesn't have to declare resources.
The browser can know them from the bundle's index section, before starting to
parse `index.html` file.

## Cache strategy

It's up-to browsers how to cache bundles. This explainer doesn't define any
formal procedure, however, there are several possible approaches:

- The browser might want to store a dependency graph of bundles in its cache
  storage, using some preferable data structures so that the browser can
  traverse a DAG efficiently.

- If the browser wants to prefetch external resource bundles, the browser can
  traverse a DAG and start to prefetch a missing _node_.

  For example, when the page declares a `bundle-A`, the browser can know its
  hash immediately, and start to traverse a DAG, starting `bunble-A` node, which
  can be looked up by its hash in the browsers' cache storage.

  For example, if the browser has the following DAG, the browser starts to
  prefetch `bundle-D` immediately.

  ```
  bundle-A
  ├── bundle-B
  └── bundle-C
      ├── bundle-D  (missing in the cache storage)
      └── bundle-E
  ```

- If the browser wants to save the cache storage space, the browser might want
  to store only index sections of bundles, which are good enough to traverse
  DAG, and doesn't store the bodies of inline resources because _body_ might be
  space-consuming.

  For example, given that browser has the following dependency graph of bundles,
  where the bodies of `bundle-C`, `bundle-D`, `bundle-E` are omitted to save the
  cache disk space, the browser can start to fetch them again, possibly in
  parallel. Note that a browser can start to fetch `bundle-D` and `bundle-E`
  immediately without waiting fetching `bundle-C` (and scanning its index
  section) because Content-Addressability guarantees that `bundle-C`'s content
  and its dependencies have not been changed. `bundle-C` always depends on the
  exactly same `bundle-D` and `bundle-E` addressed by their URLs.

  ```
  bundle-A
  ├── bundle-B
  └── bundle-C (body is missing)
      ├── bundle-D (body is missing)
      └── bundle-E (body is missing)
  ```

- It might be nice to share cache storage for CABs among origins, however,
  modern browsers split HTTPCache storage per origin for various reasons,
  especially for security reasons. Thus, sharing cache storage among origins is
  fundamentally challenging for security reasons. Security and privacy must be
  prioritized.

  TODO(hayato): Explore this problem space. Feedback is welcome.

- Individual inline resources in a bundle must not interfere a browser's HTTP
  Cache of its resolved URL.

  For example, an inline resource, `./index.js`, in the bundle,
  `https://example.com/app/index.js.abcd.wbn`, and
  `https://example.com/app/index.js` must be treated separately, in terms of
  caching.

## Key scenarios

TODO(hayato): [Description of the end-user scenario]

### Application Developers using bundlers

TODO(hayato): Describe key scenarios.

### CDNs (Content Delivery Networks)

TODO(hayato): Describe key scenarios. How this feature would make CDNs happy and
which features are important to reduce their disk space and/or the total
bandwidth? Feedback is welcome.

## Considered alternatives

### [Resource Bundles]

TODO(hayato): Mention scopes.

## FAQ

### What happens if a bundle doesn't declare a dependency to a resource which is used from its inline resources?

Nothing is wrong if you know what you are doing.

However, this proposal strongly suggests that a bundle should be
_self-contained_ as much as possible; Include every resources as inline
resources, or declare external dependencies explicitly in the bundle in order to
achieve reproducibility.

In the above example, `https://example.com/app/index.js.abcd.wbn` declares a
dependency to the _specific_ version of `date-util.js` with
`https://cdn.example.com/date-util.js.f1f2.wbn`.

It's totally fine to use an external file directly without any declaration. For
example, `index.js` might use an external module, `hello.js`, as follows:

```js
import * as hello from "https://cdn.example.com/hello.js";
```

However, we no longer guarantee reproducibility. If
`https://cdn.example.com/hello.js` changes its semantics, the web site, a user
of `index.js.abcd.wbn`, might break. Especially, this can be problematic if an
external dependency's _content_ is out of control.

If you can control an undeclared external resource's API stability somehow, this
might not be a big issue. For example, `index.js` can hard-code the specific
version of `hello.js`, such as:

```js
import * as hello from "https://cdn.example.com/hello@4.12.js";
```

However, this proposal suggests that this kind of hard-coding should be avoided,
and such a dependency information should be specified outside of source code as
much as possible.

We can see many similar patterns in other package management systems, such as
node ([package.json]), Rust ([Cargo.toml][cargo]), deno ([deps.ts]).

This proposal aims to support this pattern by making it possible to declare
dependencies inside of a bundle.

### Any recommendations for bundle's granularity? How should we group resources into bundles?

That's totally up-to you. There are various factors and trade-off to make a
decision.

For example, if you want to make an initial loading faster, with a cold start,
you probably want to make the size of a bootstrap bundle smaller; let it include
only minimum inline resources which are required for [first contentful paint].

```
toppage
└── https://example.com/app/index.js.abcd.wbn
    ├── index.js (main-resurce, which should be small)
    ├── support-file-for-initial-loading.js (which should be small)
    ├── ...
    ├── https://example.com/lazy-load-1.js.0088.wbn (the size might be big)
    │   └── ...
    ├── https://example.com/lazy-load-2.js.0089.wbn (the size might be big)
    │   └── ...
    └── https://example.com/lazy-load-3.js.008a.wbn (the size might be big)
        └── ...
```

As another example, if you have:

- resources which wouldn't be updated frequently
- and resources which would be updated frequently, such as _daily update_.

In this case, you might want to group these resources separately in order to
reduce the total bytes transferred in each day.

For example, a site may serve the following bundles today:

```
toppage
   https://example.com/app/index.js.abcd.wbn
     index.js (main-resurce)
     https://example.com/daily-hot-contents.00a0.wbn
       ...
     https://example.com/cold-contents.js.00a1.wbn
       ...
```

If the `daily-hot-contents` is updated in the next day, a site re-packages their
resources as follows:

```
toppage
└── https://example.com/app/index.js.c0e3.wbn (changed)
    ├── index.js (main-resurce)
    ├── https://example.com/daily-host-contents.00c0.wbn (changed)
    │   └── ...
    └── https://example.com/cold-contents.js.00a1.wbn (didn't change)
        └── ...
```

Then, the total byte transferred in the next day is nearly the sum of `index.js`
and `https://example.com/daily-host-contents.00c0.wbn` because the browser
probably can load `cold-contents.js.00a1.wbn`, whose hash didn't change, from
its cache.

### I would like to separate code into bundles for various reasons, but I'm afraid this causes too many round-trips if bundles are nested and discovered one bye one. Is there any good way to avoid it?

A possible workaround as a user-land solution would be to declare indirect
dependent bundles as well as direct dependent bundles.

In the example case, the top page may declare:

<!-- prettier-ignore -->
```html
<!-- This is mandarory -->
<link rel=webbundle href=https://example.com/app/index.js.abcd.wbn resources="index.js" >
<!-- The followings are optional, as a hint for a browser to prefetch -->
<link rel=webbundle href=https://cdn.example.com/date-util.js.f1f2.wbn>
<link rel=webbundle href=https://cdn.example.com/common-util.js.3b3b.wbn>
```

The second and third link elements are not mandatory because they are not a
direct dependency, but declaring them should guarantee that a browser prefetch
these bundles in parallel, instead of on-demand basis.

### How can we create CAB? Is there any tool?

Nothing yet.

In the future, we hope that js bundlers provide an option to use Content
Addressable Bundles as its output format.

Also, it would be nice that CDN supports a CAB format to serve their contents so
that we can link to their CAB from our application bundles.

### How can _mutable_ bundles and _immutable_ bundles (CAB) interact each other?

TODO(hayato): We might want to _backport_ a declarative dependency capability to
mutable bundles as well as CAB.

We might want to let them refer to each other, with some restrictions:

- Any bundles can refer to immutable bundles.
- Only mutable bundles can refer to mutable bundles.

We'll explore this problem space, and update this section.

### How can this proposal support the [WebBundles for Ad Serving](https://github.com/WICG/webpackage/issues/624) use case?

CAB is probably inappropriate for this use case, however, you can continue to
use a _mutable_ bundle for this use case.

You can always use a _mutable_ bundle if the strictness of CAB doesn't meet your
requirements.

### The Web Platform already has [Subresource integrity (SRI)][subresource integrity] feature. How is this proposal relate to SRI?

You may use SRI for a bundle as follows (Note: SRI is not currently supported):

```html
<link
  rel="webbundle"
  href="https://example.com/app/mutable-bundle.wbn"
  integrity="sha384-oqVuAfXRKap7fdgcCY5uykM6+R9GqQ8K/uxy9rx7HNQlGYl1kPzQho1wx4JwY8wC"
  ...
/>
```

However, this usage of SRI might be inappropriate for bundles use cases because:

- A bundle usually contains more than one resources. A browser wants to start to
  load and process each subresource immediately when a necessary byte range of
  each subresource arrives, utilizing streaming.

- To make a streaming possible, a bundle's hash should be conceptually a hash
  for an _index_ section of a bundle. A browser verifies an _index_ section,
  which is placed before a _responses_ section in [Web Bundles
  format][web bundles], and can get a hash for each subresource before their
  bodies arrive.

TODO(hayato): Figure out whether [Merkle Integrity Content Encoding] is
applicable or not.

### It seems weird to use a file name convention to map a main-resource URL to a bundle's URL, and vice-versa. Can we have a more flexible way?

This proposal's current approach is tentative. We may want to introduce a more
flexible way to map a main-resource URL(s) (maybe one-or-more) to a bundle's
URL. Mapping should be declarative so that static analysis tools for a bundle
can understand it easily without any _runtime_.

TODO(hayato): Any requirements or feature requests for mapping? Feedback is
welcome.

## Stakeholder Feedback / Opposition

Not yet.

## References & acknowledgements

- [Get started with Web Bundles]
- [Web Bundles]
- [Subresource Loading with Web Bundles]
- [Resource Bundles]
- [Resource Batch Preloading]
- [Bundling for the Web]
- [Subresource Integrity]
- [Webpack]
- [Browserify]
- [Import maps]
- [Dynamic bundle serving with Web Bundles]
- [NixOS]
- [Cargo]
- [Merkle Integrity Content Encoding]

[webpack]: https://webpack.js.org/
[browserify]: http://browserify.org/
[skypack]: https://www.skypack.dev/
[git]: https://git-scm.com/
[get started with web bundles]: https://web.dev/web-bundles/
[wicg/webpackage]: https://github.com/WICG/webpackage
[subresource loading with web bundles]:
  https://github.com/WICG/webpackage/blob/main/explainers/subresource-loading.md
[resource bundles]: https://github.com/WICG/resource-bundles
[bundling for the web]: https://lowentropy.net/posts/bundles/
[web bundles]:
  https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html
[dynamic bundle serving with web bundles]:
  https://docs.google.com/document/d/11t4Ix2bvF1_ZCV9HKfafGfWu82zbOD7aUhZ_FyDAgmA/edit
[nixos]: https://nixos.org/
[cargo]: https://doc.rust-lang.org/cargo/
[package.json]: https://docs.npmjs.com/files/package.json/
[deps.ts]:
  https://deno.land/manual/linking_to_external_code#it-seems-unwieldy-to-import-urls-everywhere
[first contentful paint]: https://web.dev/first-contentful-paint/
[code splitting]:
  https://developer.mozilla.org/en-US/docs/Glossary/Code_splitting
[import maps]: https://github.com/WICG/import-maps
[resource batch preloading]:
  https://gist.github.com/littledan/e01801001c277b0be03b1ca54788505e
[subresource integrity]:
  https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity
[merkle integrity content encoding]:
  https://tools.ietf.org/html/draft-thomson-http-mice-03
