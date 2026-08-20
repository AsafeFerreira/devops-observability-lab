package migrations

import "embed"

// Files contains the versioned SQL migrations executed by the services.
//
//go:embed *.sql
var Files embed.FS
