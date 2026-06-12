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
	_ "github.com/anurag925/crank/internal/bootstrap/features/crypto"
	_ "github.com/anurag925/crank/internal/bootstrap/features/mongodb"
	_ "github.com/anurag925/crank/internal/bootstrap/features/postgres"
	_ "github.com/anurag925/crank/internal/bootstrap/features/redis"
	_ "github.com/anurag925/crank/internal/bootstrap/features/temporal"
)

// allFeatures returns every feature name registered in the global registry.
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
		"postgres": true,
		"redis":    true,
		"mongodb":  true,
		"temporal": true,
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
		"internal/handler/handler.go",
		"internal/handler/user.go",
		"internal/middleware/logging.go",
		"internal/validator/validator.go",
		"internal/validator/errors.go",
		"internal/model/user.go",
		"internal/repository/user.go",
		"internal/service/user.go",
		"configs/config.yaml",
		".env.example",
		"Makefile",
		".air.toml",
		"Dockerfile",
		".gitignore",
		"go.mod",
		"README.md",
		".crank.yaml",
	}
	for _, f := range expectedFiles {
		assertFileExists(t, dir, f)
	}
}

func TestBase_GoMod_ModulePath(t *testing.T) {
	r := generateProject(t, "mymod", nil)
	content := readFile(t, r.ProjectDir, "go.mod")
	assertContains(t, content, "module github.com/example/mymod", "go.mod")
	assertContains(t, content, "go 1.25", "go.mod")

	// Dependencies are now returned via Result and installed via go get.
	assertDepsContains(t, r.Dependencies, "github.com/labstack/echo/v4", "base deps")
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

func TestBase_GoMod_NoPostgresDeps(t *testing.T) {
	r := generateProject(t, "nopg", nil)
	content := readFile(t, r.ProjectDir, "go.mod")
	assertNotContains(t, content, "uptrace/bun", "go.mod without postgres")
	assertNotContains(t, content, "golang-migrate", "go.mod without postgres")

	assertDepsNotContains(t, r.Dependencies, "uptrace/bun", "base has no bun dep")
	assertDepsNotContains(t, r.Dependencies, "golang-migrate/migrate/v4", "base has no migrate dep")
}

func TestBase_Config_ProjectName(t *testing.T) {
	r := generateProject(t, "cfgtest", nil)
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, `"cfgtest"`, "config.yaml project name")
	assertNotContains(t, content, "database:", "config.yaml without postgres")
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
	assertNotContains(t, content, "database.NewPostgres", "main.go no db init")
}

func TestBase_MainGo_NoAuthImport(t *testing.T) {
	r := generateProject(t, "noauthmain", nil)
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertNotContains(t, content, "service.NewJWTService", "main.go no jwt init")
}

func TestBase_HandlerHandler_NoDeps(t *testing.T) {
	r := generateProject(t, "nodeps", nil)
	content := readFile(t, r.ProjectDir, "internal/handler/handler.go")
	assertNotContains(t, content, "*bun.DB", "handler.go no bun dep")
	assertNotContains(t, content, "JWTService", "handler.go no jwt dep")
	assertContains(t, content, "validator", "handler.go always has validator")
}

func TestBase_HandlerUser_UsesService(t *testing.T) {
	r := generateProject(t, "usesvc", nil)
	content := readFile(t, r.ProjectDir, "internal/handler/user.go")
	assertContains(t, content, "svc *service.UserService", "user handler uses service")
	assertContains(t, content, "h.svc.List()", "user handler calls svc.List")
}

func TestBase_UserModel_NoPassword(t *testing.T) {
	r := generateProject(t, "nopw", nil)
	content := readFile(t, r.ProjectDir, "internal/model/user.go")
	assertNotContains(t, content, "Password", "user model without auth has no password")
}

func TestBase_MiddlewareLogging(t *testing.T) {
	r := generateProject(t, "mwlog", nil)
	content := readFile(t, r.ProjectDir, "internal/middleware/logging.go")
	assertContains(t, content, "func RequestLogger()", "logging middleware")
	assertContains(t, content, "log/slog", "logging middleware imports slog")
	assertContains(t, content, "slog.LevelInfo", "logging middleware uses slog")
}

