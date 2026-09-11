// Package assets embeds the static files served by the app, so the compiled
// binary is the whole deployment.
package assets

import "embed"

//go:embed static
var FS embed.FS
