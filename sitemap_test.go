package book

import (
	"encoding/xml"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func sitemapStems(t *testing.T, src fs.FS) map[string]bool {
	t.Helper()
	out, err := Sitemap(src)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("sitemap is not valid XML: %v", err)
	}
	stems := map[string]bool{}
	for _, u := range doc.URLs {
		path, ok := strings.CutPrefix(u.Loc, SiteURL)
		if !ok {
			t.Errorf("loc %q is not under %s", u.Loc, SiteURL)
			continue
		}
		if path == "/" {
			continue
		}
		stem, ok := chapterStem(path)
		if !ok {
			t.Errorf("loc %q is not a /<chapter>.html path", u.Loc)
			continue
		}
		stems[stem] = true
	}
	return stems
}

// Every sitemap URL is a chapter that exists, and every chapter is listed.
// Both directions drifted once: the file was hand-written and went stale
// across a renumbering, listing seven 404s and omitting nine live pages.
func TestSitemapMatchesChapters(t *testing.T) {
	stems := sitemapStems(t, Sources)
	for stem := range stems {
		if !chapterExists(stem) {
			t.Errorf("sitemap lists %q but chapters/%s.md does not exist", stem, stem)
		}
	}
	entries, err := fs.ReadDir(Sources, "chapters")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if name == "SUMMARY.md" || !strings.HasSuffix(name, ".md") {
			continue
		}
		if stem := strings.TrimSuffix(name, ".md"); !stems[stem] {
			t.Errorf("chapters/%s exists but the sitemap omits it", name)
		}
	}
	if stems["SUMMARY"] {
		t.Error("SUMMARY is a table of contents, not a page")
	}
}

func TestSitemapIgnoresNonChapters(t *testing.T) {
	src := fstest.MapFS{
		"chapters/preface.md":    {},
		"chapters/ch01-a.md":     {},
		"chapters/SUMMARY.md":    {},
		"chapters/appendix-a.md": {},
		"chapters/figures/x.md":  {},
	}
	got := sitemapStems(t, src)
	for _, want := range []string{"preface", "ch01-a", "appendix-a"} {
		if !got[want] {
			t.Errorf("missing %s", want)
		}
	}
	if len(got) != 3 {
		t.Errorf("got %v, want exactly preface, ch01-a, appendix-a", got)
	}
}