func TestBase_MainGo_HasValidatorImport(t *testing.T) {
	r := generateProject(t, "valmain", nil)
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, content, `"github.com/example/valmain/internal/validator"`, "main.go validator import")
	assertContains(t, content, "validator.ValidationError", "main.go checks ValidationError")
	assertContains(t, content, "model.APIError", "main.go returns APIError responses")
	assertContains(t, content, "HTTPErrorHandler", "main.go sets custom error handler")
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
	assertNotContains(t, content, "data/", "gitignore without postgres has no data/")
}

func TestBase_Makefile_NoMigrate(t *testing.T) {
	r := generateProject(t, "makenopg", nil)
	content := readFile(t, r.ProjectDir, "Makefile")
	assertNotContains(t, content, "migrate-up", "Makefile without postgres")
	assertNotContains(t, content, "migrate-down", "Makefile without postgres")
}

func TestBase_Readme_NoPostgres(t *testing.T) {
	r := generateProject(t, "readmenopg", nil)
	content := readFile(t, r.ProjectDir, "README.md")
	assertContains(t, content, "# readmenopg", "README title")
	assertNotContains(t, content, "Bun", "README without postgres")
}

func TestBase_RepositoryUser_InMemory(t *testing.T) {
	r := generateProject(t, "repomem", nil)
	content := readFile(t, r.ProjectDir, "internal/repository/user.go")
	assertContains(t, content, "sync.RWMutex", "in-memory repo uses mutex")
	assertContains(t, content, "nextID int64", "in-memory repo has auto-increment")
}

func TestBase_ServiceUser_InMemory(t *testing.T) {
	r := generateProject(t, "svcmem", nil)
	content := readFile(t, r.ProjectDir, "internal/service/user.go")
	assertContains(t, content, "UserService", "in-memory service")
	assertContains(t, content, "sync.RWMutex", "in-memory service uses mutex")
}

// ==========================================================================
// AUTH feature
// ==========================================================================

