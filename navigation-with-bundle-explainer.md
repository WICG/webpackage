# Explainer: Navigation with Web Bundles<br>(a.k.a. Bundled HTTP Exchanges)

Last updated: Oct 29, 2019

(This explainer was originally written and published as: https://docs.google.com/document/d/1KFmtiE3DHgKfQH5-nKtLiacMrXsoKIXQZ-VIMGHMje0/edit?pli=1#)

A [demo video](https://www.youtube.com/watch?v=rs-3R0ji6dA) that tries to show one of the use-case scenarios of Web Bundles (a.k.a. Bundled HTTP Exchanges):

<a href="https://www.youtube.com/watch?v=rs-3R0ji6dA">
<img src="https://img.youtube.com/vi/rs-3R0ji6dA/0.jpg">
</a>

Today, loading a website basically means retrieving multiple resources from one or multiple web servers. This gives web a great strength of linkable, indexable, composable and ephemeral, but it is also making it difficult for a website to:

* be shared and loaded in a self-contained way, similar to what packaged apps and other portable data representation formats provide,
* be composed as a distributable, installable application, or
* load reliably and quickly even when a site consists of a large number of subresources, and even when the internet access is limited.

Instead, imagine if we could bundle up a “full” website in a single resource file, so that the website could be shared via a SD card or over some p2p protocol, or could be retrieved from a fast cache or a nearby proxy. It would open up various interesting use cases.

**Web Bundle** is a proposal that tries to achieve it. It’s a format that can represent a collection of HTTP resources, and therefore can represent one or multiple web pages (which typically consist of multiple resources, e.g. HTML, javascripts, images and styles) in a single file.  It’s a part of [Web Packaging](https://github.com/WICG/webpackage) proposal and is also known as "**Bundled HTTP Exchanges** ([spec proposal](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html))."

This document explains how a Web Bundle can be a navigation target, so that a user can open a given Web Bundle in browsers (UAs) and navigate into the website represented by the Bundle.

## Example Scenarios: Save and browse an unsigned content
UA can provide **‘Save as bundle’** feature that will dynamically generate an unsigned bundle of the page that a user is currently browsing.  Then anyone can use the feature to get a bundled representation of the current page, which can be browsed later by the same user, or can be shared with a nearby friend via some person-to-person sharing mechanism.

When someone else browses the unsigned bundle, they see some notation in their URL bar that isn’t the URL of the original URL, but they can still browse around and see the JS executing.  Note that in this case the page coming from the unsigned bundle is not given any access to the cookie/storage of the original site.


## Example Scenarios: Publishing a signed content
The author of a site creates a bundle for one or a few pages of the site, signs the bundle with the site’s certificate, and then publish the bundle in the way that interested users can find it.

Later, when a user finds the bundle, they can navigate their UA client to the bundle, maybe by opening it as a local file copied over SD card or by navigating to the distribution URL where the bundle is published.  The UA should be able to parse and verify the bundle’s signature, and then to navigate to the website represented by the bundle, without actually connecting to the site as all the necessary subresources could be served by the bundle.  If the bundle represents multiple pages for the site, the user should also be able to browser those pages without worrying about the connectivity.

# Navigation

When UA [navigates](https://html.spec.whatwg.org/multipage/browsing-the-web.html#navigate) to a WBN resource, i.e. a resource that's in [Web Bundle format](https://jyasskin.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html), the UA parses the data and extracts **primaryUrl** from the [Bundle’s metadata](https://jyasskin.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#rfc.section.2.2).  This document refers to the URL where the WBN is served at as **distributorUrl** in the later sections.

The UA computes a range of URLs that's called **unsigned scope** based on the **distributorUrl** and its path, with a restriction analogous to a [Service Worker's scope restriction](https://w3c.github.io/ServiceWorker/#path-restriction).  The scope may also be explicitly given by the [manifest's scope field](https://w3c.github.io/manifest/#scope-member), where same path restriction applies (i.e. the scope can only narrow the scope but can’t specify upper path).

[TBD] We can consider allowing Servers to remove this path restriction by setting a new header that is similar to [Service-Worker-Allowed](https://w3c.github.io/ServiceWorker/#service-worker-allowed).

This scope will always be same-origin with the **distributorUrl** and controls which unsigned resources if any are drawn from the bundle.

The UA makes an internal redirect (needs definition, see [fetch/issues/576](https://github.com/whatwg/fetch/issues/576)) to the Bundle's **primaryUrl** during [process a navigate fetch](https://html.spec.whatwg.org/multipage/browsing-the-web.html#process-a-navigate-fetch). If the **distributorUrl** is [potentially trustworthy](https://w3c.github.io/webappsec-secure-contexts/#is-url-trustworthy), it also stashes the bundle and its unsigned scope in the request.

While loading the **primaryUrl**, if any of the following is true:
* The primaryUrl is within the **unsigned scope* (e.g. `https://example.com/~foo/article.wbn` cannot serve a Bundle for `https://example.com/~bar/`, but only for the resources under `https://example.com/~foo/`),
* The Bundle has a [valid signature](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#signatures-section) for the primaryUrl, or
* Loaded from an a priori trusted location. (E.g. loaded from a special pre-install location)


Then the UA performs the fetch using the response that is the result of [load a response from a bundle](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html#rfc.section.3.4) for the **primaryUrl**.  The resulting browsingContext will be created as a [Secure Context](https://w3c.github.io/webappsec-secure-contexts/) of the origin of the primaryUrl.

Otherwise (the content does not prove its authenticity), the navigation must fail unless the Bundle is loaded from a unique-origin context, e.g. from `file:///`, and the target browsingContext’s origin must be set to that of `file:/// URL`, i.e. a unique origin.

Once the navigation succeeds, the Web Bundle (metadata, signature and responses) is attached as **bundledResources** to the browsingContext the UA is navigating to.

## Fetch with Bundles

If the browsingContext that has a non-null **bundledResources** issues a fetch request that meets the following conditions:

* The request’s method is GET,
* The request’s URL matches with one of the URLs in the request map in the **bundledResources**, and the [Variants algorithm](https://httpwg.org/http-extensions/draft-ietf-httpbis-variants.html#cache) matches the request headers with the Variants header in the bundle,
* Either of the following is true:
  * The request is for same-origin and the requesting URL’s path is restricted by the path of the Bundle URL,
  * The corresponding response that is in the **bundledResources** has a valid (non-expired) signature, or
  * The initiator origin is not created as a [Secure Context](https://w3c.github.io/webappsec-secure-contexts/). In this last case the resource is loaded as coming from a unique origin.

Then the UA retrieves the corresponding response from the Bundle and attach it to the request as a [stashed exchange](https://wicg.github.io/webpackage/loading.html#request-stashed-exchange) (Note: we might want to stash the whole bundledResources, but we’re not crystal clear on that idea yet).  Note that the exchange is already a **parsedExchange** here, and only be used in [HTTP-network-or-cache](https://wicg.github.io/webpackage/loading.html#mp-http-network-or-cache-fetch) fetch step (but not in [HTTP fetch](https://wicg.github.io/webpackage/loading.html#mp-http-fetch)).

## Navigation with Inherited Bundles
If a request is a navigation request or a worker request and if the response was from the BundledResources, the initiator’s bundledResources are copied to the target browsingContext.

## Unique-origin Web Bundle Loading
When a Bundle without a valid signature is loaded from a unique origin (e.g. a local file), following URL conversion must be performed:

* Set the document’s URL to the unique URL of the Bundle package's one. 
   * See [@jyasskin](https://github.com/jyasskin)'s [Origins for Resources in Unsigned Packages](https://docs.google.com/document/d/1BYQEi8xkXDAg9lxm3PaoMzEutuQAZi1r8Y0pLaFJQoo/edit#heading=h.1fej4450b9k9) to see what's being proposed.
   * Chrome Canary 80 has an [experimental implementation of this feature](https://chromium.googlesource.com/chromium/src/+/refs/heads/master/content/browser/web_package/using_web_bundles.md), and it creates a URL for a local Bundle by concatenation of the location (URL) of the Bundle, `?` and the primaryUrl of the Bundle (that must be properly URL-encoded). (e.g. if the primaryUrl is `https://bar.com/article.html`, the document’s URL would be shown as `file:///foo/bar.wbn?https://bar.com/article.html`.)
   * TODO: also look at what TAG did in the original packaging spec: https://www.w3.org/TR/2015/WD-web-packaging-20150115/#fragment-identifiers
* Set the document’s base URL to the primaryUrl of the Bundle.  This allows the fetch with Bundles work for relative URLs, while the security origin of the document should be just remained as unique (and therefore should not be a source of XSS)
