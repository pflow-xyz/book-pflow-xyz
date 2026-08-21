// Package llms builds llms.txt / llms-full.txt (https://llmstxt.org) for
// book.pflow.xyz. llms.txt is a hand-curated table of contents kept in the
// repo root; llms-full.txt is generated from chapters/SUMMARY.md order by
// concatenating the chapter markdown, so it always matches the served book.
package llms

import (
	"bytes"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

const BaseURL = "https://book.pflow.xyz"

var (
	summaryEntry = regexp.MustCompile(`^\s*(?:-\s*)?\[([^\]]+)\]\(([^)]+\.md)\)\s*$`)
	summaryPart  = regexp.MustCompile(`^#\s+(.+?)\s*$`)
	relMDLink    = regexp.MustCompile(`\]\(([A-Za-z0-9_-]+)\.md(#[^)]*)?\)`)
	relFigure    = regexp.MustCompile(`\]\(\.\./figures/([^)]+)\)`)
)

// Entry is one row of SUMMARY.md.
type Entry struct {
	Part  string // most recent "# Part ..." heading, "" for the preface
	Title string
	File  string // e.g. ch01-why-petri-nets.md
}

// Slug is the served HTML path for the entry.
func (e Entry) Slug() string { return strings.TrimSuffix(e.File, ".md") + ".html" }

// ParseSummary returns chapter entries in SUMMARY.md order.
func ParseSummary(summary []byte) []Entry {
	var out []Entry
	part := ""
	for _, line := range strings.Split(string(summary), "\n") {
		if m := summaryEntry.FindStringSubmatch(line); m != nil {
			out = append(out, Entry{Part: part, Title: m[1], File: m[2]})
			continue
		}
		if m := summaryPart.FindStringSubmatch(line); m != nil && m[1] != "Summary" {
			part = m[1]
		}
	}
	return out
}

// Full renders llms-full.txt from the embedded chapter sources.
func Full(src fs.FS) ([]byte, error) {
	summary, err := fs.ReadFile(src, "chapters/SUMMARY.md")
	if err != nil {
		return nil, err
	}
	entries := ParseSummary(summary)

	var b bytes.Buffer
	b.WriteString("# Petri Nets as a Universal Abstraction — full text\n\n")
	b.WriteString("> A Practitioner's Guide to Modeling with pflow. This file is the complete\n")
	b.WriteString("> book, generated from the same markdown that renders " + BaseURL + ".\n")
	b.WriteString("> Each chapter is preceded by its canonical URL. Math is LaTeX ($...$),\n")
	b.WriteString("> code is fenced. The short index is at " + BaseURL + "/llms.txt.\n\n")
	b.WriteString("## Contents\n\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "- [%s](%s/%s)\n", e.Title, BaseURL, e.Slug())
	}
	b.WriteString("\nRelated: https://pflow.xyz (editor + browser solver), https://pilot.pflow.xyz (code generation, MCP analysis tools), https://blog.stackdump.com (essays), https://github.com/pflow-xyz/go-pflow (Go library).\n")

	lastPart := ""
	for _, e := range entries {
		raw, err := fs.ReadFile(src, "chapters/"+e.File)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.File, err)
		}
		if e.Part != lastPart {
			fmt.Fprintf(&b, "\n\n# %s\n", e.Part)
			lastPart = e.Part
		}
		fmt.Fprintf(&b, "\n\n---\n\n## %s\n\nSource: %s/%s\n\n", e.Title, BaseURL, e.Slug())
		b.Write(rewrite(stripH1(raw)))
		b.WriteString("\n")
	}
	return b.Bytes(), nil
}

// stripH1 removes the leading "# Title" line (we emit our own heading).
func stripH1(md []byte) []byte {
	trimmed := bytes.TrimLeft(md, "\n")
	if bytes.HasPrefix(trimmed, []byte("# ")) {
		if i := bytes.IndexByte(trimmed, '\n'); i >= 0 {
			return bytes.TrimLeft(trimmed[i+1:], "\n")
		}
		return nil
	}
	return md
}

// rewrite turns relative chapter/figure links into absolute served URLs.
func rewrite(md []byte) []byte {
	md = relMDLink.ReplaceAll(md, []byte("]("+BaseURL+"/$1.html$2)"))
	md = relFigure.ReplaceAll(md, []byte("]("+BaseURL+"/figures/$1)"))
	return md
}
