package main

import (
	"log"
	"net/http"

	"github.com/egidinas/gossamer/internal/api"
)

func main() {
	addr := "127.0.0.1:8095"
	log.Printf("gossamer API listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, api.New(".")))
}
