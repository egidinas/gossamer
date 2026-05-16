package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/egidinas/gossamer/internal/report"
)

const defaultCampaign = "thermal_acceptance_fat"

type reportOptions struct {
	campaign string
}

func parseReportOptions(args []string, output io.Writer) (reportOptions, error) {
	fs := flag.NewFlagSet("gossamer-report", flag.ContinueOnError)
	fs.SetOutput(output)
	campaign := fs.String("campaign", defaultCampaign, "campaign id")
	if err := fs.Parse(args); err != nil {
		return reportOptions{}, err
	}
	if fs.NArg() > 0 {
		return reportOptions{}, fmt.Errorf("unexpected positional argument %q; use --campaign", fs.Arg(0))
	}
	return reportOptions{campaign: *campaign}, nil
}

func main() {
	opts, err := parseReportOptions(os.Args[1:], os.Stderr)
	if err != nil {
		log.Fatal(err)
	}
	if err := report.Write(".", opts.campaign); err != nil {
		log.Fatal(err)
	}
}
