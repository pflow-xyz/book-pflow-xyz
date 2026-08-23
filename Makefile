.PHONY: build build-web book epub pdf run serve test clean

# Chapter source files in SUMMARY.md order (for Pandoc)
CHAPTERS = chapters/preface.md \
	chapters/ch01-why-petri-nets.md \
	chapters/ch02-mathematics-of-flow.md \
	chapters/ch03-discrete-to-continuous.md \
	chapters/ch04-token-language.md \
	chapters/ch05-resource-modeling.md \
	chapters/ch06-game-mechanics.md \
	chapters/ch07-constraint-satisfaction.md \
	chapters/ch08-optimization.md \
	chapters/ch09-enzyme-kinetics.md \
	chapters/ch10-complex-state-machines.md \
	chapters/ch11-process-mining.md \
	chapters/ch12-zero-knowledge-proofs.md \
	chapters/ch13-topology-driven-verification.md \
	chapters/ch14-on-chain-verification.md \
	chapters/ch15-exponential-weights.md \
	chapters/ch16-declarative-infrastructure.md \
	chapters/ch17-visual-editor.md \
	chapters/ch18-code-generation.md \
	chapters/ch19-go-pflow-library.md \
	chapters/ch20-dual-implementation.md \
	chapters/ch21-epilogue.md \
	chapters/appendix-a-solver-reference.md \
	chapters/appendix-b-token-grammar.md \
	chapters/appendix-c-json-schema.md \
	chapters/appendix-d-glossary.md

# mdBook → HTML site
book:
	mdbook build
	cp robots.txt sitemap.xml build/html/
	python3 seo-postprocess.py build/html

# Pandoc → EPUB
epub: build/book.epub

build/book.epub: $(CHAPTERS) metadata.yaml templates/epub.css
	@mkdir -p build
	pandoc --metadata-file=metadata.yaml \
		--toc --toc-depth=2 \
		--resource-path=chapters \
		--epub-cover-image=figures/cover.svg \
		--css=templates/epub.css \
		-o build/book.epub $(CHAPTERS) 2>/dev/null || \
	pandoc --metadata-file=metadata.yaml \
		--toc --toc-depth=2 \
		--resource-path=chapters \
		--css=templates/epub.css \
		-o build/book.epub $(CHAPTERS)

# Pandoc → PDF (requires xelatex)
pdf: build/book.pdf

build/book.pdf: $(CHAPTERS) metadata.yaml
	@mkdir -p build
	pandoc --metadata-file=metadata.yaml \
		--pdf-engine=xelatex \
		--resource-path=chapters \
		--toc --toc-depth=2 \
		-o build/book.pdf $(CHAPTERS)

# Full build: HTML site + epub + pdf + Go binary
build: book epub pdf
	rm -rf internal/static/public
	cp -r build/html internal/static/public
	go build -o bin/webserver ./cmd/webserver

# Web-only build: HTML site + Go binary (no pandoc/latex needed)
build-web: book
	rm -rf internal/static/public
	cp -r build/html internal/static/public
	go build -o bin/webserver ./cmd/webserver

# Run the Go server
run: build-web
	./bin/webserver -port 8087

# mdBook dev server with hot reload
serve:
	mdbook serve --open

# Toolchain parity: run here and on pflow.dev, diff the output.
# pflow.dev (2026-08-23): mdbook 0.5.2, mdbook-katex 0.10.0-alpha, pandoc 3.1.3, xelatex (texlive 2023).
# mdbook + mdbook-katex: cargo install; pandoc: static tarball into ~/.local/bin; xelatex: apt texlive-xetex.
# Math is pre-rendered by mdbook-katex at build time — no client JS; a page with raw $...$ means the preprocessor is missing.
toolchain:
	@for t in mdbook mdbook-katex pandoc xelatex go; do \
	  printf '%-14s' $$t; command -v $$t >/dev/null && ([ $$t = go ] && go version || $$t --version 2>&1 | head -1) || echo MISSING; done

test:
	mdbook test 2>/dev/null || true
	go vet ./...

clean:
	rm -rf bin/ build/ internal/static/public/
