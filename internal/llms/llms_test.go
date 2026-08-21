package llms

import (
	"strings"
	"testing"

	book "github.com/pflow-xyz/book-pflow-xyz"
)

func TestParseSummary(t *testing.T) {
	es := ParseSummary([]byte("# Summary\n\n[Preface](preface.md)\n\n# Part I\n\n- [A](a.md)\n- [B](b.md)\n"))
	if len(es) != 3 || es[0].Part != "" || es[1].Part != "Part I" || es[2].Slug() != "b.html" {
		t.Fatalf("unexpected entries: %+v", es)
	}
}

func TestFullFromEmbeddedSources(t *testing.T) {
	out, err := Full(book.Sources)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"Source: " + BaseURL + "/preface.html",
		"Source: " + BaseURL + "/ch21-epilogue.html",
		"Source: " + BaseURL + "/appendix-e-categorical-foundations.html",
		"](" + BaseURL + "/ch13-topology-driven-verification.html)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(s, "](ch13-topology-driven-verification.md)") {
		t.Error("relative .md link not rewritten")
	}
	if len(out) < 200_000 {
		t.Errorf("suspiciously short llms-full.txt: %d bytes", len(out))
	}
}

func TestIndexMatchesSummary(t *testing.T) {
	idx, err := book.Sources.ReadFile("llms.txt")
	if err != nil {
		t.Fatal(err)
	}
	sum, _ := book.Sources.ReadFile("chapters/SUMMARY.md")
	for _, e := range ParseSummary(sum) {
		if !strings.Contains(string(idx), BaseURL+"/"+e.Slug()) {
			t.Errorf("llms.txt is missing a link to %s", e.Slug())
		}
	}
}
