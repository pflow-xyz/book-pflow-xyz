# Book: Petri Nets as a Universal Abstraction

## Project Structure

```
book-pflow-xyz/
├── chapters/          # Markdown chapter files
├── figures/           # SVG diagrams and images
├── examples/          # Standalone code examples referenced in text
├── references/        # Bibliography and source material notes
└── outline.md         # Master table of contents and chapter plan
```

## Writing Conventions

- **Format**: Markdown with LaTeX math blocks where needed
- **Figures**: SVG preferred, referenced as `![caption](../figures/filename.svg)`
- **Code examples**: Inline for short snippets, separate files in `examples/` for anything >20 lines
- **Cross-references**: Use `[Chapter N: Title](chNN-slug.md)` format
- **Voice**: Technical but accessible — assume the reader knows programming but not Petri nets. Before publishing a chapter, run the `voice-audit`, `math-audit` and `model-audit` skills (`../blog-stackdump-com/.claude/skills/*/SKILL.md`) — it lists the LLM-draft tells and the argument defects (B-checks) this material is prone to.
- **Proof status of the categorical claims** lives in `PROOF-ROADMAP.md` (what is computed vs. kernel-checked vs. asserted; items P1–P4).
- **The core–observer boundary is two boundaries** (ρ, inside C; contextual arcs, outside C). The canonical statement is Appendix E, "Where the Free Structure Stops". Chapters cite it; they do not restate the table. See `../blog-stackdump-com/REVISION-ROADMAP.md` for the audit that produced it.

## Source Material

Content draws from sibling projects in this workspace:

| Source | Path | Content |
|--------|------|---------|
| blog.stackdump.com | `~/Workspace/blog-stackdump-com/content/posts/` | ~45 blog posts; `metamodel.md` is the canonical Metamodel statement |
| petri-pilot | `~/Workspace/petri-pilot/` | Docs, 13 example models, generated apps |
| go-pflow | `~/Workspace/go-pflow/` | 60+ docs, 18 examples, research paper outline |
| pflow-polyglot | `~/Workspace/pflow-polyglot/` | One model, five forms, golden trace (`FORMS.md`) |
| sim-pflow-xyz | `~/Workspace/sim-pflow-xyz/` | Business-operations instance (*The Model Is the App*) |

**Editorial rule (2026-08-23):** the book serves the Metamodel narrative
(model is a value; four primitives; declare then derive). Mathematical
connections that do not serve it are cut rather than kept for their own
sake — the comonad/coKleisli and tense-logic material went this way. ZK is
presented as a derived artifact, not a headline.

## Building & Publishing

- **HTML site**: `make book` (mdbook)
- **EPUB/PDF**: `make epub` / `make pdf` — requires pandoc and xelatex, which are only on `pflow.dev`
- **Full rebuild** (epub + pdf + webserver): `make build` — run on `pflow.dev`, not locally
- **Local dev**: `make serve` for mdbook hot-reload, or `make build-web` for the Go webserver (no pandoc needed)
- **Deploy**: push to main, then on pflow.dev: `cd ~/Workspace/book-pflow-xyz && git pull && make build && ~/services restart book-pflow-xyz`
- **Releases**: tag with `git tag vX.Y.Z`, create GitHub release with `gh release create`, attach `build/book.epub` and `build/book.pdf` built on pflow.dev

## Workflow

- Draft chapters in `chapters/` as markdown
- Pull and adapt content from source projects — don't copy verbatim, rewrite for book narrative
- Each chapter should have a clear learning objective stated at the top
- Every concept introduced should have a corresponding working example

## Decommissioning

The book at book.pflow.xyz (pflow.dev :8087).

See [Archiving, backing up and taking down a project](../stackdump-com/CLAUDE.md#archiving-backing-up-and-taking-down-a-project) for the ecosystem-wide procedure and the ordering. This section records only what **this** project holds, which is the part that differs.

**State that is not in git** (every path below is gitignored):

| Host | Path | Size | What it is |
|---|---|---|---|
| pflow.dev | `~/Workspace/book-pflow-xyz/` | - | no database found; content is in git |

**Specific to this project:**

- Content lives in the repo, so archiving the GitHub repo preserves the book itself.
