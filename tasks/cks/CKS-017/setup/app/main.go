package main

import (
	"log"
	"net/http"
	"os"
)

// A deliberately tiny payments health endpoint. It has no third-party
// dependencies, so it compiles to a fully static binary — which is exactly
// what lets it run on a distroless (or scratch) base with no libc.
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("payments-api: ok\n"))
	})

	addr := ":8080"
	log.Printf("payments-api listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
