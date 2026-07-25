package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/anurag925/crank/internal/utils"
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
	// MakefileOverride when true gives the project's Makefile precedence over
	// native crank commands. A Makefile target that shadows a crank command
	// name will be run instead of the native command.
	MakefileOverride bool
}

// Result reports the files written during a generation.
type Result struct {
	ProjectDir   string
	Files        []string
	Features     []string
	Dependencies []string // Go module paths that need `go get`
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
	if err := validateRequirements(reg, features); err != nil {
		return nil, err
	}

	ctx := NewContext(opts.ProjectName, opts.ModulePath, features, opts.MakefileOverride)

	var all []string
	var allDeps []string
	for _, name := range features {
		f, _ := reg.resolve(name)
		written, err := generateFeature(projectDir, f, ctx)
		if err != nil {
			return nil, err
		}
		all = append(all, written...)
		allDeps = append(allDeps, f.Dependencies()...)
		// Inject the feature's config snippets into config.go, config.yaml
		// and .env.example so multi-feature `crank init` calls produce a
		// fully wired project. injectConfig is idempotent and skips
		// features with no config data (e.g. "base").
		cfgWritten, err := injectConfig(projectDir, name, ctx.PackageName)
		if err != nil {
			return nil, err
		}
		all = append(all, cfgWritten...)
	}
	sort.Strings(all)
	all = unique(all)

	return &Result{
		ProjectDir:   projectDir,
		Files:        all,
		Features:     features,
		Dependencies: uniqueDeps(allDeps),
	}, nil
}

// Add applies a feature to an existing project directory. Only the new
// feature's templates are written. Files that already exist are skipped
// unless SkipIfExists is false (e.g. the manifest). The manifest is always
// re-written to reflect the updated feature set and crank version.
//
// Additionally, config files (config.go, config.yaml, .env.example) gain the
// new feature's sections via marker-based injection so existing user content
// is preserved.
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
	ctx := NewContext(manifest.ProjectName, manifest.ModulePath, features, manifest.MakefileOverride)

	// Only render the new feature's templates, not all features.
	ftr, err := reg.resolve(featureName)
	if err != nil {
		return nil, err
	}

	// Enforce the feature's requirements (e.g. outbox requires gorm).
	// The check runs against the *projected* feature set so we can detect
	// missing requirements before any templates are written.
	if err := checkRequirements(featureName, ftr.Requirements(), ctx); err != nil {
		return nil, err
	}

	written, err := generateFeature(projectDir, ftr, ctx)
	if err != nil {
		return nil, err
	}

	// Inject the feature's config sections into config.go, config.yaml,
	// and .env.example using marker-based injection. This preserves any
	// manual edits the user has already made.
	configWritten, err := injectConfig(projectDir, featureName, ctx.PackageName)
	if err != nil {
		return nil, err
	}
	written = append(written, configWritten...)

	// Always re-write the manifest with updated features + crank version.
	if err := writeManifest(projectDir, manifest, features); err != nil {
		return nil, err
	}
	written = append(written, ".crank.yaml")

	return &Result{
		ProjectDir:   projectDir,
		Files:        unique(written),
		Features:     features,
		Dependencies: uniqueDeps(ftr.Dependencies()),
	}, nil
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

// validateRequirements walks the requested feature set and refuses to
// proceed if any feature's Requirements() are not satisfied by the set
// as a whole. For example, `crank init --features=base,outbox` errors
// with a clear hint that outbox requires the gorm ORM.
func validateRequirements(reg *Registry, features []string) error {
	have := make(map[string]bool, len(features))
	for _, f := range features {
		have[f] = true
	}
	for _, name := range features {
		f, _ := reg.resolve(name)
		for _, r := range f.Requirements() {
			if !have[r] {
				return fmt.Errorf("feature %q requires %q (include both in --features or run `crank add %s` after init)", name, r, r)
			}
		}
	}
	return nil
}

func contains(list []string, target string) bool {
	return slices.Contains(list, target)
}

// uniqueDeps deduplicates dependency module paths.
func uniqueDeps(deps []string) []string {
	seen := make(map[string]bool, len(deps))
	var out []string
	for _, d := range deps {
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}

// checkRequirements checks that a feature's requirements are met by the
// context (used by the Add path).
func checkRequirements(featureName string, reqs []string, ctx *Context) error {
	for _, r := range reqs {
		if !ctx.Has(r) {
			return fmt.Errorf("feature %q requires %q (run `crank add %s` first or `crank init` with both features)", featureName, r, r)
		}
	}
	return nil
}

// manifest records which features were applied to a generated project.
type manifest struct {
	ProjectName      string   `json:"project_name" yaml:"project_name"`
	ModulePath       string   `json:"module_path"  yaml:"module_path"`
	Features         []string `json:"features"     yaml:"features"`
	CrankVersion     string   `json:"crank_version" yaml:"crank_version"`
	MakefileOverride bool     `json:"makefile_override,omitempty" yaml:"makefile_override,omitempty"`
}

func readManifest(projectDir string) (*manifest, error) {
	path := filepath.Join(projectDir, ".crank.yaml")
	if !utils.PathExists(path) {
		return nil, fmt.Errorf("no .crank.yaml manifest found in %s\n\nThis directory does not appear to be a crank-generated project. You can:\n  1. Run 'crank init' to create a new project, or\n  2. Use --project <dir> to point to an existing crank project.", projectDir)
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

// Update bumps the crank_version stamp in the project's .crank.yaml manifest
// to the current crank CLI version. This is the first step toward full
// template reconciliation; future releases will re-render feature templates
// and inject updated config sections.
func Update(projectDir string) (*Result, error) {
	if !utils.PathExists(projectDir) {
		return nil, fmt.Errorf("project directory %s does not exist", projectDir)
	}
	m, err := readManifest(projectDir)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	if m.CrankVersion == Version {
		return &Result{
			ProjectDir: projectDir,
			Files:      nil,
			Features:   m.Features,
		}, nil
	}

	oldVersion := m.CrankVersion
	if err := writeManifest(projectDir, m, m.Features); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	fmt.Printf("  crank_version: %s → %s\n", oldVersion, Version)

	return &Result{
		ProjectDir: projectDir,
		Files:      []string{".crank.yaml"},
		Features:   m.Features,
	}, nil
}

func writeManifest(projectDir string, current *manifest, updated []string) error {
	if current == nil {
		current = &manifest{}
	}
	current.Features = updated
	current.CrankVersion = Version
	body, err := encodeManifest(current)
	if err != nil {
		return err
	}
	return utils.WriteFile(filepath.Join(projectDir, ".crank.yaml"), body)
}

// SetMakefileOverride updates the makefile_override flag in the project's
// .crank.yaml manifest and writes it back. Returns true if the value changed.
func SetMakefileOverride(projectDir string, enabled bool) (bool, error) {
	m, err := readManifest(projectDir)
	if err != nil {
		return false, fmt.Errorf("read manifest: %w", err)
	}
	if m.MakefileOverride == enabled {
		return false, nil
	}
	m.MakefileOverride = enabled
	if err := writeManifest(projectDir, m, m.Features); err != nil {
		return false, err
	}
	return true, nil
}
