package scaffold

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/anurag925/crank/internal/bootstrap"
)

// routesFile is the path (relative to the project root) of the central HTTP
// adapter aggregator that wires per-resource handlers into the Echo router.
// The file lives under the DDD-shaped web/v1 package.
const routesFile = "internal/adapters/http/web/v1/routes.go"

// legacyHandlerFile is the pre-DDD wiring target. It is consulted only as a
// defensive fallback: it lets `crank make handler` work against projects
// generated before the routes.go file exists. New projects always have
// routes.go, so this fallback is rarely hit.
const legacyHandlerFile = "internal/handler/handler.go"

// Marker comments emitted by the base feature's routes.go template. New
// handlers are spliced in at these anchors.
const (
	markerHTTPFields     = "// crank:http-fields"
	markerHTTPRegister   = "// crank:http-register"
	markerTxRepoImports  = "// crank:tx-repo-imports"
	markerTxRepositories = "// crank:tx-repositories"
	markerTxRepoMethods  = "// crank:tx-repo-methods"
	markerTxRepoFields   = "// crank:tx-repo-fields"
	markerInMemOptions   = "// crank:inmem-options"
)

// wireResult reports the outcome of attempting to register a handler in the
// routes aggregator.
type wireResult struct {
	// Wired is true when the file was edited successfully.
	Wired bool
	// Hint, when non-empty, contains manual instructions the user should apply
	// because automatic wiring was not possible.
	Hint string
}

// wireHandler registers a generated handler with the project's central
// routes aggregator (`internal/adapters/http/web/routes.go`) so its routes
// are served without any manual edits. It is best-effort and never corrupts
// the file: if the resulting source does not compile/format, the edit is
// discarded and a manual hint is returned instead.
func wireHandler(projectDir string, r Resource) (wireResult, error) {
	// Prefer the DDD target. Fall back to the legacy handler.go only when the
	// caller explicitly wants to wire a handler into an old (pre-DDD) project
	// that has not yet migrated.
	path := filepath.Join(projectDir, routesFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(projectDir, legacyHandlerFile)
	}
	rel := projectRel(path, projectDir)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return wireResult{Hint: manualWireHint(r, rel)}, nil
	}
	if err != nil {
		return wireResult{}, fmt.Errorf("read %s: %w", rel, err)
	}

	content := string(data)

	// Already wired? Avoid creating duplicate registrations. The check is keyed
	// on the field name (and the route group), so the same handler can be
	// re-spliced into different routes.go files safely.
	if isAlreadyWired(content, r) {
		return wireResult{Wired: true}, nil
	}

	idx := wireGroupIndex(content, r)
	fieldLine := fmt.Sprintf("\t%sHandler *%sHandler\n", r.Pascal, r.Pascal)
	// Mount generated handlers under the shared /api/v1 group (variable `g`
	// in the base Mount()), so their routes match the UserHandler convention
	// (/api/v1/<resource>) rather than being exposed at the router root.
	registerLine := fmt.Sprintf("\tg%d := g.Group(\"/%s\")\n\tcfg.%sHandler.Register(g%d)\n",
		idx, r.KebabPlural, r.Pascal, idx)

	updated, ok := spliceAtMarkers(content, fieldLine, registerLine)
	if !ok {
		return wireResult{Hint: manualWireHint(r, rel)}, nil
	}

	formatted, err := format.Source([]byte(updated))
	if err != nil {
		// The edit produced invalid Go; do not write a broken file.
		return wireResult{Hint: manualWireHint(r, rel)}, nil
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return wireResult{}, fmt.Errorf("write %s: %w", rel, err)
	}
	return wireResult{Wired: true}, nil
}

// isAlreadyWired reports whether the routes file already contains a wiring
// line for the resource (used to keep splicing idempotent).
func isAlreadyWired(content string, r Resource) bool {
	// New style: `cfg.OrderHandler.Register(g2)` or a struct field.
	patterns := []string{
		fmt.Sprintf("cfg.%sHandler.Register(", r.Pascal),
		fmt.Sprintf("%s *%sHandler", r.Pascal, r.Pascal),
	}
	for _, p := range patterns {
		if strings.Contains(content, p) {
			return true
		}
	}
	return false
}

