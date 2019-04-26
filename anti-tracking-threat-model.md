# Anti-tracking Threat Model

WebKit and other browser engines are trying to reduce websites' ability to track
users across the web. We don't want Web Packaging to make this effort any more
difficult.

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

One argument to avoid restricting web packaging is, “you can track users in so
many ways so why care about this one?” For example, any link can convey a
user-id hidden in the URL. Because of the effort mentioned above to reduce
tracking abilities, this document assumes that browser engines will succeed in
adding all the limits and restrictions on existing technologies necessary to
eliminate tracking through them. It then analyzes what restrictions on web
packaging are necessary to prevent it from undoing that progress.

The success of new web technologies, including signed packages, relies on better
security and privacy guarantees than what we've had in the past. We want
progression in this space, not the status quo.

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
1. AdTech can convince News to let them create packages that get signed as
   `news.example`, in a couple alternate ways.

   This seems like an implausible capability at first glance, but publishers
   routinely give CDNs this ability to terminate TLS traffic, and a
   generally-trusted AdTech could convince publishers that it's merely doing
   what a good CDN would, for cheaper.

   1. News acquires a `news.example` [exchange-signing
      certificate](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html#cross-origin-cert-req)
      and gives its private key to AdTech.
   1. News acquires a `news.example` exchange-signing certificate and uses it to
      authorize a short-lived key owned by AdTech, using a system like
      [Delegated
      Credentials](https://tools.ietf.org/html/draft-ietf-tls-subcerts-03).
   1. News tells a CA that AdTech has permission to receive exchange-signing
      certificates for `news.example`. This is the model CDNs usually use.
   1. News hosts a signing service that signs packages given to News by AdTech.
      This resembles the CDN "Split-TLS" model. News would expect AdTech to
      fetch News's content, optimize it in some way, and send the result back to
      News for signing on the fly.
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

1. AdTech wants to augment its profile of the user while the user reads articles
   on `news.example`.
1. AdTech wants to use its rich profile of the user to influence the content of
   ads or articles on `news.example`.
1. More abstractly, AdTech wants to transfer its unique ID for a user to
   the Javascript environment created for `news.example`.
1. AdTech doesn't want the user agent or external auditors to be able to detect
   that AdTech is tracking users.

# Attacks and their Mitigations

This section lists potential ways AdTech might achieve its unwanted goals, along
with proposed or already-adopted mitigations to frustrate those goals. Not all
of the attacks use Web Packaging, so that we can explore the threat model's
implications for the rest of the web platform.

## Sign identifying information into the package

### The Attack

1. AdTech convinces News to let them create packages that get signed as
   `news.example`.
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
