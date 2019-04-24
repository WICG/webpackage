# Anti-tracking Threat Model

## The Actors

* **The user.** This is the human who relies on the user agent to protect their privacy.
* **The user agent.** This is the web browser that tries to protect the user's privacy.
* **AdTech or adtech.example.** This is a website that the user engages with as first-party site and that has a financial interest in 1) knowing what the user does on other websites to augment its rich profile of the user and 2) individual targeting of ads, based on its rich profile of the user.
* **News or news.example.** This is a news website which wants its articles to be served as signed packages with the user agent's URL bar showing news.example.

## The Threat

1. The user does not want AdTech to be able to augment its profile of them while reading articles on news.example.
2. The user does not want AdTech's rich profile of them to influence the content of ads or articles on news.example.

## The Attack

This is how AdTech could foil the user agent's privacy protections with the current signed packages proposal:

News wants to take part in signed package loading but thinks the actual packaging is cumbersome and costly in terms of engineering resources.

AdTech has a financial incentive to help News get going with signed packages because the technology makes AdTech's services better. Because of this incentive, AdTech decides to offer News a more convenient way to do the packaging; it offers to pull unsigned articles directly from News's servers and do packaging for them. News just has to set up a signing service that AdTech can call to get signatures back, or just hand a signing key straight to AdTech. News sees the opportunity to reduce cost and takes the offer.

AdTech also has a financial incentive to identify the user on news.example to augment its profile of the user and to earn extra money by serving the user individually targeted ads, but it can't do so because the user's user agent is protecting the user's privacy. However, the request to get a signed News package is actually made to adtech.example, containing the user's AdTech cookies. To achieve their goals and earn more money, AdTech's decides to create news.example packages on the fly, bake in individually targeted ads plus an AdTech user ID for profile enrichment, and sign the whole thing with News's key.

This is a case of cross-site tracking. The user is on a news.example webpage, convinced that their user agent protects them from AdTech tracking them on this site, but instead they got a signed package with tracking built in.

## How the Attack Relates To Other Means of Cross-Site Tracking

Often when we criticize new, technically distinct tracking vectors, we are told that “you can track users in so many ways so why care about this one?” In the case of signed packages we hear about means of tracking such as doctored links where cross-site tracking is built into the URL, or server-side exchanges of personally identifiable information such as users' email addresses.

First, we don't think past mistakes and flaws in web technologies is a valid argument for why new web technologies should enable cross-site tracking.

Second, WebKit is working hard to prevent cross-site tracking, including new limits and restrictions on old technologies. Piling on more such work is not acceptable to us.

Finally, the success of new web technologies such as signed packages relies on better security and privacy guarantees than what we've had in the past. We want progression in this space, not the status quo.

## Potential Mitigations and Fixes

A mitigation we'd like to discuss is this:

1. The server responding with the signed package is required to send the signature up front. This is to incentivize AdTech to not sign other websites' packages on the fly.
2. The user agent makes an ephemeral, cookie-less preflight request to news.example to get the signature and then validates the package from adtech.example against that signature.
3. We add a signed time stamp to the package signature to avoid AdTech telling News to get signatures from adtech.example backend and send personalized signatures back as preflight responses. With such time stamps, the user agent can decide to not accept signatures younger than, say one minute. For this to work we need signed, official time.

The above scheme would make it significantly harder to “personalize” packages.

Another potential mitigation would be some kind of public repository of signatures to check against.
