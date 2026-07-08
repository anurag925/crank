package bootstrap_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anurag925/crank/internal/bootstrap"

	// Register all features via init().
	_ "github.com/anurag925/crank/internal/bootstrap/features/auth"
	_ "github.com/anurag925/crank/internal/bootstrap/features/base"
	_ "github.com/anurag925/crank/internal/bootstrap/features/bun"
	_ "github.com/anurag925/crank/internal/bootstrap/features/crypto"
	_ "github.com/anurag925/crank/internal/bootstrap/features/gorm"
	_ "github.com/anurag925/crank/internal/bootstrap/features/mongodb"
	_ "github.com/anurag925/crank/internal/bootstrap/features/outbox"
	_ "github.com/anurag925/crank/internal/bootstrap/features/qdrant"
	_ "github.com/anurag925/crank/internal/bootstrap/features/redis"
	_ "github.com/anurag925/crank/internal/bootstrap/features/temporal"

	// Phase 4-5: new features
	_ "github.com/anurag925/crank/internal/bootstrap/features/audit"
	"github.com/anurag925/crank/internal/bootstrap/scaffold"
)

func allFeatures(t *testing.T) []string {
	t.Helper()
	return bootstrap.GlobalRegistry.Names()
}

// generateProject is a helper that scaffolds a project in a temp dir.
func generateProject(t *testing.T, name string, features []string) *bootstrap.Result {
	t.Helper()
	tmp := t.TempDir()
	result, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: name,
		ModulePath:  "github.com/example/" + name,
		TargetDir:   tmp,
		Features:    features,
	})
	if err != nil {
		t.Fatalf("Generate(%s, %v): %v", name, features, err)
	}
	return result
}

// readFile returns the content of a file or fails the test.
func readFile(t *testing.T, base, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(base, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// assertFileExists checks that the file exists in the project.
func assertFileExists(t *testing.T, base, rel string) {
	t.Helper()
	path := filepath.Join(base, rel)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", rel)
	}
}

// assertFileNotExists checks that the file does NOT exist in the project.
func assertFileNotExists(t *testing.T, base, rel string) {
	t.Helper()
	path := filepath.Join(base, rel)
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected file %s to NOT exist", rel)
	}
}

// assertContains checks that content contains substr.
func assertContains(t *testing.T, content, substr, label string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("%s: expected to contain %q", label, substr)
	}
}

// assertNotContains checks that content does NOT contain substr.
func assertNotContains(t *testing.T, content, substr, label string) {
	t.Helper()
	if strings.Contains(content, substr) {
		t.Errorf("%s: expected NOT to contain %q", label, substr)
	}
}

// assertDepsContains checks that deps contains a module path.
func assertDepsContains(t *testing.T, deps []string, module, label string) {
	t.Helper()
	for _, d := range deps {
		if strings.Contains(d, module) {
			return
		}
	}
	t.Errorf("%s: expected deps to contain %q, got %v", label, module, deps)
}

// assertDepsNotContains checks that deps does NOT contain a module path.
func assertDepsNotContains(t *testing.T, deps []string, module, label string) {
	t.Helper()
	for _, d := range deps {
		if strings.Contains(d, module) {
			t.Errorf("%s: expected deps NOT to contain %q", label, module)
			return
		}
	}
}

// --- Global registry tests ---

func TestGlobalRegistry_HasAllFeatures(t *testing.T) {
	names := allFeatures(t)
	want := map[string]bool{
		"base":     true,
		"auth":     true,
		"crypto":   true,
		"bun":      true,
		"gorm":     true,
		"redis":    true,
		"mongodb":  true,
		"qdrant":   true,
		"temporal": true,

		"audit": true,
	}
	for _, n := range names {
		delete(want, n)
	}
	if len(want) > 0 {
		t.Errorf("missing features from global registry: %v", want)
	}
}

func TestGlobalRegistry_FeatureNames(t *testing.T) {
	features := bootstrap.GlobalRegistry.All()
	for _, f := range features {
		if f.Name() == "" {
			t.Error("feature has empty name")
		}
		if f.Description() == "" {
			t.Errorf("feature %q has empty description", f.Name())
		}
		if f.Files() == nil {
			t.Errorf("feature %q has nil Files()", f.Name())
		}
	}
}

// ==========================================================================
// BASE feature — files present in every generation
// ==========================================================================

func TestBase_FilesExist_BaseOnly(t *testing.T) {
	r := generateProject(t, "baseonly", nil)
	dir := r.ProjectDir

	expectedFiles := []string{
		"cmd/server/main.go",
		"docs/docs.go",
		"internal/config/config.go",
		"internal/adapters/http/web/v1/routes.go",
		"internal/adapters/http/web/v1/user_handler.go",
		"internal/adapters/http/web/middleware/logging.go",
		"internal/validator/validator.go",
		"internal/validator/errors.go",
		"internal/domain/user/user.go",
		"internal/adapters/persistence/memory/user_repository.go",
		"internal/application/user/command_handler.go",
		"configs/config.yaml",
		".env.example",
		"Makefile",
		".air.toml",
		"Dockerfile",
		".gitignore",
		"go.mod",
		"README.md",
		".crank.yaml",
		"AGENTS.md",
		".agents/skills/crank-project/SKILL.md",
	}
	for _, f := range expectedFiles {
		assertFileExists(t, dir, f)
	}
}

func TestBase_AgentGuidance_UsesSystemCrank(t *testing.T) {
	r := generateProject(t, "agentguide", nil)

	agents := readFile(t, r.ProjectDir, "AGENTS.md")
	assertContains(t, agents, "The project root contains `.crank.yaml`", "AGENTS.md")
	assertContains(t, agents, "Use the system-installed `crank` CLI", "AGENTS.md")
	assertContains(t, agents, "Do not use `./crank`", "AGENTS.md")

	skill := readFile(t, r.ProjectDir, ".agents/skills/crank-project/SKILL.md")
	assertContains(t, skill, "name: crank-project", "crank-project skill")
	assertContains(t, skill, "A project is a Crank-generated project if it has a `.crank.yaml` file", "crank-project skill")
	assertContains(t, skill, "Use the system-installed `crank` binary", "crank-project skill")
	assertContains(t, skill, "Do not use `./crank`", "crank-project skill")
}

func TestBase_GoMod_ModulePath(t *testing.T) {
	r := generateProject(t, "mymod", nil)
	content := readFile(t, r.ProjectDir, "go.mod")
	assertContains(t, content, "module github.com/example/mymod", "go.mod")
	assertContains(t, content, "go 1.26", "go.mod")

	// Dependencies are now returned via Result and installed via go get.
	assertDepsContains(t, r.Dependencies, "github.com/labstack/echo/v5", "base deps")
	assertDepsContains(t, r.Dependencies, "github.com/spf13/viper", "base deps")
	assertDepsContains(t, r.Dependencies, "github.com/go-playground/validator/v10", "base deps")
}

func TestBase_GoMod_NoAuthDeps(t *testing.T) {
	r := generateProject(t, "noauth", nil)
	content := readFile(t, r.ProjectDir, "go.mod")
	assertNotContains(t, content, "golang-jwt", "go.mod without auth")
	assertNotContains(t, content, "golang.org/x/crypto", "go.mod without auth")

	// Dependencies are now returned via Result.
	assertDepsNotContains(t, r.Dependencies, "golang-jwt/jwt/v5", "base has no jwt dep")
	assertDepsContains(t, r.Dependencies, "github.com/go-playground/validator/v10", "base always has validator")
}

func TestBase_GoMod_NoORMDeps(t *testing.T) {
	r := generateProject(t, "nopg", nil)
	content := readFile(t, r.ProjectDir, "go.mod")
	assertNotContains(t, content, "uptrace/bun", "go.mod without bun")
	assertNotContains(t, content, "gorm.io", "go.mod without gorm")
	assertNotContains(t, content, "golang-migrate", "go.mod without migrate")

	assertDepsNotContains(t, r.Dependencies, "uptrace/bun", "base has no bun dep")
	assertDepsNotContains(t, r.Dependencies, "gorm.io", "base has no gorm dep")
	assertDepsNotContains(t, r.Dependencies, "golang-migrate/migrate/v4", "base has no migrate dep")
}

func TestBase_Config_ProjectName(t *testing.T) {
	r := generateProject(t, "cfgtest", nil)
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, `"cfgtest"`, "config.yaml project name")
	assertNotContains(t, content, "database:", "config.yaml without ORM")
	assertNotContains(t, content, "jwt:", "config.yaml without auth")
}

func TestBase_Config_HasLoggingSection(t *testing.T) {
	r := generateProject(t, "logtest", nil)
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "logging:", "config.yaml logging section")
	assertContains(t, content, "level:", "config.yaml logging level")
}

func TestBase_MainGo_NoDatabaseImport(t *testing.T) {
	r := generateProject(t, "nodbmain", nil)
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertNotContains(t, content, `internal/database`, "main.go no db import")
	assertNotContains(t, content, "bun.NewDB", "main.go no bun init")
	assertNotContains(t, content, "gorm.NewDB", "main.go no gorm init")
}

func TestBase_MainGo_NoAuthImport(t *testing.T) {
	r := generateProject(t, "noauthmain", nil)
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertNotContains(t, content, "jwt.NewTokenService", "main.go no jwt init")
}

func TestBase_HandlerHandler_NoDeps(t *testing.T) {
	// The DDD aggregator (routes.go) only depends on the MountConfig struct
	// of per-resource handlers. It has no DB or JWT dependencies of its own.
	r := generateProject(t, "nodeps", nil)
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/v1/routes.go")
	assertNotContains(t, content, "*bun.DB", "routes.go no bun dep")
	assertNotContains(t, content, "JWTTokenService", "routes.go no jwt dep")
}

func TestBase_HandlerUser_UsesService(t *testing.T) {
	// The HTTP adapter depends on the application command/query handlers,
	// not on a monolithic service.
	r := generateProject(t, "usesvc", nil)
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/v1/user_handler.go")
	assertContains(t, content, "cmd *appuser.CommandHandler", "user handler uses command handler")
	assertContains(t, content, "qry *appuser.QueryHandler", "user handler uses query handler")
	assertContains(t, content, "h.qry.HandleGet(", "user handler delegates get to query handler")
}

func TestBase_UserModel_NoPassword(t *testing.T) {
	// The base domain aggregate has no JSON/DB/validate tags and the password
	// field is unexported. Persistence adapters own the column mapping.
	r := generateProject(t, "nopw", nil)
	content := readFile(t, r.ProjectDir, "internal/domain/user/user.go")
	assertNotContains(t, content, `json:"`, "domain aggregate has no json tags")
	assertNotContains(t, content, `bun:"`, "domain aggregate has no bun tags")
	assertNotContains(t, content, `validate:"`, "domain aggregate has no validate tags")
}

