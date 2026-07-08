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
	markerHTTPFields   = "// crank:http-fields"
	markerHTTPRegister = "// crank:http-register"
	markerTxRepoImports  = "// crank:tx-repo-imports"
	markerTxRepositories = "// crank:tx-repositories"
	markerTxRepoMethods  = "// crank:tx-repo-methods"
	markerTxRepoFields   = "// crank:tx-repo-fields"
	markerInMemRepoFields = "// crank:inmem-repo-fields"
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

	fieldLine := fmt.Sprintf("\t%sHandler *%sHandler\n", r.Pascal, r.Pascal)
	registerLine := fmt.Sprintf("\tg%d := e.Group(\"/%s\")\n\tcfg.%sHandler.Register(g%d)\n",
		wireGroupIndex(content, r), r.KebabPlural, r.Pascal, wireGroupIndex(content, r))

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
// first handler wired into a file is `g` (no suffix), the second is `g2`, the
// third `g3`, and so on. The implementation just counts how many gN = e.Group
// lines already exist.
func wireGroupIndex(content string, r Resource) int {
	if !strings.Contains(content, " := e.Group(") {
		return 0
	}
	// Find the highest "gN" prefix currently in use.
	max := 0
	for _, line := range strings.Split(content, "\n") {
		if !strings.Contains(line, " := e.Group(") {
			continue
		}
		idx := strings.Index(line, ":=")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:idx])
		if name == "g" {
			if max < 0 {
				max = 0
			}
			continue
		}
		var n int
		if _, err := fmt.Sscanf(name, "g%d", &n); err == nil {
			if n > max {
				max = n
			}
		}
	}
	// The base feature's first group is named `g` (no suffix); the second
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
        g := e.Group("/%s")
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

// bunUoWFile is the path to the Bun-backed outbox UoW adapter.
const bunUoWFile = "internal/adapters/outbox/bun_uow.go"

// wireTxRepositories splices a new repository accessor into every TxRepositories
// implementation in the project. It updates:
//
//   1. uow/uow.go — TxRepositories interface + imports
//   2. adapters/outbox/gorm_uow.go — txRepositories struct + accessor method
//   3. adapters/outbox/bun_uow.go — txRepositories struct + accessor method (if bun)
//   4. adapters/uow/in_memory_uow.go — inMemoryTxRepositories struct + accessor
//
// It is idempotent; if the accessor already exists no file is modified.
func wireTxRepositories(projectDir string, r Resource) error {
	accessorName := r.PascalPlural
	interfaceMethod := fmt.Sprintf("\t%s() %s.Repository\n", accessorName, r.Snake)
	importLine := fmt.Sprintf("\t\"%s/internal/domain/%s\"\n", modulePathFromProject(projectDir), r.Snake)

	// 1. uow/uow.go — TxRepositories interface + imports
	spliceUoWInterface(projectDir, accessorName, interfaceMethod, importLine)

	// 2. in-memory UoW
	spliceInMemTxRepos(projectDir, r, accessorName)

	// 3. GORM outbox UoW
	spliceGormTxRepos(projectDir, r, accessorName)

	// 4. Bun outbox UoW
	spliceBunTxRepos(projectDir, r, accessorName)

	return nil
}

func modulePathFromProject(projectDir string) string {
	info, err := bootstrap.LoadProjectInfo(projectDir)
	if err != nil {
		return "example.com/app"
	}
	return info.ModulePath
}

func spliceUoWInterface(projectDir, accessorName, interfaceMethod, importLine string) {
	path := filepath.Join(projectDir, uowFile)
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	s := string(content)
	if strings.Contains(s, accessorName+"()") {
		return // already wired
	}

	// Add domain import
	if !strings.Contains(s, importLine) {
		s = strings.Replace(s, markerTxRepoImports, importLine+"\t"+markerTxRepoImports, 1)
	}
	// Add accessor to interface
	s = strings.Replace(s, markerTxRepositories, interfaceMethod+"\t"+markerTxRepositories, 1)

	formatted, err := format.Source([]byte(s))
	if err != nil {
		return
	}
	_ = os.WriteFile(path, formatted, 0o644)
}

func spliceInMemTxRepos(projectDir string, r Resource, accessorName string) {
	path := filepath.Join(projectDir, inMemUoWFile)
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	s := string(content)
	if strings.Contains(s, accessorName+"()") {
		return
	}

	fieldLine := fmt.Sprintf("\t%sRepo %s.Repository\n", r.Camel, r.Snake)
	methodLine := fmt.Sprintf("func (r *inMemoryTxRepositories) %s() %s.Repository { return r.%sRepo }\n", accessorName, r.Snake, r.Camel)
	constructorField := fmt.Sprintf("\t%sRepo %s.Repository\n", r.Camel, r.Snake)

	s = strings.Replace(s, markerTxRepoFields, fieldLine+"\t"+markerTxRepoFields, 1)
	s = strings.Replace(s, markerTxRepoMethods, "\n"+methodLine+"\t"+markerTxRepoMethods, 1)
	s = strings.Replace(s, markerInMemRepoFields, constructorField+"\t"+markerInMemRepoFields, 1)

	formatted, err := format.Source([]byte(s))
	if err != nil {
		return
	}
	_ = os.WriteFile(path, formatted, 0o644)
}

func spliceGormTxRepos(projectDir string, r Resource, accessorName string) {
	path := filepath.Join(projectDir, gormUoWFile)
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	s := string(content)
	if strings.Contains(s, accessorName+"()") {
		return
	}

	importLine := fmt.Sprintf("\t\"%s/internal/domain/%s\"\n", modulePathFromProject(projectDir), r.Snake)
	methodLine := fmt.Sprintf("func (r *txRepositories) %s() %s.Repository {\n\t\treturn gorm.New%sRepository(r.tx)\n\t}\n", accessorName, r.Snake, r.Pascal)

	if !strings.Contains(s, importLine) {
		s = strings.Replace(s, markerTxRepoImports, importLine+"\t"+markerTxRepoImports, 1)
	}
	s = strings.Replace(s, markerTxRepoMethods, "\n"+methodLine+"\t"+markerTxRepoMethods, 1)

	formatted, err := format.Source([]byte(s))
	if err != nil {
		return
	}
	_ = os.WriteFile(path, formatted, 0o644)
}

func spliceBunTxRepos(projectDir string, r Resource, accessorName string) {
	path := filepath.Join(projectDir, bunUoWFile)
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	s := string(content)
	if strings.Contains(s, accessorName+"()") {
		return
	}

	importLine := fmt.Sprintf("\t\"%s/internal/domain/%s\"\n", modulePathFromProject(projectDir), r.Snake)
	methodLine := fmt.Sprintf("func (r *txRepositories) %s() %s.Repository {\n\t\treturn bun.New%sRepository(r.tx)\n\t}\n", accessorName, r.Snake, r.Pascal)

	if !strings.Contains(s, importLine) {
		s = strings.Replace(s, markerTxRepoImports, importLine+"\t"+markerTxRepoImports, 1)
	}
	s = strings.Replace(s, markerTxRepoMethods, "\n"+methodLine+"\t"+markerTxRepoMethods, 1)

	formatted, err := format.Source([]byte(s))
	if err != nil {
		return
	}
	_ = os.WriteFile(path, formatted, 0o644)
}
