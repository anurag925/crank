package migrate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/bootstrap/tools"
	"github.com/anurag925/crank/internal/utils"
)

func init() {
	bootstrap.GlobalToolRegistry.MustRegister(&tool{})
}

type tool struct {
	databaseURL string
	steps       int
}

func (*tool) Name() string               { return "migrate" }
func (*tool) BinaryName() string         { return "migrate" }
func (*tool) Description() string        { return "Run database migrations via golang-migrate" }
func (*tool) RequiresFeatures() []string { return []string{"bun", "gorm"} }

func (*tool) LongDescription() string {
	return `migrate invokes the golang-migrate CLI inside the target project.
By default it applies all pending up migrations. The database URL is taken
from DATABASE_URL env var or the project's configs/config.yaml.

If --project is not specified, the current directory is used.

Examples:
  crank migrate up --project ./myapp
  crank migrate down --steps 1 --project ./myapp
  crank migrate --project ./myapp              (defaults to "up")
  cd myapp && crank migrate up                 (uses current directory)`
}

func (*tool) InstallCmd() string {
	return "go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
}

func (t *tool) Install() error {
	return tools.InstallGoTool("github.com/golang-migrate/migrate/v4/cmd/migrate@latest", t.BinaryName(), "postgres")
}

func (t *tool) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&t.databaseURL, "database-url", "", "override the database URL (defaults to DATABASE_URL or config)")
	cmd.Flags().IntVar(&t.steps, "steps", 0, "limit the number of migration steps (0 = all pending)")
}

func (t *tool) Prepare(projectDir string, cmd *cobra.Command) (*bootstrap.ToolInvocation, error) {
	if !utils.PathExists(filepath.Join(projectDir, "migrations")) {
		return nil, fmt.Errorf("no migrations/ directory found in %s", projectDir)
	}

	databaseURL := t.databaseURL
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		dsn, err := dsnFromConfig(projectDir)
		if err != nil {
			return nil, fmt.Errorf("could not determine database URL: %w", err)
		}
		databaseURL = dsn
	}

	direction := "up"
	if args := cmd.Flags().Args(); len(args) > 0 {
		d := strings.ToLower(args[0])
		if d == "up" || d == "down" {
			direction = d
		}
	}

	argv := []string{
		"-path", filepath.Join(projectDir, "migrations"),
		"-database", databaseURL,
		direction,
	}
	if t.steps > 0 {
		argv = append(argv, fmt.Sprintf("%d", t.steps))
	}

	return &bootstrap.ToolInvocation{
		Args: argv,
		Dir:  projectDir,
	}, nil
}

func dsnFromConfig(projectDir string) (string, error) {
	yamlPath := filepath.Join(projectDir, "configs", "config.yaml")
	if !utils.PathExists(yamlPath) {
		yamlPath = filepath.Join(projectDir, "config.yaml")
	}
	if !utils.PathExists(yamlPath) {
		return "", fmt.Errorf("config.yaml not found in %s/configs/ or %s/ and DATABASE_URL is unset", projectDir, projectDir)
	}
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		return "", err
	}
	db := map[string]string{}
	lines := bytes.Split(raw, []byte("\n"))
	inDB := false
	for _, line := range lines {
		s := bytes.TrimSpace(line)
		if bytes.HasPrefix(s, []byte("#")) {
			continue
		}
		if bytes.HasSuffix(s, []byte("database:")) {
			inDB = true
			continue
		}
		if inDB {
			if len(s) == 0 || (!bytes.HasPrefix(line, []byte(" ")) && !bytes.HasPrefix(line, []byte("\t"))) {
				inDB = false
				continue
			}
			parts := bytes.SplitN(s, []byte(":"), 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(string(parts[0]))
			val := strings.Trim(strings.TrimSpace(string(parts[1])), `"'`)
			db[key] = val
		}
	}
	host := pick(db, "host", "localhost")
	port := pick(db, "port", "5432")
	user := pick(db, "user", "postgres")
	pass := pick(db, "password", "postgres")
	name := pick(db, "name", filepath.Base(projectDir))
	mode := pick(db, "sslmode", "disable")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, name, mode), nil
}

func pick(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}