// wireGroupIndex returns a small integer that can be appended to the local
// group variable name to avoid clashing with other generated handlers. The
// first handler wired into a file is `g2`, the next `g3`, and so on (the base
// feature already uses the bare `g` for the /api/v1 root group). The
// implementation finds the highest existing `gN := <x>.Group(...)` suffix.
func wireGroupIndex(content string, r Resource) int {
	max := 0
	for _, line := range strings.Split(content, "\n") {
		if !strings.Contains(line, ".Group(") || !strings.Contains(line, ":=") {
			continue
		}
		idx := strings.Index(line, ":=")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:idx])
		var n int
		if _, err := fmt.Sscanf(name, "g%d", &n); err == nil {
			if n > max {
				max = n
			}
		}
	}
	// The base feature's first group is named `g` (no suffix); the first
	// generated resource is therefore `g2`.
	if max < 2 {
		return 2
	}
	return max + 1
}

// spliceAtMarkers inserts the snippets immediately before the marker
// comments. It returns false if any marker is missing.
func spliceAtMarkers(content, fieldLine, registerLine string) (string, bool) {
	if !strings.Contains(content, markerHTTPFields) ||
		!strings.Contains(content, markerHTTPRegister) {
		return "", false
	}
	content = strings.Replace(content, markerHTTPFields, "\n"+strings.TrimRight(fieldLine, "\n")+"\t"+markerHTTPFields, 1)
	content = strings.Replace(content, markerHTTPRegister, "\n"+strings.TrimRight(registerLine, "\n")+"\t"+markerHTTPRegister, 1)
	return content, true
}

// projectRel returns a project-relative path. It tolerates both relative and
// absolute projectDir values without panicking.
func projectRel(path, projectDir string) string {
	if rel, err := filepath.Rel(projectDir, path); err == nil {
		return rel
	}
	return path
}

// manualWireHint returns copy-pasteable instructions for registering a
// handler by hand when automatic wiring is not possible.
func manualWireHint(r Resource, target string) string {
	return fmt.Sprintf(`could not auto-register the handler in %s. Add these manually:

  • in the MountConfig struct:   %s *%sHandler
  • in Mount(), before the marker:
        g := g.Group("/%s")
        cfg.%sHandler.Register(g)`,
		target,
		r.Pascal, r.Pascal,
		r.KebabPlural, r.Pascal)
}

// --- TxRepositories splicing -------------------------------------------------

// uowFile is the path to the UnitOfWork + TxRepositories interface.
const uowFile = "internal/application/uow/uow.go"

// inMemUoWFile is the path to the in-memory UoW adapter.
const inMemUoWFile = "internal/adapters/uow/in_memory_uow.go"

// gormUoWFile is the path to the GORM-backed outbox UoW adapter.
const gormUoWFile = "internal/adapters/outbox/gorm_uow.go"

// wireTxRepositories splices a new repository accessor into every TxRepositories
// implementation in the project. It updates:
//
//  1. uow/uow.go — TxRepositories interface + imports
//  2. adapters/outbox/gorm_uow.go — txRepositories struct + accessor method
//  3. adapters/uow/in_memory_uow.go — inMemoryTxRepositories struct + accessor
//
// It is idempotent; if the accessor already exists no file is modified. A file
// that does not exist for the project's feature set (e.g. the GORM outbox UoW
// in a project without outbox) is skipped silently. Any other failure — a read
// error, a splice that produces invalid Go, or a write error — is returned so
// the caller does not report a broken wiring as success.
func wireTxRepositories(projectDir string, r Resource) error {
	accessorName := r.PascalPlural
	interfaceMethod := fmt.Sprintf("\t%s() %s.Repository\n", accessorName, r.Snake)
	importLine := fmt.Sprintf("\t\"%s/internal/domain/%s\"\n", modulePathFromProject(projectDir), r.Snake)

	// 1. uow/uow.go — TxRepositories interface + imports
	if err := spliceUoWInterface(projectDir, accessorName, interfaceMethod, importLine); err != nil {
		return err
	}

	// 2. in-memory UoW
	if err := spliceInMemTxRepos(projectDir, r, accessorName); err != nil {
		return err
	}

	// 3. GORM outbox UoW
	if err := spliceGormTxRepos(projectDir, r, accessorName); err != nil {
		return err
	}

	return nil
}

