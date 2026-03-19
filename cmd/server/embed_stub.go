//go:build noembed

package main

import "io/fs"

// StaticFS returns nil when building without embedded files.
// This is used in development mode where the frontend is served separately by Vite.
func StaticFS() fs.FS {
	return nil
}
