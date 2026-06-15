package bootstrap

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dirFeature is used only for tests that don't call generateFeature
// (e.g. validateFeatures, register, etc.)
type dirFeature struct {
	name string
	dir  string
}

func (f *dirFeature) Name() string           { return f.name }
func (f *dirFeature) Description() string    { return "dir feature " + f.name }
func (f *dirFeature) Files() []FileMapping   { return nil }
func (f *dirFeature) Templates() embed.FS    { return embed.FS{} }
func (f *dirFeature) Dependencies() []string { return nil }
func (f *dirFeature) Requirements() []string { return nil }

// --- Generate tests (using real features from GlobalRegistry) ---

func TestGenerate_EmptyProjectName(t *testing.T) {
	_, err := Generate(GlobalRegistry, Options{ProjectName: ""})
	if err == nil {
		t.Fatal("expected error for empty project name")
	}
}

func TestGenerate_NonEmptyDir_WithoutForce(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "existing")
	os.MkdirAll(projectDir, 0o755)
	os.WriteFile(filepath.Join(projectDir, "file.txt"), []byte("x"), 0o644)

	_, err := Generate(GlobalRegistry, Options{
		ProjectName: "existing",
		TargetDir:   tmp,
	})
	if err == nil {
		t.Fatal("expected error for non-empty dir without --force")
	}
}

func TestGenerate_NonEmptyDir_WithForce(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "forced")
	os.MkdirAll(projectDir, 0o755)
	os.WriteFile(filepath.Join(projectDir, "old.txt"), []byte("old"), 0o644)

	result, err := Generate(GlobalRegistry, Options{
		ProjectName: "forced",
		TargetDir:   tmp,
		Force:       true,
	})
	if err != nil {
		t.Fatalf("Generate with force: %v", err)
	}

	// old file should be gone
	if _, err := os.Stat(filepath.Join(result.ProjectDir, "old.txt")); err == nil {
		t.Error("expected old.txt to be removed with --force")
	}

	// base files should exist
	if _, err := os.Stat(filepath.Join(result.ProjectDir, "go.mod")); os.IsNotExist(err) {
		t.Error("expected go.mod to exist after force generate")
	}
}

