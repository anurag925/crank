package bootstrap

import (
	"fmt"
	"strings"
)

// Context holds the data passed to every template during project generation.
type Context struct {
	// ProjectName is the user-supplied name (e.g. "myapp").
	ProjectName string
	// ModulePath is the Go module path used in go.mod and import statements.
	ModulePath string
	// PackageName is the last segment of the module path, used as a directory name.
	PackageName string
	// Features is the list of feature names the user opted into.
	Features []string
	// CrankVersion is the crank CLI version that generated this project.
	CrankVersion string

	featureSet map[string]bool
}

// NewContext builds a Context from the project name, module path, and feature list.
// If modulePath is empty, the project name is used as the module path.
func NewContext(projectName, modulePath string, features []string) *Context {
	if modulePath == "" {
		modulePath = projectName
	}
	modulePath = strings.TrimSpace(modulePath)
	modulePath = strings.TrimSuffix(modulePath, "/")

	set := make(map[string]bool, len(features))
	for _, f := range features {
		set[f] = true
	}

	pkg := modulePath
	if idx := strings.LastIndex(pkg, "/"); idx >= 0 {
		pkg = pkg[idx+1:]
	}
	if pkg == "" {
		pkg = projectName
	}

	return &Context{
		ProjectName:  projectName,
		ModulePath:   modulePath,
		PackageName:  pkg,
		Features:     features,
		CrankVersion: Version,
		featureSet:   set,
	}
}

// Has reports whether the named feature is included in this generation.
func (c *Context) Has(name string) bool {
	return c.featureSet[name]
}

// Require returns an error if any of the supplied features is missing from the context.
func (c *Context) Require(names ...string) error {
	var missing []string
	for _, n := range names {
		if !c.featureSet[n] {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required features: %s", strings.Join(missing, ", "))
	}
	return nil
}
