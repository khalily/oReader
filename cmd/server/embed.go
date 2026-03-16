package main

import (
	"embed"
	"io/fs"
)

// embedFS contains the production frontend build.
// The dist/ directory is a symlink to ../../web/dist created by `make frontend-build`
//
//go:embed all:dist
var embedFS embed.FS

// StaticFS returns a sub-filesystem that strips the "dist" prefix
// from embedded files, allowing Gin to serve them correctly.
func StaticFS() fs.FS {
	fsys, err := fs.Sub(embedFS, "dist")
	if err != nil {
		// This should never happen in production since the embed is compile-time
		panic(err)
	}
	return fsys
}
