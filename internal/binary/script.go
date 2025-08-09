package binary

import (
	_ "embed"
)

//go:generate go run generate.go

//go:embed ghostscript
var GhostscriptBinary []byte