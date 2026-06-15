package bootstrap

import (
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/anurag925/crank/internal/utils"
)

// FileMapping describes a single file produced by a feature: the template inside the
// feature's embedded FS and the destination path inside the generated project.
type FileMapping struct {
	// TemplatePath is the path of the .tmpl file inside the feature's embedded FS.
	TemplatePath string
	// OutputPath is the destination path relative to the project root.
	OutputPath string
	// SkipIfExists causes the generator to leave an existing file untouched.
	SkipIfExists bool
}

// Feature is implemented by every installable module in crank.
type Feature interface {
	// Name returns the short identifier used in --features lists.
	Name() string
	// Description is shown by `crank list`.
	Description() string
	// Files enumerates the templates the feature contributes. Each template is rendered
	// against the project context and written to the corresponding output path.
	Files() []FileMapping
	// Templates returns the embedded FS containing the feature's .tmpl files.
	Templates() embed.FS
	// Dependencies returns the Go module paths this feature requires.
	// These are fetched via `go get` in the generated project after scaffolding.
	Dependencies() []string
	// Requirements returns the names of other features that must be
	// installed alongside this one. `crank add` and `crank init` will
	// refuse to install the feature if any requirement is missing.
	// Returning nil or an empty slice is valid.
	Requirements() []string
}

// Registry holds the set of features known to crank.
type Registry struct {
	features map[string]Feature
	order    []string
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{features: make(map[string]Feature)}
}

// Register adds a feature to the registry. Names must be unique.
func (r *Registry) Register(f Feature) error {
	if _, exists := r.features[f.Name()]; exists {
		return fmt.Errorf("feature %q already registered", f.Name())
	}
	r.features[f.Name()] = f
	r.order = append(r.order, f.Name())
	return nil
}

// MustRegister panics if Register returns an error. Useful for package-level init.
func (r *Registry) MustRegister(f Feature) {
	if err := r.Register(f); err != nil {
		panic(err)
	}
}

// Get returns the feature with the given name, or false if unknown.
func (r *Registry) Get(name string) (Feature, bool) {
	f, ok := r.features[name]
	return f, ok
}

// Names returns the feature names in the order they were registered.
func (r *Registry) Names() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// All returns every registered feature, in registration order.
func (r *Registry) All() []Feature {
	out := make([]Feature, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.features[n])
	}
	return out
}

// resolve maps a feature name to the Feature, returning an error if unknown.
func (r *Registry) resolve(name string) (Feature, error) {
	f, ok := r.features[name]
	if !ok {
		return nil, fmt.Errorf("unknown feature %q (run `crank list` to see available features)", name)
	}
	return f, nil
}

// renderTemplate parses and executes a template with the supplied context.
func renderTemplate(name, body string, ctx *Context) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(body)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, ctx); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return sb.String(), nil
}

// generateFeature processes every template owned by a feature, writing the results into
// the destination project directory.
func generateFeature(projectDir string, f Feature, ctx *Context) ([]string, error) {
	fsys := f.Templates()
	var written []string
	for _, m := range f.Files() {
		dest := filepath.Join(projectDir, m.OutputPath)
		if m.SkipIfExists && utils.PathExists(dest) {
			continue
		}
		body, err := fs.ReadFile(fsys, m.TemplatePath)
		if err != nil {
			return written, fmt.Errorf("feature %s: read %s: %w", f.Name(), m.TemplatePath, err)
		}
		rendered, err := renderTemplate(m.OutputPath, string(body), ctx)
		if err != nil {
			return written, fmt.Errorf("feature %s: render %s: %w", f.Name(), m.OutputPath, err)
		}
		// Run gofmt on .go files so generated code is always aligned and
		// import-ordered correctly, regardless of whitespace emitted by
		// templates. Without this, conditional field insertions in struct
		// blocks drift out of column alignment.
		if strings.HasSuffix(m.OutputPath, ".go") {
			if formatted, ferr := format.Source([]byte(rendered)); ferr == nil {
				rendered = string(formatted)
			} else {
				return written, fmt.Errorf("feature %s: format %s: %w", f.Name(), m.OutputPath, ferr)
			}
		}
		if err := utils.WriteFile(dest, rendered); err != nil {
			return written, err
		}
		written = append(written, m.OutputPath)
	}
	sort.Strings(written)
	return written, nil
}
