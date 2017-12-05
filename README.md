# Web Packaging Format

This repository holds the in-progress specification for a web packaging format.
This format replaces the
~~[W3C TAG's Web Packaging Draft](https://w3ctag.github.io/packaging-on-the-web/)~~.
It will allow people to bundle together the resources that make up a website, so
they can be shared offline, either with or without a proof that they came from
the original website.

See the [explainer](explainer.md) for an overview of what the format does and
how it works.

Work on this format is happening in both the IETF and the W3C. We presented the
format to the DISPATCH WG at IETF99, and
received [feedback](https://datatracker.ietf.org/doc/minutes-99-dispatch/). We
have Internet Drafts for:

* [Use cases](https://tools.ietf.org/html/draft-yasskin-webpackage-use-cases) ([Editor's draft](https://wicg.github.io/webpackage/draft-yasskin-webpackage-use-cases.html))
* [Requirements on individual origin-signed request/response pairs](https://tools.ietf.org/html/draft-yasskin-http-origin-signed-responses) ([Editor's draft](https://wicg.github.io/webpackage/draft-yasskin-http-origin-signed-responses.html)):
  This is one layer split out of the full packaging proposal. The other layer,
  bundling these signed responses, isn't drafted yet.
* [The full packaging format](https://tools.ietf.org/html/draft-yasskin-dispatch-web-packaging) ([Editor's draft](https://wicg.github.io/webpackage/draft-yasskin-dispatch-web-packaging.html))

At the W3C, we'll pursue a specification on how browsers load this format. If
the IETF turns out not to be interested in the format itself, that'll come back
to the W3C too.

## Building this repository

### Building the Draft

Formatted text and HTML versions of the draft can be built using `make`.

```sh
$ make
```

This requires that you have software installed as described in
https://github.com/martinthomson/i-d-template/blob/master/doc/SETUP.md.

### Packaging tool.

Install this with `go install
github.com/WICG/webpackage/tree/internet-draft/go/webpack/cmd/wpktext2cbor`.
This tool is not yet documented well.
