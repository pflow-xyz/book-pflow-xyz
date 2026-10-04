// Package book exposes the chapter sources embedded in the binary so the
// web server can derive machine-readable views (llms-full.txt) from the
// same markdown mdBook renders, with no chance of drift.
package book

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed chapters/*.md llms.txt
var Sources embed.FS

// redirects.json maps retired chapter URLs to their current ones, e.g.
// "/ch13-exponential-weights.html" -> "/ch15-exponential-weights.html".
// Renumbering chapters changes their URLs, and every link published before
// the renumbering (blog posts, search results) keeps pointing at the old
// name. The console MCP's book_add_redirects writes this file from git's
// rename history; edit it there rather than by hand.
//
//go:embed redirects.json
var redirectsJSON []byte

// Redirects returns the old-path -> new-path table, refusing one that would
// shadow a chapter that exists or point at one that does not. Both mistakes
// are silent at request time — a shadowing redirect hides a live chapter, a
// dangling one trades a 404 for a 301-then-404 — so they fail the build here
// (via the test) and fail startup rather than serving.
func Redirects() (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal(redirectsJSON, &m); err != nil {
		return nil, fmt.Errorf("redirects.json: %w", err)
	}
	for from, to := range m {
		if stem, ok := chapterStem(from); !ok {
			return nil, fmt.Errorf("redirects.json: %q is not a /<chapter>.html path", from)
		} else if chapterExists(stem) {
			return nil, fmt.Errorf("redirects.json: %q would shadow an existing chapter", from)
		}
		stem, ok := chapterStem(strings.SplitN(to, "#", 2)[0])
		if !ok || !chapterExists(stem) {
			return nil, fmt.Errorf("redirects.json: %q -> %q: target chapter does not exist", from, to)
		}
		if _, chained := m[strings.SplitN(to, "#", 2)[0]]; chained {
			return nil, fmt.Errorf("redirects.json: %q -> %q is a chain; point it at the final chapter", from, to)
		}
	}
	return m, nil
}

func chapterStem(path string) (string, bool) {
	if !strings.HasPrefix(path, "/") || !strings.HasSuffix(path, ".html") {
		return "", false
	}
	stem := strings.TrimSuffix(strings.TrimPrefix(path, "/"), ".html")
	if stem == "" || strings.Contains(stem, "/") {
		return "", false
	}
	return stem, true
}

func chapterExists(stem string) bool {
	_, err := Sources.Open("chapters/" + stem + ".md")
	return err == nil
}