func TestBase_MiddlewareLogging(t *testing.T) {
	r := generateProject(t, "mwlog", nil)
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/middleware/logging.go")
	assertContains(t, content, "func RequestLogger()", "logging middleware")
	assertContains(t, content, "log/slog", "logging middleware imports slog")
	assertContains(t, content, "slog.LevelInfo", "logging middleware uses slog")
}

func TestBase_MainGo_HasValidatorImport(t *testing.T) {
	r := generateProject(t, "valmain", nil)
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertNotContains(t, content, `"github.com/example/valmain/internal/validator"`, "main.go no longer imports validator (moved to server.go)")
	assertNotContains(t, content, "model.APIError", "main.go no longer imports model")
	assertNotContains(t, content, "HTTPErrorHandler", "main.go no longer sets error handler (moved to server.go)")
}

func TestBase_Dockerfile(t *testing.T) {
	r := generateProject(t, "dockertest", nil)
	content := readFile(t, r.ProjectDir, "Dockerfile")
	assertContains(t, content, "golang:1.25-alpine", "Dockerfile golang image")
	assertContains(t, content, "cmd/server", "Dockerfile build target")
	assertContains(t, content, "EXPOSE 8080", "Dockerfile expose")
}

func TestBase_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "manifesttest", []string{"base"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "project_name: manifesttest", "manifest project name")
	assertContains(t, content, "module_path: github.com/example/manifesttest", "manifest module path")
	assertContains(t, content, "- base", "manifest features")
}

func TestBase_Gitignore(t *testing.T) {
	r := generateProject(t, "gitignoretest", nil)
	content := readFile(t, r.ProjectDir, ".gitignore")
	assertContains(t, content, "/bin/", "gitignore bin")
	assertContains(t, content, ".env", "gitignore .env")
	assertNotContains(t, content, "data/", "gitignore without ORM has no data/")
}

func TestBase_Makefile_NoMigrate(t *testing.T) {
	r := generateProject(t, "makenorm", nil)
	content := readFile(t, r.ProjectDir, "Makefile")
	assertNotContains(t, content, "migrate-up", "Makefile without ORM")
	assertNotContains(t, content, "migrate-down", "Makefile without ORM")
}

func TestBase_Readme_NoORM(t *testing.T) {
	r := generateProject(t, "readmenorm", nil)
	content := readFile(t, r.ProjectDir, "README.md")
	assertContains(t, content, "# readmenorm", "README title")
	assertNotContains(t, content, "Bun", "README without ORM")
	assertNotContains(t, content, "GORM", "README without ORM")
}

func TestBase_RepositoryUser_InMemory(t *testing.T) {
	r := generateProject(t, "repomem", nil)
	content := readFile(t, r.ProjectDir, "internal/adapters/persistence/memory/user_repository.go")
	assertContains(t, content, "sync.RWMutex", "in-memory repo uses mutex")
	assertContains(t, content, "byID", "in-memory repo has id-keyed map")
}

func TestBase_ServiceUser_InMemory(t *testing.T) {
	// In the DDD layout the application service is split into CommandHandler
	// and QueryHandler. Both live under internal/application/user.
	r := generateProject(t, "svcmem", nil)
	contents := readFile(t, r.ProjectDir, "internal/application/user/command_handler.go")
	assertContains(t, contents, "CommandHandler", "in-memory service is a CommandHandler")
	contents = readFile(t, r.ProjectDir, "internal/application/user/query_handler.go")
	assertContains(t, contents, "QueryHandler", "in-memory service is a QueryHandler")
}

// ==========================================================================
// AUTH feature
// ==========================================================================

func TestAuth_FilesExist(t *testing.T) {
	r := generateProject(t, "authtest", []string{"auth"})
	dir := r.ProjectDir

	expectedFiles := []string{
		"internal/adapters/http/web/middleware/auth.go",
		"pkg/crypto/bcrypt_hasher.go",
		"internal/adapters/auth/jwt/token_service.go",
		"internal/adapters/http/web/auth_handler.go",
		"internal/domain/user/user.go",
		"internal/ports/hasher.go",
		"internal/ports/tokenservice.go",
	}
	for _, f := range expectedFiles {
		assertFileExists(t, dir, f)
	}
}

func TestAuth_GoMod_Deps(t *testing.T) {
	r := generateProject(t, "authgomod", []string{"auth"})
	content := readFile(t, r.ProjectDir, "go.mod")
	assertContains(t, content, "module", "go.mod")
	// Dependencies are returned via Result and installed via go get.
	assertDepsContains(t, r.Dependencies, "github.com/golang-jwt/jwt/v5", "auth deps")
	assertDepsContains(t, r.Dependencies, "github.com/google/uuid", "auth deps")
	assertDepsContains(t, r.Dependencies, "golang.org/x/crypto", "auth deps")
}

func TestAuth_Config_HasJWTSection(t *testing.T) {
	r := generateProject(t, "authcfg", []string{"auth"})
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "jwt:", "config.yaml jwt section")
	assertContains(t, content, "secret:", "config.yaml jwt secret")
	assertContains(t, content, "expiration:", "config.yaml jwt expiration")
	assertContains(t, content, "refresh_expiration:", "config.yaml jwt refresh")
}

func TestAuth_ConfigGo_HasJWTConfig(t *testing.T) {
	r := generateProject(t, "authcfggo", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "JWTConfig", "config.go JWTConfig struct")
	assertContains(t, content, "time.Duration", "config.go uses time.Duration")
}

func TestAuth_MiddlewareAuth(t *testing.T) {
	r := generateProject(t, "authmw", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/middleware/auth.go")
	assertContains(t, content, "func JWTAuth(", "auth middleware function")
	assertContains(t, content, "Bearer", "auth middleware checks bearer")
	assertContains(t, content, "ports.TokenService", "auth middleware uses TokenService port")
}

func TestAuth_JWTTokenService(t *testing.T) {
	// The auth feature ships a ports.TokenService adapter in the crypto
	// package — that is where JWT issue/parse lives in the DDD layout.
	r := generateProject(t, "authsvc", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/adapters/auth/jwt/token_service.go")
	assertContains(t, content, "TokenService", "JWT token service struct")
	assertContains(t, content, "TokenPair", "JWT token service returns TokenPair")
	assertContains(t, content, "jwt.ParseWithClaims", "JWT token service parses with claims")
}

func TestAuth_HandlerAuth(t *testing.T) {
	r := generateProject(t, "authhandler", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/auth_handler.go")
	assertContains(t, content, "AuthHandler", "auth handler struct")
	assertContains(t, content, "RegisterUser", "auth handler register endpoint")
	assertContains(t, content, "Login", "auth handler login endpoint")
	assertContains(t, content, "Refresh", "auth handler refresh endpoint")
	assertContains(t, content, "/me", "auth handler /me endpoint")
	assertContains(t, content, `validate:"required,email"`, "credentials email validation tag")
	assertContains(t, content, `validate:"required,min=8"`, "credentials password validation tag")
	assertContains(t, content, `validate:"omitempty,min=2,max=100"`, "credentials name validation tag")
}

func TestAuth_UserModel_HasPassword(t *testing.T) {
	// In the DDD layout the User aggregate has an unexported password field
	// plus PasswordHash() / SetPasswordHash() accessors. The DTOs that decide
	// what hits the wire live in the HTTP adapter, not on the aggregate.
	r := generateProject(t, "authmodel", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/domain/user/user.go")
	assertContains(t, content, "PasswordHash", "user aggregate has PasswordHash accessor")
	assertContains(t, content, "SetPasswordHash", "user aggregate has SetPasswordHash setter")
	assertNotContains(t, content, `json:"`, "domain aggregate still has no json tags")
}

func TestAuth_MainGo_ImportsService(t *testing.T) {
	r := generateProject(t, "authmain", []string{"auth"})
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, content, "jwt.NewTokenService", "main.go creates JWTTokenService")
	assertContains(t, content, "web.NewAuthHandler", "main.go creates AuthHandler")
	assertNotContains(t, content, "validator", "main.go no longer imports validator (moved to server.go)")
}

func TestAuth_HandlerHandler_HasJWTDep(t *testing.T) {
	// The HTTP aggregator (routes.go) does not depend on JWT — the auth
	// handler is mounted on its own group. Verify the auth handler carries
	// the token service instead.
	r := generateProject(t, "authdeps", []string{"auth"})
	routes := readFile(t, r.ProjectDir, "internal/adapters/http/web/v1/routes.go")
	assertNotContains(t, routes, "JWTTokenService", "routes.go has no JWT dep")
	auth := readFile(t, r.ProjectDir, "internal/adapters/http/web/auth_handler.go")
	assertContains(t, auth, "tokens ports.TokenService", "auth handler has TokenService dep")
}

func TestAuth_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "authmanifest", []string{"auth"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- auth", "manifest includes auth")
}

// ==========================================================================
// POSTGRES feature
// ==========================================================================

func TestBun_FilesExist(t *testing.T) {
	r := generateProject(t, "pgtest", []string{"bun"})
	dir := r.ProjectDir

	expectedFiles := []string{
		"internal/adapters/persistence/bun/db.go",
		"internal/adapters/persistence/bun/migrate.go",
		"internal/domain/user/user.go",
		"internal/adapters/persistence/memory/user_repository.go",
		"db/migrations/000001_init.up.sql",
		"db/migrations/000001_init.down.sql",
	}
	for _, f := range expectedFiles {
		assertFileExists(t, dir, f)
	}
}

func TestBun_GoMod_Deps(t *testing.T) {
	r := generateProject(t, "pggomod", []string{"bun"})
	content := readFile(t, r.ProjectDir, "go.mod")
	assertContains(t, content, "module", "go.mod")
	// Dependencies are returned via Result and installed via go get.
	assertDepsContains(t, r.Dependencies, "github.com/uptrace/bun", "bun deps")
	assertDepsContains(t, r.Dependencies, "github.com/uptrace/bun/dialect/pgdialect", "bun deps")
	assertDepsContains(t, r.Dependencies, "github.com/uptrace/bun/driver/pgdriver", "bun deps")
	assertDepsContains(t, r.Dependencies, "github.com/golang-migrate/migrate/v4", "bun deps")
}

func TestBun_Config_HasDatabaseSection(t *testing.T) {
	r := generateProject(t, "pgcfg", []string{"bun"})
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "database:", "config.yaml database section")
	assertContains(t, content, "host:", "config.yaml db host")
	assertContains(t, content, "port: 5432", "config.yaml db port")
	assertContains(t, content, "sslmode:", "config.yaml db sslmode")
}

func TestBun_ConfigGo_HasDatabaseConfig(t *testing.T) {
	r := generateProject(t, "pgcfggo", []string{"bun"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "DatabaseConfig", "config.go DatabaseConfig struct")
	assertContains(t, content, "func (d DatabaseConfig) DSN()", "config.go DSN method")
}

func TestBun_DatabaseBun(t *testing.T) {
	r := generateProject(t, "pgdb", []string{"bun"})
	content := readFile(t, r.ProjectDir, "internal/adapters/persistence/bun/db.go")
	assertContains(t, content, "func NewDB(", "db.go NewDB function")
	assertContains(t, content, "pgdriver", "db.go uses pgdriver")
	assertContains(t, content, "bun.NewDB", "db.go creates bun.DB")
}

func TestBun_DatabaseMigrate(t *testing.T) {
	r := generateProject(t, "pgmigrate", []string{"bun"})
	content := readFile(t, r.ProjectDir, "internal/adapters/persistence/bun/migrate.go")
	assertContains(t, content, "func MigrateUp(", "migrate.go MigrateUp function")
	assertContains(t, content, "func MigrateDown(", "migrate.go MigrateDown function")
	assertContains(t, content, "migrate.New", "migrate.go uses migrate.New")
}

func TestBun_MigrationUp(t *testing.T) {
	r := generateProject(t, "pgmigup", []string{"bun"})
	content := readFile(t, r.ProjectDir, "db/migrations/000001_init.up.sql")
	assertContains(t, content, "CREATE TABLE IF NOT EXISTS users", "migration up creates users table")
	assertContains(t, content, "UUID PRIMARY KEY", "migration up has uuid pk")
	assertContains(t, content, "email TEXT NOT NULL UNIQUE", "migration up email unique")
}

func TestBun_MigrationDown(t *testing.T) {
	r := generateProject(t, "pgmigdown", []string{"bun"})
	content := readFile(t, r.ProjectDir, "db/migrations/000001_init.down.sql")
	assertContains(t, content, "DROP TABLE IF EXISTS users", "migration down drops users")
}

func TestBun_UserModel_HasBunTags(t *testing.T) {
	// Bun tags live in the bun adapter's row DTO, not on the domain
	// aggregate. The aggregate stays tag-free.
	r := generateProject(t, "pgmodel", []string{"bun"})
	aggregate := readFile(t, r.ProjectDir, "internal/domain/user/user.go")
	assertNotContains(t, aggregate, `bun:"`, "domain aggregate has no bun tags")
	row := readFile(t, r.ProjectDir, "internal/adapters/persistence/bun/user_repository.go")
	assertContains(t, row, "type userRow struct", "bun row DTO is private")
	assertContains(t, row, `bun:"id,pk,type:uuid"`, "row DTO carries bun tags")
	assertContains(t, row, "toAggregate", "row DTO has toAggregate")
}

func TestBun_UserModel_NoAuth_NoPassword(t *testing.T) {
	r := generateProject(t, "pgmodelnopw", []string{"bun"})
	row := readFile(t, r.ProjectDir, "internal/adapters/persistence/bun/user_repository.go")
	assertNotContains(t, row, `Password`, "row DTO without auth has no password column")
}

func TestBun_RepositoryUser_BunBacked(t *testing.T) {
	r := generateProject(t, "pgrepo", []string{"bun"})
	content := readFile(t, r.ProjectDir, "internal/adapters/persistence/bun/user_repository.go")
	assertContains(t, content, "bun.IDB", "repo uses bun.IDB")
	assertContains(t, content, "func NewUserRepository(db bun.IDB)", "repo constructor takes bun.IDB")
	assertContains(t, content, "GetByEmail", "repo has GetByEmail method")
}

func TestBun_HandlerUser_UsesRepo(t *testing.T) {
	// The HTTP handler does not depend on the repository directly; it
	// delegates to the application command/query handlers. The composition
	// root wires the repository into the application layer.
	r := generateProject(t, "pghandler", []string{"bun"})
	handler := readFile(t, r.ProjectDir, "internal/adapters/http/web/v1/user_handler.go")
	assertNotContains(t, handler, "repo *repository.UserRepository", "handler has no repo dep")
	assertContains(t, handler, "h.cmd.HandleCreate(", "handler delegates to command handler")
	main := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, main, "bun.NewUserRepository", "main wires the bun repo into the app layer")
}

func TestBun_HandlerHandler_HasDBDep(t *testing.T) {
	// Routes.go is DB-agnostic. The DB dependency lives in main.go where the
	// bun connection is opened and passed to the bun adapter.
	r := generateProject(t, "pghandlerdep", []string{"bun"})
	routes := readFile(t, r.ProjectDir, "internal/adapters/http/web/v1/routes.go")
	assertNotContains(t, routes, "*bun.DB", "routes.go has no DB dep")
	main := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, main, "bun.NewDB", "main opens the bun DB")
}

func TestBun_MainGo_ImportsDatabase(t *testing.T) {
	r := generateProject(t, "pgmain", []string{"bun"})
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, content, "bun.NewDB", "main.go creates Bun connection")
	assertContains(t, content, "db.Close()", "main.go defers db close")
}

