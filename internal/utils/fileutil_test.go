package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir_CreatesNestedDirs(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")
	if err := EnsureDir(target); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", target)
	}
}

func TestEnsureDir_Idempotent(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("second call: %v", err)
	}
}

func TestWriteFile_CreatesFileAndDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deep", "nested", "file.txt")
	if err := WriteFile(path, "hello world"); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("got %q, want %q", string(data), "hello world")
	}
}

func TestWriteFile_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := WriteFile(path, "first"); err != nil {
		t.Fatalf("WriteFile first: %v", err)
	}
	if err := WriteFile(path, "second"); err != nil {
		t.Fatalf("WriteFile second: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("got %q, want %q", string(data), "second")
	}
}

func TestPathExists(t *testing.T) {
	dir := t.TempDir()
	existingFile := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing file", existingFile, true},
		{"existing dir", dir, true},
		{"nonexistent", filepath.Join(dir, "nope"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathExists(tt.path); got != tt.want {
				t.Errorf("PathExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	emptyDir := filepath.Join(dir, "empty")
	if err := EnsureDir(emptyDir); err != nil {
		t.Fatal(err)
	}
	nonEmptyDir := filepath.Join(dir, "nonempty")
	if err := EnsureDir(nonEmptyDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		want    bool
		wantErr bool
	}{
		{"empty dir", emptyDir, true, false},
		{"non-empty dir", nonEmptyDir, false, false},
		{"nonexistent dir", filepath.Join(dir, "nope"), true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsEmptyDir(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("IsEmptyDir() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("IsEmptyDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToPackageName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"myapp", "myapp"},
		{"my-app", "myapp"},
		{"my_app", "myapp"},
		{"My-App", "myapp"},
		{"APP123", "app123"},
		{"hello world", "helloworld"},
		{"---", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ToPackageName(tt.input); got != tt.want {
				t.Errorf("ToPackageName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
