package bootstrap

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

// Marker comments embedded in generated config files. When crank add injects
// a new feature's config, it inserts text immediately before these anchors so
// existing custom content is preserved.
const (
	markerConfigFields   = "// crank:config-fields"
	markerConfigStructs  = "// crank:config-structs"
	markerConfigDefaults = "// crank:config-defaults"
	markerConfigYAML     = "# crank:config-section"
	markerEnvSection     = "# crank:env-section"
)

// configInjection holds the Go, YAML and env snippets for a single feature.
type configInjection struct {
	// Config struct field (e.g. "\tRedis RedisConfig `mapstructure:\"redis\"`\n")
	StructField string
	// Config struct definition (e.g. "// RedisConfig holds...\ntype RedisConfig struct {...}\n")
	StructDef string
	// setDefaults lines (e.g. "\tv.SetDefault(\"redis.addr\", \"localhost:6379\")\n")
	Defaults string
	// Additional Go imports needed (e.g. `"time"`)
	Imports []string
	// YAML section (e.g. "\nredis:\n  addr: ...\n")
	YAMLSection string
	// .env.example section (e.g. "\nREDIS_ADDR=localhost:6379\n")
	EnvSection string
}

// bt is a helper for writing Go struct tags in string literals.
// Go struct tags use backticks which cannot appear inside raw string literals,
// so we use the BT constant as a placeholder that we replace at runtime.
const bt = "`"