func TestBun_Makefile_NoMigrateTargets(t *testing.T) {
	// Common commands (including migrate) are provided by the crank CLI, not
	// duplicated in the Makefile.
	r := generateProject(t, "pgmake", []string{"bun"})
	content := readFile(t, r.ProjectDir, "Makefile")
	assertNotContains(t, content, "migrate-up:", "Makefile no longer defines migrate-up target")
	assertNotContains(t, content, "migrate-down:", "Makefile no longer defines migrate-down target")
	assertNotContains(t, content, "migrate-create:", "Makefile no longer defines migrate-create target")
}

func TestBun_Gitignore_HasDataDir(t *testing.T) {
	r := generateProject(t, "pggitignore", []string{"bun"})
	content := readFile(t, r.ProjectDir, ".gitignore")
	assertContains(t, content, "data/", "gitignore with bun has data/")
}

func TestBun_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "pgmanifest", []string{"bun"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- bun", "manifest includes bun")
}

func TestBun_Readme_HasDBInfo(t *testing.T) {
	r := generateProject(t, "pgreadme", []string{"bun"})
	content := readFile(t, r.ProjectDir, "README.md")
	assertContains(t, content, "Bun", "README mentions Bun")
	assertContains(t, content, "golang-migrate", "README mentions migrate")
	assertContains(t, content, "crank migrate", "README documents the crank migrate command")
}

// ==========================================================================
// GORM feature (default ORM)
// ==========================================================================

func TestGorm_FilesExist(t *testing.T) {
	r := generateProject(t, "gormtest", []string{"gorm"})
	dir := r.ProjectDir

	expectedFiles := []string{
		"internal/adapters/persistence/gorm/db.go",
		"internal/adapters/persistence/gorm/migrate.go",
		"internal/adapters/persistence/gorm/user_repository.go",
		"internal/domain/user/user.go",
		"internal/adapters/persistence/memory/user_repository.go",
		"db/migrations/000001_init.up.sql",
		"db/migrations/000001_init.down.sql",
	}
	for _, f := range expectedFiles {
		assertFileExists(t, dir, f)
	}
}

func TestGorm_GoMod_Deps(t *testing.T) {
	r := generateProject(t, "gormgomod", []string{"gorm"})
	assertDepsContains(t, r.Dependencies, "gorm.io/gorm", "gorm deps")
	assertDepsContains(t, r.Dependencies, "gorm.io/driver/postgres", "gorm deps")
	assertDepsContains(t, r.Dependencies, "github.com/golang-migrate/migrate/v4", "gorm deps")
}

func TestGorm_Config_HasDatabaseSection(t *testing.T) {
	r := generateProject(t, "gormcfg", []string{"gorm"})
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "database:", "config.yaml database section")
	assertContains(t, content, "host:", "config.yaml db host")
	assertContains(t, content, "port: 5432", "config.yaml db port")
	assertContains(t, content, "sslmode:", "config.yaml db sslmode")
}

func TestGorm_ConfigGo_HasDatabaseConfig(t *testing.T) {
	r := generateProject(t, "gormcfggo", []string{"gorm"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "DatabaseConfig", "config.go DatabaseConfig struct")
	assertContains(t, content, "func (d DatabaseConfig) DSN()", "config.go DSN method")
}

func TestGorm_DatabaseGorm(t *testing.T) {
	r := generateProject(t, "gormdb", []string{"gorm"})
	content := readFile(t, r.ProjectDir, "internal/adapters/persistence/gorm/db.go")
	assertContains(t, content, "func NewDB(", "db.go NewDB function")
	assertContains(t, content, "gorm.io/driver/postgres", "db.go uses gorm postgres driver")
	assertContains(t, content, "gorm.Open", "db.go opens gorm")
}

func TestGorm_DatabaseMigrate(t *testing.T) {
	r := generateProject(t, "gormmigrate", []string{"gorm"})
	content := readFile(t, r.ProjectDir, "internal/adapters/persistence/gorm/migrate.go")
	assertContains(t, content, "func MigrateUp(", "migrate.go MigrateUp function")
	assertContains(t, content, "func MigrateDown(", "migrate.go MigrateDown function")
	assertContains(t, content, "migrate.New", "migrate.go uses migrate.New")
}

func TestGorm_MigrationUp(t *testing.T) {
	r := generateProject(t, "gormmigup", []string{"gorm"})
	content := readFile(t, r.ProjectDir, "db/migrations/000001_init.up.sql")
	assertContains(t, content, "CREATE TABLE IF NOT EXISTS users", "migration up creates users table")
	assertContains(t, content, "UUID PRIMARY KEY", "migration up has uuid pk")
	assertContains(t, content, "email TEXT NOT NULL UNIQUE", "migration up email unique")
}

func TestGorm_UserRepository_GormBacked(t *testing.T) {
	r := generateProject(t, "gormrepo", []string{"gorm"})
	content := readFile(t, r.ProjectDir, "internal/adapters/persistence/gorm/user_repository.go")
	assertContains(t, content, "*gorm.DB", "repo uses gorm.DB")
	assertContains(t, content, "func NewUserRepository(db *gorm.DB)", "repo constructor takes gorm.DB")
	assertContains(t, content, "GetByEmail", "repo has GetByEmail method")
	assertContains(t, content, "gorm.ErrRecordNotFound", "repo maps ErrRecordNotFound")
	assertContains(t, content, "TableName", "row DTO declares table name")
}

func TestGorm_UserModel_HasGormTags(t *testing.T) {
	// GORM tags live in the gorm adapter's row DTO, not on the domain
	// aggregate. The aggregate stays tag-free.
	r := generateProject(t, "gormmodel", []string{"gorm"})
	aggregate := readFile(t, r.ProjectDir, "internal/domain/user/user.go")
	assertNotContains(t, aggregate, `gorm:"`, "domain aggregate has no gorm tags")
	row := readFile(t, r.ProjectDir, "internal/adapters/persistence/gorm/user_repository.go")
	assertContains(t, row, "type userRow struct", "gorm row DTO is private")
	assertContains(t, row, `gorm:"column:id;type:uuid;primaryKey"`, "row DTO carries gorm tags")
	assertContains(t, row, "toAggregate", "row DTO has toAggregate")
}

func TestGorm_MainGo_ImportsGorm(t *testing.T) {
	r := generateProject(t, "gormmain", []string{"gorm"})
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, content, "gorm.NewDB", "main.go creates gorm connection")
	assertContains(t, content, "gorm.NewUserRepository", "main.go wires gorm user repo")
	assertContains(t, content, "sqlDB.Close()", "main.go defers gorm sqlDB close")
}

