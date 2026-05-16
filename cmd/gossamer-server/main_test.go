package main

import (
	"os"
	"path/filepath"
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

func TestOpenWrtRemoteListenInvocationPreservesAllowRemote(t *testing.T) {
	initPath := filepath.Join("..", "..", "deploy", "openwrt", "gossamer.init")
	contents, err := os.ReadFile(initPath)
	if err != nil {
		t.Fatalf("read %s: %v", initPath, err)
	}
	initScript := string(contents)
	if !strings.Contains(initScript, `GOSSAMER_ADDR="${GOSSAMER_ADDR:-0.0.0.0:8095}"`) {
		t.Fatalf("test expects OpenWrt profile to default to wildcard listen address")
	}
	if !strings.Contains(initScript, `-allow-remote`) {
		t.Fatalf("OpenWrt wildcard listen invocation must pass -allow-remote")
	}
}
