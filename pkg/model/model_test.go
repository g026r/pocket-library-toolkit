package model

import "embed"

// Tests are as follows:
// count_mismatch: contains 229 entries but the count value in the file is 4
// invalid_header: header magic number is invalid; should not parse
// valid: contains 229 entries, all valid
//
//go:embed tests
var files embed.FS
