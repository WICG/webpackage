# Anti-tracking Threat Model

WebKit and other browsers are trying to reduce websites' ability to track users
across the internet. While the success of their effort remains to be seen, we
don't want Web Packaging to make the problem any more difficult.

This document contains a proposed threat model for the best-case outcome of that anti-tracking effort. We do not yet claim there is consensus about any of:

1. Whether the [attacker capabilities](#attacker-capabilities) are plausible.
1. Whether it will eventually be possible to frustrate the [attacker goals that
   we want to frustrate](#attacker-goals-that-we-want-to-frustrate).
1. Whether the mitigations actually mitigate the attacks.
1. Whether the costs of the mitigations are worth the benefits.
1. Probably other things.

Nonetheless, we feel it's important to start with a concrete proposal in mind
when discussing how to evolve both this threat model and the Web Packaging
proposals.

Often when we criticize new, technically distinct tracking vectors, we are told
that “you can track users in so many ways so why care about this one?” In the
case of signed packages we hear about means of tracking such as doctored links
where cross-site tracking is built into the URL, or server-side exchanges of
personally identifiable information such as users' email addresses.

Browsers are working hard to prevent cross-site tracking, including with new
limits and restrictions on old technologies. Web Packaging must not add to that work, although it's acceptable for Web Packaging to prevent tracking only after that other work has been done.

Finally, the success of new web technologies, including signed packages, relies
on better security and privacy guarantees than what we've had in the past. We
want progression in this space, not the status quo.

## The Actors

* **The user.** This is the human who relies on the user agent to protect their
  privacy.
* **The user agent.** This is the web browser that tries to protect the user's
  privacy.
* **Distributor.** An entity that delivers a signed package to either the user or another distributor.
  * **AdTech or `adtech.example`.** This is a distributor that the user also
    engages with as a first-party site and that has a financial interest in
    1. knowing what the user does on other websites to augment its rich profile
       of the user and
    2. individual targeting of ads, based on its rich profile of the user.
* **Publisher.** An entity that owns a domain and publishes content there. The
  publisher may not actually author the content, but they put their name on it.
  * **News or `news.example`.** This is a publisher for a news website which
    wants its articles to be served as signed packages with the user agent's URL
    bar showing `news.example`.

## Use Cases

See https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html
for use cases. It's possible some mitigations will sacrifice some use cases, but
those mitigations should call out the use cases they break.

## Attacker Capabilities

1. AdTech has significant first-party traffic which means most users have an
   `adtech.example` cookie holding a unique ID, even in browsers with
   multi-keyed caches.
1. AdTech can convince News to give them a `news.example` certificate's private
   key or to tell a CA that AdTech has permission to receive certificates valid
   to sign exchanges for `news.example`.

   This seems like an implausible capability at first glance, but publishers
   routinely give CDNs this ability to terminate TLS traffic, and a
   generally-trusted AdTech could convince publishers that it's merely doing
   what a good CDN would, for cheaper.
1. `adtech.example` can serve a link to `news.example` and expect users to click
   on it and then browse around `news.example`.
   1. This link can point to a resource that redirects to `news.example`.
   1. This link can point to a signed exchange or web package containing
      content signed by `news.example`'s certificate.
1. AdTech can make any number of signatures with private keys it controls.

## Attacker Non-capabilities

1. AdTech cannot manipulate the DNS system on a per-user basis.
1. There is a limit to the complexity AdTech can convince News to add to its
   serving infrastructure. For example, News is willing to ignore unknown query
   parameters and fragments, but is not willing to ignore unexpected path
   segments. TODO: Can we describe this limit any more precisely?

## Attacker goals that we want to frustrate

1. The user does not want AdTech to be able to augment its profile of them while
   reading articles on `news.example`.
1. The user does not want AdTech's rich profile of them to influence the content
   of ads or articles on `news.example`.
1. More abstractly, AdTech cannot transfer its unique ID for a user to
   the Javascript environment created for `news.example`.
   1. Failing this, the user agent or external auditors should be able to detect
      that AdTech is tracking users.

# Attacks and their Mitigations

This section lists potential ways AdTech might achieve its unwanted goals, along
with proposed or already-adopted mitigations to frustrate those goals. Not all
of the attacks use Web Packaging, so that we can explore the threat model's
implications for the rest of the web platform.

## Sign identifying information into the package

### The Attack

1. AdTech convinces News to give AdTech a package-signing certificate for
   `news.example`, perhaps by offering to handle the technical complications of
   signing packages.
   1. Alternately, News could set up a service that signs packages AdTech sends
      it.
1. When a user clicks a link from `adtech.example` to
   `https://adtech.example/news.example.sxg`, they send identifying information
   to the `adtech.example` server. This could be cookies or a user ID encoded in
   the query, path, or even hostname.
1. Instead of signing `news.example`'s content directly, AdTech embeds the
   user's identity in that content and signs the result on the fly.
1. This successfully transfers the unique ID to the `news.example` JS
   environment, where it can be picked up by advertising code there.

### Proposed Mitigations

#### Preflight to publisher

1. The server responding with the signed package is required to send the
   signature up front. This does not prevent any attacks but increases the
   user-visible latency, which AdTech experiences as a cost.
2. The user agent makes an ephemeral, cookie-less preflight request to
   `news.example` to get the signature and then validates the package from
   `adtech.example` against that signature.

   Both the
   [fully-offline](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#fully-offline-use)
   use case and [cryptographic
   agility](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html#crypto-agility)
   require `news.example` to be able to declare that multiple signatures are
   currently valid. This requires AdTech to send all of its signatures to
   `news.example` and for News to list them all in the now-quite-large preflight
   response. Having enough signatures that each one identifies a user would be
   detectable by clients and might be enough cost that News wouldn't be willing
   to do it.
3. We add a signed time stamp to the package signature to avoid AdTech telling
   News to get signatures from `adtech.example` backend and send personalized
   signatures back as preflight responses. With such time stamps, the user agent
   can decide to not accept signatures younger than, say one minute. For this to
   work we need signed, official time.

   TODO: Figure out how this blocks the attack.

#### Public signature repository

Another potential mitigation would be some kind of public repository of
signatures to check against.

This does not prevent any attacks, but could make them detectable.

#### Make package loads stateless

When requesting a signed package:

1. The request to the distributor must be credential-less. i.e. the [credentials
   mode](https://fetch.spec.whatwg.org/#concept-request-credentials-mode) must
   be `"omit"`. This prevents AdTech from learning the user's identity from its
   first-party cookies.
1. It must be an HTTP GET request to prevent, for example, a POST request body
   from including a user ID.
1. The request URL on the distributor must not have a query string. Fragments
   aren't sent to the server and would be blocked after the redirect, if
   necessary, by anti-tracking measures that are independent of web packages.
1. The path of the package request must be the same as the path on the target
   domain to prevent the distributor from encoding a user ID in the path.

Note that this still allows AdTech to encode a user ID in the *signed* path and
inject Javascript to decode it into local variables and `pushState()` the URL
back to what the publisher's Javascript expects.

This also still allows AdTech to encode a user ID into the hostname.

## ORIGIN frame and shared connections

### The Attack

Sketch: AdTech gets a certificate that covers both `adtech.example` and
`news.example`. They associate the user's ID with the HTTP/2 connection, use the
ORIGIN frame to convince the client to make a `news.example` request over that
connection, and return a `Set-Cookie` header with the user's ID.

## AdTech subdomain

### The Attack

Sketch: AdTech convinces News to point `*.adtech.news.example` to AdTech's
servers, perhaps by offering News higher rates on ads. AdTech has users click on
a link to `userid.adtech.news.example` and returns a `Set-Cookie` header setting
a user-id cookie for all of `news.example`, followed by a redirect to the real
URL.
