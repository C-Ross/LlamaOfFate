package web

import "embed"

// Dist embeds the Vite build output (web/dist/).
// Run "just web-build" before building the server binary.
//
//go:embed all:dist
var Dist embed.FS
