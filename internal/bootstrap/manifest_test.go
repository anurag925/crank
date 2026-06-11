package bootstrap

import (
	"testing"
)

func TestParseManifest(t *testing.T) {
	data := []byte(`project_name: myapp
module_path: github.com/example/myapp
features:
  - base
  - auth
`)
	m, err := parseManifest(data)
	if err != nil {
		t.Fatalf("parseManifest: %v", err)
	}
	if m.ProjectName != "myapp" {
		t.Errorf("ProjectName = %q, want %q", m.ProjectName, "myapp")
	}
	if m.ModulePath != "github.com/example/myapp" {
		t.Errorf("ModulePath = %q, want %q", m.ModulePath, "github.com/example/myapp")
	}
	if len(m.Features) != 2 || m.Features[0] != "base" || m.Features[1] != "auth" {
		t.Errorf("Features = %v, want [base auth]", m.Features)
	}
}

func TestParseManifest_Invalid(t *testing.T) {
	_, err := parseManifest([]byte("{{invalid yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestEncodeManifest(t *testing.T) {
	m := &manifest{
		ProjectName: "myapp",
		ModulePath:  "github.com/example/myapp",
		Features:    []string{"base", "postgres"},
	}
	body, err := encodeManifest(m)
	if err != nil {
		t.Fatalf("encodeManifest: %v", err)
	}

	// round-trip
	parsed, err := parseManifest([]byte(body))
	if err != nil {
		t.Fatalf("parseManifest round-trip: %v", err)
	}
	if parsed.ProjectName != m.ProjectName {
		t.Errorf("ProjectName = %q, want %q", parsed.ProjectName, m.ProjectName)
	}
	if parsed.ModulePath != m.ModulePath {
		t.Errorf("ModulePath = %q, want %q", parsed.ModulePath, m.ModulePath)
	}
	if len(parsed.Features) != len(m.Features) {
		t.Fatalf("Features len = %d, want %d", len(parsed.Features), len(m.Features))
	}
	for i, f := range parsed.Features {
		if f != m.Features[i] {
			t.Errorf("Features[%d] = %q, want %q", i, f, m.Features[i])
		}
	}
}

func TestEncodeManifest_Nil(t *testing.T) {
	// should not panic
	body, err := encodeManifest(nil)
	if err != nil {
		t.Fatalf("encodeManifest(nil): %v", err)
	}
	if body == "" {
		t.Error("expected non-empty output")
	}
}
