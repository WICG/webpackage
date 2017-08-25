---
coding: utf-8

title: Origin-signed HTTP Responses
docname: draft-yasskin-http-origin-signed-responses
category: std

ipr: trust200902
area: gen
workgroup: http
keyword: Internet-Draft

stand_alone: yes
pi: [comments, sortrefs, strict, symrefs, toc]

author:
 -
    name: Jeffrey Yasskin
    organization: Google
    email: jyasskin@chromium.org

informative:
  SRI: W3C.REC-SRI-20160623

--- abstract

This document explores how a server can send particular responses that are
authoritative for an origin, when the server itself is not authoritative for
that origin. For now, it focuses on the constraints covering any such mechanism.

--- note_Note_to_Readers

Discussion of this draft takes place on the HTTP working group mailing list
(ietf-http-wg@w3.org), which is archived
at <https://lists.w3.org/Archives/Public/ietf-http-wg/>.

The source code and issues list for this draft can be found
in <https://github.com/WICG/webpackage>.

--- middle

# Introduction

When
I
[presented Web Packaging to DISPATCH](https://datatracker.ietf.org/doc/minutes-99-dispatch/),
folks thought it would make sense to split it into a way to sign individual HTTP
responses as coming from a particular origin, and separately a way to bundle a
collection of HTTP responses. This document explores the constraints on any
method of signing HTTP responses and briefly sketches a possible solution to the
constraints.

# Terminology

Author
: The entity that controls the server for a particular origin {{?RFC6454}}. The
  author can get a CA to issue certificates for their private keys and can run a
  TLS server for their origin.

Intermediate
: An entity that fetches signed HTTP responses from an author or another
  intermediate and forwards them to another intermediate or a client.

Client
: An entity that uses a signed HTTP response and needs to be able to prove that
  the author vouched for it as coming from its claimed origin.

# Use cases

## PUSHed subresources {#uc-pushed-subresources}

To reduce round trips, a server might use HTTP/2 PUSH to inject a subresource
from another server into the client's cache. If anything about the subresource
is expired or can't be verified, the client would fetch it from the original
server.

For example, if `https://example.com/index.html` includes

~~~html
<script src="https://jquery.com/jquery-1.2.3.min.js">
~~~

Then to avoid the need to look up and connect to `jquery.com` in the critical
path, `example.com` might PUSH that resource ({{?RFC7540}}, section 8.2), signed
by `jquery.com`.

## Explicit use of a CDN for subresources {#uc-explicit-cdn}

In order to speed up loading but still maintain control over its content, an
HTML page in a particular origin `O.com` could tell clients to load its
subresources from an intermediate CDN that's not authoritative, but require that
those resources be signed by `O.com` so that the CDN couldn't modify the
resources.

~~~html
<img logicalsrc="https://O.com/img.png"
     physicalsrc="https://cdn.com/O.com/img.png">
~~~

This could be used for some of the same purposes as SRI ({{uc-sri}}).

## Subresource Integrity {#uc-sri}

The W3C WebAppSec group is investigating
[using signatures](https://github.com/w3c/webappsec-subresource-integrity/blob/master/signature-based-restrictions-explainer.markdown) in
{{SRI}}. They need a way to transmit the signature with the
response, which this proposal could provide.

However, their needs also differ in some significant ways:

1. The `integrity="ed25519-[public-key]"` attribute and CSP-based ways of
   expressing a public key don't need the signing key to be also trusted to sign
   arbitrary content for an origin.
2. Some uses of SRI want to constrain subresources to be vouched for by a
   third-party, rather than just being signed by the subresource's author.

While we can design this system to cover both origin-trusted and simple-key
signatures, we should check that this is better than having two separate systems
for the two kinds of signatures.

## Offline websites {#uc-offline-websites}

See <https://github.com/WICG/webpackage> and
{{?I-D.yasskin-dispatch-web-packaging}}. This use requires origin-signed
resources to be bundled.

# Requirements and open questions

## Proof of origin

To verify that a thing came from a particular origin, for use in the same
context as a TLS connection, we need someone to vouch for the signing key with
as much verification as the signing keys used in TLS. The obvious way to do this
is to re-use the web PKI and CA ecosystem.

We'll improve adoption if we can re-use existing TLS server certificates, but if
the risks of doing that are too high, we could define a new Extended Key Usage
({{?RFC5280}}, section 4.2.1.12) that requires CAs to issue new keys for this
purpose. The rest of this document will assume we can re-use existing TLS server
certificates.

It's essential that an attacker who can convince the server to sign some
resource, cannot cause those signed bytes to be interpreted as something else.
As a result, signatures here need to use the same format as TLS's signatures,
specified in {{?I-D.ietf-tls-tls13}} section 4.4.3, with a context string that's
specific to this use.

### The certificate and its chain {#certificate-chain}

The client needs to be able to find the certificate vouching for the signing key
and a chain from that certificate to a trusted root. One approach would be to
include the certificate and its chain in the signature metadata itself, but this
wastes bytes when the same certificate is used for multiple HTTP responses. If
we decide to put the signature in an HTTP header, certificates are also
unusually large for that context.

Another option is to pass a URL that the client can fetch to retrieve the
certificate and chain. To avoid extra round trips in fetching that URL, it could
be [bundled](#uc-offline-websites) with the signed content
or [PUSHed](#uc-pushed-subresources) with it.

## How much to sign

The previous {{?I-D.thomson-http-content-signature}} and
{{?I-D.burke-content-signature}} schemes signed just the content, while
({{?I-D.cavage-http-signatures}} could also sign the response headers and the
request method and path. However, the same path, response headers, and content
may mean something very different when retrieved from a different server, so
this document expects to include the whole URL in the signed data as well.

The question of whether to include other request headers---primarily the
`accept*` family---is still open. These headers need to be represented so that
clients wanting a different language, say, can avoid using the wrong-language
response, but it's not obvious that there's a security vulnerability if an
attacker can spoof them. That said, it's always safer to include everything in
the signature.

In order to allow multiple clients to consume the same signed response, the
response shouldn't include the exact headers that any particular client sends.
For example, a Japanese resource wouldn't include

~~~http
accept-language: ja-JP, ja;q=0.9, en;q=0.8, zh;q=0.7, *;q=0.5
~~~

Instead, it would probably include just

~~~http
accept-language: ja-JP, ja
~~~

and clients would use the same matching logic as
for [PUSH_PROMISE](https://tools.ietf.org/html/rfc7540#section-8.2) frame
headers.

The rest of this document will assume that all request headers are included in
the signature.

## Response lifespan

A normal HTTPS response is authoritative only for one client, for as long as its
cache headers say it should live. A signed response can be re-used for many
clients, and if it was generated while a server was compromised, it can continue
compromising clients even if their requests happen after the server recovers.
This signing scheme needs to mitigate that risk.

### Certificate revocation

Certificates are mis-issued and private keys are stolen, and in response clients
need to be able to stop trusting these certificates as promptly as possible.
Online revocation
checks [don't work](https://www.imperialviolet.org/2012/02/05/crlsets.html), so
the industry has moved to pushed revocation lists and stapled OCSP responses
{{?RFC6066}}.

Pushed revocation lists work as-is to block trust in the certificate signing a
response, but the signatures need an explicit strategy to staple OCSP responses.
One option is to extend the certificate download ({{certificate-chain}}) to
include the OCSP response too, perhaps in the
[TLS 1.3 CertificateEntry](https://tlswg.github.io/tls13-spec/draft-ietf-tls-tls13.html#ocsp-and-sct) format.

### Response downgrade attacks {#downgrade}

The signed content in a response might be vulnerable to attacks, such as XSS, or
might simply be discovered to be incorrect after publication. Once the author
fixes those vulnerabilities or mistakes, clients should stop trusting the old
signed content in a reasonable amount of time. Similar to certificate
revocation, I expect the best option to be stapled "this version is still valid"
assertions with short expiration times.

These assertions could be structured as:

1. A signed minimum version number or timestamp for a set of request headers:
   This requires that signed responses need to include a version number or
   timestamp, but allows a server to provide a single signature covering all
   valid versions.
2. A replacement for the whole response's signature. This requires the author to
   separately re-sign each valid version and requires each version to include a
   strong validator {{?RFC7232}}, but allows intermediates to serve less data.
3. A replacement for the response's signature and an update for the embedded
   `expires` and related cache-control HTTP headers {{?RFC7234}}. This naturally
   extends authors' intuitions about cache expiration and the existing cache
   revalidation behavior to signed responses. However, it also requires that the
   update procedure for the response headers on the client must produce
   something that's bytewise identical to the updated headers on the server.

The signature also needs to include instructions to intermediates for how to
fetch updated validity assertions.

## Conveying the signed headers

HTTP headers are traditionally munged by proxies, making it impossible to
guarantee that the client will see the same sequence of bytes as the author
wrote. In the HTTPS world, we have more end-to-end header integrity, but it's
still likely that there are enough TLS-terminating proxies that the author's
signatures would tend to break before getting to the client.

Since proxies don't modify unknown content types, I expect to propose an
`application/http2` format to serialize the request headers, response headers,
and body so they can be signed. This could be as simple as a series of HTTP/2
frames, or could

1. Allow longer contiguous bodies than
   [HTTP/2's 16MB frame limit](https://tools.ietf.org/html/rfc7540#section-4.2), and
2. Use better compression than {{?RFC7541}} for the non-confidential headers.

To help the PUSHed subresources use case ({{uc-pushed-subresources}}), we might
also want to extend the `PUSH_PROMISE` frame type to include a signature, and
then there might be some way to include the request headers directly in that
frame.

# Straw proposal

TBD

# Security considerations

Authors MUST NOT include confidential information in a signed response that an
untrusted intermediate could forward, since the response is only signed and not
encrypted. Intermediates can read the content.

Relaxing the requirement to consult DNS when determining authority for an origin
means that an attacker who possesses a valid certificate no longer needs to be
on-path to redirect traffic to them; instead of modifying DNS, they need only
convince the user to visit another Web site in order to serve responses signed
as the target. This consideration and mitigations for it are shared by
{{?I-D.ietf-httpbis-origin-frame}}.

Signing a bad response can affect more users than simply serving a bad response,
since a served response will only affect users who make a request while the bad
version is live, while an attacker can forward a signed response until its
signature expires. Authors should consider shorter signature expiration times
than they use for cache expiration times.

# Privacy considerations

Normally, when a client fetches `https://o1.com/resource.js`,
`o1.com` learns that the client is interested in the resource. If
`o1.com` signs `resource.js`, `o2.com` serves it as
`https://o2.com/o1resource.js`, and the client fetches it from there,
then `o2.com` learns that the client is interested, and if the client
executes the Javascript, that could also report the client's interest back to
`o1.com`.

Often, `o2.com` already knew about the client's interest, because it's the
entity that directed the client to `o1resource.js`, but there may be cases where
this leaks extra information.

For non-executable resource types, a signed response can improve the privacy
situation by hiding the client's interest from the original author.


--- back

# Acknowledgements

