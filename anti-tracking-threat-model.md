# Anti-tracking Threat Model

WebKit and other browsers are trying to reduce websites' ability to track users
across the internet. While the success of their effort remains to be seen, we
don't want Web Packaging to make the problem any more difficult.

This document contains a proposed threat model for the best-case outcome of that anti-tracking effort. We do not yet claim there is consensus about any of:

1. Whether the [attacker capabilities](#attacker-capabilities) are plausible.
1. Whether it will eventually be possible to frustrate the [ttacker goals that we want to frustrate](#attacker-goals-that-we-want-to-frustrate).
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

1. AdTech cannot manipulate the DNS system.
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

# Attacks and their Mitigations

This section lists potential ways AdTech might achieve its unwanted goals, along
with proposed or already-adopted mitigations to frustrate those goals. Not all
of the attacks use Web Packaging, so that we can explore the threat model's
implications for the rest of the web platform.

## Sign identifying information into the package

### The Attack

TODO: Simplify this description now that bits are in the threat model above.

This is how AdTech could foil the user agent's privacy protections with the
current signed packages proposal:

News wants to take part in signed package loading but thinks the actual
packaging is cumbersome and costly in terms of engineering resources.

AdTech has a financial incentive to help News get going with signed packages
because the technology makes AdTech's services better. Because of this
incentive, AdTech decides to offer News a more convenient way to do the
packaging; it offers to pull unsigned articles directly from News's servers and
do packaging for them. News just has to set up a signing service that AdTech can
call to get signatures back, or just hand a signing key straight to AdTech. News
sees the opportunity to reduce cost and takes the offer.

AdTech also has a financial incentive to identify the user on `news.example` to
augment its profile of the user and to earn extra money by serving the user
individually targeted ads, but it can't do so because the user's user agent is
protecting the user's privacy. However, the request to get a signed News package
is actually made to `adtech.example`, containing the user's AdTech cookies. To
achieve their goals and earn more money, AdTech's decides to create
`news.example` packages on the fly, bake in individually targeted ads plus an
AdTech user ID for profile enrichment, and sign the whole thing with News's key.

This is a case of cross-site tracking. The user is on a `news.example` webpage,
convinced that their user agent protects them from AdTech tracking them on this
site, but instead they got a signed package with tracking built in.

### Mitigations

#### Preflight to publisher

1. The server responding with the signed package is required to send the
   signature up front. This is to incentivize AdTech to not sign other websites'
   packages on the fly.
2. The user agent makes an ephemeral, cookie-less preflight request to
   `news.example` to get the signature and then validates the package from
   `adtech.example` against that signature.
3. We add a signed time stamp to the package signature to avoid AdTech telling
   News to get signatures from `adtech.example` backend and send personalized
   signatures back as preflight responses. With such time stamps, the user agent
   can decide to not accept signatures younger than, say one minute. For this to
   work we need signed, official time.

The above scheme would make it significantly harder to “personalize” packages.

#### Public signature repository

Another potential mitigation would be some kind of public repository of
signatures to check against.
