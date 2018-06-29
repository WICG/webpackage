# go/bundle
This directory contains a reference implementation of [Bundled HTTP Exchanges](https://wicg.github.io/webpackage/draft-yasskin-wpack-bundled-exchanges.html) spec.

## Overview
We currently provide two command-line tools: `gen-bundle` and `dump-bundle`.

`gen-bundle` command is a bundle generator tool. `gen-bundle` consumes a set of http exchanges (currently in the form of [HAR format](https://w3c.github.io/web-performance/specs/HAR/Overview.html)), and emits a bundled exchange file.

`dump-bundle` command is a bundle inspector tool. `dump-bundle` dumps the enclosed http exchanges of a given bundled exchange file in a human readable form.

You are also welcome to use the code as golang lib (e.g. `import "github.com/WICG/webpackage/go/bundle"`), but please be aware that the API is not yet stable and is subject to change any time.

## Getting Started

### Prerequisite
golang environment needs to be set up in prior to using the tool. We are testing the tool on latest golang. Please refer to [Go Getting Started documentation](https://golang.org/doc/install) for the details.

### Installation
We recommend using `go get` to install the command-line tool.

```
go get -u github.com/WICG/webpackage/go/bundle/cmd/...
```

### Usage
`gen-bundle` generates a bundled exchange file from a HAR file.

One convenient way to generate HAR file is via Chrome Devtools. Navigate to "Network" panel, and right-click on any resource and select "Save as HAR with content".
![generating har with devtools](https://raw.githubusercontent.com/WICG/webpackage/master/go/bundle/har-devtools.png)

Once you have the har file, generate the bundled exchange file via:
```
gen-bundle -i foo.har -o foo.wbn
```

`dump-bundle` dumps the content of a bundled exchange in a human readable form. To display content of a har file, invoke:
```
dump-bundle -i foo.har
```
