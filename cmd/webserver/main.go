package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

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