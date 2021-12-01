# Bundle Dependencies

Last update: Dec 2021

Status: Proposal (Draft)

This proposal outlines the format and usage of the `dependencies` optional
section as a part of a [Web Bundle][Web Bundle].

# Authors

- Hayato Ito (hayato@google.com)

# Introduction

[Subresource loading with Web Bundles][Subresource loading with Web Bundles]
provides a way to load a large number of resources efficiently using a
[Web Bundle][Web Bundle]. However, it's not always appropriate for a web site to
deliver their resources in one giant web bundle.

There are several reasons to split their resources into several bundles.
[Code Splitting][Code Splitting], splitting code into various bundles which can
be loaded on demand or in parallel, instead of having one giant bundle, if used
correctly, has many advantages, such as:

- To prevent downloading ginormous files on initial load.
- On demand lazy-loading only when a user requests it.
- Separate common code into a separate bundle so that they can be reloaded from
  a browser's cache, instead of from the network, on page re-loading or page
  navigation.

Some bundlers like [Webpack][Webpack] and [Browserify][Browserify] support
[Code Splitting][code splitting], however,
[Subresource loading with Web Bundles][subresource loading with web bundles]
doesn't provide any support for [Code Splitting][code splitting] yet.

This proposal explain a new `dependencies` section, which provides a way to
express dependencies between _this_ bundle and _other_ bundles, aiming more
flexibility of how resources are grouped together as bundles. Bundlers can
express a dependency graph explicitly in bundles themselves.

In actual loading time, a browser reads the `dependencies` section in a web
bundle and can get benefits. A browser can preload dependent bundles,
recursively, or fetch dependent bundles on-demand basics. Behaviors can be
specified in `dependencies` section in a declarative way. No configuration is
required outside of a bundle, thus a bundle becomes more _self-contained_.

# Format

TODO(hayato): Define a `dependencies` section format in [CDDL][CDDL]. Meanwhile,
this section explains a non-normative _conceptual_ format.

The `dependencies` section maps a `resource URL` to a tuple of
(`web bundle URL`, `load type`). That declares that `resource URL` should be
loaded from an external bundle, `web bundle url`.

For example,

| resource URL                    | web bundle URL                   | load type |
| ------------------------------- | -------------------------------- | --------- |
| ./a.js                          | ./a.wbn                          | preload   |
| ./b.js                          | ./b.wbn                          | preload   |
| ./optional.js                   | ./optional.wbn                   | lazy      |
| https://cdn.example.com/util.js | https://cdn.example.com/util.wbn | preload   |

- `load type` is either `preload` or `lazy`.
- If `resource URL` is absolute, `web bundle URL` must be absolute.
- If `resource URL` is relative, `web bundle URL` must be relative.
- `resource URL` and its mapped `web bundle URL` must follow a _path
  restriction_ rule.

  For example, the following is fine.

  | resource URL | web bundle URL | load type |
  | ------------ | -------------- | --------- |
  | ./dir/foo.js | ./foo.wbn      | preload   |

  The following is illegal.

  | resource URL | web bundle URL | load type |
  | ------------ | -------------- | --------- |
  | ./foo.js     | ./dir/foo.wbn  | preload   |

# Usage

Suppose that a web site has two top pages, `main1.html` and `main2.html`. They
use several resources.

The resource dependency graph is:

![resource dependencies](./assets/dependencies-section-resource-dep.svg)

Note that each `*.js` file here might be the result of combining one-or-more
fine-grained raw ES module files used in development time. This proposal doesn't
care the content. It's totally up to a web site how they decide granularity of
each resource, considering various factors.

Considering the usage patterns, they split their resources into the following
five bundles:

![bundles dependencies](./assets/dependencies-section-bundle-dep.svg)

## A (a.wbn)

The bundle, `a.wbn`, has the following resources, which are required for FCP
(First Contentful Paint), for `main1.html`:

| URL     |
| ------- |
| ./a1.js |
| ./a2.js |
| ./a3.js |
| ./a.png |
| ./a.css |

and the following `dependencies` section:

| resource URL | web bundle URL | load type |
| ------------ | -------------- | --------- |
| ./b1.js      | ./b.wbn        | preload   |
| ./c1.js      | ./c.wbn        | preload   |

## B (b.wbn)

