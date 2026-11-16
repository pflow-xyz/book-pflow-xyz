package book

import (
	"strings"
	"testing"
)

// The committed redirects.json must load: every target is a real chapter,
// no entry hides a live one, no chains.
func TestRedirectsValid(t *testing.T) {
	if _, err := Redirects(); err != nil {
		t.Fatal(err)
	}
}

func TestChapterStem(t *testing.T) {
	for in, want := range map[string]string{
		"/ch15-exponential-weights.html": "ch15-exponential-weights",
		"/ch15.html":                     "ch15",
	} {
		if got, ok := chapterStem(in); !ok || got != want {
			t.Errorf("chapterStem(%q) = %q, %v", in, got, ok)
		}
	}
	for _, bad := range []string{"ch15.html", "/a/b.html", "/ch15.md", "/.html"} {
		if _, ok := chapterStem(bad); ok {
			t.Errorf("chapterStem(%q) accepted", bad)
		}
	}
}

func TestChapterExists(t *testing.T) {
	if !chapterExists("ch15-exponential-weights") {
		t.Error("ch15-exponential-weights should exist")
	}
	if chapterExists("ch13-exponential-weights") {
		t.Error("ch13-exponential-weights should not exist")
	}
}

// A retired chapter URL must land on a page the sitemap lists, so a
// crawler following the redirect ends on an indexed page. (Redirects()
// itself already rejects dangling, shadowing and chained entries.)
func TestRedirectTargetsAreInSitemap(t *testing.T) {
	r, err := Redirects()
	if err != nil {
		t.Fatal(err)
	}
	stems := sitemapStems(t, Sources)
	for from, to := range r {
		if stem, _ := chapterStem(strings.SplitN(to, "#", 2)[0]); !stems[stem] {
			t.Errorf("%s -> %s: target not in sitemap", from, to)
		}
	}
	if len(r) == 0 {
		t.Error("redirects.json is empty")
	}
}
