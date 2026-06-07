package assets

import _ "embed"

//go:embed logo.png
var LogoPNG []byte

// HasAssets reports whether embedded resources are present.
func HasAssets() bool { return len(LogoPNG) > 0 }
