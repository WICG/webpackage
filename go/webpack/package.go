// Package webpack parses and serializes Web Packages in text and binary formats.
//
// Web Packages are defined in https://github.com/WICG/webpackage.
package webpack

type Package struct {
	manifest Manifest
	parts    []*PackPart
}
