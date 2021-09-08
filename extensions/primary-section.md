# Primary URL optional section

This document outlines the format and usage of the "Primary URL" optional section as a part of a [WebBundle](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html).

## Format

~~~ cddl
primary = whatwg-url
~~~

The "primary" section records a single URL identifying the primary URL of the
bundle. The URL MUST refer to a resource with representations contained in the bundle itself.

## Usage

The URL enclosed in the section identifies both a fallback when the recipient doesn't
understand the bundle and a default resource inside the bundle to use when the
recipient doesn't have more specific instructions. This field MAY be an empty
string, although protocols using bundles MAY themselves forbid that empty value.

While this section is not a part of the main spec, it is required for the
'Navigation to WebBundle' use-case as it acts as a default resource to load.