// featureConfigData returns the config injection snippets for a given feature.
// Each key maps to the Go struct field, struct definition, viper defaults,
// YAML section, and .env section that the feature contributes.
func featureConfigData(pkgName string) map[string]configInjection {
	return map[string]configInjection{
		"bun": {
			StructField: "\tDatabase DatabaseConfig " + bt + "mapstructure:\"database\"" + bt + "\n",
			StructDef: "// DatabaseConfig holds PostgreSQL connection settings.\n" +
				"type DatabaseConfig struct {\n" +
				"\tHost     string " + bt + "mapstructure:\"host\"" + bt + "\n" +
				"\tPort     int    " + bt + "mapstructure:\"port\"" + bt + "\n" +
				"\tUser     string " + bt + "mapstructure:\"user\"" + bt + "\n" +
				"\tPassword string " + bt + "mapstructure:\"password\"" + bt + "\n" +
				"\tName     string " + bt + "mapstructure:\"name\"" + bt + "\n" +
				"\tSSLMode  string " + bt + "mapstructure:\"sslmode\"" + bt + "\n" +
				"}\n\n" +
				"// DSN returns a libpq-style connection string.\n" +
				"func (d DatabaseConfig) DSN() string {\n" +
				"\treturn fmt.Sprintf(\"postgres://%s:%s@%s:%d/%s?sslmode=%s\",\n" +
				"\t\td.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)\n" +
				"}\n",
			Defaults:    "\tv.SetDefault(\"database.host\", \"localhost\")\n" + "\tv.SetDefault(\"database.port\", 5432)\n" + "\tv.SetDefault(\"database.user\", \"postgres\")\n" + "\tv.SetDefault(\"database.password\", \"postgres\")\n" + "\tv.SetDefault(\"database.name\", \"" + pkgName + "\")\n" + "\tv.SetDefault(\"database.sslmode\", \"disable\")\n",
			YAMLSection: "\ndatabase:\n  host: \"localhost\"\n  port: 5432\n  user: \"postgres\"\n  password: \"postgres\"\n  name: \"" + pkgName + "\"\n  sslmode: \"disable\"\n",
			EnvSection:  "\nDB_HOST=localhost\nDB_PORT=5432\nDB_USER=postgres\nDB_PASSWORD=postgres\nDB_NAME=" + pkgName + "\nDB_SSLMODE=disable\n",
		},
		"gorm": {
			StructField: "\tDatabase DatabaseConfig " + bt + "mapstructure:\"database\"" + bt + "\n",
			StructDef: "// DatabaseConfig holds PostgreSQL connection settings.\n" +
				"type DatabaseConfig struct {\n" +
				"\tHost     string " + bt + "mapstructure:\"host\"" + bt + "\n" +
				"\tPort     int    " + bt + "mapstructure:\"port\"" + bt + "\n" +
				"\tUser     string " + bt + "mapstructure:\"user\"" + bt + "\n" +
				"\tPassword string " + bt + "mapstructure:\"password\"" + bt + "\n" +
				"\tName     string " + bt + "mapstructure:\"name\"" + bt + "\n" +
				"\tSSLMode  string " + bt + "mapstructure:\"sslmode\"" + bt + "\n" +
				"}\n\n" +
				"// DSN returns a libpq-style connection string.\n" +
				"func (d DatabaseConfig) DSN() string {\n" +
				"\treturn fmt.Sprintf(\"postgres://%s:%s@%s:%d/%s?sslmode=%s\",\n" +
				"\t\td.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)\n" +
				"}\n",
			Defaults:    "\tv.SetDefault(\"database.host\", \"localhost\")\n" + "\tv.SetDefault(\"database.port\", 5432)\n" + "\tv.SetDefault(\"database.user\", \"postgres\")\n" + "\tv.SetDefault(\"database.password\", \"postgres\")\n" + "\tv.SetDefault(\"database.name\", \"" + pkgName + "\")\n" + "\tv.SetDefault(\"database.sslmode\", \"disable\")\n",
			YAMLSection: "\ndatabase:\n  host: \"localhost\"\n  port: 5432\n  user: \"postgres\"\n  password: \"postgres\"\n  name: \"" + pkgName + "\"\n  sslmode: \"disable\"\n",
			EnvSection:  "\nDB_HOST=localhost\nDB_PORT=5432\nDB_USER=postgres\nDB_PASSWORD=postgres\nDB_NAME=" + pkgName + "\nDB_SSLMODE=disable\n",
		},
		"auth": {
			StructField: "\tJWT JWTConfig " + bt + "mapstructure:\"jwt\"" + bt + "\n",
			StructDef: "// JWTConfig holds settings for JWT issuance and validation.\n" +
				"type JWTConfig struct {\n" +
				"\tSecret            string        " + bt + "mapstructure:\"secret\"" + bt + "\n" +
				"\tExpiration        time.Duration " + bt + "mapstructure:\"expiration\"" + bt + "\n" +
				"\tRefreshExpiration time.Duration " + bt + "mapstructure:\"refresh_expiration\"" + bt + "\n" +
				"}\n",
			Defaults:    "\tv.SetDefault(\"jwt.secret\", \"change-me-in-production\")\n" + "\tv.SetDefault(\"jwt.expiration\", \"24h\")\n" + "\tv.SetDefault(\"jwt.refresh_expiration\", \"168h\")\n",
			Imports:     []string{"\"time\""},
			YAMLSection: "\njwt:\n  secret: \"change-me-in-production\"\n  expiration: 24h\n  refresh_expiration: 168h\n",
			EnvSection:  "\nJWT_SECRET=change-me-in-production\nJWT_EXPIRATION=24h\nJWT_REFRESH_EXPIRATION=168h\n",
		},
		"crypto": {
			StructField: "\tCrypto CryptoConfig " + bt + "mapstructure:\"crypto\"" + bt + "\n",
			StructDef: "// CryptoConfig holds the encryption secret used by the crypto package.\n" +
				"// The secret is read from the CRYPTO_SECRET environment variable.\n" +
				"// Generate a strong secret with: openssl rand -base64 32\n//\n" +
				"// IMPORTANT: Never commit the real secret. Use .env for local development\n" +
				"// and your platform's secret manager in production.\n" +
				"type CryptoConfig struct {\n" +
				"\tSecret string " + bt + "mapstructure:\"secret\"" + bt + "\n" +
				"}\n",
			Defaults:    "\tv.SetDefault(\"crypto.secret\", \"change-me-in-production\")\n",
			YAMLSection: "\ncrypto:\n  # Generate a strong secret with: openssl rand -base64 32\n  secret: \"change-me-in-production\"\n",
			EnvSection:  "\n# Generate a strong secret with: openssl rand -base64 32\nCRYPTO_SECRET=change-me-in-production\n",
		},
		"redis": {
			StructField: "\tRedis RedisConfig " + bt + "mapstructure:\"redis\"" + bt + "\n",
			StructDef: "// RedisConfig holds the Redis connection settings.\n" +
				"type RedisConfig struct {\n" +
				"\tAddr     string " + bt + "mapstructure:\"addr\"" + bt + "\n" +
				"\tPassword string " + bt + "mapstructure:\"password\"" + bt + "\n" +
				"\tDB       int    " + bt + "mapstructure:\"db\"" + bt + "\n" +
				"}\n",
			Defaults:    "\tv.SetDefault(\"redis.addr\", \"localhost:6379\")\n" + "\tv.SetDefault(\"redis.password\", \"\")\n" + "\tv.SetDefault(\"redis.db\", 0)\n",
			YAMLSection: "\nredis:\n  addr: \"localhost:6379\"\n  password: \"\"\n  db: 0\n",
			EnvSection:  "\nREDIS_ADDR=localhost:6379\nREDIS_PASSWORD=\nREDIS_DB=0\n",
		},
		"mongodb": {
			StructField: "\tMongoDB MongoDBConfig " + bt + "mapstructure:\"mongodb\"" + bt + "\n",
			StructDef: "// MongoDBConfig holds the MongoDB connection settings.\n" +
				"type MongoDBConfig struct {\n" +
				"\tURI      string " + bt + "mapstructure:\"uri\"" + bt + "\n" +
				"\tDatabase string " + bt + "mapstructure:\"database\"" + bt + "\n" +
				"}\n",
			Defaults:    "\tv.SetDefault(\"mongodb.uri\", \"mongodb://localhost:27017\")\n" + "\tv.SetDefault(\"mongodb.database\", \"" + pkgName + "\")\n",
			YAMLSection: "\nmongodb:\n  uri: \"mongodb://localhost:27017\"\n  database: \"" + pkgName + "\"\n",
			EnvSection:  "\nMONGODB_URI=mongodb://localhost:27017\nMONGODB_DATABASE=" + pkgName + "\n",
		},
		"temporal": {
			StructField: "\tTemporal TemporalConfig " + bt + "mapstructure:\"temporal\"" + bt + "\n",
			StructDef: "// TemporalConfig holds the Temporal client and worker settings.\n" +
				"type TemporalConfig struct {\n" +
				"\tHostPort  string " + bt + "mapstructure:\"host_port\"" + bt + "\n" +
				"\tNamespace string " + bt + "mapstructure:\"namespace\"" + bt + "\n" +
				"\tTaskQueue string " + bt + "mapstructure:\"task_queue\"" + bt + "\n" +
				"}\n",
			Defaults:    "\tv.SetDefault(\"temporal.host_port\", \"127.0.0.1:7233\")\n" + "\tv.SetDefault(\"temporal.namespace\", \"default\")\n" + "\tv.SetDefault(\"temporal.task_queue\", \"" + pkgName + "-task-queue\")\n",
			YAMLSection: "\ntemporal:\n  host_port: \"127.0.0.1:7233\"\n  namespace: \"default\"\n  task_queue: \"" + pkgName + "-task-queue\"\n",
			EnvSection:  "\nTEMPORAL_HOST_PORT=127.0.0.1:7233\nTEMPORAL_NAMESPACE=default\nTEMPORAL_TASK_QUEUE=" + pkgName + "-task-queue\n",
		},
		"otel": {
			StructField: "\tTelemetry TelemetryConfig " + bt + "mapstructure:\"telemetry\"" + bt + "\n",
			StructDef: "// TelemetryConfig holds the OpenTelemetry settings.\n" +
				"type TelemetryConfig struct {\n" +
				"\tServiceName string " + bt + "mapstructure:\"service_name\"" + bt + "\n" +
				"\tExporter    string " + bt + "mapstructure:\"exporter\"" + bt + "\n" +
				"}\n",
			Defaults:    "\tv.SetDefault(\"telemetry.service_name\", \"" + pkgName + "\")\n" + "\tv.SetDefault(\"telemetry.exporter\", \"stdout\")\n",
			YAMLSection: "\ntelemetry:\n  service_name: \"" + pkgName + "\"\n  exporter: \"stdout\"\n",
			EnvSection:  "\nTELEMETRY_SERVICE_NAME=" + pkgName + "\nTELEMETRY_EXPORTER=stdout\n",
		},
		"outbox": {
			StructField: "\tOutbox OutboxConfig " + bt + "mapstructure:\"outbox\"" + bt + "\n",
			StructDef: "// OutboxConfig holds the transactional outbox worker settings.\n" +
				"type OutboxConfig struct {\n" +
				"\tPollInterval time.Duration " + bt + "mapstructure:\"poll_interval\"" + bt + "\n" +
				"\tBatchSize    int           " + bt + "mapstructure:\"batch_size\"" + bt + "\n" +
				"}\n",
			Defaults:    "\tv.SetDefault(\"outbox.poll_interval\", \"1s\")\n" + "\tv.SetDefault(\"outbox.batch_size\", 100)\n",
			Imports:     []string{"\"time\""},
			YAMLSection: "\noutbox:\n  poll_interval: 1s\n  batch_size: 100\n",
			EnvSection:  "\nOUTBOX_POLL_INTERVAL=1s\nOUTBOX_BATCH_SIZE=100\n",
		},
	}
}

