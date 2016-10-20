**TODO list**
- this is work in progress
- needs more links


# Web Packaging Format Explainer
This document intends to list use cases motivating the improvements of the current [Web Packaging Draft](https://w3ctag.github.io/packaging-on-the-web/). It serves similar role as typical "Introduction" or "Using" and other non-normative sections of specs.

## Background
Recently, some new use cases for using Web technology arose that motivate thinking about a multi-resource packaging format. Those new opportunity include:
1. Local file sharing (typically over local Bluetooth/WiFi, as in ShareIt or Xander). This is widely used in EM countries where the cellular data is expensive, limited and slow. Sharing bundles of content and whole apps locally opens up many possibilities for such users.
2. Physical Web. Beacons _(need link)_ and other physical web devices often want to 'share' content and potentially an app with devices nearby. It often can only happen over Bluetooth or some other connection which does not allow regular HTTP(s) protocol.
3. Content Distribution Networks and Web caches. The CDNs can provide service of hosting web content that should be delivered at scale. This includes both hosting subreasources (JS libraries, images) or even top-level frame content (AMP). Unfortunately decoupling of origin (as in NYT content hosted on amp.google

