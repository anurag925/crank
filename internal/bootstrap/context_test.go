package bootstrap

import (
	"testing"
)

func TestNewContext_BasicFields(t *testing.T) {
	ctx := NewContext("myapp", "github.com/example/myapp", []string{"base"})

	if ctx.ProjectName != "myapp" {
		t.Errorf("ProjectName = %q, want %q", ctx.ProjectName, "myapp")
	}
	if ctx.ModulePath != "github.com/example/myapp" {
		t.Errorf("ModulePath = %q, want %q", ctx.ModulePath, "github.com/example/myapp")
	}
	if ctx.PackageName != "myapp" {
		t.Errorf("PackageName = %q, want %q", ctx.PackageName, "myapp")
	}
}

func TestNewContext_ModulePathDefaultsToProjectName(t *testing.T) {
	ctx := NewContext("myapp", "", []string{"base"})

	if ctx.ModulePath != "myapp" {
		t.Errorf("ModulePath = %q, want %q", ctx.ModulePath, "myapp")
	}
	if ctx.PackageName != "myapp" {
		t.Errorf("PackageName = %q, want %q", ctx.PackageName, "myapp")
	}
}

func TestNewContext_TrimsAndStripsTrailingSlash(t *testing.T) {
	ctx := NewContext("app", " github.com/x/app/ ", nil)

	if ctx.ModulePath != "github.com/x/app" {
		t.Errorf("ModulePath = %q, want %q", ctx.ModulePath, "github.com/x/app")
	}
	if ctx.PackageName != "app" {
		t.Errorf("PackageName = %q, want %q", ctx.PackageName, "app")
	}
}

func TestNewContext_PackageNameFromLastSegment(t *testing.T) {
	ctx := NewContext("proj", "github.com/org/special-pkg", nil)

	if ctx.PackageName != "special-pkg" {
		t.Errorf("PackageName = %q, want %q", ctx.PackageName, "special-pkg")
	}
}

func TestHas(t *testing.T) {
	ctx := NewContext("app", "app", []string{"base", "auth", "postgres"})

	tests := []struct {
		name string
		want bool
	}{
		{"base", true},
		{"auth", true},
		{"postgres", true},
		{"redis", false},
		{"mongodb", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ctx.Has(tt.name); got != tt.want {
				t.Errorf("Has(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestRequire_AllPresent(t *testing.T) {
	ctx := NewContext("app", "app", []string{"base", "auth"})
	if err := ctx.Require("base", "auth"); err != nil {
		t.Errorf("Require() unexpected error: %v", err)
	}
}

func TestRequire_Missing(t *testing.T) {
	ctx := NewContext("app", "app", []string{"base"})
	err := ctx.Require("base", "auth", "postgres")
	if err == nil {
		t.Fatal("Require() expected error, got nil")
	}
	want := "missing required features: auth, postgres"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