func TestAuth_FilesExist(t *testing.T) {
	r := generateProject(t, "authtest", []string{"auth"})
	dir := r.ProjectDir

	expectedFiles := []string{
		"internal/middleware/auth.go",
		"internal/service/auth.go",
		"internal/handler/auth.go",
		"internal/model/user.go",
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
	content := readFile(t, r.ProjectDir, "internal/middleware/auth.go")
	assertContains(t, content, "func JWTAuth(", "auth middleware function")
	assertContains(t, content, "Bearer", "auth middleware checks bearer")
	assertContains(t, content, "jwt.Parse", "auth middleware parses jwt")
}

func TestAuth_ServiceAuth(t *testing.T) {
	r := generateProject(t, "authsvc", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/service/auth.go")
	assertContains(t, content, "JWTService", "auth service struct")
	assertContains(t, content, "TokenPair", "auth service TokenPair")
	assertContains(t, content, "HashPassword", "auth service HashPassword")
	assertContains(t, content, "bcrypt", "auth service uses bcrypt")
}

func TestAuth_HandlerAuth(t *testing.T) {
	r := generateProject(t, "authhandler", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/handler/auth.go")
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
	r := generateProject(t, "authmodel", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/model/user.go")
	assertContains(t, content, "Password", "user model with auth has password")
	assertContains(t, content, `json:"-"`, "password not in JSON")
}

func TestAuth_MainGo_ImportsService(t *testing.T) {
	r := generateProject(t, "authmain", []string{"auth"})
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, content, "service.NewJWTService", "main.go creates JWTService")
	assertContains(t, content, "handler.NewAuthHandler", "main.go creates AuthHandler")
	assertContains(t, content, "validator", "main.go still imports validator")
}

func TestAuth_HandlerHandler_HasJWTDep(t *testing.T) {
	r := generateProject(t, "authdeps", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/handler/handler.go")
	assertContains(t, content, "JWT *service.JWTService", "handler.go has JWT dep")
}

func TestAuth_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "authmanifest", []string{"auth"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- auth", "manifest includes auth")
}

// ==========================================================================
// POSTGRES feature
// ==========================================================================

func TestPostgres_FilesExist(t *testing.T) {
	r := generateProject(t, "pgtest", []string{"postgres"})
	dir := r.ProjectDir

	expectedFiles := []string{
		"internal/database/postgres.go",
		"internal/database/migrate.go",
		"internal/model/user.go",
		"internal/repository/user.go",
		"migrations/000001_init.up.sql",
		"migrations/000001_init.down.sql",
	}
	for _, f := range expectedFiles {
		assertFileExists(t, dir, f)
	}
}

func TestPostgres_GoMod_Deps(t *testing.T) {
	r := generateProject(t, "pggomod", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "go.mod")
	assertContains(t, content, "module", "go.mod")
	// Dependencies are returned via Result and installed via go get.
	assertDepsContains(t, r.Dependencies, "github.com/uptrace/bun", "postgres deps")
	assertDepsContains(t, r.Dependencies, "github.com/uptrace/bun/dialect/pgdialect", "postgres deps")
	assertDepsContains(t, r.Dependencies, "github.com/uptrace/bun/driver/pgdriver", "postgres deps")
	assertDepsContains(t, r.Dependencies, "github.com/golang-migrate/migrate/v4", "postgres deps")
}

func TestPostgres_Config_HasDatabaseSection(t *testing.T) {
	r := generateProject(t, "pgcfg", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, content, "database:", "config.yaml database section")
	assertContains(t, content, "host:", "config.yaml db host")
	assertContains(t, content, "port: 5432", "config.yaml db port")
	assertContains(t, content, "sslmode:", "config.yaml db sslmode")
}

func TestPostgres_ConfigGo_HasDatabaseConfig(t *testing.T) {
	r := generateProject(t, "pgcfggo", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "DatabaseConfig", "config.go DatabaseConfig struct")
	assertContains(t, content, "func (d DatabaseConfig) DSN()", "config.go DSN method")
}

func TestPostgres_DatabasePostgres(t *testing.T) {
	r := generateProject(t, "pgdb", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "internal/database/postgres.go")
	assertContains(t, content, "func NewPostgres(", "postgres.go NewPostgres function")
	assertContains(t, content, "pgdriver", "postgres.go uses pgdriver")
	assertContains(t, content, "bun.NewDB", "postgres.go creates bun.DB")
}

func TestPostgres_DatabaseMigrate(t *testing.T) {
	r := generateProject(t, "pgmigrate", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "internal/database/migrate.go")
	assertContains(t, content, "func MigrateUp(", "migrate.go MigrateUp function")
	assertContains(t, content, "func MigrateDown(", "migrate.go MigrateDown function")
	assertContains(t, content, "migrate.New", "migrate.go uses migrate.New")
}

func TestPostgres_MigrationUp(t *testing.T) {
	r := generateProject(t, "pgmigup", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "migrations/000001_init.up.sql")
	assertContains(t, content, "CREATE TABLE IF NOT EXISTS users", "migration up creates users table")
	assertContains(t, content, "BIGSERIAL PRIMARY KEY", "migration up has bigserial pk")
	assertContains(t, content, "email TEXT NOT NULL UNIQUE", "migration up email unique")
}

func TestPostgres_MigrationDown(t *testing.T) {
	r := generateProject(t, "pgmigdown", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "migrations/000001_init.down.sql")
	assertContains(t, content, "DROP TABLE IF EXISTS users", "migration down drops users")
}

func TestPostgres_UserModel_HasBunTags(t *testing.T) {
	r := generateProject(t, "pgmodel", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "internal/model/user.go")
	assertContains(t, content, "bun.BaseModel", "model has bun.BaseModel")
	assertContains(t, content, `bun:"table:users`, "model has bun table tag")
	assertContains(t, content, "CreatedAt time.Time", "model has CreatedAt")
	assertContains(t, content, "UpdatedAt time.Time", "model has UpdatedAt")
}

func TestPostgres_UserModel_NoAuth_NoPassword(t *testing.T) {
	r := generateProject(t, "pgmodelnopw", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "internal/model/user.go")
	assertNotContains(t, content, "Password", "pg model without auth has no password")
}

func TestPostgres_RepositoryUser_BunBacked(t *testing.T) {
	r := generateProject(t, "pgrepo", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "internal/repository/user.go")
	assertContains(t, content, "*bun.DB", "repo uses bun.DB")
	assertContains(t, content, "func NewUserRepository(db *bun.DB)", "repo constructor takes bun.DB")
	assertContains(t, content, "GetByEmail", "repo has GetByEmail method")
}

func TestPostgres_HandlerUser_UsesRepo(t *testing.T) {
	r := generateProject(t, "pghandler", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "internal/handler/user.go")
	assertContains(t, content, "repo *repository.UserRepository", "handler uses repo")
	assertContains(t, content, "h.repo.List(", "handler calls repo.List")
}

func TestPostgres_HandlerHandler_HasDBDep(t *testing.T) {
	r := generateProject(t, "pghandlerdep", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "internal/handler/handler.go")
	assertContains(t, content, "DB *bun.DB", "handler.go has DB dep")
}

func TestPostgres_MainGo_ImportsDatabase(t *testing.T) {
	r := generateProject(t, "pgmain", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, content, "database.NewPostgres", "main.go creates Postgres connection")
	assertContains(t, content, "db.Close()", "main.go defers db close")
}

func TestPostgres_Makefile_NoMigrateTargets(t *testing.T) {
	// Common commands (including migrate) are provided by the crank CLI, not
	// duplicated in the Makefile.
	r := generateProject(t, "pgmake", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "Makefile")
	assertNotContains(t, content, "migrate-up:", "Makefile no longer defines migrate-up target")
	assertNotContains(t, content, "migrate-down:", "Makefile no longer defines migrate-down target")
	assertNotContains(t, content, "migrate-create:", "Makefile no longer defines migrate-create target")
}

func TestPostgres_Gitignore_HasDataDir(t *testing.T) {
	r := generateProject(t, "pggitignore", []string{"postgres"})
	content := readFile(t, r.ProjectDir, ".gitignore")
	assertContains(t, content, "data/", "gitignore with postgres has data/")
}

func TestPostgres_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "pgmanifest", []string{"postgres"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- postgres", "manifest includes postgres")
}

func TestPostgres_Readme_HasDBInfo(t *testing.T) {
	r := generateProject(t, "pgreadme", []string{"postgres"})
	content := readFile(t, r.ProjectDir, "README.md")
	assertContains(t, content, "Bun", "README mentions Bun")
	assertContains(t, content, "golang-migrate", "README mentions migrate")
	assertContains(t, content, "crank migrate", "README documents the crank migrate command")
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
	r := generateProject(t, "valbinder", nil)
	content := readFile(t, r.ProjectDir, "internal/handler/handler.go")
	assertContains(t, content, "echoBinder", "custom echoBinder type")
	assertContains(t, content, "defaultBinder echo.Binder", "binder wraps default")
	assertContains(t, content, "validator.Struct(i)", "binder calls validator.Struct")
	assertContains(t, content, "e.Binder = &echoBinder", "Register installs custom binder")
}

func TestValidator_MainGo_ErrorHandler(t *testing.T) {
	r := generateProject(t, "valmaineh", nil)
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, content, "e.HTTPErrorHandler", "sets custom HTTP error handler")
	assertContains(t, content, `err.(*validator.ValidationError)`, "checks for ValidationError type")
	assertContains(t, content, `err.(*echo.HTTPError)`, "checks for Echo HTTPError type")
	assertContains(t, content, "internal server error", "fallback error message")
}

func TestValidator_UserHandler_CreateDelegatesToBind(t *testing.T) {
	r := generateProject(t, "valuserh", nil)
	content := readFile(t, r.ProjectDir, "internal/handler/user.go")
	// Create handler relies on the custom binder for validation
	assertContains(t, content, "c.Bind(&in)", "Create uses Bind (which validates)")
}

func TestValidator_AuthHandler_ValidationTags(t *testing.T) {
	r := generateProject(t, "valauthtags", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/handler/auth.go")
	assertContains(t, content, `validate:"required,email"`, "email validation tag")
	assertContains(t, content, `validate:"required,min=8"`, "password min 8 validation tag")
	assertContains(t, content, `validate:"omitempty,min=2,max=100"`, "name optional validation tag")
	assertContains(t, content, `validate:"required"`, "refresh_token required validation")
}

func TestValidator_AuthHandler_CredentialsStruct(t *testing.T) {
	r := generateProject(t, "valauthcred", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/handler/auth.go")
	assertContains(t, content, "type credentials struct", "credentials struct defined")
	assertContains(t, content, "Email", "credentials has Email")
	assertContains(t, content, "Password", "credentials has Password")
	assertContains(t, content, "Name", "credentials has Name")
}

func TestValidator_AuthHandler_RefreshStruct(t *testing.T) {
	r := generateProject(t, "valauthref", []string{"auth"})
	content := readFile(t, r.ProjectDir, "internal/handler/auth.go")
	assertContains(t, content, "type refreshRequest struct", "refreshRequest struct defined")
	assertContains(t, content, "RefreshToken", "refreshRequest has RefreshToken")
	assertContains(t, content, `json:"refresh_token"`, "refreshRequest has json tag")
}

// ==========================================================================
// AUTH + POSTGRES together
// ==========================================================================

func TestAuthPostgres_MigrationUp_HasPassword(t *testing.T) {
	r := generateProject(t, "authpg", []string{"auth", "postgres"})
	content := readFile(t, r.ProjectDir, "migrations/000001_init.up.sql")
	assertContains(t, content, "password TEXT NOT NULL", "migration has password column")
}

func TestAuthPostgres_UserModel_HasBunAndPassword(t *testing.T) {
	r := generateProject(t, "authpgmodel", []string{"auth", "postgres"})
	content := readFile(t, r.ProjectDir, "internal/model/user.go")
	assertContains(t, content, "bun.BaseModel", "model has bun tags")
	assertContains(t, content, "Password", "model has password")
	assertContains(t, content, `json:"-"`, "password hidden from JSON")
}

func TestAuthPostgres_MainGo_HasBothDeps(t *testing.T) {
	r := generateProject(t, "authpgmain", []string{"auth", "postgres"})
	content := readFile(t, r.ProjectDir, "cmd/server/main.go")
	assertContains(t, content, "database.NewPostgres", "main.go creates db")
	assertContains(t, content, "service.NewJWTService", "main.go creates jwt")
	assertContains(t, content, "handler.NewAuthHandler", "main.go creates auth handler")
}

func TestAuthPostgres_ConfigGo_HasBothConfigs(t *testing.T) {
	r := generateProject(t, "authpgcfg", []string{"auth", "postgres"})
	content := readFile(t, r.ProjectDir, "internal/config/config.go")
	assertContains(t, content, "DatabaseConfig", "config has DatabaseConfig")
	assertContains(t, content, "JWTConfig", "config has JWTConfig")
}

func TestAuthPostgres_GoMod_HasAllDeps(t *testing.T) {
	r := generateProject(t, "authpggomod", []string{"auth", "postgres"})
	content := readFile(t, r.ProjectDir, "go.mod")
	assertContains(t, content, "module", "go.mod")
	assertDepsContains(t, r.Dependencies, "golang-jwt/jwt/v5", "auth+pg deps")
	assertDepsContains(t, r.Dependencies, "golang.org/x/crypto", "auth+pg deps")
	assertDepsContains(t, r.Dependencies, "uptrace/bun", "auth+pg deps")
	assertDepsContains(t, r.Dependencies, "golang-migrate/migrate/v4", "auth+pg deps")
}

func TestAuthPostgres_BootstrapManifest(t *testing.T) {
	r := generateProject(t, "authpgmanifest", []string{"auth", "postgres"})
	content := readFile(t, r.ProjectDir, ".crank.yaml")
	assertContains(t, content, "- auth", "manifest has auth")
	assertContains(t, content, "- postgres", "manifest has postgres")
}

// ==========================================================================
// REDIS feature (placeholder)
// ==========================================================================

func TestRedis_FilesExist(t *testing.T) {
	r := generateProject(t, "redistest", []string{"redis"})
	assertFileExists(t, r.ProjectDir, "internal/redis/client.go")
}

func TestRedis_Client(t *testing.T) {
	r := generateProject(t, "redisclient", []string{"redis"})
	content := readFile(t, r.ProjectDir, "internal/redis/client.go")
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
	assertFileExists(t, r.ProjectDir, "internal/crypto/crypto.go")
}

func TestCrypto_CryptoGo_PackageAndType(t *testing.T) {
	r := generateProject(t, "cryptopkg", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "internal/crypto/crypto.go")
	assertContains(t, content, "package crypto", "crypto package")
	assertContains(t, content, "type Crypto struct", "Crypto struct")
	assertContains(t, content, "aead cipher.AEAD", "Crypto has AEAD field")
}

func TestCrypto_CryptoGo_NewFunction(t *testing.T) {
	r := generateProject(t, "cryptonew", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "internal/crypto/crypto.go")
	assertContains(t, content, "func New(secret string) (*Crypto, error)", "New function signature")
	assertContains(t, content, "secret must not be empty", "New rejects empty secret")
	assertContains(t, content, "sha256.Sum256", "New derives key via SHA-256")
	assertContains(t, content, "aes.NewCipher", "New creates AES cipher")
	assertContains(t, content, "cipher.NewGCM", "New creates GCM AEAD")
}

func TestCrypto_CryptoGo_EncryptFunction(t *testing.T) {
	r := generateProject(t, "cryptoenc", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "internal/crypto/crypto.go")
	assertContains(t, content, "func (c *Crypto) Encrypt(plaintext string) (string, error)", "Encrypt signature")
	assertContains(t, content, "c.aead.NonceSize()", "Encrypt uses nonce")
	assertContains(t, content, "rand.Reader", "Encrypt uses crypto/rand")
	assertContains(t, content, "c.aead.Seal", "Encrypt calls Seal")
	assertContains(t, content, "base64.RawURLEncoding.EncodeToString", "Encrypt encodes base64-url")
}

func TestCrypto_CryptoGo_DecryptFunction(t *testing.T) {
	r := generateProject(t, "cryptodec", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "internal/crypto/crypto.go")
	assertContains(t, content, "func (c *Crypto) Decrypt(encoded string) (string, error)", "Decrypt signature")
	assertContains(t, content, "base64.RawURLEncoding.DecodeString", "Decrypt decodes base64-url")
	assertContains(t, content, "ciphertext too short", "Decrypt rejects short ciphertext")
	assertContains(t, content, "c.aead.Open", "Decrypt calls Open")
}

func TestCrypto_CryptoGo_Imports(t *testing.T) {
	r := generateProject(t, "cryptoimports", []string{"crypto"})
	content := readFile(t, r.ProjectDir, "internal/crypto/crypto.go")
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

func TestCrypto_PostgresCombined_AllSections(t *testing.T) {
	r := generateProject(t, "cryptopg", []string{"crypto", "postgres"})
	cfg := readFile(t, r.ProjectDir, "configs/config.yaml")
	assertContains(t, cfg, "crypto:", "config has crypto section")
	assertContains(t, cfg, "database:", "config has database section")
	assertFileExists(t, r.ProjectDir, "internal/crypto/crypto.go")
	assertFileExists(t, r.ProjectDir, "internal/database/postgres.go")
}

// ==========================================================================
// MONGODB feature (placeholder)
// ==========================================================================

func TestMongodb_FilesExist(t *testing.T) {
	r := generateProject(t, "mongotest", []string{"mongodb"})
	assertFileExists(t, r.ProjectDir, "internal/mongo/client.go")
}

func TestMongodb_Client(t *testing.T) {
	r := generateProject(t, "mongoclient", []string{"mongodb"})
	content := readFile(t, r.ProjectDir, "internal/mongo/client.go")
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
// TEMPORAL feature
// ==========================================================================

func TestTemporal_FilesExist(t *testing.T) {
	r := generateProject(t, "temporaltest", []string{"temporal"})
	for _, rel := range []string{
		"internal/temporal/client.go",
		"internal/temporal/logger.go",
		"internal/temporal/worker.go",
		"internal/workflow/greeting.go",
		"internal/activity/greeting.go",
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
	content := readFile(t, r.ProjectDir, "internal/temporal/worker.go")
	assertContains(t, content, "// crank:workflow-register", "worker workflow marker")
	assertContains(t, content, "// crank:activity-register", "worker activity marker")
	assertContains(t, content, "w.RegisterWorkflow(workflow.GreetingWorkflow)", "worker registers example workflow")
	assertContains(t, content, "w.RegisterActivity(activity.Greet)", "worker registers example activity")
	assertContains(t, content, "config.TemporalConfig", "worker uses shared config type")
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
	content := readFile(t, r.ProjectDir, "internal/temporal/logger.go")
	assertContains(t, content, "go.temporal.io/sdk/log", "logger imports temporal log")
	assertContains(t, content, "func NewLogger(logger *slog.Logger) log.Logger", "logger adapter constructor")
}

func TestTemporal_Client_UsesDial(t *testing.T) {
	r := generateProject(t, "temporalclient", []string{"temporal"})
	content := readFile(t, r.ProjectDir, "internal/temporal/client.go")
	assertContains(t, content, "client.Dial(client.Options{", "client dials temporal")
	assertContains(t, content, "config.TemporalConfig", "client uses shared config type")
	assertContains(t, content, "Logger:    NewLogger(logger)", "client wires slog logger")
}

func TestTemporal_WorkerMain_UsesConfigAndLogging(t *testing.T) {
	r := generateProject(t, "temporalmain", []string{"temporal"})
	content := readFile(t, r.ProjectDir, "cmd/worker/main.go")
	assertContains(t, content, "/internal/config", "worker main imports config")
	assertContains(t, content, "/internal/temporal", "worker main imports temporal pkg")
	assertContains(t, content, "/internal/logging", "worker main imports logging")
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
	assertFileNotExists(t, r.ProjectDir, "internal/temporal/worker.go")
	assertDepsNotContains(t, r.Dependencies, "go.temporal.io/sdk", "base-only deps")
}

// ==========================================================================
// ALL features together
// ==========================================================================

func TestAll_Features(t *testing.T) {
	names := allFeatures(t)
	r := generateProject(t, "allfeatures", names)
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
	assertFileExists(t, dir, "internal/crypto/crypto.go")

	// auth files
	assertFileExists(t, dir, "internal/middleware/auth.go")
	assertFileExists(t, dir, "internal/service/auth.go")
	assertFileExists(t, dir, "internal/handler/auth.go")

	// postgres files
	assertFileExists(t, dir, "internal/database/postgres.go")
	assertFileExists(t, dir, "internal/database/migrate.go")
	assertFileExists(t, dir, "migrations/000001_init.up.sql")
	assertFileExists(t, dir, "migrations/000001_init.down.sql")

	// redis files
	assertFileExists(t, dir, "internal/redis/client.go")

	// mongodb files
	assertFileExists(t, dir, "internal/mongo/client.go")

	// temporal files
	assertFileExists(t, dir, "internal/temporal/worker.go")
	assertFileExists(t, dir, "internal/workflow/greeting.go")
	assertFileExists(t, dir, "internal/activity/greeting.go")
	assertFileExists(t, dir, "cmd/worker/main.go")

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
	assertContains(t, cfg, "temporal:", "config temporal")

	// Verify manifest has all features
	manifest := readFile(t, dir, ".crank.yaml")
	for _, name := range names {
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
	assertFileNotExists(t, r.ProjectDir, "internal/middleware/auth.go")
	assertFileNotExists(t, r.ProjectDir, "internal/service/auth.go")

	// Add auth
	r2, err := bootstrap.Add(bootstrap.GlobalRegistry, r.ProjectDir, "auth")
	if err != nil {
		t.Fatalf("Add auth: %v", err)
	}

	// Now auth files should exist
	assertFileExists(t, r2.ProjectDir, "internal/middleware/auth.go")
	assertFileExists(t, r2.ProjectDir, "internal/service/auth.go")
	assertFileExists(t, r2.ProjectDir, "internal/handler/auth.go")

	// Manifest should include auth
	manifest := readFile(t, r2.ProjectDir, ".crank.yaml")
	assertContains(t, manifest, "- auth", "manifest after add auth")

	// go.mod should now include auth deps
	assertDepsContains(t, r2.Dependencies, "golang-jwt/jwt/v5", "add auth deps")
}

func TestAdd_PostgresToBaseProject(t *testing.T) {
	tmp := t.TempDir()
	r, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "addpg",
		ModulePath:  "github.com/example/addpg",
		TargetDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Add postgres
	r2, err := bootstrap.Add(bootstrap.GlobalRegistry, r.ProjectDir, "postgres")
	if err != nil {
		t.Fatalf("Add postgres: %v", err)
	}

	// Postgres files should exist
	assertFileExists(t, r2.ProjectDir, "internal/database/postgres.go")
	assertFileExists(t, r2.ProjectDir, "internal/database/migrate.go")
	assertFileExists(t, r2.ProjectDir, "migrations/000001_init.up.sql")

	// Manifest should include postgres
	manifest := readFile(t, r2.ProjectDir, ".crank.yaml")
	assertContains(t, manifest, "- postgres", "manifest after add postgres")

	// Config should be re-rendered with postgres section
	cfgGo := readFile(t, r2.ProjectDir, "internal/config/config.go")
	assertContains(t, cfgGo, "DatabaseConfig", "config.go has DatabaseConfig after add postgres")
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
	assertFileNotExists(t, r2.ProjectDir, "internal/temporal/config.go")
}