func TestGorm_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "gormmanifest", []string{"gorm"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- gorm", "manifest includes gorm")
}

func TestGorm_Readme_HasDBInfo(t *testing.T) {
	r := generateProject(t, "gormreadme", []string{"gorm"})
	content := readFile(t, r.ProjectDir, "README.md")
	assertContains(t, content, "GORM", "README mentions GORM")
	assertContains(t, content, "golang-migrate", "README mentions migrate")
	assertContains(t, content, "crank migrate", "README documents the crank migrate command")
}

func TestGorm_NotPresent_NoBunImports(t *testing.T) {
	// A base + gorm project should not have any bun references in main.go.
	r := generateProject(t, "gormnobun", []string{"gorm"})
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertNotContains(t, content, "bun.NewDB", "gorm-only main has no bun init")
	assertNotContains(t, content, "bun.NewUserRepository", "gorm-only main has no bun user repo")
}

func TestBun_NotPresent_NoGormImports(t *testing.T) {
	// A base + bun project should not have any gorm references in main.go.
	r := generateProject(t, "bunnogorm", []string{"bun"})
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertNotContains(t, content, "gorm.NewDB", "bun-only main has no gorm init")
	assertNotContains(t, content, "gorm.NewUserRepository", "bun-only main has no gorm user repo")
}

func TestGorm_Scaffold_GormRepository(t *testing.T) {
	// A gorm-based project should get a GORM-backed repository via `crank make`.
	r := generateProject(t, "gormscaff", []string{"gorm"})
	dir := r.ProjectDir

	// Generate a handler (which pulls in model + repo + service + wiring).
	res, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindHandler,
		Name:       "Order",
		Fields:     []string{"customer:string", "total:float"},
	})
	if err != nil {
		t.Fatalf("Generate handler on gorm project: %v", err)
	}

	// GORM adapter should exist.
	assertFileExists(t, dir, "internal/adapters/persistence/gorm/order_repository.go")
	// No bun adapter (gorm-only project).
	assertFileNotExists(t, dir, "internal/adapters/persistence/bun/order_repository.go")

	// In-memory adapter always ships.
	assertFileExists(t, dir, "internal/adapters/persistence/memory/order_repository.go")

	// Repository uses gorm.DB and GORM tags.
	repo := readFile(t, dir, "internal/adapters/persistence/gorm/order_repository.go")
	assertContains(t, repo, "*gorm.DB", "gorm repo uses *gorm.DB")
	assertContains(t, repo, `gorm:"column:id;primaryKey;type:uuid"`, "row DTO carries gorm tags")
	assertContains(t, repo, "TableName()", "row DTO declares table name")
	assertContains(t, repo, "func (r *OrderRepository) Save(", "gorm repo has Save")
	assertContains(t, repo, "gorm.ErrRecordNotFound", "gorm repo maps ErrRecordNotFound")

	// A create-table migration was produced.
	matches, _ := filepath.Glob(filepath.Join(dir, "db/migrations", "*_create_orders.up.sql"))
	if len(matches) != 1 {
		t.Errorf("expected one create_orders migration, got %d", len(matches))
	}

	// Handler was wired.
	if !res.Wired {
		t.Errorf("expected handler to be wired")
	}
}

func TestGorm_Outbox_FilesExist(t *testing.T) {
	// A base + gorm + outbox project should have GORM outbox files, not bun ones.
	r := generateProject(t, "gormoutbox", []string{"gorm", "outbox"})
	dir := r.ProjectDir

	// Domain outbox files (ORM-agnostic).
	assertFileExists(t, dir, "internal/domain/outbox/event.go")
	assertFileExists(t, dir, "internal/domain/outbox/repository.go")

	// GORM outbox adapter.
	assertFileExists(t, dir, "internal/adapters/persistence/gorm/outbox_repository.go")
	assertFileExists(t, dir, "internal/adapters/outbox/gorm_uow.go")
	outboxRepo := readFile(t, dir, "internal/adapters/persistence/gorm/outbox_repository.go")
	assertContains(t, outboxRepo, "*gorm.DB", "gorm outbox repo uses *gorm.DB")
	assertContains(t, outboxRepo, "NewOutboxRepository", "gorm outbox repo constructor")

	// Bun outbox adapter should NOT exist (gorm-only project).
	assertFileNotExists(t, dir, "internal/adapters/persistence/bun/outbox_repository.go")
	assertFileNotExists(t, dir, "internal/adapters/outbox/bun_uow.go")

	// Worker is ORM-agnostic.
	assertFileExists(t, dir, "internal/adapters/outbox/worker.go")

	// Outbox migrations exist.
	assertFileExists(t, dir, "db/migrations/000002_add_outbox_events.up.sql")
	assertFileExists(t, dir, "db/migrations/000002_add_outbox_events.down.sql")

	// Main.go wires the GORM outbox.
	main := readFile(t, dir, "cmd/server/main.go")
	assertContains(t, main, "gorm.NewOutboxRepository", "main.go wires gorm outbox repo")
	assertContains(t, main, "outboxadapter.NewGormUoW", "main.go wires gorm UoW")
	assertNotContains(t, main, "bun.NewOutboxRepository", "main.go has no bun outbox")
}

func TestBun_Outbox_FilesExist(t *testing.T) {
	// A base + bun + outbox project should have bun outbox files, not gorm ones.
	r := generateProject(t, "bunoutbox", []string{"bun", "outbox"})
	dir := r.ProjectDir

	// Bun outbox adapter exists.
	assertFileExists(t, dir, "internal/adapters/persistence/bun/outbox_repository.go")
	assertFileExists(t, dir, "internal/adapters/outbox/bun_uow.go")

	// GORM outbox adapter should NOT exist.
	assertFileNotExists(t, dir, "internal/adapters/persistence/gorm/outbox_repository.go")
	assertFileNotExists(t, dir, "internal/adapters/outbox/gorm_uow.go")

	// Main.go wires the bun outbox.
	main := readFile(t, dir, "cmd/server/main.go")
	assertContains(t, main, "bun.NewOutboxRepository", "main.go wires bun outbox repo")
	assertContains(t, main, "outboxadapter.NewBunUoW", "main.go wires bun UoW")
	assertNotContains(t, main, "gorm.NewOutboxRepository", "main.go has no gorm outbox")
}

func TestOutbox_NoORM_Fails(t *testing.T) {
	// Trying to add outbox without either bun or gorm must fail.
	_, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "noormoutbox",
		ModulePath:  "github.com/example/noormoutbox",
		TargetDir:   t.TempDir(),
		Features:    []string{"base", "outbox"},
	})
	if err == nil {
		t.Fatal("expected Generate to refuse outbox without an ORM")
	}
	if !strings.Contains(err.Error(), "requires a database ORM") {
		t.Errorf("expected ORM requirement error, got: %v", err)
	}
}

func TestBunAndGorm_MutuallyExclusive(t *testing.T) {
	// Both ORM features in the same project is a hard error.
	_, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "tworms",
		ModulePath:  "github.com/example/tworms",
		TargetDir:   t.TempDir(),
		Features:    []string{"base", "bun", "gorm"},
	})
	if err == nil {
		t.Fatal("expected Generate to refuse when both bun and gorm are selected")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got: %v", err)
	}
}

// ==========================================================================
// VALIDATOR feature (always included via base)
// ==========================================================================

func TestValidator_FilesExist(t *testing.T) {
	r := generateProject(t, "valfiles", nil)
	assertFileExists(t, r.ProjectDir, "internal/validator/validator.go")
	assertFileExists(t, r.ProjectDir, "internal/validator/errors.go")
}

func TestValidator_GoMod_HasDep(t *testing.T) {
	r := generateProject(t, "valgomod", nil)
	content := readFile(t, r.ProjectDir, "go.mod")
	assertContains(t, content, "module", "go.mod")
	assertDepsContains(t, r.Dependencies, "github.com/go-playground/validator/v10", "validator dep")
}

func TestValidator_ValidatorGo_Structures(t *testing.T) {
	r := generateProject(t, "valstruct", nil)
	content := readFile(t, r.ProjectDir, "internal/validator/validator.go")
	assertContains(t, content, "package validator", "validator package")
	assertContains(t, content, "func Init()", "validator Init function")
	assertContains(t, content, "func Validate()", "validator Validate accessor")
	assertContains(t, content, "func Struct(", "validator Struct function")
	assertContains(t, content, "validator.New(", "validator creates new instance")
	assertContains(t, content, "RegisterTagNameFunc", "validator registers JSON tag name func")
	assertContains(t, content, "json", "validator uses json tag for field names")
	assertContains(t, content, "validate.Struct(s)", "Struct calls validate.Struct")
}

func TestValidator_ValidatorGo_CustomValidatorRegistration(t *testing.T) {
	r := generateProject(t, "valcustom", nil)
	content := readFile(t, r.ProjectDir, "internal/validator/validator.go")
	assertContains(t, content, "Register custom validators here", "custom validator registration placeholder")
	assertContains(t, content, "notblank", "notblank example validator")
	assertContains(t, content, "RegisterValidation", "shows how to register validators")
}

func TestValidator_ErrorsGo_Structures(t *testing.T) {
	r := generateProject(t, "valerrors", nil)
	content := readFile(t, r.ProjectDir, "internal/validator/errors.go")
	assertContains(t, content, "package validator", "errors package")
	assertContains(t, content, "ValidationError", "ValidationError type")
	assertContains(t, content, "HTTPStatus int", "ValidationError has HTTPStatus")
	assertContains(t, content, `Message    string`, "ValidationError has Message")
	assertContains(t, content, `Errors     map[string]string`, "ValidationError has Errors map")
	assertContains(t, content, "ErrorResponse", "ErrorResponse type")
	assertContains(t, content, "toValidationError", "toValidationError conversion function")
	assertContains(t, content, "humanMessage", "humanMessage function")
}

