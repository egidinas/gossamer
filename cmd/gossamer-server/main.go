package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/egidinas/gossamer/internal/api"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8095", "HTTP listen address")
	root := flag.String("root", ".", "repository or fixture root")
	webDir := flag.String("web-dir", "", "optional built web/dist directory to serve with the API")
	flag.Parse()

	var handler http.Handler = api.New(*root)
	if *webDir != "" {
		handler = api.NewWithStatic(*root, *webDir)
		log.Printf("gossamer demo listening on http://%s with web assets from %s", *addr, *webDir)
	} else {
		log.Printf("gossamer API listening on http://%s", *addr)
	}
	log.Fatal(http.ListenAndServe(*addr, handler))
}
