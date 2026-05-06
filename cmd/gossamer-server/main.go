package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/egidinas/gossamer/internal/api"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8095", "HTTP listen address")
	flag.Parse()

	log.Printf("gossamer API listening on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, api.New(".")))
}