func TestValidator_ErrorsGo_HumanMessages(t *testing.T) {
	r := generateProject(t, "valhuman", nil)
	content := readFile(t, r.ProjectDir, "internal/validator/errors.go")

	// Verify all built-in human message cases
	expectedMessages := []struct {
		tag  string
		text string
	}{
		{"required", "is required"},
		{"email", "must be a valid email address"},
		{"min", "must be at least"},
		{"max", "must be at most"},
		{"len", "must be exactly"},
		{"oneof", "must be one of:"},
		{"url", "must be a valid URL"},
		{"uuid", "must be a valid UUID"},
		{"gt", "must be greater than"},
		{"gte", "must be greater than or equal to"},
		{"lt", "must be less than"},
		{"lte", "must be less than or equal to"},
	}
	for _, em := range expectedMessages {
		assertContains(t, content, em.tag, "humanMessage case: "+em.tag)
		assertContains(t, content, em.text, "humanMessage text: "+em.text)
	}

	// Default fallback
	assertContains(t, content, "failed validation:", "humanMessage default fallback")
}

func TestValidator_ErrorsGo_ErrorMethod(t *testing.T) {
	r := generateProject(t, "valerr", nil)
	content := readFile(t, r.ProjectDir, "internal/validator/errors.go")
	assertContains(t, content, "func (e *ValidationError) Error()", "Error() method on ValidationError")
}

func TestValidator_ErrorsGo_HTTPStatus(t *testing.T) {
	r := generateProject(t, "valstatus", nil)
	content := readFile(t, r.ProjectDir, "internal/validator/errors.go")
	assertContains(t, content, "StatusUnprocessableEntity", "uses 422 for validation errors")
	assertContains(t, content, "validation failed", "default validation failed message")
}

func TestValidator_HandlerBinder_Integration(t *testing.T) {
	// In the DDD layout, validation lives in the DTOs themselves (validate
	// struct tags). The custom binder is no longer needed because each
	// handler's input DTO self-validates when c.Bind is called.
	r := generateProject(t, "valbinder", nil)
	dto := readFile(t, r.ProjectDir, "internal/adapters/http/web/v1/user_handler.go")
	assertContains(t, dto, `validate:"required,email"`, "userDTO has email validate tag")
	assertContains(t, dto, `validate:"required,min=2,max=100"`, "userDTO has name validate tag")
}

func TestValidator_MainGo_ErrorHandler(t *testing.T) {
	r := generateProject(t, "valmaineh", nil)
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/server.go")
	assertContains(t, content, `err.(*validator.ValidationError)`, "checks for ValidationError type")
	assertContains(t, content, `err.(*echo.HTTPError)`, "checks for Echo HTTPError type")
	assertContains(t, content, "internal server error", "fallback error message")
}

func TestValidator_UserHandler_CreateDelegatesToBind(t *testing.T) {
	r := generateProject(t, "valuserh", nil)
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/v1/user_handler.go")
	// Create handler relies on the custom binder for validation
	assertContains(t, content, "c.Bind(&in)", "Create uses Bind (which validates)")
}

func TestValidator_AuthHandler_ValidationTags(t *testing.T) {
	r := generateProject(t, "valauthtags", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/auth_handler.go")
	assertContains(t, content, `validate:"required,email"`, "email validation tag")
	assertContains(t, content, `validate:"required,min=8"`, "password min 8 validation tag")
	assertContains(t, content, `validate:"omitempty,min=2,max=100"`, "name optional validation tag")
	assertContains(t, content, `validate:"required"`, "refresh_token required validation")
}

func TestValidator_AuthHandler_CredentialsStruct(t *testing.T) {
	r := generateProject(t, "valauthcred", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/auth_handler.go")
	assertContains(t, content, "type credentials struct", "credentials struct defined")
	assertContains(t, content, "Email", "credentials has Email")
	assertContains(t, content, "Password", "credentials has Password")
	assertContains(t, content, "Name", "credentials has Name")
}

func TestValidator_AuthHandler_RefreshStruct(t *testing.T) {
	r := generateProject(t, "valauthref", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/adapters/http/web/auth_handler.go")
	assertContains(t, content, "type refreshRequest struct", "refreshRequest struct defined")
	assertContains(t, content, "RefreshToken", "refreshRequest has RefreshToken")
	assertContains(t, content, `json:"refresh_token"`, "refreshRequest has json tag")
}

// ==========================================================================
// AUTH + POSTGRES together
// ==========================================================================

func TestAuthBun_MigrationUp_HasPassword(t *testing.T) {
	r := generateProject(t, "authpg", []string{"auth", "bun"})
	content := readFile(t, r.ProjectDir, "db/migrations/000001_init.up.sql")
	assertContains(t, content, "password TEXT NOT NULL", "migration has password column")
}

func TestAuthBun_UserModel_HasBunAndPassword(t *testing.T) {
	// Password is on the domain aggregate (as an unexported field) and the
	// bun row DTO carries the bun:"password,notnull" tag. The
	// aggregate itself stays free of ORM tags.
	r := generateProject(t, "authpgmodel", []string{"auth", "bun"})
	agg := readFile(t, r.ProjectDir, "internal/domain/user/user.go")
	assertNotContains(t, agg, `bun:"`, "domain aggregate has no bun tags")
	assertContains(t, agg, "PasswordHash", "domain aggregate has PasswordHash accessor")
	row := readFile(t, r.ProjectDir, "internal/adapters/persistence/bun/user_repository.go")
	assertContains(t, row, `bun:"password,notnull"`, "row DTO has password bun tag")
}

func TestAuthBun_MainGo_HasBothDeps(t *testing.T) {
	r := generateProject(t, "authpgmain", []string{"auth", "bun"})
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, content, "bun.NewDB", "main.go creates bun db")
	assertContains(t, content, "jwt.NewTokenService", "main.go creates jwt")
	assertContains(t, content, "web.NewAuthHandler", "main.go creates auth handler")
}

func TestAuthBun_ConfigGo_HasBothConfigs(t *testing.T) {
	r := generateProject(t, "authpgcfg", []string{"auth", "bun"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "DatabaseConfig", "config has DatabaseConfig")
	assertContains(t, content, "JWTConfig", "config has JWTConfig")
}

func TestAuthBun_GoMod_HasAllDeps(t *testing.T) {
	r := generateProject(t, "authpggomod", []string{"auth", "bun"})
	content := readFile(t, r.ProjectDir, "go.mod")
	assertContains(t, content, "module", "go.mod")
	assertDepsContains(t, r.Dependencies, "golang-jwt/jwt/v5", "auth+pg deps")
	assertDepsContains(t, r.Dependencies, "golang.org/x/crypto", "auth+pg deps")
	assertDepsContains(t, r.Dependencies, "uptrace/bun", "auth+pg deps")
	assertDepsContains(t, r.Dependencies, "golang-migrate/migrate/v4", "auth+pg deps")
}

func TestAuthBun_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "authpgmanifest", []string{"auth", "bun"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- auth", "manifest has auth")
	assertContains(t, content, "- bun", "manifest has bun")
}

// ==========================================================================
// REDIS feature (placeholder)
// ==========================================================================

func TestRedis_FilesExist(t *testing.T) {
	r := generateProject(t, "redistest", []string{"redis"})
	assertFileExists(t, r.ProjectDir, "internal/adapters/cache/redis/client.go")
}

func TestRedis_Client(t *testing.T) {
	r := generateProject(t, "redisclient", []string{"redis"})
	content := readFile(t, r.ProjectDir, "internal/adapters/cache/redis/client.go")
	assertContains(t, content, "package redis", "redis client package")
	assertContains(t, content, "func NewClient(", "redis client NewClient")
	assertContains(t, content, "redis.NewClient", "redis client uses go-redis")
	assertContains(t, content, "client.Ping", "redis client pings")
	assertContains(t, content, "config.RedisConfig", "redis client uses shared config type")
}

func TestRedis_Config_HasRedisSection(t *testing.T) {
	r := generateProject(t, "rediscfg", []string{"redis"})
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "redis:", "config.yaml redis section")
	assertContains(t, content, "addr:", "config.yaml redis addr key")
}

func TestRedis_ConfigGo_HasRedisConfig(t *testing.T) {
	r := generateProject(t, "rediscfggo", []string{"redis"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "Redis RedisConfig", "Config struct has Redis field")
	assertContains(t, content, "RedisConfig struct", "RedisConfig struct defined")
	assertContains(t, content, "Addr", "RedisConfig has Addr field")
	assertContains(t, content, "Password", "RedisConfig has Password field")
	assertContains(t, content, "DB", "RedisConfig has DB field")
}

func TestRedis_ConfigGo_NoRedisConfig(t *testing.T) {
	r := generateProject(t, "norediscfggo", nil)
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertNotContains(t, content, "RedisConfig", "config.go without redis has no RedisConfig")
}

func TestRedis_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "redismanifest", []string{"redis"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- redis", "manifest includes redis")
}

// ==========================================================================
// CRYPTO feature
// ==========================================================================

func TestCrypto_FilesExist(t *testing.T) {
	r := generateProject(t, "cryptotest", []string{"crypto"})
	assertFileExists(t, r.ProjectDir, "pkg/crypto/aesgcm_cipher.go")
}

func TestCrypto_CryptoGo_PackageAndType(t *testing.T) {
	r := generateProject(t, "cryptopkg", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "pkg/crypto/aesgcm_cipher.go")
	assertContains(t, content, "package crypto", "crypto package")
	assertContains(t, content, "type AESGCMCipher struct", "Crypto struct")
	assertContains(t, content, "aead cipher.AEAD", "Crypto has AEAD field")
}

func TestCrypto_CryptoGo_NewFunction(t *testing.T) {
	r := generateProject(t, "cryptonew", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "pkg/crypto/aesgcm_cipher.go")
	assertContains(t, content, "func NewAESGCMCipher(secret string) (*AESGCMCipher, error)", "NewAESGCMCipher function signature")
	assertContains(t, content, "secret must not be empty", "New rejects empty secret")
	assertContains(t, content, "sha256.Sum256", "New derives key via SHA-256")
	assertContains(t, content, "aes.NewCipher", "New creates AES cipher")
	assertContains(t, content, "cipher.NewGCM", "New creates GCM AEAD")
}

func TestCrypto_CryptoGo_EncryptFunction(t *testing.T) {
	r := generateProject(t, "cryptoenc", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "pkg/crypto/aesgcm_cipher.go")
	assertContains(t, content, "func (c *AESGCMCipher) Encrypt(plaintext string) (string, error)", "AESGCMCipher Encrypt signature")
	assertContains(t, content, "c.aead.NonceSize()", "Encrypt uses nonce")
	assertContains(t, content, "rand.Reader", "Encrypt uses crypto/rand")
	assertContains(t, content, "c.aead.Seal", "Encrypt calls Seal")
	assertContains(t, content, "base64.RawURLEncoding.EncodeToString", "Encrypt encodes base64-url")
}

func TestCrypto_CryptoGo_DecryptFunction(t *testing.T) {
	r := generateProject(t, "cryptodec", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "pkg/crypto/aesgcm_cipher.go")
	assertContains(t, content, "func (c *AESGCMCipher) Decrypt(encoded string) (string, error)", "AESGCMCipher Decrypt signature")
	assertContains(t, content, "base64.RawURLEncoding.DecodeString", "Decrypt decodes base64-url")
	assertContains(t, content, "ciphertext too short", "Decrypt rejects short ciphertext")
	assertContains(t, content, "c.aead.Open", "Decrypt calls Open")
}

func TestCrypto_CryptoGo_Imports(t *testing.T) {
	r := generateProject(t, "cryptoimports", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "pkg/crypto/aesgcm_cipher.go")
	assertContains(t, content, "crypto/aes", "imports crypto/aes")
	assertContains(t, content, "crypto/cipher", "imports crypto/cipher")
	assertContains(t, content, "crypto/rand", "imports crypto/rand")
	assertContains(t, content, "crypto/sha256", "imports crypto/sha256")
	assertContains(t, content, "encoding/base64", "imports encoding/base64")
	assertContains(t, content, "errors", "imports errors")
}

func TestCrypto_Config_HasCryptoSection(t *testing.T) {
	r := generateProject(t, "cryptocfg", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "crypto:", "config.yaml crypto section")
	assertContains(t, content, "secret:", "config.yaml crypto secret key")
	assertContains(t, content, "change-me-in-production", "config.yaml crypto default secret")
}

func TestCrypto_Config_NoCryptoSection(t *testing.T) {
	r := generateProject(t, "nocryptocfg", nil)
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertNotContains(t, content, "crypto:", "config.yaml without crypto has no crypto section")
}

func TestCrypto_ConfigGo_HasCryptoConfig(t *testing.T) {
	r := generateProject(t, "cryptocfggo", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "Crypto CryptoConfig", "Config struct has Crypto field")
	assertContains(t, content, "CryptoConfig struct", "CryptoConfig struct defined")
	assertContains(t, content, `Secret string`, "CryptoConfig has Secret field")
	assertContains(t, content, `mapstructure:"crypto"`, "Crypto field has mapstructure tag")
}

func TestCrypto_ConfigGo_NoCryptoConfig(t *testing.T) {
	r := generateProject(t, "nocryptocfggo", nil)
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertNotContains(t, content, "CryptoConfig", "config.go without crypto has no CryptoConfig")
}

func TestCrypto_ConfigGo_Defaults(t *testing.T) {
	r := generateProject(t, "cryptodefaults", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, `crypto.secret`, "viper default for crypto.secret")
	assertContains(t, content, "change-me-in-production", "default crypto secret value")
}

func TestCrypto_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "cryptomanifest", []string{"crypto"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- crypto", "manifest includes crypto")
}

func TestCrypto_NoExtraGoModDeps(t *testing.T) {
	// Crypto uses only stdlib — verify it adds no deps beyond what base provides.
	r := generateProject(t, "cryptogomod", []string{"crypto"})
	// base is auto-included and has its own deps; crypto should add none.
	baseResult := generateProject(t, "cryptobaseonly", nil)
	if len(r.Dependencies) != len(baseResult.Dependencies) {
		t.Errorf("crypto should not add deps beyond base: got %d deps, base has %d", len(r.Dependencies), len(baseResult.Dependencies))
	}
}

func TestCrypto_AuthCombined_CryptoConfigPresent(t *testing.T) {
	r := generateProject(t, "cryptoauth", []string{"crypto", "auth"})
	cfg := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, cfg, "crypto:", "config has crypto section")
	assertContains(t, cfg, "jwt:", "config has jwt section")
	cfgGo := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, cfgGo, "CryptoConfig", "config.go has CryptoConfig")
	assertContains(t, cfgGo, "JWTConfig", "config.go has JWTConfig")
}

func TestCrypto_BunCombined_AllSections(t *testing.T) {
	r := generateProject(t, "cryptopg", []string{"crypto", "bun"})
	cfg := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, cfg, "crypto:", "config has crypto section")
	assertContains(t, cfg, "database:", "config has database section")
	assertFileExists(t, r.ProjectDir, "pkg/crypto/aesgcm_cipher.go")
	assertFileExists(t, r.ProjectDir, "internal/adapters/persistence/bun/db.go")
}

