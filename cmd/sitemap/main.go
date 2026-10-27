// Command sitemap writes sitemap.xml, derived from chapters/*.md, to stdout.
// The Makefile redirects it into build/html/ so the file is never committed.
package main

import (
	"log"
	"os"

	book "github.com/pflow-xyz/book-pflow-xyz"
)

func main() {
	out, err := book.Sitemap(book.Sources)
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(out)
}
