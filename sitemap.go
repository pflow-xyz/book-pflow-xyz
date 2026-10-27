package book

import (
	"bytes"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// SiteURL is the origin the sitemap's <loc> entries are written against.
const SiteURL = "https://book.pflow.xyz"

// Sitemap renders sitemap.xml from the chapter files in src, so the list of
// pages is derived from chapters/*.md and cannot drift from what mdBook
// builds. Order: home page, preface, chapters, appendices. SUMMARY.md is the
// table of contents, not a page, and is skipped.
func Sitemap(src fs.FS) ([]byte, error) {
	entries, err := fs.ReadDir(src, "chapters")
	if err != nil {
		return nil, err
	}
	var chapters, appendices []string
	hasPreface := false
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || name == "SUMMARY.md" {
			continue
		}
		stem := strings.TrimSuffix(name, ".md")
		switch {
		case stem == "preface":
			hasPreface = true
		case strings.HasPrefix(stem, "appendix-"):
			appendices = append(appendices, stem)
		default:
			chapters = append(chapters, stem)
		}
	}
	sort.Strings(chapters)
	sort.Strings(appendices)

	var b bytes.Buffer
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	url := func(path, priority string) {
		fmt.Fprintf(&b, "  <url><loc>%s%s</loc><priority>%s</priority></url>\n", SiteURL, path, priority)
	}
	url("/", "1.0")
	if hasPreface {
		url("/preface.html", "0.9")
	}
	for _, s := range chapters {
		url("/"+s+".html", "0.8")
	}
	for _, s := range appendices {
		url("/"+s+".html", "0.5")
	}
	b.WriteString("</urlset>\n")
	return b.Bytes(), nil
}