// injectConfig adds a feature's configuration sections (struct field, struct
// definition, defaults, YAML section, env vars) to an existing project's config
// files using marker-based injection. It never overwrites the entire file — only
// the new snippets are inserted at the marker anchors. If a snippet is already
// present (idempotency check), it is skipped.
func injectConfig(projectDir, featureName, pkgName string) ([]string, error) {
	data, ok := featureConfigData(pkgName)[featureName]
	if !ok {
		return nil, nil // feature has no config injection data
	}

	var written []string

	// config.go
	cfgPath := filepath.Join(projectDir, "internal/config/config.go")
	if cfgWritten, err := injectGoConfig(cfgPath, data); err != nil {
		return nil, fmt.Errorf("inject config.go: %w", err)
	} else if cfgWritten {
		written = append(written, "internal/config/config.go")
	}

	// config.yaml
	yamlPath := filepath.Join(projectDir, "configs/config.yaml")
	if yamlWritten, err := injectTextConfig(yamlPath, markerConfigYAML, data.YAMLSection); err != nil {
		return nil, fmt.Errorf("inject config.yaml: %w", err)
	} else if yamlWritten {
		written = append(written, "configs/config.yaml")
	}

	// .env.example
	envPath := filepath.Join(projectDir, ".env.example")
	if envWritten, err := injectTextConfig(envPath, markerEnvSection, data.EnvSection); err != nil {
		return nil, fmt.Errorf("inject .env.example: %w", err)
	} else if envWritten {
		written = append(written, ".env.example")
	}

	return written, nil
}

