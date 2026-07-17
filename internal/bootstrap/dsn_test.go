package bootstrap

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDSNFromConfig_EncodesSpecialChars guards the fix for credentials that
// contain URL-reserved characters. A password like "p@ss:w/rd?" must be
// percent-encoded so the resulting DSN parses back to the original value
// rather than silently corrupting the connection string.
func TestDSNFromConfig_EncodesSpecialChars(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "database:\n" +
		"  host: \"db.internal\"\n" +
		"  port: \"5432\"\n" +
		"  user: \"admin\"\n" +
		"  password: \"p@ss:w/rd?#1\"\n" +
		"  name: \"mydb\"\n" +
		"  sslmode: \"require\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	dsn, err := DSNFromConfig(dir)
	if err != nil {
		t.Fatalf("DSNFromConfig: %v", err)
	}

	// The raw (unencoded) password must never appear verbatim in the DSN.
	if strings.Contains(dsn, "p@ss:w/rd?#1") {
		t.Fatalf("password was not percent-encoded: %s", dsn)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("resulting DSN does not parse: %v (dsn=%s)", err, dsn)
	}
	if u.Scheme != "postgres" {
		t.Errorf("scheme = %q, want postgres", u.Scheme)
	}
	if got := u.Hostname(); got != "db.internal" {
		t.Errorf("host = %q, want db.internal", got)
	}
	if got := u.Port(); got != "5432" {
		t.Errorf("port = %q, want 5432", got)
	}
	if u.User.Username() != "admin" {
		t.Errorf("user = %q, want admin", u.User.Username())
	}
	pw, _ := u.User.Password()
	if pw != "p@ss:w/rd?#1" {
		t.Errorf("decoded password = %q, want p@ss:w/rd?#1", pw)
	}
	if got := strings.TrimPrefix(u.Path, "/"); got != "mydb" {
		t.Errorf("db name = %q, want mydb", got)
	}
	if got := u.Query().Get("sslmode"); got != "require" {
		t.Errorf("sslmode = %q, want require", got)
	}
}
