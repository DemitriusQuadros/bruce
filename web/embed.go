// Package web exposes the embedded static assets for the Bruce web UI.
package web

import "embed"

// Public holds the contents of the web/public directory, embedded at compile time.
//
//go:embed public
var Public embed.FS