func TestGenerate_DuplicateFeatures(t *testing.T) {
	tmp := t.TempDir()
	_, err := Generate(GlobalRegistry, Options{
		ProjectName: "dup",
		TargetDir:   tmp,
		Features:    []string{"base", "base"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate features")
	}
}

func TestGenerate_UnknownFeature(t *testing.T) {
	tmp := t.TempDir()
	_, err := Generate(GlobalRegistry, Options{
		ProjectName: "unk",
		TargetDir:   tmp,
		Features:    []string{"nope"},
	})
	if err == nil {
		t.Fatal("expected error for unknown feature")
	}
}

func TestGenerate_InvalidProjectName(t *testing.T) {
	_, err := Generate(GlobalRegistry, Options{ProjectName: "---"})
	if err == nil {
		t.Fatal("expected error for project name that resolves to empty package")
	}
}

func TestGenerate_BaseAutoIncluded(t *testing.T) {
	tmp := t.TempDir()
	result, err := Generate(GlobalRegistry, Options{
		ProjectName: "autobase",
		TargetDir:   tmp,
		Features:    []string{"auth"}, // no explicit "base"
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	found := false
	for _, f := range result.Features {
		if f == "base" {
			found = true
		}
	}
	if !found {
		t.Errorf("base not auto-included: %v", result.Features)
	}
}

func TestGenerate_FilesSorted(t *testing.T) {
	tmp := t.TempDir()
	result, err := Generate(GlobalRegistry, Options{
		ProjectName: "sorted",
		TargetDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for i := 1; i < len(result.Files); i++ {
		if result.Files[i-1] > result.Files[i] {
			t.Errorf("files not sorted: %q > %q at index %d", result.Files[i-1], result.Files[i], i)
			break
		}
	}
}

// --- Add tests ---

func TestAdd_AlreadyInstalled(t *testing.T) {
	tmp := t.TempDir()
	result, err := Generate(GlobalRegistry, Options{
		ProjectName: "dupadd",
		TargetDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	_, err = Add(GlobalRegistry, result.ProjectDir, "base")
	if err == nil {
		t.Fatal("expected error adding already-installed feature")
	}
}

func TestAdd_NonexistentDir(t *testing.T) {
	_, err := Add(GlobalRegistry, "/nonexistent/path", "auth")
	if err == nil {
		t.Fatal("expected error for nonexistent project dir")
	}
}

func TestAdd_AuthToBase(t *testing.T) {
	tmp := t.TempDir()
	result, err := Generate(GlobalRegistry, Options{
		ProjectName: "addauth",
		TargetDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// auth files should NOT exist yet
	if _, err := os.Stat(filepath.Join(result.ProjectDir, "internal/adapters/http/web/middleware/auth.go")); err == nil {
		t.Error("auth middleware should not exist before Add")
	}

	result2, err := Add(GlobalRegistry, result.ProjectDir, "auth")
	if err != nil {
		t.Fatalf("Add auth: %v", err)
	}

	// auth files should now exist
	for _, f := range []string{
		"internal/adapters/http/web/middleware/auth.go",
		"internal/adapters/crypto/bcrypt_hasher.go",
		"internal/adapters/http/web/auth_handler.go",
	} {
		if _, err := os.Stat(filepath.Join(result2.ProjectDir, f)); os.IsNotExist(err) {
			t.Errorf("expected %s after Add auth", f)
		}
	}

	// manifest should include auth
	data, _ := os.ReadFile(filepath.Join(result2.ProjectDir, ".crank.yaml"))
	if !strings.Contains(string(data), "auth") {
		t.Error("manifest missing auth after Add")
	}
}

func TestAdd_PostgresToBase(t *testing.T) {
	tmp := t.TempDir()
	result, err := Generate(GlobalRegistry, Options{
		ProjectName: "addpg",
		TargetDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	result2, err := Add(GlobalRegistry, result.ProjectDir, "postgres")
	if err != nil {
		t.Fatalf("Add postgres: %v", err)
	}

	// postgres files should exist
	for _, f := range []string{
		"internal/adapters/persistence/postgres/db.go",
		"internal/adapters/persistence/postgres/migrate.go",
		"migrations/000001_init.up.sql",
	} {
		if _, err := os.Stat(filepath.Join(result2.ProjectDir, f)); os.IsNotExist(err) {
			t.Errorf("expected %s after Add postgres", f)
		}
	}
}

// --- renderTemplate tests ---

func TestRenderTemplate_Basic(t *testing.T) {
	ctx := NewContext("world", "github.com/example/world", []string{"base"})
	rendered, err := renderTemplate("hello.txt.tmpl", "Hello {{.ProjectName}}!", ctx)
	if err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	if rendered != "Hello world!" {
		t.Errorf("rendered = %q, want %q", rendered, "Hello world!")
	}
}

func TestRenderTemplate_ConditionalBlocks(t *testing.T) {
	tmpl := `start
{{- if .Has "auth"}}
AUTH
{{- end}}
{{- if .Has "postgres"}}
PG
{{- end}}
end`

	t.Run("auth only", func(t *testing.T) {
		ctx := NewContext("app", "app", []string{"base", "auth"})
		out, err := renderTemplate("test", tmpl, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "AUTH") {
			t.Error("expected AUTH in output")
		}
		if strings.Contains(out, "PG") {
			t.Error("unexpected PG in output")
		}
	})

	t.Run("postgres only", func(t *testing.T) {
		ctx := NewContext("app", "app", []string{"base", "postgres"})
		out, err := renderTemplate("test", tmpl, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "AUTH") {
			t.Error("unexpected AUTH in output")
		}
		if !strings.Contains(out, "PG") {
			t.Error("expected PG in output")
		}
	})

	t.Run("both", func(t *testing.T) {
		ctx := NewContext("app", "app", []string{"base", "auth", "postgres"})
		out, err := renderTemplate("test", tmpl, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "AUTH") || !strings.Contains(out, "PG") {
			t.Errorf("expected both AUTH and PG, got:\n%s", out)
		}
	})
}

func TestRenderTemplate_RangeLoop(t *testing.T) {
	tmpl := `features:
{{- range .Features}}
  - {{.}}
{{- end}}`
	ctx := NewContext("app", "app", []string{"base", "auth"})
	out, err := renderTemplate("test", tmpl, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "- base") || !strings.Contains(out, "- auth") {
		t.Errorf("expected feature list, got:\n%s", out)
	}
}

func TestRenderTemplate_MissingKey(t *testing.T) {
	ctx := NewContext("app", "app", nil)
	_, err := renderTemplate("test", "{{.NoSuchField}}", ctx)
	if err == nil {
		t.Fatal("expected error for missing key (missingkey=error)")
	}
}

func TestRenderTemplate_ModulePath(t *testing.T) {
	ctx := NewContext("myapp", "github.com/org/myapp", []string{"base"})
	out, err := renderTemplate("test", "import \"{{.ModulePath}}/internal/config\"", ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := "import \"github.com/org/myapp/internal/config\""
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// --- helper function tests ---

func TestContains(t *testing.T) {
	tests := []struct {
		list   []string
		target string
		want   bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{nil, "a", false},
		{[]string{}, "a", false},
	}
	for _, tt := range tests {
		if got := contains(tt.list, tt.target); got != tt.want {
			t.Errorf("contains(%v, %q) = %v, want %v", tt.list, tt.target, got, tt.want)
		}
	}
}

func TestEnsureBase(t *testing.T) {
	tests := []struct {
		input []string
		want  []string
	}{
		{nil, []string{"base"}},
		{[]string{"auth"}, []string{"base", "auth"}},
		{[]string{"base", "auth"}, []string{"base", "auth"}},
	}
	for _, tt := range tests {
		got := ensureBase(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("ensureBase(%v) len = %d, want %d", tt.input, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ensureBase(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestAppendUnique(t *testing.T) {
	got := appendUnique([]string{"a", "b"}, "b", "c", "a")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestUnique(t *testing.T) {
	got := unique([]string{"a", "b", "a", "c", "b"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestValidateFeatures(t *testing.T) {
	reg := NewRegistry()
	reg.MustRegister(&dirFeature{name: "base"})
	reg.MustRegister(&dirFeature{name: "auth"})

	t.Run("valid", func(t *testing.T) {
		if err := validateFeatures(reg, []string{"base", "auth"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		if err := validateFeatures(reg, []string{"base", "nope"}); err == nil {
			t.Error("expected error for unknown feature")
		}
	})
	t.Run("duplicate", func(t *testing.T) {
		if err := validateFeatures(reg, []string{"base", "base"}); err == nil {
			t.Error("expected error for duplicate feature")
		}
	})
}
