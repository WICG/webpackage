**TODO list**
- this is work in progress
- needs more links


# Web Packaging Format Explainer
This document intends to list use cases motivating the improvements of the current [Web Packaging Draft](https://w3ctag.github.io/packaging-on-the-web/). It serves similar role as typical "Introduction" or "Using" and other non-normative sections of specs.

## Background
Some new use cases for Web technology have motivated thinking about a multi-resource packaging format. Those new opportunity include:

### Local Sharing

Local sharing is quite popular, especially in Emerging Markets countries, due to cost and limitations on cellular data and relatively spotty WiFi availability. It is typically done over local Bluetooth/WiFi, by either built-in OS features like [Android Beam](https://en.wikipedia.org/wiki/Android_Beam) or with popular 3-rd party apps, such as [ShareIt](https://play.google.com/store/apps/details?id=com.lenovo.anyshare.gps) or [Xender](https://play.google.com/store/apps/details?id=cn.xender)). Typically, the locally stored media files and apps (APK files for Android for example) are shared this way. Extending sharing to bundles of content and web apps (Progressive Web Apps in particular) opens up new possibilities, especially if combined with a form of signing the content. Cryptographic signing could make it possible to afford the shared content the treatment normally reserved for the origin that content is claiming to be in - by verifying that the content indeed was produced by the holder of the corresponding certificate.

### Physical Web.
[Beacons](https://google.github.io/physical-web/) and other physical web devices often want to 'broadcast' various content locally. Today, they broadcast a URL and make the user's device go to a web site. This delivers the trusted content to the user's browser (user cna observe the address bar to verify) and allow web apps to talk back to their services. It can be useful to be able to broadcast a package containing several pages or even a simple web app, even without a need to immediately have a Web connection - for example, via Bluetooth. If combined with signature form the publisher, the loaded pages may be treated as if they were laoded via TLS connection with a valid certificate, in terms of [origin-based security model](https://tools.ietf.org/html/rfc6454). For example, they can use XMLHttpRequest against its service or use "Add To Homescreen" for the convenience of the user.

### Content Distribution Networks and Web caches.
The CDNs can provide service of hosting web content that should be delivered at scale. This includes both hosting subresources (JS libraries, images) as well as entire content ([Google AMP](https://developers.google.com/amp/cache/overview)) on network of servers, often provided as a service by 3rd party. Unfortunately, origin-based security model of the Web limits the ways a 3rd-party caches/servers can be used. Indeed, for example in case of hosting JS subresourcves, the original document must explicitly trust the CDN origin to serve the trusted script. The user agent must use protocol-based means to verify the subresource is coming from the trusted CDN. Another example is a CDN that caches the whole content. Because the origin of CDN is different from the origin of the site, the browser normally can't afford the origin treatment of the site tot he loaded content. Look at how an article from USA Today is represented:

<img align="center" width=350 src="https://github.com/dimich-g/webpackage/blob/initial/buick.png">

Note the address bar indicating google.com. Also, since the content of USA Today is hosted in an iframe, it can't use all the functionality typically afforded to a top-level document:
- Can't request permissions
- Can't be added to homescreen
- Can't install a Service Worker (true???).

## Proposal

We propose to introduce a packaging format for the Web that would be able to contain multiple resources (HTML, CSS, Images, Media files, JS files etc) in a "bundle". That bundle can be distributed as regular Web resource (over HTTP[S]) and also by non-Web means, which includes local storage, local sharing, non-HTTP protocols like Bluetooth, etc. Being a single "bundle", it facilitates various modes of transfer.

In addition, that format would include optional signing of the resources, which can be used to verify authenticity and integrity of the content. Once and if verified (this may or may not require network connection), the content can be afforded the treatment of the claimed origin - for example showing a "green lock" with URL in a browser, or being able to send network request to the origin's server.

There is already a [packaging format proposal](https://w3ctag.github.io/packaging-on-the-web/) which we will base upon.
We are proposing to improve on the spec, in particular by introducing 2 major additions:
- Hierarchical structure of the sub-packages to allow resources from multiple origins.
- Index of content to facilitate local resource fetching from the package.
- Signature block, to cryptographically sign the content of the package.

Following are some example usages that correspond to these additions:

### Use Case: a copy of web site in a file.

```html
Content-Type: application/package
Content-Location: http://example.org/examplePack.pack
Link: <index.html>; rel=describedby

--j38n02qryf9n0eqny8cq0
Content-Location: index.html
Content-Type: text/html
<h1>Awesome page</h1>
<head>
<link rel=style
<body>

[ html content of the index page ]
--j38n02qryf9n0eqny8cq0
Content-Location: TOC
Content-Type: application/x-content-index 

http://example.org/images/image1.png 16100/4096
http://example.org/styles/first.css 25124/1000 
cid://e12eff3e4r5fg6g780 320/10240
--j38n02qryf9n0eqny8cq0--
```

```html
Content-Type: application/package
Content-Location: http://example.org/examplePack.pack
Link: <index.html>; rel=describedby
Link: <TOC>; rel=index; file-location: 267/312


--j38n02qryf9n0eqny8cq0
Content-Location: index.html
Content-Type: text/html
<h1>Table Of Contents</h1>
<body>

[ html content of the index page ]
--j38n02qryf9n0eqny8cq0
Content-Location: TOC
Content-Type: application/x-content-index 

http://example.org/images/image1.png 16100/4096
http://example.org/styles/first.css 25124/1000 
cid://e12eff3e4r5fg6g780 320/10240
--j38n02qryf9n0eqny8cq0--
```

