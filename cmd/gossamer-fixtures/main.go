package main

import (
	"log"

	"github.com/egidinas/gossamer/internal/report"
	"github.com/egidinas/gossamer/internal/synthetic"
)

func main() {
	if err := synthetic.WritePublicFixtures("."); err != nil {
		log.Fatal(err)
	}
	for _, id := range report.CampaignIDs() {
		if err := report.Write(".", id); err != nil {
			log.Fatal(err)
		}
	}
}
