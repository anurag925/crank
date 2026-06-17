package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindBinary_FallsBackToGOBIN(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("GOPATH", "")

	gobin := t.TempDir()
	t.Setenv("GOBIN", gobin)

	bin := filepath.Join(gobin, "fake-tool")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	got, err := FindBinary("fake-tool", "install fake-tool")
	if err != nil {
		t.Fatalf("FindBinary() error = %v", err)
	}
	if got != bin {
		t.Fatalf("FindBinary() = %q, want %q", got, bin)
	}
}

func TestFindBinary_FallsBackToGOPATHBin(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("GOBIN", "")

	gopath := t.TempDir()
	t.Setenv("GOPATH", gopath)

	binDir := filepath.Join(gopath, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir gopath bin: %v", err)
	}

	bin := filepath.Join(binDir, "fake-tool")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	got, err := FindBinary("fake-tool", "install fake-tool")
	if err != nil {
		t.Fatalf("FindBinary() error = %v", err)
	}
	if got != bin {
		t.Fatalf("FindBinary() = %q, want %q", got, bin)
	}
}
