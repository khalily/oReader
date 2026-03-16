package main

import (
	"embed"
	"io/fs"
)

// embedFS contains the production frontend build.
// The build output is in web/dist after running `make frontend-build`
//
//go:embed all:web/dist
var embedFS embed.FS

// StaticFS returns a sub-filesystem that strips the "web/dist" prefix
// from embedded files, allowing Gin to serve them correctly.
func StaticFS() fs.FS {
	fsys, err := fs.Sub(embedFS, "web/dist")
	if err != nil {
		// This should never happen in production since the embed is compile-time
		panic(err)
	}
	return fsys
}
