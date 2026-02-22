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
	chapters/ch13-exponential-weights.md \
	chapters/ch14-declarative-infrastructure.md \
	chapters/ch15-visual-editor.md \
	chapters/ch16-code-generation.md \
	chapters/ch17-go-pflow-library.md \
	chapters/ch18-dual-implementation.md \
	chapters/ch19-epilogue.md \
	chapters/appendix-a-solver-reference.md \
	chapters/appendix-b-token-grammar.md \
	chapters/appendix-c-json-schema.md \
	chapters/appendix-d-glossary.md

# mdBook → HTML site
book:
	mdbook build

# Pandoc → EPUB
epub: build/book.epub

build/book.epub: $(CHAPTERS) metadata.yaml templates/epub.css
	@mkdir -p build
	pandoc --metadata-file=metadata.yaml \
		--toc --toc-depth=2 \
		--epub-cover-image=figures/cover.svg \
		--css=templates/epub.css \
		-o build/book.epub $(CHAPTERS) 2>/dev/null || \
	pandoc --metadata-file=metadata.yaml \
		--toc --toc-depth=2 \
		--css=templates/epub.css \
		-o build/book.epub $(CHAPTERS)

# Pandoc → PDF (requires xelatex)
pdf: build/book.pdf

build/book.pdf: $(CHAPTERS) metadata.yaml
	@mkdir -p build
	pandoc --metadata-file=metadata.yaml \
		--pdf-engine=xelatex \
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

test:
	mdbook test 2>/dev/null || true
	go vet ./...

clean:
	rm -rf bin/ build/ internal/static/public/
