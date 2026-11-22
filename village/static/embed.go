package static

import "embed"

//go:embed *.html *.css
var FS embed.FS
