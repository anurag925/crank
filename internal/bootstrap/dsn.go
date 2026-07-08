package bootstrap

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anurag925/crank/internal/utils"
)

// DSNFromConfig reads the project's config.yaml and builds a postgres DSN
// from the database section. It searches configs/config.yaml first, then
// config.yaml in the project root. Returns an error if neither file exists
// and DATABASE_URL is expected to have been tried first by the caller.
func DSNFromConfig(projectDir string) (string, error) {
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
	host := Pick(db, "host", "localhost")
	port := Pick(db, "port", "5432")
	user := Pick(db, "user", "postgres")
	pass := Pick(db, "password", "postgres")
	name := Pick(db, "name", filepath.Base(projectDir))
	mode := Pick(db, "sslmode", "disable")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, name, mode), nil
}

// Pick returns the value for key from m if present and non-empty, otherwise def.
// It is a small helper used when reading optional config values.
func Pick(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}
