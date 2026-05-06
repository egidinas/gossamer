package main

import (
	"log"

	"github.com/egidinas/gossamer/internal/synthetic"
)

func main() {
	if err := synthetic.WritePublicFixtures("."); err != nil {
		log.Fatal(err)
	}
}