// ==========================================================================
// MONGODB feature (placeholder)
// ==========================================================================

func TestMongodb_FilesExist(t *testing.T) {
	r := generateProject(t, "mongotest", []string{"mongodb"})
	assertFileExists(t, r.ProjectDir, "internal/adapters/persistence/mongodb/client.go")
}

func TestMongodb_Client(t *testing.T) {
	r := generateProject(t, "mongoclient", []string{"mongodb"})
	content := readFile(t, r.ProjectDir, "internal/adapters/persistence/mongodb/client.go")
	assertContains(t, content, "package mongo", "mongo client package")
	assertContains(t, content, "func NewClient(", "mongo client NewClient")
	assertContains(t, content, "mongo.Connect", "mongo client uses mongo driver")
	assertContains(t, content, "client.Ping", "mongo client pings")
	assertContains(t, content, "config.MongoDBConfig", "mongo client uses shared config type")
}

func TestMongodb_Config_HasMongodbSection(t *testing.T) {
	r := generateProject(t, "mongocfg", []string{"mongodb"})
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "mongodb:", "config.yaml mongodb section")
	assertContains(t, content, "uri:", "config.yaml mongodb uri key")
}

func TestMongodb_ConfigGo_HasMongodbConfig(t *testing.T) {
	r := generateProject(t, "mongocfggo", []string{"mongodb"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "MongoDB MongoDBConfig", "Config struct has MongoDB field")
	assertContains(t, content, "MongoDBConfig struct", "MongoDBConfig struct defined")
}

func TestMongodb_ConfigGo_NoMongodbConfig(t *testing.T) {
	r := generateProject(t, "nomongocfggo", nil)
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertNotContains(t, content, "MongoDBConfig", "config.go without mongodb has no MongoDBConfig")
}

func TestMongodb_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "mongomanifest", []string{"mongodb"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- mongodb", "manifest includes mongodb")
}

// ==========================================================================
// QDRANT feature
// ==========================================================================

func TestQdrant_FilesExist(t *testing.T) {
	r := generateProject(t, "qdranttest", []string{"qdrant"})
	assertFileExists(t, r.ProjectDir, "internal/adapters/persistence/qdrant/client.go")
}

func TestQdrant_Client(t *testing.T) {
	r := generateProject(t, "qdrantclient", []string{"qdrant"})
	content := readFile(t, r.ProjectDir, "internal/adapters/persistence/qdrant/client.go")
	assertContains(t, content, "package qdrant", "qdrant client package")
	assertContains(t, content, "func NewClient(", "qdrant client NewClient")
	assertContains(t, content, "qdrantgo.NewClient", "qdrant client uses go-client")
	assertContains(t, content, "client.HealthCheck", "qdrant client health checks")
	assertContains(t, content, "config.QdrantConfig", "qdrant client uses shared config type")
}

func TestQdrant_Config_HasQdrantSection(t *testing.T) {
	r := generateProject(t, "qdrantcfg", []string{"qdrant"})
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "qdrant:", "config.yaml qdrant section")
	assertContains(t, content, "port: 6334", "config.yaml qdrant port key")
}

func TestQdrant_ConfigGo_HasQdrantConfig(t *testing.T) {
	r := generateProject(t, "qdrantcfggo", []string{"qdrant"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "Qdrant QdrantConfig", "Config struct has Qdrant field")
	assertContains(t, content, "QdrantConfig struct", "QdrantConfig struct defined")
}

func TestQdrant_ConfigGo_NoQdrantConfig(t *testing.T) {
	r := generateProject(t, "noqdrantcfggo", nil)
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertNotContains(t, content, "QdrantConfig", "config.go without qdrant has no QdrantConfig")
}

func TestQdrant_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "qdrantmanifest", []string{"qdrant"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- qdrant", "manifest includes qdrant")
}

// ==========================================================================
// TEMPORAL feature
// ==========================================================================

func TestTemporal_FilesExist(t *testing.T) {
	r := generateProject(t, "temporaltest", []string{"temporal"})
	// In the DDD layout the Temporal client and worker share a single file
	// (worker.go) since they are the only two top-level types in the package.
	for _, rel := range []string{
		"internal/adapters/temporal/logger.go",
		"internal/adapters/temporal/worker.go",
		"internal/adapters/temporal/workflow/greeting.go",
		"internal/adapters/temporal/activity/greeting.go",
		"cmd/worker/main.go",
	} {
		assertFileExists(t, r.ProjectDir, rel)
	}
}

func TestTemporal_GoMod_Deps(t *testing.T) {
	r := generateProject(t, "temporaldeps", []string{"temporal"})
	assertDepsContains(t, r.Dependencies, "go.temporal.io/sdk", "temporal deps")
}

func TestTemporal_Worker_RegistersAndHasMarkers(t *testing.T) {
	r := generateProject(t, "temporalworker", []string{"temporal"})
	dir := r.ProjectDir

	content := readFile(t, dir, "internal/adapters/temporal/worker.go")
	assertContains(t, content, "// crank:workflow-register", "worker workflow marker")
	assertContains(t, content, "w.RegisterWorkflow(workflow.GreetingWorkflow)", "worker registers example workflow")
	assertNotContains(t, content, "// crank:activity-register", "worker no longer has activity marker")
	assertNotContains(t, content, "w.RegisterActivity", "worker no longer registers activities directly")
	assertContains(t, content, "config.TemporalConfig", "worker uses shared config type")
	assertContains(t, content, "func NewClient(", "worker file exposes NewClient")
	assertContains(t, content, "func NewWorker(", "worker file exposes NewWorker")

	// Activity registration is now in the Activities container.
	acts := readFile(t, dir, "internal/adapters/temporal/activity/activities.go")
	assertContains(t, acts, "// crank:activity-register", "activities has activity marker")
	assertContains(t, acts, "// crank:activity-fields", "activities has activity-fields marker")
	assertContains(t, acts, "w.RegisterActivity(Greet)", "activities registers example activity")
	assertContains(t, acts, "type Activities struct", "activities container")
	assertContains(t, acts, "func (a *Activities) Register", "activities Register method")
}

func TestTemporal_Config_HasTemporalSection(t *testing.T) {
	r := generateProject(t, "temporalcfg", []string{"temporal"})
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "temporal:", "config.yaml temporal section")
	assertContains(t, content, "host_port:", "config.yaml temporal host_port key")
	assertContains(t, content, "namespace:", "config.yaml temporal namespace key")
}

