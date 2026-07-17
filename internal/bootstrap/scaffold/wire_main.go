package scaffold

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/anurag925/crank/internal/bootstrap"
)

// mainFile is the path (relative to the project root) of the composition root
// where generated handlers must be assembled and passed to v1.Mount.
const mainFile = "cmd/server/main.go"

// Marker comments emitted by the base feature's cmd/server/main.go template.
// Generated per-resource wiring is spliced in at these anchors.
const (
	markerCompositionImports = "// crank:composition-imports"
	markerRepos              = "// crank:repos"
	markerUoWRepos           = "// crank:uow-repos"
	markerCompositionRoot    = "// crank:composition-root"
	markerMountConfig        = "// crank:mount-config"
)

// wireCompositionRoot assembles a generated resource into cmd/server/main.go so
// its HTTP handler is actually constructed and passed to v1.Mount — without
// this the handler struct field in MountConfig stays nil and every generated
// endpoint nil-panics at runtime.
//
// It splices, in order:
//
//  1. the application package import (`<resource>app "…/internal/application/<resource>"`)
//  2. the repository constructor (`<resource>Repo := gorm.New…` or `memory.New…`),
//     placed BEFORE the UnitOfWork construction so it can be handed to the
//     in-memory UoW as a functional option
//  3. (non-outbox projects only) a `uow.With<Resource>Repo(<resource>Repo)`
//     option so the in-memory UoW shares the same repository instance
//     (read-your-writes)
//  4. the command/query/handler construction block
//  5. the `<Resource>Handler: <resource>Handler` field in the MountConfig literal
//
// The splice is best-effort and idempotent: if the handler variable already
// exists it is a no-op. If the edit would produce invalid Go it is discarded
// and an error is returned (the on-disk file is left untouched) so a broken
// wiring is never silently reported as success.
func wireCompositionRoot(projectDir string, r Resource) error {
	path := filepath.Join(projectDir, mainFile)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no composition root in this project shape; nothing to wire
		}
		return fmt.Errorf("read %s: %w", mainFile, err)
	}
	s := string(content)

	handlerVar := r.Camel + "Handler"
	// Idempotent: the handler variable is the last thing we splice, so its
	// presence means this resource is already fully wired.
	if strings.Contains(s, handlerVar+" :=") {
		return nil
	}

	// Without the composition markers this is an older project the base
	// template predates; leave it untouched rather than corrupting it.
	if !strings.Contains(s, markerCompositionRoot) || !strings.Contains(s, markerMountConfig) {
		return nil
	}

	info, err := bootstrap.LoadProjectInfo(projectDir)
	if err != nil {
		return fmt.Errorf("load project %s: %w", projectDir, err)
	}
	module := info.ModulePath
	hasGorm := info.Has("gorm")

	appAlias := r.Camel + "app"
	repoVar := r.Camel + "Repo"

	// 1. application package import.
	importLine := fmt.Sprintf("\t%s \"%s/internal/application/%s\"\n", appAlias, module, r.Snake)
	if strings.Contains(s, markerCompositionImports) && !strings.Contains(s, importLine) {
		s = strings.Replace(s, markerCompositionImports, importLine+"\t"+markerCompositionImports, 1)
	}

	// 2. repository constructor, spliced before the UoW construction.
	var repoLine string
	if hasGorm {
		repoLine = fmt.Sprintf("%s := gorm.New%sRepository(gormDB)\n\t", repoVar, r.Pascal)
	} else {
		repoLine = fmt.Sprintf("%s := memory.New%sRepository()\n\t", repoVar, r.Pascal)
	}
	s = strings.Replace(s, markerRepos, repoLine+markerRepos, 1)

	// 3. in-memory UoW option (only present in non-outbox projects).
	if strings.Contains(s, markerUoWRepos) {
		optLine := fmt.Sprintf("uow.With%sRepo(%s),\n\t\t", r.Pascal, repoVar)
		s = strings.Replace(s, markerUoWRepos, optLine+markerUoWRepos, 1)
	}

	// 4. command/query/handler construction.
	rootBlock := fmt.Sprintf(
		"%sCmd := %s.NewCommandHandler(%s, uow)\n\t%sQry := %s.NewQueryHandler(%s)\n\t%s := v1.New%sHandler(%sCmd, %sQry)\n\t",
		r.Camel, appAlias, repoVar,
		r.Camel, appAlias, repoVar,
		handlerVar, r.Pascal, r.Camel, r.Camel,
	)
	s = strings.Replace(s, markerCompositionRoot, rootBlock+markerCompositionRoot, 1)

	// 5. MountConfig field.
	mountLine := fmt.Sprintf("%s: %s,\n\t\t", r.Pascal+"Handler", handlerVar)
	s = strings.Replace(s, markerMountConfig, mountLine+markerMountConfig, 1)

	formatted, err := format.Source([]byte(s))
	if err != nil {
		return fmt.Errorf("wiring %s produced invalid Go (leaving it untouched): %w", mainFile, err)
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", mainFile, err)
	}
	return nil
}
