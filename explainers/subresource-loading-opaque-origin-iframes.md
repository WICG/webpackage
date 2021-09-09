# Subresource loading with Web Bundles: Support opaque origin iframes

Last updated: Sep 2021

This is an extension to [Subresource loading with Web Bundles]. This extension
allows a bundle to include `uuid-in-package:` URL resources, which will be used to
create an opaque origin iframe.

## Goals

Support the use case of
[WebBundles for Ad Serving](https://github.com/WICG/webpackage/issues/624).

## Extension to [Subresource loading with Web Bundles]

In this section, _the explainer_ means the [Subresource loading with Web
Bundles] explainer.

### Allow `uuid-in-package:` resources

This extension introduces a new URL scheme `uuid-in-package:` that can be used in
resource URLs in web bundles. The `uuid-in-package:` URL has the following syntax:
```
<uuid-in-package> ::= "uuid-in-package:" <UUID>
```
Where `<UUID>` is a UUID as specified in
[RFC 4122](https://datatracker.ietf.org/doc/html/rfc4122).

In addition to the same origin subresource explained in the
[`<link>`-based API](https://github.com/WICG/webpackage/blob/main/explainers/subresource-loading.md#link-based-api)
section in the explainer, this extension allows a bundle to include a
`uuid-in-package:` URL subresource.

### Opaque origin iframes

If a `<iframe>`'s `src` attribute is a `uuid-in-package:` URL subresource in the
bundle, the iframe must be instantiated as an
[opaque origin](https://html.spec.whatwg.org/multipage/origin.html#concept-origin-opaque)
iframe.

## Example

### The bundle

Suppose that the bundle, `subresources.wbn`, includes the following resources:

```
- uuid-in-package:f81d4fae-7dec-11d0-a765-00a0c91e6bf6
- … (omitted)
```

### The main document

```html
<link
  rel="webbundle"
  href="https://example.com/dir/subresources.wbn"
  resources="uuid-in-package:f81d4fae-7dec-11d0-a765-00a0c91e6bf6"
/>

<iframe src="uuid-in-package:f81d4fae-7dec-11d0-a765-00a0c91e6bf6"></iframe>
```

`uuid-in-package:f81d4fae-7dec-11d0-a765-00a0c91e6bf6` is loaded from the bundle, and a
subframe is instantiated as an
[opaque origin](https://html.spec.whatwg.org/multipage/origin.html#concept-origin-opaque)
iframe.

Note:

- `uuid-in-package:` resource must be explicitly specified in `resources` attribute in
  `<link>` elements, similar to other subresources. `scopes` attribute can be
  also used for `uuid-in-package:` resources. For example, `scopes=uuid-in-package:` allows all
  `uuid-in-package:` resources.

### Content Security Policy (CSP) for `uuid-in-package` resources

Using the `uuid-in-package` URLs in CSP's
[matching rule](https://w3c.github.io/webappsec-csp/#match-url-to-source-expression)
is almost useless from a security standpoint, because anyone can use arbitrary
`uuid-in-package` URLs.
So the CSP restrictions must be evaluated against the source of the bundle
instead of to the `uuid-in-package` URL.

For example, given this CSP header,

```
Content-Security-Policy: script-src https://cdn.example; frame-src https://cdn.example
```

the page can load `uuid-in-package` resources in web bundles served from
`https://cdn.example`.

```
<link rel="webbundle"
  href="https://cdn.example/subresources.wbn"
  resources="uuid-in-package:429fcc4e-0696-4bad-b099-ee9175f023ae
             uuid-in-package:f81d4fae-7dec-11d0-a765-00a0c91e6bf6"
/>

<script src="uuid-in-package:429fcc4e-0696-4bad-b099-ee9175f023ae"></script>
<iframe src="uuid-in-package:f81d4fae-7dec-11d0-a765-00a0c91e6bf6"></iframe>
```

Note:
- When loading `HTTPS` resources from web bundles, the CSP restrictions must be
  evaluated against the resource URL, not against the bundle URL.
- Loading `uuid-in-package` resources from web bundles served from HTTPS server is
  allowed when "\*" is set in the CSP
  [source expression](https://w3c.github.io/webappsec-csp/#source-expression).
  This is different from the CSP behavior that `data:` and `blob:` schemes are
  excluded from matching a policy of "\*". Loading `uuid-in-package` resources from web
  bundles is safer than using `data:` or `blob:` URL resources which are
  directly under the control of the page, because a `uuid-in-package` resource is a
  reference to a component of something with a globally-accessible URL. So we
  don't need to exclude `uuid-in-package` resources in a web bundle from matching the
  policy of "\*".
- See an issue [#651](https://github.com/WICG/webpackage/issues/651) for the
  detailed motivation.  

## Alternatives Considered

### urn:uuid: resources
Previous version of this document used
[`urn:uuid:` URLs](https://datatracker.ietf.org/doc/html/rfc4122) instead of
`uuid-in-package:`. However, the `urn:` scheme is included in the
[safelisted schemes](https://html.spec.whatwg.org/multipage/system-state.html#safelisted-scheme)
of the HTML spec, meaning that web sites can
[register a custom protocol handler](https://html.spec.whatwg.org/multipage/system-state.html#custom-handlers)
that handles `urn:` scheme. To avoid the potential for conflict, this extension
introduces the new `uuid-in-package:` scheme.

Note that Chromium's experimental implementation currently supports only
`urn:uuid:` as of M95.

[subresource loading with web bundles]:
  https://github.com/WICG/webpackage/blob/main/explainers/subresource-loading.md
