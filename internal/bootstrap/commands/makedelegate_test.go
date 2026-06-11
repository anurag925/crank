package commands

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMakefileTargets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Makefile")
	content := `APP_NAME := myapp
MODULE   = github.com/example/myapp

.PHONY: help build migrate-up

help:
	@echo "help"

build: tidy
	go build ./...

migrate-up:
	migrate up

# a comment line
lint:
	golangci-lint run
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	targets, err := makefileTargets(path)
	if err != nil {
		t.Fatalf("makefileTargets: %v", err)
	}

	want := map[string]bool{
		"help":       true,
		"build":      true,
		"migrate-up": true,
		"lint":       true,
	}
	if !reflect.DeepEqual(targets, want) {
		t.Fatalf("targets mismatch:\n got: %v\nwant: %v", targets, want)
	}

	// Variable assignments and special targets must not be treated as targets.
	for _, notTarget := range []string{"APP_NAME", "MODULE", ".PHONY"} {
		if targets[notTarget] {
			t.Errorf("%q should not be parsed as a target", notTarget)
		}
	}
}

func TestSplitProjectFlag(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantProject string
		wantRest    []string
	}{
		{
			name:        "no project flag",
			args:        []string{"FOO=bar"},
			wantProject: ".",
			wantRest:    []string{"FOO=bar"},
		},
		{
			name:        "space separated",
			args:        []string{"--project", "./app", "FOO=bar"},
			wantProject: "./app",
			wantRest:    []string{"FOO=bar"},
		},
		{
			name:        "equals separated",
			args:        []string{"--project=./app", "-n"},
			wantProject: "./app",
			wantRest:    []string{"-n"},
		},
		{
			name:        "project flag with no remaining args",
			args:        []string{"--project", "./app"},
			wantProject: "./app",
			wantRest:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotProject, gotRest := splitProjectFlag(tc.args)
			if gotProject != tc.wantProject {
				t.Errorf("project: got %q want %q", gotProject, tc.wantProject)
			}
			if !reflect.DeepEqual(gotRest, tc.wantRest) {
				t.Errorf("rest: got %v want %v", gotRest, tc.wantRest)
			}
		})
	}
}
