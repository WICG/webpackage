# Signatures optional section

This document outlines the format and usage of the "signatures" optional section
as a part of a [WebBundle](https://wpack-wg.github.io/bundled-responses/draft-ietf-wpack-bundled-responses.html).


## Parsing the signatures section

The "signatures" section vouches for the resources in the bundle.

The section can contain as many signatures as needed, each by some authority,
and each covering an arbitrary subset of the resources in the bundle.
Intermediates, including attackers, can remove signatures from the bundle
without breaking the other signatures.

The bundle parser's client is responsible to determine the validity and meaning
of each authority's signatures. In particular, the algorithm below does not
check that signatures are valid. For example, a client might:

* Use the ecdsa_secp256r1_sha256 algorithm defined in Section 4.2.3 of
  [TLS1.3](https://datatracker.ietf.org/doc/html/rfc8446) to check the validity
  of any signature with an EC public key on the secp256r1 curve.
* Reject all signatures by an RSA public key.
* Treat an X.509 certificate with the CanSignHttpExchanges extension (Section
  4.2 of [I-D.yasskin-http-origin-signed-responses](I-D.yasskin-http-origin-signed-responses))
  and a valid chain to a trusted root as an authority that vouches for the
  authenticity of resources claimed to come from that certificate's domains.
* Treat an X.509 certificate with another extension or EKU as vouching that a
  particular analysis has run over the signed resources without finding
  malicious behavior.

A client might also choose different behavior for those kinds of authorities and
keys.

~~~ cddl
signatures = [
  authorities: [*authority],
  vouched-subsets: [*{
    authority: index-in-authorities,
    sig: bstr,
    signed: bstr  ; Expected to hold a signed-subset item.
  }],
]
authority = augmented-certificate
index-in-authorities = uint

signed-subset = {
  validity-url: whatwg-url,
  auth-sha256: bstr,
  date: uint,
  expires: uint,
  subset-hashes: {+
    whatwg-url => [variants-value, +resource-integrity]
  },
  * tstr => any,
}
resource-integrity = (
  header-sha256: bstr,
  payload-integrity-header: tstr
)
~~~

The `augmented-certificate` CDDL rule comes from Section 3.3 of [I-D.yasskin-http-origin-signed-responses](I-D.yasskin-http-origin-signed-responses).

To parse the signatures section, given its `sectionContents`, the `sectionOffsets`
map, and the `metadata` map to fill in, the parser MUST do the following:

1. Let `signatures` be the result of parsing `sectionContents` as a CBOR item
   matching the `signatures` rule in the above CDDL.
1. Set `metadata["authorities"]` to the list of authorities in the first element
   of the `signatures` array.
1. Set `metadata["vouched-subsets"]` to the second element of the `signatures`
   array.

## Note

This extension document doesn't follow the latest changes on the Web Bundles
spec now. The content negotiation part was removed by this
[PR](https://github.com/wpack-wg/bundled-responses/pull/7/). So the
`variants-value` in the avove CDDL is nonsense now. (TODO: Need to change the
CDDL not to use `variants-value`.)

[I-D.yasskin-http-origin-signed-responses]:https://datatracker.ietf.org/doc/html/draft-yasskin-http-origin-signed-responses-09