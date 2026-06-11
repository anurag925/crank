package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/anurag925/rev/internal/utils"
)

// Options control how a project is generated.
type Options struct {
	// ProjectName is the directory name (and default module path) for the new project.
	ProjectName string
	// ModulePath overrides the Go module path. Empty means use ProjectName.
	ModulePath string
	// TargetDir is the parent directory where the project will be created. Defaults to "."
	TargetDir string
	// Features is the ordered list of feature names to apply.
	Features []string
	// Force overwrites an existing non-empty directory.
	Force bool
}

// Result reports the files written during a generation.
type Result struct {
	ProjectDir string
	Files      []string
	Features   []string
}

// Generate creates a new project at TargetDir/ProjectName using the named features.
// The "base" feature is always applied first; remaining features run in the order supplied.
func Generate(reg *Registry, opts Options) (*Result, error) {
	if opts.ProjectName == "" {
		return nil, fmt.Errorf("project name is required")
	}
	if utils.ToPackageName(opts.ProjectName) == "" {
		return nil, fmt.Errorf("project name resolves to an empty package")
	}

	target := opts.TargetDir
	if target == "" {
		target = "."
	}
	projectDir := filepath.Join(target, opts.ProjectName)

	if !opts.Force {
		empty, err := utils.IsEmptyDir(projectDir)
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", projectDir, err)
		}
		if !empty {
			return nil, fmt.Errorf("target directory %s is not empty (use --force to overwrite)", projectDir)
		}
	} else if utils.PathExists(projectDir) {
		if err := os.RemoveAll(projectDir); err != nil {
			return nil, fmt.Errorf("clean %s: %w", projectDir, err)
		}
	}

	if err := utils.EnsureDir(projectDir); err != nil {
		return nil, fmt.Errorf("create project dir: %w", err)
	}

	features := ensureBase(opts.Features)
	if err := validateFeatures(reg, features); err != nil {
		return nil, err
	}

	ctx := NewContext(opts.ProjectName, opts.ModulePath, features)

	var all []string
	for _, name := range features {
		f, _ := reg.resolve(name)
		written, err := generateFeature(projectDir, f, ctx)
		if err != nil {
			return nil, err
		}
		all = append(all, written...)
	}
	sort.Strings(all)

	return &Result{ProjectDir: projectDir, Files: all, Features: features}, nil
}

// Add applies a feature to an existing project directory. The base feature is
// always re-rendered so its conditional sections (config schema, go.mod deps, ...)
// pick up the updated feature set. Other features are re-rendered in case their
// state is conditional too.
func Add(reg *Registry, projectDir, featureName string) (*Result, error) {
	if !utils.PathExists(projectDir) {
		return nil, fmt.Errorf("project directory %s does not exist", projectDir)
	}
	manifest, err := readManifest(projectDir)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	if contains(manifest.Features, featureName) {
		return nil, fmt.Errorf("feature %q is already installed", featureName)
	}

	features := ensureBase(append([]string{}, manifest.Features...))
	features = appendUnique(features, featureName)
	ctx := NewContext(manifest.ProjectName, manifest.ModulePath, features)
	if err := ctx.Require(featureName); err != nil {
		return nil, err
	}

	var written []string
	for _, name := range features {
		ftr, _ := reg.resolve(name)
		out, err := generateFeature(projectDir, ftr, ctx)
		if err != nil {
			return nil, err
		}
		written = append(written, out...)
	}
	sort.Strings(written)
	written = unique(written)

	if err := writeManifest(projectDir, manifest, features); err != nil {
		return nil, err
	}
	return &Result{ProjectDir: projectDir, Files: written, Features: features}, nil
}

func appendUnique(list []string, items ...string) []string {
	for _, it := range items {
		if !contains(list, it) {
			list = append(list, it)
		}
	}
	return list
}

func unique(list []string) []string {
	seen := make(map[string]bool, len(list))
	out := list[:0]
	for _, v := range list {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func ensureBase(features []string) []string {
	if contains(features, "base") {
		return features
	}
	out := make([]string, 0, len(features)+1)
	out = append(out, "base")
	out = append(out, features...)
	return out
}

func validateFeatures(reg *Registry, features []string) error {
	seen := make(map[string]bool, len(features))
	for _, name := range features {
		if seen[name] {
			return fmt.Errorf("feature %q listed more than once", name)
		}
		seen[name] = true
		if _, err := reg.resolve(name); err != nil {
			return err
		}
	}
	return nil
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

// manifest records which features were applied to a generated project.
type manifest struct {
	ProjectName string   `json:"project_name" yaml:"project_name"`
	ModulePath  string   `json:"module_path"  yaml:"module_path"`
	Features    []string `json:"features"     yaml:"features"`
}

func readManifest(projectDir string) (*manifest, error) {
	path := filepath.Join(projectDir, ".bootstrap.yaml")
	if !utils.PathExists(path) {
		return nil, fmt.Errorf("no .bootstrap.yaml manifest found in %s; is this a bootstrap-generated project?", projectDir)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m, err := parseManifest(data)
	if err != nil {
		return nil, err
	}
	if m.ProjectName == "" {
		// fall back to directory name for projects created before manifests
		m.ProjectName = filepath.Base(projectDir)
	}
	return m, nil
}

func writeManifest(projectDir string, current *manifest, updated []string) error {
	if current == nil {
		current = &manifest{}
	}
	current.Features = updated
	body, err := encodeManifest(current)
	if err != nil {
		return err
	}
	return utils.WriteFile(filepath.Join(projectDir, ".bootstrap.yaml"), body)
}
