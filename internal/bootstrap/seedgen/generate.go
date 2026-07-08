package seedgen

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/anurag925/crank/internal/utils"
)

// Options controls seed file generation.
type Options struct {
	// ProjectDir is the root of the target crank-generated project.
	ProjectDir string
	// ModelName is the PascalCase struct name. When empty, an empty seed
	// file with a placeholder INSERT is generated.
	ModelName string
	// Count is the number of seed rows to generate. Defaults to 10.
	Count int
	// Force overwrites an existing seed file with the same timestamp.
	Force bool
}

// GeneratedFile reports a seed file that was created or skipped.
type GeneratedFile struct {
	Path    string
	Skipped bool
}

// GenerateSeed produces one or more timestamped SQL seed files in
// db/seeds/ of the target project. When ModelName is empty, a single
// placeholder file is generated. When a model is provided with exported
// fields, a data-driven INSERT statement is produced.
//
// Returns the generated file(s) or an error.
func GenerateSeed(opts Options) ([]GeneratedFile, error) {
	if opts.ProjectDir == "" {
		return nil, fmt.Errorf("project directory is required")
	}
	if opts.Count <= 0 {
		opts.Count = 10
	}

	seedsDir := filepath.Join(opts.ProjectDir, "db/seeds")
	if err := utils.EnsureDir(seedsDir); err != nil {
		return nil, fmt.Errorf("create db/seeds directory: %w", err)
	}

	stamp := time.Now().UTC().Format("20060102150405")

	if opts.ModelName == "" {
		// Empty seed file with a template INSERT.
		name := fmt.Sprintf("%s_empty_seed.up.sql", stamp)
		path := filepath.Join(seedsDir, name)
		if !opts.Force {
			if utils.PathExists(path) {
				return []GeneratedFile{{Path: name, Skipped: true}}, nil
			}
		}
		content := emptySeedContent(stamp)
		if err := utils.WriteFile(path, content); err != nil {
			return nil, err
		}
		return []GeneratedFile{{Path: name}}, nil
	}

	// Find the struct and generate data-driven seed.
	info, err := FindStruct(opts.ProjectDir, opts.ModelName)
	if err != nil {
		return nil, fmt.Errorf("find struct %q: %w", opts.ModelName, err)
	}
	if info == nil {
		return nil, fmt.Errorf("struct %q not found in %s/internal/domain/<name>/ (no exported fields found)",
			opts.ModelName, opts.ProjectDir)
	}

	// Build INSERT with generated data.
	upContent := generateInsert(info, opts.Count, stamp)

	upName := fmt.Sprintf("%s_seed_%s.up.sql", stamp, info.TableName)
	upPath := filepath.Join(seedsDir, upName)

	if utils.PathExists(upPath) && !opts.Force {
		return []GeneratedFile{{Path: upName, Skipped: true}}, nil
	}

	if err := utils.WriteFile(upPath, upContent); err != nil {
		return nil, err
	}

	// Down file: TRUNCATE the table (safe for seed data).
	downName := fmt.Sprintf("%s_seed_%s.down.sql", stamp, info.TableName)
	downPath := filepath.Join(seedsDir, downName)
	downContent := fmt.Sprintf("-- +migrate Down\nTRUNCATE TABLE %s;\n", info.TableName)
	if err := utils.WriteFile(downPath, downContent); err != nil {
		return nil, err
	}

	return []GeneratedFile{
		{Path: upName},
		{Path: downName},
	}, nil
}

// generateInsert builds a multi-row INSERT statement with fake data.
func generateInsert(info *StructInfo, count int, stamp string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("-- Seed data for %s\n", info.Name))
	b.WriteString(fmt.Sprintf("-- Generated at %s\n", stamp))
	b.WriteString(fmt.Sprintf("-- Rows: %d\n", count))
	b.WriteString("\n")

	// Build column list.
	colNames := make([]string, len(info.ExportedFields))
	for i, f := range info.ExportedFields {
		colNames[i] = f.ColumnName
	}

	b.WriteString("INSERT INTO ")
	b.WriteString(info.TableName)
	b.WriteString(" (")
	b.WriteString(strings.Join(colNames, ", "))
	b.WriteString(") VALUES\n")

	// Build row values.
	for i := 0; i < count; i++ {
		vals := make([]string, len(info.ExportedFields))
		for j, f := range info.ExportedFields {
			vals[j] = fakeValue(f)
		}
		b.WriteString("    (")
		b.WriteString(strings.Join(vals, ", "))
		if i < count-1 {
			b.WriteString("),\n")
		} else {
			b.WriteString(");\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

// emptySeedContent generates a placeholder seed file for manual editing.
func emptySeedContent(stamp string) string {
	var b strings.Builder
	b.WriteString("-- Seed data\n")
	b.WriteString("-- Generated at ")
	b.WriteString(stamp)
	b.WriteString("\n")
	b.WriteString("-- TODO: Replace this with your seed data.\n")
	b.WriteString("--\n")
	b.WriteString("-- Examples:\n")
	b.WriteString("--   INSERT INTO users (id, name, email) VALUES\n")
	b.WriteString("--       ('a1b2c3d4-...', 'Alice', 'alice@example.com'),\n")
	b.WriteString("--       ('e5f6g7h8-...', 'Bob', 'bob@example.com');\n")
	b.WriteString("\n")
	b.WriteString("-- +migrate Up\n")
	b.WriteString("\n")
	b.WriteString("-- INSERT INTO <table> (<columns>) VALUES\n")
	b.WriteString("--     (<values>);\n")
	b.WriteString("\n")
	return b.String()
}