// injectGoConfig inserts a feature's Go config snippets into the config.go
// file at the marker positions. It also adds any needed imports. The result is
// validated with go/format before writing — if formatting fails, the file is
// left untouched and an error is returned.
func injectGoConfig(path string, data configInjection) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	content := string(raw)

	// Idempotency: skip if the struct field is already present. We
	// check for the field name + mapstructure tag rather than the whole
	// struct so gofmt-driven whitespace changes don't break the check.
	fieldTag := fieldSignature(data.StructField)
	if fieldTag != "" && strings.Contains(content, fieldTag) {
		return false, nil
	}

	// Insert struct field at marker.
	if strings.Contains(content, markerConfigFields) && data.StructField != "" {
		content = strings.Replace(content, markerConfigFields, data.StructField+"\n\t"+markerConfigFields, 1)
	}

	// Insert struct definition at marker.
	if strings.Contains(content, markerConfigStructs) && data.StructDef != "" {
		content = strings.Replace(content, markerConfigStructs, data.StructDef+"\n"+markerConfigStructs, 1)
	}

	// Insert defaults at marker.
	if strings.Contains(content, markerConfigDefaults) && data.Defaults != "" {
		content = strings.Replace(content, markerConfigDefaults, data.Defaults+"\n\t"+markerConfigDefaults, 1)
	}

	// Insert imports if needed.
	if len(data.Imports) > 0 {
		newContent, importErr := injectImports(content, data.Imports)
		if importErr == nil {
			content = newContent
		}
		// If import injection fails, proceed anyway; go/format will report
		// the resulting error which gives better diagnostics.
	}

	// Validate with go/format.
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return false, fmt.Errorf("config injection produced invalid Go (%s); manual changes may be needed: %w", path, err)
	}

	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	return true, nil
}

