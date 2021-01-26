# Web Packaging Explainers

The explainers in this directory collectively motivate the cluster of
specifications in this repository. This document is an index of the rest of
those explainers.

<!-- TOC -->

- [Use cases](#use-cases)
- [Maintaining security and privacy constraints](#maintaining-security-and-privacy-constraints)

<!-- /TOC -->

## Use cases

* [Packaging whole pages or sites](./authoritative-site-sharing.md): can be
  signed or unsigned.
   * [Serving signed subresources of a signed top-level HTML
     page](./signed-exchange-subresource-substitution.md)
   * [Navigating to top-level unsigned pages](./navigation-to-unsigned-bundles.md)
* [Packaging subresources](./subresource-loading.md). This can include groups of
  JS modules, stylesheets, images, or fonts. This is also getting fleshed out at
  https://github.com/littledan/resource-bundles/

## Maintaining security and privacy constraints

* [How to avoid third-party tracking](./anti-tracking.md)
* [What origin does an unsigned bundle have?](./bundle-urls-and-origins.md)
