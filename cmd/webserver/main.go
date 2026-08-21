package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	book "github.com/pflow-xyz/book-pflow-xyz"
	"github.com/pflow-xyz/book-pflow-xyz/internal/llms"
	"github.com/pflow-xyz/book-pflow-xyz/internal/static"
)

func main() {
	port := flag.Int("port", 8087, "Port to listen on")
	flag.Parse()

	publicFS, err := static.Public()
	if err != nil {
		log.Fatalf("Failed to load static files: %v", err)
	}

	server := &http.ServeMux{}
	registerDeployRoutes(server)
	server.HandleFunc("/metrics", handleMetrics)
	registerLLMSRoutes(server)
	server.Handle("/", http.FileServerFS(publicFS))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("book.pflow.xyz listening on %s", addr)
	if err := http.ListenAndServe(addr, logRequests(server)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// registerLLMSRoutes serves llms.txt (curated index, embedded from the repo
// root) and llms-full.txt (the whole book, generated once at startup from
// the embedded chapter markdown so it cannot drift from the rendered site).
func registerLLMSRoutes(mux *http.ServeMux) {
	index, err := book.Sources.ReadFile("llms.txt")
	if err != nil {
		log.Fatalf("llms.txt: %v", err)
	}
	full, err := llms.Full(book.Sources)
	if err != nil {
		log.Fatalf("llms-full.txt: %v", err)
	}
	serve := func(body []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Cache-Control", "public, max-age=3600")
			w.Write(body)
		}
	}
	mux.HandleFunc("/llms.txt", serve(index))
	mux.HandleFunc("/llms-full.txt", serve(full))
}
