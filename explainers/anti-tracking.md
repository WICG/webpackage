# Anti-tracking in signed packages

<!-- TOC depthTo:3 -->

- [Blocking credentials](#blocking-credentials)
  - [Possible spellings](#possible-spellings)
  - [Tentatively-rejected spellings](#tentatively-rejected-spellings)
- [Blocking normal link decoration](#blocking-normal-link-decoration)
- [Blocking privileged link decoration](#blocking-privileged-link-decoration)
- [Privileged server-side collaboration](#privileged-server-side-collaboration)
- [Appendices](#appendices)
  - [Security concerns](#security-concerns)
  - [Behavior on an unexpected response](#behavior-on-an-unexpected-response)

<!-- /TOC -->

This explainer attempts to solve the problems identified in issues
[#422](https://github.com/WICG/webpackage/issues/422) and
[#423](https://github.com/WICG/webpackage/issues/423). Specifically, we're
worried that a distributor might be able to use the fact that it can choose
which exchange or package, signed by a publisher, to serve in response to a
client's request, to transfer its notion of the user's identity to the
publisher.

The approach this document takes is to prevent the distributor from accessing
its notion of the user's identity. There are several routes by which a
distributor could learn the user's identity:

1. The browser could send the distributor its top-level credentials. [Countermeasures](#blocking-credentials)
1. The source of a link could use the same link-decoration tools as it might use
   in a cross-origin link, which other efforts are blocking. Note that the
   source of a link and the distributor of a package could be the same origin.
   [Countermeasures](#blocking-normal-link-decoration)
1. The source of a link could decorate links to its distributor to send a user
   ID in ways an unrelated cross-origin target couldn't be expected to receive.
   [Countermeasures](#blocking-privileged-link-decoration)
1. The source of a link could collaborate with its distributor on the server
   side to send a user ID in ways an unrelated cross-origin target couldn't be
   induced to collaborate.
   [Discussion](#privileged-server-side-collaboration)

These are tackled individually in the following sections.

## Blocking credentials

The navigation to a signed package (or exchange) must be done without
credentials. This is roughly [credentials
mode](https://fetch.spec.whatwg.org/#concept-request-credentials-mode) ==
`"omit"`, with changes to Fetch to make it work for navigations.

There are [security concerns](#security-concerns) for allowing an attacker to
load an arbitrary site without credentials, so if the target of an
uncredentialed navigation isn't a signed package, the navigation should
[fail](#behavior-on-an-unexpected-response).

There are lots of possible spellings for a navigation that omits credentials:

### Possible spellings

<a id="credentials-omit"></a>

#### An attribute to set [credentials-mode](https://fetch.spec.whatwg.org/#concept-request-credentials-mode) to "omit"

```html
<a href="https://target" credentials="omit">
```

This seems like the most straightforward way to say what we currently want, but
it doesn't automatically adapt if we later decide that package navigations
should differ from other navigations in an additional way.

#### An attribute to set the [_init_ parameter to `fetch()`](https://developer.mozilla.org/en-US/docs/Web/API/WindowOrWorkerGlobalScope/fetch#Parameters)

```html
<a href="https://target" fetchoptions='{"credentials": "omit"}'>
<a href="https://target" fetchoptions="credentials: omit">
```

We'd add this to all fetch-causing elements, which would unify the current
`crossorigin=""`, `referrpolicy=""`, and `integrity=""` attributes, and be a
better future extension point. The two hard parts here are

1. Getting people to swallow putting JSON in an attribute, or alternately coming
   up with a new microsyntax that has enough capabilities.
2. Checking all of the options for possible security holes.

This is either more attractive than the [`credentials`
attribute](#credentials-omit) because it solves a larger class of problems, or
less attractive because it introduces more possible avenues of abuse.

#### An attribute that declares the target is a package

```html
<a href="https://sxg" ispackage>
```

Just tell the browser directly to expect a package. For now, this would do the
same thing as [credentials="omit"](#credentials-omit) above.

#### Rely on prenavigate

```html
<a href="https://signed_inner_url">
```

If there has already been a
[prenavigate](https://github.com/w3c/resource-hints/issues/82#issuecomment-529951528)
that found a package holding the mentioned URL, it uses the content of that
package for the navigation. If there hasn't been such a prenavigate, it acts
like a normal link.

#### An attribute to announce the expected inner URL

```html
<a href="https://package/path" publisherurl="https://signed/inner/url">
```

An attribute declares the expected `start_url` of the package (or inner url of a
signed exchange). If these don't match, [the navigation
fails](#behavior-on-an-unexpected-response).

<a id="fetchfrom-attribute">

#### An attribute to declare a package that contains the link target

```html
<a href="https://signed/inner/url" fetchfrom="https://package/path">
```

This naturally works on browsers that haven't implemented packages. It also
allows any browser to skip the package if it wants to make sure the publisher is
notified/checked/etc. Again, if the package's content doesn't match the href,
[the navigation fails](#behavior-on-an-unexpected-response).

#### An attribute to declare only a distributor origin

```html
<a href="https://signed/inner/url" distributor="https://distributor.origin">
```

This relies on us defining a single path within `distributor.origin` that serves
a given signed URL, as [Blocking privileged link
decoration](#blocking-privileged-link-decoration) suggests. It has similar
properties to the [`fetchfrom`](#fetchfrom-attribute) option otherwise.

This is neither better or worse from the [`fetchfrom`
attribute](#fetchfrom-attribute) from a technical perspective: `fetchfrom` can
reject paths that don't match the one required by [Blocking privileged link
decoration](#blocking-privileged-link-decoration), and if we ever decide to
relax the single-path restriction, the `distributor` attribute could begin to
allow a full path.

### Tentatively-rejected spellings

#### Re-use the `crossorigin` attribute

```html
<a href="https://target" crossorigin="no-credentials">
```

This would have the same semantics as `<a href="https://target"
credentials="omit">`, but it would also apply to same-origin navigations. It
seems really confusing to use a "crossorigin" attribute to modify same-origin
navigations.

#### Mark packages with a scheme

```html
<a href="package://package">
```

Basically encodes the `ispackage` flag into the scheme of the URL. The package
is fetched by changing the scheme to `https`. We're [developing a more general
package scheme](https://lists.w3.org/Archives/Public/uri/2019Nov/0000.html), but
as it's useful for unsigned packages, which don't necessarily need the same
restrictions as signed package, it seems like the scheme won't do the right
thing.

## Blocking normal link decoration

Same-origin links to packages should use the same restrictions as cross-origin
links to non-packages.

## Blocking privileged link decoration

A link source run by the same entity as the distributor can encode a user ID in
ways that might break an arbitrary publisher's serving code. So, we need to
define a deterministic function from the inner URL to the acceptable URL within
a distributor used to fetch that inner URL's package.

The two main options are

```url
https://distributor.origin/.well-known/package/<hash(innerUrl)>
https://distributor.origin/.well-known/package/innerAuthority/innerPath?innerQuery
```

The hash-based path is a bit easier to compute, but leaving the inner URL mostly
intact makes it easier for a human to guess where the link is going to go.

## Privileged server-side collaboration

The primary mechanism by which a distributor's server could collaborate with a
link source's server is by deciding that if there's a click by user `A` at time
`T` on a link to `https://distributor.example/package`, then a subsequent
uncredentialed load of that package at time `T+ε` is probably user `A`. Servers
can optionally refine this by limiting it to clicks and loads from the same IP
address. This same mechanism allows cross-origin user ID transfer between
arbitrary servers, and we don't know any ways to block it in that context, so we
only need to make the package-assisted transfer at least as difficult as the
cross-server transfer.

We know of two ways for a distributor to pass their user ID onto the publisher
of a package, and in either case—the publisher shares their signing key with the
distributor, or the publisher generates 2^N variants of a package to encode N
bits of a user ID—it seems safer or simpler for the publisher to share their
logs instead.

## Appendices

### Security concerns

There are worries that uncredentialed navigation could be used in an attack on
the target site, since the main request won't send or save cookies, but
subrequests will. We don't actually know a concrete attack that's enabled by
uncredentialed navigations, but we'd like to prevent surprises anyway.

Similar worries came up in the discussion of cross-origin prefetch, which led to
the new `"prenavigate"` operation, for which the target site opts in using an
[`Allow-Uncredentialed-Navigation`
header](https://github.com/w3c/resource-hints/issues/82#issuecomment-529951528).

### Behavior on an unexpected response

If a link declares that its target is a package or has a particular inner URL,
and then the actual response isn't a package or has a different inner URL, what
should the browser do?

* If it fails the navigation with a network error, link sources would quickly
  discover their error and correct it, but it means that a distributor could
  never change the content of a URL that once contained a package.
* If it reloads the outer URL as a normal navigation or redirects to the claimed
  inner URL, that wastes network traffic and time, but gives users a smoother
  experience.

Security folks generally prefer to fail fast, so the rest of this explainer
suggests that option.
