package bootstrap

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/anurag925/crank/internal/utils"
)

// ProjectInfo is the public view of a crank-generated project's manifest. It is
// used by tooling (e.g. the `make` code generators) that needs to know the
// project's module path and enabled feature set.
type ProjectInfo struct {
	ProjectName  string
	ModulePath   string
	Features     []string
	CrankVersion string
}

// LoadProjectInfo reads the .crank.yaml manifest from projectDir and returns a
// public ProjectInfo. When the manifest omits the module path (older projects),
// it falls back to the `module` line of go.mod.
func LoadProjectInfo(projectDir string) (*ProjectInfo, error) {
	m, err := readManifest(projectDir)
	if err != nil {
		return nil, fmt.Errorf("load project info: %w", err)
	}
	mod := strings.TrimSpace(m.ModulePath)
	if mod == "" {
		mod = moduleFromGoMod(projectDir)
	}
	if mod == "" {
		mod = m.ProjectName
	}
	return &ProjectInfo{
		ProjectName:  m.ProjectName,
		ModulePath:   mod,
		Features:     m.Features,
		CrankVersion: m.CrankVersion,
	}, nil
}

// Has reports whether the named feature is enabled in the project.
func (p *ProjectInfo) Has(name string) bool {
	return slices.Contains(p.Features, name)
}

// moduleFromGoMod returns the module path declared in projectDir/go.mod, or ""
// if it cannot be determined.
func moduleFromGoMod(projectDir string) string {
	path := filepath.Join(projectDir, "go.mod")
	if !utils.PathExists(path) {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	if scanner.Err() != nil {
		slog.Error("failed to read go.mod", "err", scanner.Err())
		return ""
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}
