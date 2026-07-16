package seedgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options controls seed file generation.
type Options struct {
	ProjectDir string
	// ModelName is the PascalCase struct name. When empty, only scaffolding
	// files (main.go + seeder.go) are generated.
	ModelName string
	// Count is the number of seed rows to generate. Defaults to 10.
	Count int
	// Force overwrites existing files.
	Force bool
	// ModulePath is the Go module path for import statements.
	ModulePath string
	// Orm is the ORM in use ("gorm" or "bun").
	Orm string
}

// GeneratedFile reports a seed file that was created or skipped.
type GeneratedFile struct {
	Path    string
	Skipped bool
}

// GenerateSeed produces Go seed files in db/seeds/ of the target project.
// When ModelName is empty, only scaffolding (main.go + seeder.go) is
// generated. When a model is provided, a per-model seed file is created
// and the seeder is updated to register it.
func GenerateSeed(opts Options) ([]GeneratedFile, error) {
	if opts.ProjectDir == "" {
		return nil, fmt.Errorf("project directory is required")
	}
	if opts.Count <= 0 {
		opts.Count = 10
	}
	if opts.Orm == "" {
		opts.Orm = "gorm"
	}
	if opts.ModulePath == "" {
		return nil, fmt.Errorf("module path is required for Go seed generation")
	}

	var files []GeneratedFile

	seedsDir := filepath.Join(opts.ProjectDir, "db/seeds")

	// 1. Always ensure main.go exists (never force-overwrite scaffolding).
	mainPath := filepath.Join(seedsDir, "main.go")
	if !fileExists(mainPath) {
		if err := writeSeedFile(mainPath, generateMain(opts), false); err != nil {
			return nil, err
		}
		files = append(files, GeneratedFile{Path: "db/seeds/main.go"})
	} else {
		files = append(files, GeneratedFile{Path: "db/seeds/main.go", Skipped: true})
	}

	// 2. Always ensure seeder.go exists (never force-overwrite scaffolding).
	ormDir := filepath.Join(seedsDir, opts.Orm)
	seederPath := filepath.Join(ormDir, "seeder.go")
	if !fileExists(seederPath) {
		if err := writeSeedFile(seederPath, generateSeeder(opts), false); err != nil {
			return nil, err
		}
		files = append(files, GeneratedFile{Path: fmt.Sprintf("db/seeds/%s/seeder.go", opts.Orm)})
	} else {
		files = append(files, GeneratedFile{Path: fmt.Sprintf("db/seeds/%s/seeder.go", opts.Orm), Skipped: true})
	}

	if opts.ModelName == "" {
		return files, nil
	}

	// 3. Find the struct.
	info, err := FindStruct(opts.ProjectDir, opts.ModelName)
	if err != nil {
		return nil, fmt.Errorf("find struct %q: %w", opts.ModelName, err)
	}
	if info == nil {
		return nil, fmt.Errorf("struct %q not found in %s/internal/domain/<name>/ or persistence adapters",
			opts.ModelName, opts.ProjectDir)
	}

	// 4. Generate per-model seed file.
	seedFileName := fmt.Sprintf("seed_%s.go", info.TableName)
	seedPath := filepath.Join(ormDir, seedFileName)
	seedRelPath := fmt.Sprintf("db/seeds/%s/%s", opts.Orm, seedFileName)
	if fileExists(seedPath) && !opts.Force {
		files = append(files, GeneratedFile{Path: seedRelPath, Skipped: true})
		return files, nil
	}
	if err := writeSeedFile(seedPath, generateModelSeed(info, opts), opts.Force); err != nil {
		return nil, err
	}
	files = append(files, GeneratedFile{Path: seedRelPath})

	// 5. Update seeder.go to register the new model.
	if err := updateSeeder(seederPath, info, opts); err != nil {
		return nil, fmt.Errorf("update seeder: %w", err)
	}

	return files, nil
}

// generateMain returns the content for db/seeds/main.go.
func generateMain(opts Options) string {
	return fmt.Sprintf(`package main

import (
	"flag"
	"fmt"
	"log"

	seed%s "%s/db/seeds/%s"
	"%s/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.Load()

	var dir string
	flag.StringVar(&dir, "dir", "up", "Seed direction: up or down")
	flag.Parse()

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %%v", err)
	}

	seeder := seed%s.NewSeeder(db)
	switch dir {
	case "up":
		if err := seeder.SeedUp(); err != nil {
			log.Fatalf("seed up failed: %%v", err)
		}
		fmt.Println("✓ All seeds applied successfully")
	case "down":
		if err := seeder.SeedDown(); err != nil {
			log.Fatalf("seed down failed: %%v", err)
		}
		fmt.Println("✓ All seeds rolled back successfully")
	default:
		log.Fatalf("invalid direction: %%s (use 'up' or 'down')", dir)
	}
}
`, ormPkg(opts.Orm), opts.ModulePath, opts.Orm, opts.ModulePath, ormPkg(opts.Orm))
}

