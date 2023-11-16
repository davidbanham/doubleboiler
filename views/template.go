package views

import "embed"

//go:embed pages/*.html layouts/*.html
var TmplFS embed.FS
