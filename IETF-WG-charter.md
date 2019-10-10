# Background

There is a long history of webpages grouping multiple subresources into a single
combined resource because this allows cross-resource compression and because
HTTP/1 made requests expensive. This encouraged the W3C TAG (Technical
Architecture Group) to propose a web packaging format based on multipart/* to
give web browsers visibility into the substructure of these combined resources.
HTTP/2 was expected to make these bundles unnecessary, but that hasn't panned
out.

In countries with expensive and/or unreliable mobile data, there is an
established practice of sharing content and native applications peer-to-peer
using SD cards, WiFi Direct, and similar transmission channels. Untrusted web
content could already be shared, albeit awkwardly, using existing formats.
However, with the web's move to HTTPS, it became impossible to share web apps
over these channels, so a team within Google Chrome began adding an index and
origin-trusted signatures to the TAG's packaging format.

The AMP project realized that this new format could solve the problem that pages
served by the AMP cache got the wrong URLs. This eventually led to Google Chrome
shipping an evolution of the format and Google Search using it in search
results.

# WPACK

The WPACK working group will develop a specification for a web packaging format
that efficiently bundles multiple HTTP resources. It will also specify a way to
optionally sign these resources such that a user agent can trust that they came
from their claimed web origins. Key goals for WPACK are:

* Efficiently storing a few or many, small or large resources, whose URLs are
  from one to many origins. Three examples of the sets of resources that should
  be supported are: a client-generated snapshot of a complete web page, a web
  page's tree of JavaScript modules, and El Paquete Semanal from Cuba.
* Allowing web apps to be safely installed after having been retrieved from a
  peer.
* Minimizing the latency to load a subresource from a package, whether the
  package is signed or unsigned, and whether the package is streamed or loaded
  from random-access storage.
* Being able to smoothly migrate to support future requirements such as random
  access within subresources, larger numbers of subresources, or avoidance of
  obsolete cryptography.
* Losing as little security and privacy as practical relative to TLS 1.3 and
  documenting exactly what is lost and how content authors can compensate. Part
  of this is analyzing how the shift from transport security to object security
  changes the security properties of the web's existing features.
* Minimizing the likelihood that the new format increases centralization or
  power imbalances on the web.

The packaging format will also achieve the following secondary goals as long as
they don't compromise or delay the above properties.

* Support more-efficient signing of a single, possibly same-origin HTTP
  resource.
* Support signed statements about subresources beyond just assertions that
  they're accurate representations of particular URLs. For example, that they
  appear on a transparency log, that they have passed a certain kind of static
  analysis, that a particular real-world entity vouches for them, etc.
* Address the threat model of a website whose frontend might be compromised
  after a user first uses the site.
* Support books being published in the format.
* Support long-lived archival storage.
* Optimize transport of large numbers of small same-origin resources.
* Allow the format to be used in self-extracting executables.
* Allow publishers to efficiently combine sub-packages from other publishers.

The following potential goals are out of scope under this charter:

* DRM
* A way to distribute the private portions of a website. For example, WPACK
  might define a way to distribute Gmail's application but wouldn't define a way
  to distribute individual emails without a direct connection to Gmail's origin
  server.
* Defining the details of how web browsers load the formats and interact with
  any protocols we define here. Other standards bodies are more appropriate fora
  for this.
* A way to automatically discover the URL for an accessible package that
  includes content for a blocked or expensive-to-access URL that the user wants
  to browse.

Note that consensus is required both for changes to the current protocol
mechanisms and retention of current mechanisms. In particular, because something
is in the initial document set (consisting of
draft-yasskin-webpackage-use-cases, draft-yasskin-wpack-bundled-exchanges, and
draft-yasskin-http-origin-signed-responses) does not imply that there is
consensus around the feature or around how it is specified.

# Relationship to Other WGs and SDOs

WPACK will work with the W3C and WHATWG to identify the existing security and
privacy models for the web, and to ensure those SDOs can define how this format
is used by web browsers.

# Milestones

* Chartering + 3 Months: Working group adoption of use cases document
* Chartering + 3 Months: Working group adoption of bundling document
* Chartering + 3 Months: Working group adoption of security analysis document
* Chartering + 3 Months: Working group adoption of privacy analysis document
* Chartering + 3 Months: Working group adoption of signing document
* Chartering + 18 Months: Use cases document to IESG
* Chartering + 18 Months: Bundling document to IESG
* Chartering + 24 Months: Security analysis document to IESG
* Chartering + 24 Months: Privacy analysis document to IESG
* Chartering + 24 Months: Signing document to IESG
