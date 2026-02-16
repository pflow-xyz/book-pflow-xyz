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

## Workflow

- Draft chapters in `chapters/` as markdown
- Pull and adapt content from source projects — don't copy verbatim, rewrite for book narrative
- Each chapter should have a clear learning objective stated at the top
- Every concept introduced should have a corresponding working example
