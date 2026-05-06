package main

import (
	"flag"
	"log"

	"github.com/egidinas/gossamer/internal/report"
)

func main() {
	campaign := flag.String("campaign", "thermal_acceptance_fat", "campaign id")
	flag.Parse()
	if err := report.Write(".", *campaign); err != nil {
		log.Fatal(err)
	}
}
