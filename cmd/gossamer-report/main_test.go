package main

import (
	"io"
	"strings"
	"testing"
)

func TestParseReportOptionsDefaultsCampaign(t *testing.T) {
	opts, err := parseReportOptions(nil, io.Discard)
	if err != nil {
		t.Fatalf("parseReportOptions returned error: %v", err)
	}
	if opts.campaign != defaultCampaign {
		t.Fatalf("campaign = %q, want %q", opts.campaign, defaultCampaign)
	}
}

func TestParseReportOptionsAcceptsCampaignFlag(t *testing.T) {
	opts, err := parseReportOptions([]string{"--campaign", "tvac_qualification"}, io.Discard)
	if err != nil {
		t.Fatalf("parseReportOptions returned error: %v", err)
	}
	if opts.campaign != "tvac_qualification" {
		t.Fatalf("campaign = %q, want tvac_qualification", opts.campaign)
	}
}

func TestParseReportOptionsRejectsPositionalCampaign(t *testing.T) {
	_, err := parseReportOptions([]string{"tvac_qualification"}, io.Discard)
	if err == nil {
		t.Fatal("parseReportOptions returned nil error for positional argument")
	}
	if !strings.Contains(err.Error(), "unexpected positional argument") {
		t.Fatalf("error = %q, want unexpected positional argument", err.Error())
	}
}