// generateSeeder returns the content for db/seeds/<orm>/seeder.go.
func generateSeeder(opts Options) string {
	return fmt.Sprintf(`package %s

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Seeder orchestrates seed operations in dependency order.
type Seeder struct {
	db *gorm.DB
}

// NewSeeder creates a new Seeder.
func NewSeeder(db *gorm.DB) *Seeder {
	return &Seeder{db: db}
}

// SeedUp runs all seed operations in order (respecting FK dependencies).
func (s *Seeder) SeedUp() error {
	seeders := []struct {
		name string
		fn   func(*gorm.DB) error
	}{
		// crank:seed-up-begin
		// crank:seed-up-end
	}

	for _, se := range seeders {
		log.Printf("seeding %%s ...", se.name)
		if err := se.fn(s.db); err != nil {
			return fmt.Errorf("seed %%s: %%w", se.name, err)
		}
	}
	return nil
}

// SeedDown tears down all seeded data in reverse dependency order.
func (s *Seeder) SeedDown() error {
	seeders := []struct {
		name string
		fn   func(*gorm.DB) error
	}{
		// crank:seed-down-begin
		// crank:seed-down-end
	}

	for _, se := range seeders {
		log.Printf("tearing down %%s ...", se.name)
		if err := se.fn(s.db); err != nil {
			return fmt.Errorf("seed down %%s: %%w", se.name, err)
		}
	}
	return nil
}
`, ormPkg(opts.Orm))
}

// generateModelSeed returns the content for db/seeds/<orm>/seed_<table>.go.
func generateModelSeed(info *StructInfo, opts Options) string {
	if opts.Orm == "bun" {
		return generateBunModelSeed(info, opts)
	}
	return generateGormModelSeed(info, opts)
}

func generateGormModelSeed(info *StructInfo, opts Options) string {
	var b strings.Builder

	needsTime := hasTimeField(info)

	b.WriteString(fmt.Sprintf(`package %s

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
`, ormPkg(opts.Orm)))

	if needsTime {
		b.WriteString("\t\"time\"\n")
	}

	b.WriteString(fmt.Sprintf(`)

type seed%s struct {
`, info.Name))

	for _, f := range info.ExportedFields {
		b.WriteString(fmt.Sprintf("\t%s %s `gorm:\"%s\"`\n",
			f.Name, goTypeForField(f), gormTag(f)))
	}

	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("func (seed%s) TableName() string { return %q }\n\n", info.Name, info.TableName))

	// SeedUp
	b.WriteString(fmt.Sprintf(`func Seed%sUp(db *gorm.DB) error {
	entries := []seed%s{
`, pascalPlural(info.Name), info.Name))

	for i := 0; i < opts.Count; i++ {
		b.WriteString("\t\t{")
		for j, f := range info.ExportedFields {
			if j > 0 {
				b.WriteString(", ")
			}
			val := goValueForField(f, i)
			if f.Name == "ID" || (strings.HasSuffix(strings.ToLower(f.Name), "id") && isUUIDType(f)) {
				val = goIDValue(i)
			}
			b.WriteString(fmt.Sprintf("%s: %s", f.Name, val))
		}
		b.WriteString("},\n")
	}

	b.WriteString(`	}
	for _, e := range entries {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&e).Error; err != nil {
			return err
		}
		log.Printf("  ✓ ` + info.TableName + ` %%s", e.`)

	nameField := findNameField(info)
	if nameField != "" {
		b.WriteString(nameField)
	} else {
		b.WriteString("ID")
	}
	b.WriteString(`)
	}
	return nil
}
`)

	// SeedDown
	b.WriteString(fmt.Sprintf(`func Seed%sDown(db *gorm.DB) error {
	return db.Where("id LIKE 'a0000000-0000-4000-a000-00000000000%%%%'").Delete(&seed%s{}).Error
}
`, pascalPlural(info.Name), info.Name))

	return b.String()
}

