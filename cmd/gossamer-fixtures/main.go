package main

import (
	"log"
	"sort"

	"github.com/egidinas/gossamer/internal/report"
	"github.com/egidinas/gossamer/internal/synthetic"
)

func main() {
	if err := synthetic.WritePublicFixtures("."); err != nil {
		log.Fatal(err)
	}
	set := synthetic.Build()
	campaignIDs := make([]string, 0, len(set.Campaigns))
	for id := range set.Campaigns {
		campaignIDs = append(campaignIDs, id)
	}
	sort.Strings(campaignIDs)
	for _, id := range campaignIDs {
		if err := report.Write(".", id); err != nil {
			log.Fatal(err)
		}
	}
}
