package main

import (
	"strings"
	"testing"
)

func TestValidateListenAddressRequiresLoopbackUnlessAllowed(t *testing.T) {
	if err := validateListenAddress("127.0.0.1:8095", false); err != nil {
		t.Fatalf("loopback rejected: %v", err)
	}
	err := validateListenAddress("0.0.0.0:8095", false)
	if err == nil || !strings.Contains(err.Error(), "allow-remote") {
		t.Fatalf("wildcard err = %v, want allow-remote guidance", err)
	}
	if err := validateListenAddress("0.0.0.0:8095", true); err != nil {
		t.Fatalf("allowed wildcard rejected: %v", err)
	}
	if err := validateListenAddress("not-a-hostport", true); err == nil || !strings.Contains(err.Error(), "invalid listen address") {
		t.Fatalf("malformed allow-remote err = %v, want invalid listen address", err)
	}
}