func generateBunModelSeed(info *StructInfo, opts Options) string {
	var b strings.Builder

	needsTime := hasTimeField(info)

	b.WriteString(fmt.Sprintf(`package %s

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
`, ormPkg(opts.Orm)))

	if needsTime {
		b.WriteString("\t\"time\"\n")
	}

	b.WriteString(fmt.Sprintf(`)

type seed%s struct {
	bun.BaseModel `+"`"+`bun:"table:%s"`+"`"+`
`, info.Name, info.TableName))

	for _, f := range info.ExportedFields {
		b.WriteString(fmt.Sprintf("\t%s %s `bun:\"%s\"`\n",
			f.Name, goTypeForField(f), bunTag(f)))
	}

	b.WriteString("}\n\n")

	// SeedUp
	b.WriteString(fmt.Sprintf(`func Seed%sUp(db *bun.DB) error {
	ctx := context.Background()
	entries := []seed%s{
`, pascalPlural(info.Name), info.Name))

	for i := 0; i < opts.Count; i++ {
		b.WriteString("\t\t{")
		for j, f := range info.ExportedFields {
			if j > 0 {
				b.WriteString(", ")
			}
			val := goValueForField(f, i)
			if f.Name == "ID" || (strings.HasSuffix(strings.ToLower(f.Name), "id") && isUUIDType(f)) {
				val = goIDValue(i)
			}
			b.WriteString(fmt.Sprintf("%s: %s", f.Name, val))
		}
		b.WriteString("},\n")
	}

	b.WriteString(`	}
	for _, e := range entries {
		if _, err := db.NewInsert().Model(&e).On("CONFLICT (id) DO NOTHING").Exec(ctx); err != nil {
			return err
		}
		log.Printf("  ✓ ` + info.TableName + ` %%s", e.`)

	nameField := findNameField(info)
	if nameField != "" {
		b.WriteString(nameField)
	} else {
		b.WriteString("ID")
	}
	b.WriteString(`)
	}
	return nil
}
`)

	// SeedDown
	b.WriteString(fmt.Sprintf(`func Seed%sDown(db *bun.DB) error {
	ctx := context.Background()
	_, err := db.NewDelete().Model((*seed%s)(nil)).
		Where("id LIKE 'a0000000-0000-4000-a000-00000000000%%%%'").
		Exec(ctx)
	return err
}
`, pascalPlural(info.Name), info.Name))

	return b.String()
}

// updateSeeder reads an existing seeder.go and injects entries for the new
// model into the SeedUp and SeedDown lists. If already registered, it is a
// no-op.
func updateSeeder(seederPath string, info *StructInfo, opts Options) error {
	content, err := os.ReadFile(seederPath)
	if err != nil {
		return err
	}
	text := string(content)

	snakePlural := info.TableName
	upEntry := fmt.Sprintf(`{"%s", Seed%sUp},`, snakePlural, pascalPlural(info.Name))
	downEntry := fmt.Sprintf(`{"%s", Seed%sDown},`, snakePlural, pascalPlural(info.Name))

	if strings.Contains(text, upEntry) {
		return nil
	}

	// Inject into SeedUp (before // crank:seed-up-end).
	upEndMarker := "// crank:seed-up-end"
	if strings.Contains(text, upEndMarker) {
		text = strings.Replace(text, upEndMarker,
			"\t\t"+upEntry+"\n\t\t"+upEndMarker, 1)
	}

	// Inject into SeedDown (after // crank:seed-down-begin, so it's torn
	// down first — safe default for new models).
	downBeginMarker := "// crank:seed-down-begin"
	if strings.Contains(text, downBeginMarker) {
		text = strings.Replace(text, downBeginMarker,
			downBeginMarker+"\n\t\t"+downEntry, 1)
	}

	return os.WriteFile(seederPath, []byte(text), 0o644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writeSeedFile(path, content string, force bool) error {
	if fileExists(path) && !force {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func ormPkg(orm string) string {
	if orm == "bun" {
		return "bun"
	}
	return "gorm"
}

func pascalPlural(name string) string {
	return name + "s"
}

func findNameField(info *StructInfo) string {
	for _, f := range info.ExportedFields {
		lower := strings.ToLower(f.Name)
		if lower == "name" {
			return f.Name
		}
	}
	return ""
}

func hasTimeField(info *StructInfo) bool {
	for _, f := range info.ExportedFields {
		if f.GoType == "time.Time" || strings.HasSuffix(f.GoType, ".Time") {
			return true
		}
	}
	return false
}

func isUUIDType(f FieldInfo) bool {
	lower := strings.ToLower(f.GoType)
	return strings.Contains(lower, "uuid")
}

func goTypeForField(f FieldInfo) string {
	switch {
	case f.GoType == "uuid.UUID" || strings.HasSuffix(f.GoType, ".UUID"):
		return "uuid.UUID"
	case strings.HasPrefix(f.GoType, "*"):
		return "*" + goTypeForField(FieldInfo{GoType: strings.TrimPrefix(f.GoType, "*")})
	case f.GoType == "time.Time":
		return "time.Time"
	default:
		return f.GoType
	}
}

func gormTag(f FieldInfo) string {
	tag := fmt.Sprintf("column:%s", f.ColumnName)
	lower := strings.ToLower(f.GoType)
	if strings.Contains(lower, "uuid") || strings.Contains(strings.ToLower(f.ORMType), "uuid") {
		tag += ";type:uuid"
		if strings.EqualFold(f.ColumnName, "id") {
			tag += ";primaryKey"
		}
	}
	return tag
}

func bunTag(f FieldInfo) string {
	tag := f.ColumnName
	lower := strings.ToLower(f.GoType)
	if strings.Contains(lower, "uuid") || strings.Contains(strings.ToLower(f.ORMType), "uuid") {
		tag += ",type:uuid"
		if strings.EqualFold(f.ColumnName, "id") {
			tag += ",pk"
		}
	}
	return tag
}

func goIDValue(idx int) string {
	return fmt.Sprintf("uuid.MustParse(\"a0000000-0000-4000-a000-%012x\")", idx+1)
}