func TestTemporal_ConfigGo_HasTemporalConfig(t *testing.T) {
	r := generateProject(t, "temporalcfggo", []string{"temporal"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "Temporal TemporalConfig", "Config struct has Temporal field")
	assertContains(t, content, "TemporalConfig struct", "TemporalConfig struct defined")
	assertContains(t, content, "HostPort", "TemporalConfig has HostPort field")
	assertContains(t, content, "TaskQueue", "TemporalConfig has TaskQueue field")
}

func TestTemporal_ConfigGo_NoTemporalConfig(t *testing.T) {
	r := generateProject(t, "notemporalcfggo", nil)
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertNotContains(t, content, "TemporalConfig", "config.go without temporal has no TemporalConfig")
}

func TestTemporal_Logger_BridgesSlog(t *testing.T) {
	r := generateProject(t, "temporallogger", []string{"temporal"})
	content := readFile(t, r.ProjectDir, "internal/adapters/temporal/logger.go")
	// In the DDD layout the temporal logger is an unexported slogAdapter
	// type that satisfies the SDK's log.Logger interface implicitly, so
	// there is no separate constructor.
	assertContains(t, content, "type slogAdapter struct", "logger adapter type")
	assertContains(t, content, "logger *slog.Logger", "logger holds slog logger")
	assertContains(t, content, "func (a slogAdapter) Debug(", "adapter implements Debug")
	assertContains(t, content, "func (a slogAdapter) Info(", "adapter implements Info")
}

func TestTemporal_Client_UsesDial(t *testing.T) {
	// NewClient lives next to NewWorker in the DDD layout.
	r := generateProject(t, "temporalclient", []string{"temporal"})
	content := readFile(t, r.ProjectDir, "internal/adapters/temporal/worker.go")
	assertContains(t, content, "client.Dial(client.Options{", "client dials temporal")
	assertContains(t, content, "config.TemporalConfig", "client uses shared config type")
	assertContains(t, content, "Logger:    slogAdapter{logger: logger}", "client wires slog logger")
}

func TestTemporal_WorkerMain_UsesConfigAndLogging(t *testing.T) {
	r := generateProject(t, "temporalmain", []string{"temporal"})
	content := readFile(t, r.ProjectDir, "cmd/worker/main.go")
	assertContains(t, content, "/internal/config", "worker main imports config")
	assertContains(t, content, "/internal/adapters/temporal", "worker main imports temporal pkg")
	assertContains(t, content, "/pkg/logging", "worker main imports logging")
	assertContains(t, content, "cfg.Temporal", "worker uses shared temporal config")
	assertContains(t, content, "worker.InterruptCh()", "worker main runs until interrupt")
}

func TestTemporal_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "temporalmanifest", []string{"temporal"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- temporal", "manifest includes temporal")
}

// Temporal config now lives in the shared config package. A project without temporal
// has no TemporalConfig section and no worker entrypoint.
func TestTemporal_NotPresent_NoWorker(t *testing.T) {
	r := generateProject(t, "notemporal", []string{"base"})
	assertFileNotExists(t, r.ProjectDir, "cmd/worker/main.go")
	assertFileNotExists(t, r.ProjectDir, "internal/adapters/temporal/worker.go")
	assertDepsNotContains(t, r.Dependencies, "go.temporal.io/sdk", "base-only deps")
}

// ==========================================================================
// ALL features together
// ==========================================================================

func TestAll_Features(t *testing.T) {
	// bun and gorm are mutually exclusive, so the "all features" matrix uses
	// bun (the older ORM) and exercises every other feature.
	names := allFeatures(t)
	allButGorm := make([]string, 0, len(names))
	for _, n := range names {
		if n == "gorm" {
			continue
		}
		allButGorm = append(allButGorm, n)
	}
	r := generateProject(t, "allfeatures", allButGorm)
	dir := r.ProjectDir

	// base files
	assertFileExists(t, dir, "cmd/server/main.go")
	assertFileExists(t, dir, "go.mod")
	assertFileExists(t, dir, "Makefile")
	assertFileExists(t, dir, "Dockerfile")

	// validator files
	assertFileExists(t, dir, "internal/validator/validator.go")
	assertFileExists(t, dir, "internal/validator/errors.go")

	// crypto files
	assertFileExists(t, dir, "pkg/crypto/aesgcm_cipher.go")

	// auth files
	assertFileExists(t, dir, "internal/adapters/http/web/middleware/auth.go")
	assertFileExists(t, dir, "pkg/crypto/bcrypt_hasher.go")
	assertFileExists(t, dir, "internal/adapters/auth/jwt/token_service.go")
	assertFileExists(t, dir, "internal/adapters/http/web/auth_handler.go")

	// bun files
	assertFileExists(t, dir, "internal/adapters/persistence/bun/db.go")
	assertFileExists(t, dir, "internal/adapters/persistence/bun/migrate.go")
	assertFileExists(t, dir, "db/migrations/000001_init.up.sql")
	assertFileExists(t, dir, "db/migrations/000001_init.down.sql")

	// redis files
	assertFileExists(t, dir, "internal/adapters/cache/redis/client.go")

	// mongodb files
	assertFileExists(t, dir, "internal/adapters/persistence/mongodb/client.go")

	// qdrant files
	assertFileExists(t, dir, "internal/adapters/persistence/qdrant/client.go")

	// temporal files
	assertFileExists(t, dir, "internal/adapters/temporal/worker.go")
	assertFileExists(t, dir, "internal/adapters/temporal/workflow/greeting.go")
	assertFileExists(t, dir, "internal/adapters/temporal/activity/greeting.go")
	assertFileExists(t, dir, "cmd/worker/main.go")

	// audit files
	assertFileExists(t, dir, "internal/domain/audit/event.go")
	assertFileExists(t, dir, "internal/domain/audit/repository.go")
	assertFileExists(t, dir, "internal/ports/audit.go")
	assertFileExists(t, dir, "internal/adapters/audit/logger.go")
	assertFileExists(t, dir, "internal/application/audit/query_handler.go")
	assertFileExists(t, dir, "internal/adapters/http/web/v1/audit_handler.go")
	assertFileExists(t, dir, "db/migrations/000003_add_audit_events.up.sql")

	// Verify deps are returned via Result
	assertDepsContains(t, r.Dependencies, "golang-jwt", "all features deps")
	assertDepsContains(t, r.Dependencies, "uptrace/bun", "all features deps")
	assertDepsContains(t, r.Dependencies, "golang-migrate", "all features deps")
	assertDepsContains(t, r.Dependencies, "go-playground/validator", "all features deps")
	assertDepsContains(t, r.Dependencies, "go.temporal.io/sdk", "all features deps")

	// Verify config has all sections
	cfg := readFile(t, dir, "configs/config.yaml")
	assertContains(t, cfg, "database:", "config database")
	assertContains(t, cfg, "jwt:", "config jwt")
	assertContains(t, cfg, "crypto:", "config crypto")
	assertContains(t, cfg, "redis:", "config redis")
	assertContains(t, cfg, "mongodb:", "config mongodb")
	assertContains(t, cfg, "qdrant:", "config qdrant")
	assertContains(t, cfg, "temporal:", "config temporal")

	// Verify manifest has all features (gorm is excluded from this matrix —
	// see comment at the top of TestAll_Features).
	manifest := readFile(t, dir, ".crank.yaml")
	for _, name := range allButGorm {
		assertContains(t, manifest, "- "+name, "manifest "+name)
	}
}

// ==========================================================================
// ADD feature to existing project
// ==========================================================================

func TestAdd_AuthToBaseProject(t *testing.T) {
	tmp := t.TempDir()
	r, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "addauth",
		ModulePath:  "github.com/example/addauth",
		TargetDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Verify auth files don't exist yet
	assertFileNotExists(t, r.ProjectDir, "internal/adapters/http/web/middleware/auth.go")
	assertFileNotExists(t, r.ProjectDir, "pkg/crypto/bcrypt_hasher.go")

	// Add auth
	r2, err := bootstrap.Add(bootstrap.GlobalRegistry, r.ProjectDir, "auth")
	if err != nil {
		t.Fatalf("Add auth: %v", err)
	}

	// Now auth files should exist
	assertFileExists(t, r2.ProjectDir, "internal/adapters/http/web/middleware/auth.go")
	assertFileExists(t, r2.ProjectDir, "pkg/crypto/bcrypt_hasher.go")
	assertFileExists(t, r2.ProjectDir, "internal/adapters/http/web/auth_handler.go")

	// Manifest should include auth
	manifest := readFile(t, r2.ProjectDir, ".crank.yaml")
	assertContains(t, manifest, "- auth", "manifest after add auth")

	// go.mod should now include auth deps
	assertDepsContains(t, r2.Dependencies, "golang-jwt/jwt/v5", "add auth deps")
}

func TestAdd_BunToBaseProject(t *testing.T) {
	tmp := t.TempDir()
	r, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "addbun",
		ModulePath:  "github.com/example/addbun",
		TargetDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Add bun
	r2, err := bootstrap.Add(bootstrap.GlobalRegistry, r.ProjectDir, "bun")
	if err != nil {
		t.Fatalf("Add bun: %v", err)
	}

	// bun files should exist
	assertFileExists(t, r2.ProjectDir, "internal/adapters/persistence/bun/db.go")
	assertFileExists(t, r2.ProjectDir, "internal/adapters/persistence/bun/migrate.go")
	assertFileExists(t, r2.ProjectDir, "db/migrations/000001_init.up.sql")

	// Manifest should include bun
	manifest := readFile(t, r2.ProjectDir, ".crank.yaml")
	assertContains(t, manifest, "- bun", "manifest after add bun")

	// Config should be re-rendered with database section
	cfgGo := readFile(t, r2.ProjectDir, "internal/config/config.go")
	assertContains(t, cfgGo, "DatabaseConfig", "config.go has DatabaseConfig after add bun")
	assertNotContains(t, cfgGo, "JWTConfig", "config.go has no JWTConfig before adding auth")
}

func TestAdd_RedisToBaseProject_UpdatesConfig(t *testing.T) {
	tmp := t.TempDir()
	r, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "addredis",
		ModulePath:  "github.com/example/addredis",
		TargetDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Base project should not have Redis config
	cfgGo := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertNotContains(t, cfgGo, "RedisConfig", "base config has no RedisConfig")

	// Add redis
	r2, err := bootstrap.Add(bootstrap.GlobalRegistry, r.ProjectDir, "redis")
	if err != nil {
		t.Fatalf("Add redis: %v", err)
	}

	// Config should be re-rendered with Redis section
	cfgGo2 := readFile(t, r2.ProjectDir, "internal/config/config.go")
	assertContains(t, cfgGo2, "RedisConfig", "config.go has RedisConfig after add redis")
	assertContains(t, cfgGo2, `Redis RedisConfig`, "config.go has Redis field")

	// Config YAML should also have redis section
	yaml := readFile(t, r2.ProjectDir, "configs/config.yaml")
	assertContains(t, yaml, "redis:", "config.yaml has redis section")
}

func TestAdd_TemporalToBaseProject_UpdatesConfig(t *testing.T) {
	tmp := t.TempDir()
	r, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "addtemp",
		ModulePath:  "github.com/example/addtemp",
		TargetDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Base project should not have Temporal config
	cfgGo := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertNotContains(t, cfgGo, "TemporalConfig", "base config has no TemporalConfig")

	// Add temporal
	r2, err := bootstrap.Add(bootstrap.GlobalRegistry, r.ProjectDir, "temporal")
	if err != nil {
		t.Fatalf("Add temporal: %v", err)
	}

	// Config should be re-rendered with Temporal section
	cfgGo2 := readFile(t, r2.ProjectDir, "internal/config/config.go")
	assertContains(t, cfgGo2, "TemporalConfig", "config.go has TemporalConfig after add temporal")
	assertContains(t, cfgGo2, `Temporal TemporalConfig`, "config.go has Temporal field")

	// Config YAML should also have temporal section
	yaml := readFile(t, r2.ProjectDir, "configs/config.yaml")
	assertContains(t, yaml, "temporal:", "config.yaml has temporal section")

	// No standalone config.go should exist in temporal package
	assertFileNotExists(t, r2.ProjectDir, "internal/adapters/temporal/config.go")
}

// ==========================================================================
// PHASE 1: Logging ContextHandler
// ==========================================================================

func TestBase_Logging_ContextHandler(t *testing.T) {
	t.Run("ContextHandler injected in New()", func(t *testing.T) {
		project := generateProject(t, "logctx", nil).ProjectDir
		logger := readFile(t, project, "pkg/logging/logger.go")
		assertContains(t, logger, "ContextHandler", "logger.go has ContextHandler")
		assertContains(t, logger, "func (h *ContextHandler) Handle", "logger.go has Handle method")
		assertNotContains(t, logger, "func L(", "logger.go has no L(ctx)")
	})

	t.Run("no L(ctx) in middleware", func(t *testing.T) {
		project := generateProject(t, "logmw", nil).ProjectDir
		mw := readFile(t, project, "internal/adapters/http/web/middleware/logging.go")
		assertNotContains(t, mw, "logging.L(", "middleware has no logging.L")
		assertContains(t, mw, "slog.LogAttrs(ctx", "middleware uses slog.LogAttrs")
	})

	t.Run("no L(ctx) in main.go error handler", func(t *testing.T) {
		project := generateProject(t, "logmain", nil).ProjectDir
		main := readFile(t, project, "internal/adapters/http/web/server.go")
		assertContains(t, main, "logger.WarnContext", "server.go uses logger.WarnContext")
		assertContains(t, main, "logger.ErrorContext", "server.go uses logger.ErrorContext")
	})

	t.Run("ParseLevel exists", func(t *testing.T) {
		project := generateProject(t, "loglevel", nil).ProjectDir
		logger := readFile(t, project, "pkg/logging/logger.go")
		assertContains(t, logger, "func ParseLevel", "logger.go has ParseLevel")
	})

	t.Run("no parseLevel in main.go", func(t *testing.T) {
		project := generateProject(t, "lognoparse", nil).ProjectDir
		main := readFile(t, project, "cmd/server/main.go")
		assertNotContains(t, main, "func parseLevel", "main has no local parseLevel")
		assertContains(t, main, "logging.ParseLevel", "main uses logging.ParseLevel")
	})

	t.Run("no L(ctx) in user_handler.go", func(t *testing.T) {
		project := generateProject(t, "loguser", nil).ProjectDir
		handler := readFile(t, project, "internal/adapters/http/web/v1/user_handler.go")
		assertNotContains(t, handler, "logging.L(", "user handler has no logging.L")
		assertNotContains(t, handler, "pkg/logging", "user handler imports no logging")
		assertContains(t, handler, "c.Request().Context()", "user handler uses c.Request().Context()")
	})
}

// ==========================================================================
// PHASE 2: UoW log publish errors
// ==========================================================================

func TestBase_UoW_LogsPublishErrors(t *testing.T) {
	project := generateProject(t, "uowlog", nil).ProjectDir
	uow := readFile(t, project, "internal/adapters/uow/in_memory_uow.go")
	assertContains(t, uow, "slog.ErrorContext", "uow logs publish errors")
	assertContains(t, uow, "failed to publish domain events", "uow has error message")
	assertNotContains(t, uow, "_ = u.bus.Publish", "uow does not silently discard")
}

// ==========================================================================
// PHASE 3: Two-process architecture
// ==========================================================================

func TestTemporal_TwoProcess_WorkerSignature(t *testing.T) {
	project := generateProject(t, "twoproc", []string{"base", "gorm", "temporal"}).ProjectDir

	worker := readFile(t, project, "internal/adapters/temporal/worker.go")
	assertContains(t, worker, "acts *activity.Activities", "worker has acts parameter")
	assertContains(t, worker, "if acts != nil", "worker checks acts for nil")
	assertContains(t, worker, "acts.Register(w)", "worker calls acts.Register")
	assertNotContains(t, worker, "func registerActivities", "worker has no registerActivities func")

	activitiesFile := readFile(t, project, "internal/adapters/temporal/activity/activities.go")
	assertContains(t, activitiesFile, "type Activities struct", "activities has struct")
	assertContains(t, activitiesFile, "func (a *Activities) Register", "activities has Register method")
	assertContains(t, activitiesFile, "// crank:activity-register", "activities has activity-register marker")
	assertContains(t, activitiesFile, "// crank:activity-fields", "activities has activity-fields marker")

	workerMain := readFile(t, project, "cmd/worker/main.go")
	assertContains(t, workerMain, "activity.NewActivities()", "worker main creates activities")
	assertContains(t, workerMain, "temporal.NewWorker(c, cfg.Temporal, acts)", "worker main passes acts")
	assertNotContains(t, workerMain, "func parseLevel", "worker main has no local parseLevel")
	assertContains(t, workerMain, "logging.ParseLevel", "worker main uses logging.ParseLevel")

	serverMain := readFile(t, project, "cmd/server/main.go")
	assertContains(t, serverMain, "temporal.NewWorker(tc, cfg.Temporal, nil)", "server passes nil")
}

func TestTemporal_DockerfileWorker(t *testing.T) {
	project := generateProject(t, "dockworker", []string{"base", "gorm", "temporal"}).ProjectDir
	assertFileExists(t, project, "Dockerfile.worker")
	dockerfile := readFile(t, project, "Dockerfile.worker")
	assertContains(t, dockerfile, "go build -o /out/worker ./cmd/worker", "worker dockerfile builds worker")
}

// ==========================================================================
// PHASE 4: Platform client pattern
// ==========================================================================

// ==========================================================================
// PHASE 5: Audit trail feature
// ==========================================================================

func TestAudit_FilesExist(t *testing.T) {
	project := generateProject(t, "audit", []string{"base", "gorm", "audit"}).ProjectDir

	for _, rel := range []string{
		"internal/domain/audit/event.go",
		"internal/domain/audit/repository.go",
		"internal/ports/audit.go",
		"internal/adapters/persistence/gorm/audit_repository.go",
		"internal/adapters/audit/logger.go",
		"internal/application/audit/query_handler.go",
		"internal/adapters/http/web/v1/audit_handler.go",
		"db/migrations/000003_add_audit_events.up.sql",
		"db/migrations/000003_add_audit_events.down.sql",
	} {
		assertFileExists(t, project, rel)
	}
}

func TestAudit_BunRepository(t *testing.T) {
	project := generateProject(t, "auditbun", []string{"base", "bun", "audit"}).ProjectDir
	assertFileExists(t, project, "internal/adapters/persistence/bun/audit_repository.go")
}

func TestAudit_NoORM_Fails(t *testing.T) {
	_, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "auditnoorm",
		ModulePath:  "github.com/example/auditnoorm",
		TargetDir:   t.TempDir(),
		Features:    []string{"base", "audit"},
	})
	if err == nil {
		t.Fatal("expected Generate to refuse audit without an ORM")
	}
	if !strings.Contains(err.Error(), "requires a database ORM") {
		t.Errorf("expected ORM requirement error, got: %v", err)
	}
}