The bundle, `b.wbn`, has the following resources:

| URL     |
| ------- |
| ./b1.js |
| ./b2.js |
| ./b3.js |

## C (c.wbn)

The bundle, `c.wbn`, has the following resources:

| URL     |
| ------- |
| ./c1.js |
| ./c2.js |
| ./c3.js |

and the following `dependencies` section:

| resource URL | web bundle URL | load type |
| ------------ | -------------- | --------- |
| ./d1.js      | ./d.wbn        | lazy      |

## D (d.wbn)

The bundle, `d.wbn`, has the following resources:

| URL     |
| ------- |
| ./d1.js |
| ./d2.js |
| ./d3.js |

## E (e.wbn)

The bundle, `e.wbn`, has the following resources, which are required for FCP
(First Contentful Paint), for `main2.html`:

| URL     |
| ------- |
| ./e1.js |
| ./e.png |
| ./e.css |

and the following `dependencies` section:

| resource URL | web bundle URL | load type |
| ------------ | -------------- | --------- |
| ./c1.js      | ./c.wbn        | preload   |

## main page (main1.html)

The main page, `main1.html` includes a _bootstrap_ `<script type=webbundle>`
declaration.

```html
<script type="webbundle">
{
   source: "a.wbn",
   resources: ["a1.js", "a.css", "a.png"]
}
</script>

....

<link rel="style" href="a.css" />

...

<img src="a.png" />

...

<script type="module" src="a1.js" />
```

1. After parsing `<script type=webbundle>`, a browser updates its internal
   `resource-to-webbundle` map as follows, and starts to preload `a.wbn`.

   | resource URL | web bundle URL    |
   | ------------ | ----------------- |
   | ./a1.js      | ./a.wbn (preload) |
   | ./a.css      | ./a.wbn           |
   | ./a.png      | ./a.wbn           |

2. When fetching `a.wbn` is done, the map is updated:

   | resource URL | web bundle URL    |
   | ------------ | ----------------- |
   | ./a1.js      | ./a.wbn           |
   | ./a.css      | ./a.wbn           |
   | ./a.png      | ./a.wbn           |
   | ./a2.js      | ./a.wbn           |
   | ./a3.js      | ./a.wbn           |
   | ./b1.js      | ./b.wbn (preload) |
   | ./c1.js      | ./c.wbn (preload) |

   Note that `a2.js` and `a3.js` are also added. The current
   [Subresource Loading with Web Bundles] doesn't add that. This proposal
   extends the current behavior.

3. Since `load type` of `b.wbn` and `c.wbn` are both `preload`, a browser starts
   to preload `b.wbn` and `c.wbn`.

   TODO: There is an optimization opportunity to fetching `b.wbn` and `c.wbn` in
   _one_ HTTP GET request. That's currently out of scope in this proposal. We
   will explore this idea later.

4. When fetching `b.wbn` and `c.wbn` is done, the map is updated:

   | resource URL | web bundle URL |
   | ------------ | -------------- |
   | ./a1.js      | ./a.wbn        |
   | ./a.css      | ./a.wbn        |
   | ./a.png      | ./a.wbn        |
   | ./a2.js      | ./a.wbn        |
   | ./a3.js      | ./a.wbn        |
   | ./b1.js      | ./b.wbn        |
   | ./c1.js      | ./c.wbn        |
   | ./b2.js      | ./b.wbn        |
   | ./b3.js      | ./b.wbn        |
   | ./c2.js      | ./c.wbn        |
   | ./c3.js      | ./c.wbn        |
   | ./d1.js      | ./d.wbn (lazy) |

5. Although `d1.js` -> `d.wbn` mapping is added to the map, a browser doesn't
   start to preload `d.wbn` because its `load type` is lazy.

   A browser start to fetch `d.wbn` only when a user actually requests `d1.js`,
   such as dynamic import.

   For example, when `userAction()` function, which is defined in `c2.js`, is
   called,

   ```js
   function userAction() {
     const d1 = import("d1.js");
     // ...
   }
   ```

   a browser starts to fetch `d.wbn` and loads `d1.js` from the fetched bundle.

## main page (main2.html)

Suppose that a user visits _another_ main page `main2.html` after a user visits
`main1.html`:

`main2.html`:

```html
<script type="webbundle">
{
   source: "e.wbn",
   resources: ["e1.js", "e.css", "e.png"]
}
</script>

....

<link rel="style" href="e.css" />

...

<img src="e.png" />

...

<script type="module" src="e1.js" />
```

1. A browser updates the internal `resource-to-webbundle` map as follows:

   | resource URL | web bundle URL    |
   | ------------ | ----------------- |
   | ./e1.js      | ./e.wbn (preload) |
   | ./e1.css     | ./e.wbn           |
   | ./e2.png     | ./e.wbn           |

2. When fetching `e.wbn` is done, the map is updated:

   | resource URL | web bundle URL    |
   | ------------ | ----------------- |
   | ./e1.js      | ./e.wbn           |
   | ./e1.css     | ./e.wbn           |
   | ./e2.png     | ./e.wbn           |
   | ./c1.js      | ./c.wbn (preload) |

3. Since a browser caches `c.wbn` when a user visits `main1.html`, a browser
   retrieves `c.wbn` from the cache. A network request doesn't happen.

   | resource URL | web bundle URL       |
   | ------------ | -------------------- |
   | ./e1.js      | ./e.wbn              |
   | ./e1.css     | ./e.wbn              |
   | ./e2.png     | ./e.wbn              |
   | ./c1.js      | ./c.wbn (from cache) |
   | ./c2.js      | ./c.wbn              |
   | ./c3.js      | ./c.wbn              |
   | ./d1.js      | ./d.wbn (lazy)       |

4. Similarly, a browser retrieves `d.wbn`, if `d1.js` is requested, from the
   cache.

   | resource URL | web bundle URL       |
   | ------------ | -------------------- |
   | ./e1.js      | ./e.wbn              |
   | ./e1.css     | ./e.wbn              |
   | ./e2.png     | ./e.wbn              |
   | ./c1.js      | ./c.wbn (from cache) |
   | ./c2.js      | ./c.wbn              |
   | ./c3.js      | ./c.wbn              |
   | ./d1.js      | ./d.wbn (from cache) |
   | ./d2.js      | ./d.wbn              |
   | ./d3.js      | ./d.wbn              |

# FAQ

## If the depth of a bundle dependency graph is deep, this may cause many network round-trip?

Yes. For example, in the following cases:

![bundle dep deep](./assets/dependencies-section-bundle-dep-deep.svg)

There can be 5 network round-trips,
`[main.html -> A], [A -> C], [C -> F], [F -> G], [G -> H]`.

If a web site wants to keep their dependency graph, and wants to reduce the
number of network round-trip, they can let `A`'s `dependencies` section have all
_descendant_ bundles, including indirect dependencies as well as direct
dependencies.

For example, `A (a.wbn)` can have the following `dependencies` section:

| resource URL | web bundle URL | load type |
| ------------ | -------------- | --------- |
| ./b1.js      | ./b.wbn        | preload   |
| ./c1.js      | ./c.wbn        | preload   |
| ./d1.js      | ./d.wbn        | preload   |
| ./f1.js      | ./f.wbn        | preload   |
| ./g1.js      | ./g.wbn        | preload   |
| ./h1.js      | ./h.wbn        | preload   |
| ./i1.js      | ./i.wbn        | preload   |

This is feasible because a server knows the dependency graph statically.

As previously mentioned, there is an optimization opportunity to fetch all
dependent bundles in one HTTP GET request, however, that's out of scope of this
proposal.

<!-- prettier-ignore-start -->
<!-- deno-fmt-ignore-start -->

[cddl]: https://datatracker.ietf.org/doc/html/rfc8610
[web bundle]: https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html
[webpack]: https://webpack.js.org/
[browserify]: http://browserify.org/
[skypack]: https://www.skypack.dev/
[git]: https://git-scm.com/
[wicg/webpackage]: https://github.com/WICG/webpackage
[subresource loading with web bundles]: https://github.com/WICG/webpackage/blob/main/explainers/subresource-loading.md
[bundling for the web]: https://lowentropy.net/posts/bundles/
[code splitting]: https://developer.mozilla.org/en-US/docs/Glossary/Code_splitting
[import maps]: https://github.com/WICG/import-maps
<!-- prettier-ignore-end -->

<!-- deno-fmt-ignore-end -->