// injectImports adds the given import paths to the import block if they are
// not already present. It finds the import block and inserts lines before the
// closing parenthesis.
func injectImports(content string, imports []string) (string, error) {
	// Find the import block.
	importStart := strings.Index(content, "import (")
	if importStart < 0 {
		return content, fmt.Errorf("no import block found")
	}
	importEnd := strings.Index(content[importStart:], "\n)")
	if importEnd < 0 {
		return content, fmt.Errorf("import block not closed")
	}
	absEnd := importStart + importEnd

	importBlock := content[importStart : absEnd+2] // includes "\n)"

	var lines []string
	for _, imp := range imports {
		// Only add if not already present.
		if !strings.Contains(importBlock, imp) {
			lines = append(lines, "\t"+imp)
		}
	}
	if len(lines) == 0 {
		return content, nil
	}

	insertion := "\n" + strings.Join(lines, "\n")
	newContent := content[:absEnd] + insertion + content[absEnd:]
	return newContent, nil
}

// injectTextConfig inserts snippet before the marker in a text (YAML or env)
// config file. It is idempotent — if the snippet is already present, it
// returns false without error.
func injectTextConfig(path, marker, snippet string) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	content := string(raw)

	// Idempotency: skip if a representative line from the snippet is
	// already present. We use the first non-empty, non-comment line.
	representative := firstDataLine(snippet)
	if representative != "" && strings.Contains(content, representative) {
		return false, nil
	}

	if !strings.Contains(content, marker) {
		// Marker not found — cannot inject automatically.
		return false, nil
	}

	content = strings.Replace(content, marker, snippet+"\n"+marker, 1)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	return true, nil
}

// firstDataLine returns the first non-empty, non-comment line from text.
func firstDataLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			return trimmed
		}
	}
	return ""
}

// fieldSignature returns the "Name mapstructure:" tag fragment from a
// Go struct field declaration. This is the whitespace-independent
// fingerprint used for idempotency checks — gofmt may insert or remove
// extra alignment spaces, but the field name and the mapstructure tag
// string are stable.
func fieldSignature(fieldDecl string) string {
	trimmed := strings.TrimSpace(fieldDecl)
	idx := strings.Index(trimmed, "mapstructure:")
	if idx < 0 {
		return ""
	}
	// Find the field name (first whitespace-delimited token) and
	// concatenate it with the mapstructure tag.
	fields := strings.Fields(trimmed)
	if len(fields) < 2 {
		return ""
	}
	name := fields[0]
	// Walk back to include the mapstructure tag (and its closing backtick).
	tag := trimmed[idx:]
	if !strings.HasSuffix(tag, "`") {
		tag = tag + "`"
	}
	return name + " " + tag
}