func modulePathFromProject(projectDir string) string {
	info, err := bootstrap.LoadProjectInfo(projectDir)
	if err != nil {
		return "example.com/app"
	}
	return info.ModulePath
}

func spliceUoWInterface(projectDir, accessorName, interfaceMethod, importLine string) error {
	path := filepath.Join(projectDir, uowFile)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no UoW interface in this project shape; nothing to wire
		}
		return fmt.Errorf("read %s: %w", uowFile, err)
	}
	s := string(content)
	if strings.Contains(s, accessorName+"()") {
		return nil // already wired
	}

	// Add domain import
	if !strings.Contains(s, importLine) {
		s = strings.Replace(s, markerTxRepoImports, importLine+"\t"+markerTxRepoImports, 1)
	}
	// Add accessor to interface
	s = strings.Replace(s, markerTxRepositories, interfaceMethod+"\t"+markerTxRepositories, 1)

	formatted, err := format.Source([]byte(s))
	if err != nil {
		return fmt.Errorf("wiring %s produced invalid Go (leaving it untouched): %w", uowFile, err)
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", uowFile, err)
	}
	return nil
}

func spliceInMemTxRepos(projectDir string, r Resource, accessorName string) error {
	path := filepath.Join(projectDir, inMemUoWFile)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", inMemUoWFile, err)
	}
	s := string(content)
	if strings.Contains(s, accessorName+"()") {
		return nil
	}

	fieldLine := fmt.Sprintf("\t%sRepo %s.Repository\n", r.Camel, r.Snake)
	methodLine := fmt.Sprintf("func (r *inMemoryTxRepositories) %s() %s.Repository { return r.%sRepo }\n", accessorName, r.Snake, r.Camel)
	optionFunc := fmt.Sprintf("func With%sRepo(r %s.Repository) Option {\n\treturn func(repos *inMemoryTxRepositories) { repos.%sRepo = r }\n}\n", r.Pascal, r.Snake, r.Camel)
	importLine := fmt.Sprintf("\t\"%s/internal/domain/%s\"\n", modulePathFromProject(projectDir), r.Snake)

	if !strings.Contains(s, importLine) {
		s = strings.Replace(s, markerTxRepoImports, importLine+"\t"+markerTxRepoImports, 1)
	}
	s = strings.Replace(s, markerTxRepoFields, fieldLine+"\t"+markerTxRepoFields, 1)
	s = strings.Replace(s, markerTxRepoMethods, "\n"+methodLine+"\t"+markerTxRepoMethods, 1)
	s = strings.Replace(s, markerInMemOptions, "\n"+optionFunc+"\n"+markerInMemOptions, 1)

	formatted, err := format.Source([]byte(s))
	if err != nil {
		return fmt.Errorf("wiring %s produced invalid Go (leaving it untouched): %w", inMemUoWFile, err)
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", inMemUoWFile, err)
	}
	return nil
}

func spliceGormTxRepos(projectDir string, r Resource, accessorName string) error {
	path := filepath.Join(projectDir, gormUoWFile)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", gormUoWFile, err)
	}
	s := string(content)
	if strings.Contains(s, accessorName+"()") {
		return nil
	}

	importLine := fmt.Sprintf("\t\"%s/internal/domain/%s\"\n", modulePathFromProject(projectDir), r.Snake)
	methodLine := fmt.Sprintf("func (r *txRepositories) %s() %s.Repository {\n\t\treturn gormadapter.New%sRepository(r.tx)\n\t}\n", accessorName, r.Snake, r.Pascal)

	if !strings.Contains(s, importLine) {
		s = strings.Replace(s, markerTxRepoImports, importLine+"\t"+markerTxRepoImports, 1)
	}
	s = strings.Replace(s, markerTxRepoMethods, "\n"+methodLine+"\t"+markerTxRepoMethods, 1)

	formatted, err := format.Source([]byte(s))
	if err != nil {
		return fmt.Errorf("wiring %s produced invalid Go (leaving it untouched): %w", gormUoWFile, err)
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", gormUoWFile, err)
	}
	return nil
}
