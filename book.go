// Package book exposes the chapter sources embedded in the binary so the
// web server can derive machine-readable views (llms-full.txt) from the
// same markdown mdBook renders, with no chance of drift.
package book

import "embed"

//go:embed chapters/*.md llms.txt
var Sources embed.FS
