package web

import "embed"

// DistFS embeds the compiled React 19 production build.
//
//go:embed all:dist
var DistFS embed.FS
