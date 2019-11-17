# Background
Webpages sometimes group multiple subresources into a single
combined resource to allow cross-resource compression and to reduce the overhead
of HTTP/1 requests. The W3C TAG (Technical Architecture Group) proposed a web
packaging format based on multipart/* , to give web browsers visibility into
the substructure of these combined resources. That has not seen deployment
and HTTP/2 did not make these bundles unnecessary as was once expected.

These bundles are still needed.  In countries with expensive and/or unreliable
mobile data, there is an established practice of sharing content and native
applications peer-to-peer. Untrusted web content can generally be shared, but
with the web's move to HTTPS, it is no longer possible to share web apps
over these channels

# WPACK

The WPACK working group will develop a specification for a web packaging format
that efficiently bundles multiple HTTP resources. It will also specify a way to
optionally sign these resources such that a user agent can trust that they came
from their claimed web origins. Key goals for WPACK are:

* Efficient storage across a range of resource combinations. Three examples to
  be supported are: a client-generated snapshot of a complete web page, a web
  page's tree of JavaScript modules, and El Paquete Semanal from Cuba.
* Safe web app installation after having been retrieved from a peer.
* Low latency to load a subresource from a package, whether the
  package is signed or unsigned, and whether the package is streamed or loaded
  from random-access storage.
* Being extensible, including to avoid cryptography that becomes obsolete.
* Security and privacy properties of using bundles as close as practical to TLS
  1.3 transport of the same resources.  Where properties do change, the group
  will document exactly what changed and how content authors can compensate.
* A low likelihood that the new format increases centralization or
  power imbalances on the web.

The packaging format will also aim to achieve the following secondary goals as long as
they don't compromise or delay the above properties.

* Support more-efficient signing of a single, possibly same-origin HTTP
  resource.
* Support signed statements about subresources beyond just assertions that
  they're accurate representations of particular URLs.
* Address the threat model of a website compromised after a user first uses the site.
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
  any protocols we define here.
* A way to automatically discover the URL for an accessible package that
  includes specific content.

Note that consensus is required both for changes to the current protocol
mechanisms and retention of current mechanisms. In particular, because something
is in the initial document set (consisting of
draft-yasskin-wpack-use-cases, draft-yasskin-wpack-bundled-exchanges, and
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
