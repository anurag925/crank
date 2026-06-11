package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResult_FeaturesUsed_WithFeatures(t *testing.T) {
	r := &Result{
		ProjectDir: "/tmp/test",
		Features:   []string{"base", "auth"},
	}
	got := r.FeaturesUsed()
	if len(got) != 2 || got[0] != "base" || got[1] != "auth" {
		t.Errorf("FeaturesUsed() = %v, want [base auth]", got)
	}
}

func TestResult_FeaturesUsed_Nil(t *testing.T) {
	var r *Result
	got := r.FeaturesUsed()
	if got != nil {
		t.Errorf("nil Result.FeaturesUsed() = %v, want nil", got)
	}
}

func TestResult_FeaturesUsed_FallbackToManifest(t *testing.T) {
	dir := t.TempDir()
	manifestContent := `project_name: test
module_path: github.com/test
features:
  - base
  - postgres
`
	os.WriteFile(filepath.Join(dir, ".bootstrap.yaml"), []byte(manifestContent), 0o644)

	r := &Result{ProjectDir: dir}
	got := r.FeaturesUsed()
	if len(got) != 2 {
		t.Fatalf("FeaturesUsed() len = %d, want 2", len(got))
	}
	if got[0] != "base" || got[1] != "postgres" {
		t.Errorf("FeaturesUsed() = %v, want [base postgres]", got)
	}
}

func TestResult_FeaturesUsed_NoManifest(t *testing.T) {
	r := &Result{ProjectDir: "/nonexistent"}
	got := r.FeaturesUsed()
	if got != nil {
		t.Errorf("FeaturesUsed() = %v, want nil", got)
	}
}
