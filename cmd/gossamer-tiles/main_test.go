package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDataVersionRejectsUnsafePathSegments(t *testing.T) {
	invalid := []string{
		"",
		".",
		"..",
		"../outside",
		"..\\outside",
		"nested/version",
		"nested\\version",
		"version with space",
	}
	for _, version := range invalid {
		if err := validateDataVersion(version); err == nil {
			t.Fatalf("validateDataVersion(%q) succeeded, want error", version)
		}
	}
}

func TestValidateDataVersionAcceptsConservativeSegments(t *testing.T) {
	valid := []string{
		"v7b7e73e7",
		"physics-static-20260507T072127Z",
		"bundle_1.2.3",
	}
	for _, version := range valid {
		if err := validateDataVersion(version); err != nil {
			t.Fatalf("validateDataVersion(%q) = %v, want nil", version, err)
		}
	}
}

func TestReplaceDirRestoresExistingDestinationOnFinalRenameFailure(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "current")

	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalRename := renamePath
	defer func() { renamePath = originalRename }()
	renamePath = func(oldPath, newPath string) error {
		if oldPath == dst+".tmp" && newPath == dst {
			return errors.New("forced final rename failure")
		}
		return originalRename(oldPath, newPath)
	}

	if err := replaceDir(dst, src); err == nil {
		t.Fatal("replaceDir succeeded, want forced rename error")
	}
	data, err := os.ReadFile(filepath.Join(dst, "old.txt"))
	if err != nil {
		t.Fatalf("existing destination was not restored: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("restored destination content = %q, want old", string(data))
	}
	if _, err := os.Stat(filepath.Join(dst, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("new file exists after failed replace, err=%v", err)
	}
}