func TestAudit_LoggerSubscribes(t *testing.T) {
	project := generateProject(t, "auditlog", []string{"base", "gorm", "audit"}).ProjectDir
	logger := readFile(t, project, "internal/adapters/audit/logger.go")
	assertContains(t, logger, "func (l *Logger) Subscribe", "audit logger has Subscribe")
	assertContains(t, logger, "bus.Subscribe", "audit logger subscribes to bus")
}

func TestAudit_MigrationContent(t *testing.T) {
	project := generateProject(t, "auditmig", []string{"base", "gorm", "audit"}).ProjectDir
	up := readFile(t, project, "db/migrations/000003_add_audit_events.up.sql")
	assertContains(t, up, "CREATE TABLE IF NOT EXISTS audit_events", "audit migration creates table")
	assertContains(t, up, "entity_type", "audit migration has entity_type")
	assertContains(t, up, "entity_id", "audit migration has entity_id")
	assertContains(t, up, "event_type", "audit migration has event_type")
	assertContains(t, up, "payload", "audit migration has payload")
	assertContains(t, up, "idx_audit_entity", "audit migration has index")
}

func TestAudit_MainWiring(t *testing.T) {
	project := generateProject(t, "auditwire", []string{"base", "gorm", "audit"}).ProjectDir
	main := readFile(t, project, "cmd/server/main.go")
	assertContains(t, main, "auditadapter", "main imports audit adapter")
	assertContains(t, main, "auditLogger.Subscribe(bus)", "main subscribes audit logger")
	assertContains(t, main, "NewAuditHandler", "main creates audit handler")
}

func TestAudit_OutboxCoexistence(t *testing.T) {
	project := generateProject(t, "auditoutbox", []string{"base", "gorm", "audit", "outbox"}).ProjectDir
	assertFileExists(t, project, "internal/adapters/audit/logger.go")
	assertFileExists(t, project, "internal/adapters/outbox/worker.go")
}
