package migrations

import "embed"

//go:embed all:*.sql
var FS embed.FS
