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
- **Voice**: Technical but accessible — assume the reader knows programming but not Petri nets

## Source Material

Content draws from three sibling projects in this workspace:

| Source | Path | Content |
|--------|------|---------|
| blog.stackdump.com | `~/Workspace/blog-stackdump-com/content/posts/` | 21 blog posts (~21k words) |
| petri-pilot | `~/Workspace/petri-pilot/` | Docs, 13 example models, generated apps |
| go-pflow | `~/Workspace/go-pflow/` | 60+ docs, 18 examples, research paper outline |

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
