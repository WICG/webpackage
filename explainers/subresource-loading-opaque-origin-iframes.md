# Subresource loading with Web Bundles: Support opaque origin iframes

Last updated: Apr 2021

This is an extension to [Subresource loading with Web Bundles]. This extension
allows a bundle to include `urn:uuid:` URL resources, which will be used to
create an opaque origin iframe.

## Goals

Support the use case of
[WebBundles for Ad Serving](https://github.com/WICG/webpackage/issues/624).

## Extension to [Subresource loading with Web Bundles]

In this section, _the explainer_ means the [Subresource loading with Web
Bundles] explainer.

### Allow `urn:uuid:` resources

In addition to the same origin subresource explained in the
[`<link>`-based API](https://github.com/WICG/webpackage/blob/main/explainers/subresource-loading.md#link-based-api)
section in the explainer, this extension allows a bundle to include a
[`urn:uuid:`](https://tools.ietf.org/html/rfc4122) URL subresource.

### Opaque origin iframes

If a `<iframe>`'s `src` attribute is a `urn:uuid:` URL subresource in the
bundle, the iframe must be instantiated as an
[opaque origin](https://html.spec.whatwg.org/multipage/origin.html#concept-origin-opaque)
iframe.

## Example

### The bundle

Suppose that the bundle, `subresources.wbn`, includes the following resources:

```
- urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6
- … (omitted)
```

### The main document

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  resources="urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6"
/>

<iframe src="urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6"></iframe>
```

`urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6` is loaded from the bundle, and a
subframe is instantiated as an
[opaque origin](https://html.spec.whatwg.org/multipage/origin.html#concept-origin-opaque)
iframe.

Note:

- `urn:uuid:` resource must be explicitly specified in `resources` attribute in
  `<link>` elements, similar to other subresources. `scopes` attribute can be
  also used for `urn:uuid:` resources. For example, `scopes=urn:` allows all
  `urn:` resources.

### Content Security Policy (CSP)

Regarding `urn:uuid` resources in a bundle, a CSP restriction must be
evaluated against the source of the bundle, instead of the literal
`urn:uuid` URL.

See an issue [#651](https://github.com/WICG/webpackage/issues/651) for
the motivation.

[subresource loading with web bundles]:
  https://github.com/WICG/webpackage/blob/main/explainers/subresource-loading.md
