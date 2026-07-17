package seedgen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Parser tests (unchanged logic)
// ---------------------------------------------------------------------------

func TestParseStruct_DomainAggregate_ParseRowDTO(t *testing.T) {
	dir := t.TempDir()

	domainDir := filepath.Join(dir, "internal/domain/product")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "product.go"), []byte(`package product

import "github.com/google/uuid"

type Product struct {
	id    uuid.UUID
	name  string
	price float64
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	gormDir := filepath.Join(dir, "internal/adapters/persistence/gorm")
	if err := os.MkdirAll(gormDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gormDir, "product_repository.go"), []byte(`package gorm

import "github.com/google/uuid"

type productRow struct {
	ID    uuid.UUID `+"`"+`gorm:"column:id;type:uuid;primaryKey"`+"`"+`
	Name  string    `+"`"+`gorm:"column:name"`+"`"+`
	Price float64   `+"`"+`gorm:"column:price"`+"`"+`
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := FindStruct(dir, "Product")
	if err != nil {
		t.Fatalf("FindStruct: %v", err)
	}
	if info == nil {
		t.Fatal("FindStruct returned nil, expected row DTO to be found")
	}
	if info.Name != "Product" {
		t.Errorf("Name = %q, want %q", info.Name, "Product")
	}
	if info.TableName != "products" {
		t.Errorf("TableName = %q, want %q", info.TableName, "products")
	}
	if len(info.ExportedFields) != 3 {
		t.Fatalf("got %d exported fields, want 3", len(info.ExportedFields))
	}
	if info.ExportedFields[0].Name != "ID" {
		t.Errorf("field[0].Name = %q, want %q", info.ExportedFields[0].Name, "ID")
	}
	if info.ExportedFields[0].ColumnName != "id" {
		t.Errorf("field[0].ColumnName = %q, want %q", info.ExportedFields[0].ColumnName, "id")
	}
	if info.ExportedFields[0].ORMType != "uuid" {
		t.Errorf("field[0].ORMType = %q, want %q", info.ExportedFields[0].ORMType, "uuid")
	}
}

func TestParseStruct_DomainWithExportedFields(t *testing.T) {
	dir := t.TempDir()
	domainDir := filepath.Join(dir, "internal/domain/user")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "user.go"), []byte(`package user

type User struct {
	Name  string
	Email string
	Age   int
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := FindStruct(dir, "User")
	if err != nil {
		t.Fatalf("FindStruct: %v", err)
	}
	if info == nil {
		t.Fatal("FindStruct returned nil")
	}
	if len(info.ExportedFields) != 3 {
		t.Fatalf("got %d fields, want 3", len(info.ExportedFields))
	}
}

func TestParseStruct_MixedVisibility(t *testing.T) {
	dir := t.TempDir()
	domainDir := filepath.Join(dir, "internal/domain/staff")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "staff.go"), []byte(`package staff

type Staff struct {
	ID        string
	name      string // private, should be skipped
	Email     string
	salary    float64 // private, should be skipped
	Department string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := FindStruct(dir, "Staff")
	if err != nil {
		t.Fatalf("FindStruct: %v", err)
	}
	if info == nil {
		t.Fatal("FindStruct returned nil")
	}
	if len(info.ExportedFields) != 3 {
		t.Fatalf("got %d fields, want 3 (ID, Email, Department)", len(info.ExportedFields))
	}
}

func TestParseStruct_NotFound(t *testing.T) {
	dir := t.TempDir()
	info, err := FindStruct(dir, "NonExistent")
	if err != nil {
		t.Fatalf("FindStruct: %v", err)
	}
	if info != nil {
		t.Fatal("expected nil for non-existent struct")
	}
}

func TestParseStruct_NoDomainDir(t *testing.T) {
	dir := t.TempDir()
	domainDir := filepath.Join(dir, "internal/domain/other")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "other.go"), []byte(`package other

type Other struct {
	X string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := FindStruct(dir, "NonExistent")
	if err != nil {
		t.Fatalf("FindStruct: %v", err)
	}
	if info != nil {
		t.Fatal("expected nil for non-existent struct name")
	}
}

func TestParseStruct_GormTagColumnName(t *testing.T) {
	dir := t.TempDir()
	gormDir := filepath.Join(dir, "internal/adapters/persistence/gorm")
	if err := os.MkdirAll(gormDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gormDir, "user_repository.go"), []byte(`package gorm

type UserRow struct {
	UserID    string `+"`"+`gorm:"column:user_id;primaryKey"`+"`"+`
	FullName  string `+"`"+`gorm:"column:full_name"`+"`"+`
	CreatedAt string `+"`"+`gorm:"column:created_at"`+"`"+`
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := FindStruct(dir, "User")
	if err != nil {
		t.Fatalf("FindStruct: %v", err)
	}
	if info == nil {
		t.Fatal("FindStruct returned nil")
	}
	if len(info.ExportedFields) != 3 {
		t.Fatalf("got %d fields, want 3", len(info.ExportedFields))
	}
	if info.ExportedFields[0].ColumnName != "user_id" {
		t.Errorf("ColumnName = %q, want %q", info.ExportedFields[0].ColumnName, "user_id")
	}
	if info.ExportedFields[1].ColumnName != "full_name" {
		t.Errorf("ColumnName = %q, want %q", info.ExportedFields[1].ColumnName, "full_name")
	}
}

func TestParseStruct_EnumDetection(t *testing.T) {
	dir := t.TempDir()
	domainDir := filepath.Join(dir, "internal/domain/ticket")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "ticket.go"), []byte(`package ticket

type Status string

const (
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
	StatusPending Status = "pending"
)

type Ticket struct {
	ID     string
	Title  string
	Status Status
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := FindStruct(dir, "Ticket")
	if err != nil {
		t.Fatalf("FindStruct: %v", err)
	}
	if info == nil {
		t.Fatal("FindStruct returned nil")
	}
	if len(info.ExportedFields) != 3 {
		t.Fatalf("got %d fields, want 3", len(info.ExportedFields))
	}

	statusField := info.ExportedFields[2]
	if statusField.Name != "Status" {
		t.Errorf("field[2].Name = %q, want %q", statusField.Name, "Status")
	}
	if !statusField.IsEnum {
		t.Fatal("Status field should be detected as enum")
	}
	if len(statusField.EnumValues) != 3 {
		t.Fatalf("expected 3 enum values, got %d: %v", len(statusField.EnumValues), statusField.EnumValues)
	}
	want := map[string]bool{"open": true, "closed": true, "pending": true}
	for _, v := range statusField.EnumValues {
		if !want[v] {
			t.Errorf("unexpected enum value %q", v)
		}
		delete(want, v)
	}
	if len(want) > 0 {
		t.Errorf("missing enum values: %v", want)
	}
}

// ---------------------------------------------------------------------------
// Package parsing edge cases
// ---------------------------------------------------------------------------

func TestParsePackage_NonExistentDir(t *testing.T) {
	pi, err := parsePackage("/tmp/nonexistent-path-12345")
	if err != nil {
		t.Fatalf("parsePackage should not error on non-existent dir: %v", err)
	}
	if pi != nil {
		t.Fatal("expected nil result for non-existent directory")
	}
}

func TestParsePackage_NoGoFiles(t *testing.T) {
	dir := t.TempDir()
	pi, err := parsePackage(dir)
	if err != nil {
		t.Fatalf("parsePackage should not error on empty dir: %v", err)
	}
	if pi != nil {
		t.Fatal("expected nil result for dir with no .go files")
	}
}

func TestParsePackage_OnlyNonGoFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# not go"), 0o644); err != nil {
		t.Fatal(err)
	}
	pi, err := parsePackage(dir)
	if err != nil {
		t.Fatalf("parsePackage should not error: %v", err)
	}
	if pi != nil {
		t.Fatal("expected nil for dir with only non-Go files")
	}
}

// ---------------------------------------------------------------------------
// extractTagValue tests
// ---------------------------------------------------------------------------

func TestExtractTagValue(t *testing.T) {
	cases := []struct {
		tag  string
		key  string
		want string
	}{
		{`gorm:"column:id;primaryKey"`, "gorm:", "column:id;primaryKey"},
		{`gorm:"column:name" json:"name"`, "gorm:", "column:name"},
		{`gorm:"column:name" json:"name"`, "json:", "name"},
		{`gorm:"column:user_id;primaryKey"`, "gorm:", "column:user_id;primaryKey"},
		{`json:"name"`, "gorm:", ""},
		{``, "gorm:", ""},
		{`gorm:""`, "gorm:", ""},
	}
	for _, c := range cases {
		got := extractTagValue(c.tag, c.key)
		if got != c.want {
			t.Errorf("extractTagValue(%q, %q) = %q, want %q", c.tag, c.key, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// columnName tests
// ---------------------------------------------------------------------------

func TestColumnName(t *testing.T) {
	makeTag := func(v string) *ast.BasicLit {
		return &ast.BasicLit{Value: "`" + v + "`"}
	}

	cases := []struct {
		fieldName string
		tag       *ast.BasicLit
		want      string
	}{
		{"UserID", makeTag(`gorm:"column:user_id;primaryKey"`), "user_id"},
		{"UserName", makeTag(`gorm:"column:user_name"`), "user_name"},
		{"Email", makeTag(`gorm:"-"`), "email"},
		{"CreatedAt", nil, "created_at"},
		{"ID", nil, "id"},
		{"MyField", makeTag(`gorm:"column:my_field"`), "my_field"},
	}
	for _, c := range cases {
		got := columnName(c.fieldName, c.tag)
		if got != c.want {
			t.Errorf("columnName(%q, %v) = %q, want %q", c.fieldName, c.tag, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// ormType tests
// ---------------------------------------------------------------------------

func TestOrmType(t *testing.T) {
	makeTag := func(v string) *ast.BasicLit {
		return &ast.BasicLit{Value: "`" + v + "`"}
	}

	cases := []struct {
		tag  *ast.BasicLit
		want string
	}{
		{makeTag(`gorm:"column:id;type:uuid;primaryKey"`), "uuid"},
		{makeTag(`gorm:"column:name;type:TEXT"`), "TEXT"},
		{makeTag(`gorm:"column:name"`), ""},
		{nil, ""},
		{makeTag(`json:"name"`), ""},
	}
	for _, c := range cases {
		got := ormType(c.tag)
		if got != c.want {
			t.Errorf("ormType(%v) = %q, want %q", c.tag, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// shortenImport tests
// ---------------------------------------------------------------------------

func TestShortenImport(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"github.com/google/uuid", "uuid"},
		{"github.com/gofrs/uuid", "uuid"},
		{"time", "time"},
		{"gorm.io/gorm", "gorm"},
	}
	for _, c := range cases {
		got := shortenImport(c.path)
		if got != c.want {
			t.Errorf("shortenImport(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// typeString tests
// ---------------------------------------------------------------------------

func TestTypeString(t *testing.T) {
	exprStr := `package p; type X struct { A uuid.UUID; B *string; C []int; E time.Time }`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", exprStr, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	pi := &packageInfo{imports: map[string]string{"uuid": "github.com/google/uuid", "time": "time"}}

	ts := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	st := ts.Type.(*ast.StructType)
	fields := st.Fields.List

	tests := []struct {
		index int
		want  string
	}{
		{0, "uuid.UUID"},
		{1, "*string"},
		{2, "[]int"},
		{3, "time.Time"},
	}
	for _, tc := range tests {
		got := typeString(fields[tc.index].Type, pi)
		if got != tc.want {
			t.Errorf("field[%d] typeString = %q, want %q", tc.index, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// pascalToSnake tests
// ---------------------------------------------------------------------------

func TestPascalToSnake(t *testing.T) {
	cases := []struct{ in, want string }{
		{"User", "user"},
		{"OrderItem", "order_item"},
		{"ABC", "abc"},
		{"UserID", "user_id"},
		{"UserIDField", "user_id_field"},
		{"HTTPServer", "http_server"},
		{"GenerateSQL", "generate_sql"},
		{"", ""},
	}
	for _, c := range cases {
		got := pascalToSnake(c.in)
		if got != c.want {
			t.Errorf("pascalToSnake(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// snakePlural tests
// ---------------------------------------------------------------------------

func TestSnakePlural(t *testing.T) {
	cases := []struct{ in, want string }{
		{"user", "users"},
		{"product", "products"},
		{"category", "categories"},
		{"address", "addresses"},
		{"box", "boxes"},
		{"quiz", "quizzes"},
		{"status", "statuses"},
		{"vowel_day", "vowel_days"},
		{"", ""},
	}
	for _, c := range cases {
		got := snakePlural(c.in)
		if got != c.want {
			t.Errorf("snakePlural(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// GenerateSeed tests — Go file generation
// ---------------------------------------------------------------------------

func TestGenerateSeed_WithModel(t *testing.T) {
	dir := t.TempDir()
	domainDir := filepath.Join(dir, "internal/domain/product")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "product.go"), []byte(`package product

type Product struct {
	ID    string
	Name  string
	Price float64
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		ProjectDir: dir,
		ModelName:  "Product",
		Count:      3,
		ModulePath: "example.com/myapp",
	}
	_, err := GenerateSeed(opts)
	if err != nil {
		t.Fatalf("GenerateSeed: %v", err)
	}

	// Check main.go was generated.
	mainPath := filepath.Join(dir, "db/seeds/main.go")
	mainData, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal("main.go not generated:", err)
	}
	if !strings.Contains(string(mainData), "package main") {
		t.Error("main.go missing package main")
	}
	if !strings.Contains(string(mainData), "config.Load()") {
		t.Error("main.go missing config.Load()")
	}

	// Check seeder.go was generated with marker comments.
	seederPath := filepath.Join(dir, "db/seeds/gorm/seeder.go")
	seederData, err := os.ReadFile(seederPath)
	if err != nil {
		t.Fatal("seeder.go not generated:", err)
	}
	seederStr := string(seederData)
	if !strings.Contains(seederStr, "crank:seed-up-begin") {
		t.Error("seeder.go missing crank:seed-up-begin marker")
	}
	if !strings.Contains(seederStr, "crank:seed-up-end") {
		t.Error("seeder.go missing crank:seed-up-end marker")
	}
	if !strings.Contains(seederStr, `{"products", SeedProductsUp}`) {
		t.Error("seeder.go should have SeedProductsUp registered")
	}
	if !strings.Contains(seederStr, `{"products", SeedProductsDown}`) {
		t.Error("seeder.go should have SeedProductsDown registered")
	}

	// Check seed_products.go was generated.
	seedPath := filepath.Join(dir, "db/seeds/gorm/seed_products.go")
	seedData, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatal("seed_products.go not generated:", err)
	}
	seedStr := string(seedData)
	if !strings.Contains(seedStr, "func SeedProductsUp") {
		t.Error("seed_products.go missing SeedProductsUp")
	}
	if !strings.Contains(seedStr, "func SeedProductsDown") {
		t.Error("seed_products.go missing SeedProductsDown")
	}
	if !strings.Contains(seedStr, "clause.OnConflict{DoNothing: true}") {
		t.Error("seed_products.go missing OnConflict clause")
	}
	if !strings.Contains(seedStr, `gorm:"column:id`) {
		t.Error("seed_products.go missing gorm tag")
	}
}

func TestGenerateSeed_ScaffoldOnly(t *testing.T) {
	dir := t.TempDir()

	opts := Options{
		ProjectDir: dir,
		ModelName:  "",
		Count:      0,
		ModulePath: "example.com/myapp",
	}
	files, err := GenerateSeed(opts)
	if err != nil {
		t.Fatalf("GenerateSeed: %v", err)
	}

	foundMain := false
	foundSeeder := false
	for _, f := range files {
		if strings.Contains(f.Path, "main.go") {
			foundMain = true
		}
		if strings.Contains(f.Path, "seeder.go") {
			foundSeeder = true
		}
	}
	if !foundMain {
		t.Error("scaffold should generate main.go")
	}
	if !foundSeeder {
		t.Error("scaffold should generate seeder.go")
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files (main.go + seeder.go), got %d", len(files))
	}

	// Verify main.go exists on disk.
	mainPath := filepath.Join(dir, "db/seeds/main.go")
	if _, err := os.Stat(mainPath); err != nil {
		t.Error("main.go not written to disk")
	}
	seederPath := filepath.Join(dir, "db/seeds/gorm/seeder.go")
	if _, err := os.Stat(seederPath); err != nil {
		t.Error("seeder.go not written to disk")
	}
}

func TestGenerateSeed_EmptyProjectDir(t *testing.T) {
	_, err := GenerateSeed(Options{
		ProjectDir: "",
		ModelName:  "User",
	})
	if err == nil {
		t.Fatal("expected error for empty project directory")
	}
	if !strings.Contains(err.Error(), "project directory is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerateSeed_MissingModulePath(t *testing.T) {
	dir := t.TempDir()
	_, err := GenerateSeed(Options{
		ProjectDir: dir,
		ModelName:  "User",
	})
	if err == nil {
		t.Fatal("expected error for missing module path")
	}
	if !strings.Contains(err.Error(), "module path") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerateSeed_ModelNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := GenerateSeed(Options{
		ProjectDir: dir,
		ModelName:  "NonExistent",
		ModulePath: "example.com/myapp",
	})
	if err == nil {
		t.Fatal("expected error for non-existent model")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerateSeed_DefaultCount(t *testing.T) {
	dir := t.TempDir()
	domainDir := filepath.Join(dir, "internal/domain/item")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "item.go"), []byte(`package item

type Item struct {
	ID   string
	Name string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		ProjectDir: dir,
		ModelName:  "Item",
		Count:      0, // should default to 10
		ModulePath: "example.com/myapp",
	}
	_, err := GenerateSeed(opts)
	if err != nil {
		t.Fatalf("GenerateSeed: %v", err)
	}

	seedPath := filepath.Join(dir, "db/seeds/gorm/seed_items.go")
	data, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	// 10 entries in the entries slice.
	count := strings.Count(content, "uuid.MustParse")
	if count < 10 {
		t.Errorf("expected at least 10 rows (uuid.MustParse calls), got %d", count)
	}
}

func TestGenerateSeed_ForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	domainDir := filepath.Join(dir, "internal/domain/item")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "item.go"), []byte(`package item

type Item struct {
	ID string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		ProjectDir: dir,
		ModelName:  "Item",
		Count:      1,
		ModulePath: "example.com/myapp",
	}

	// First generation.
	files1, err := GenerateSeed(opts)
	if err != nil {
		t.Fatal(err)
	}
	_ = files1

	// Second without force should skip the model seed file.
	opts.Force = false
	files2, err := GenerateSeed(opts)
	if err != nil {
		t.Fatal(err)
	}
	skipped := false
	for _, f := range files2 {
		if f.Skipped {
			skipped = true
		}
	}
	if !skipped {
		t.Error("expected second call to skip existing file without --force")
	}

	// Third with force should not skip.
	opts.Force = true
	files3, err := GenerateSeed(opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files3 {
		if f.Skipped && strings.Contains(f.Path, "seed_items.go") {
			t.Error("expected --force to overwrite model seed file, but it was skipped")
		}
	}
}

func TestGenerateSeed_AdditionalModel(t *testing.T) {
	dir := t.TempDir()

	// Create domain structs for User and Product.
	userDir := filepath.Join(dir, "internal/domain/user")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "user.go"), []byte(`package user

type User struct {
	ID    string
	Name  string
	Email string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	productDir := filepath.Join(dir, "internal/domain/product")
	if err := os.MkdirAll(productDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(productDir, "product.go"), []byte(`package product

type Product struct {
	ID    string
	Name  string
	Price float64
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	baseOpts := Options{
		ProjectDir: dir,
		Count:      2,
		ModulePath: "example.com/myapp",
	}

	// Generate User seed.
	opts := baseOpts
	opts.ModelName = "User"
	_, err := GenerateSeed(opts)
	if err != nil {
		t.Fatalf("GenerateSeed User: %v", err)
	}

	// Generate Product seed (should append to seeder, not overwrite).
	opts.ModelName = "Product"
	_, err = GenerateSeed(opts)
	if err != nil {
		t.Fatalf("GenerateSeed Product: %v", err)
	}

	// Verify seeder.go has both models registered.
	seederData, err := os.ReadFile(filepath.Join(dir, "db/seeds/gorm/seeder.go"))
	if err != nil {
		t.Fatal(err)
	}
	seederStr := string(seederData)
	if !strings.Contains(seederStr, `{"users", SeedUsersUp}`) {
		t.Error("seeder.go should have SeedUsersUp")
	}
	if !strings.Contains(seederStr, `{"products", SeedProductsUp}`) {
		t.Error("seeder.go should have SeedProductsUp")
	}
	if !strings.Contains(seederStr, `{"users", SeedUsersDown}`) {
		t.Error("seeder.go should have SeedUsersDown")
	}
	if !strings.Contains(seederStr, `{"products", SeedProductsDown}`) {
		t.Error("seeder.go should have SeedProductsDown")
	}

	// Repeat generation should not duplicate entries.
	opts.ModelName = "Product"
	_, err = GenerateSeed(opts)
	if err != nil {
		t.Fatalf("GenerateSeed Product (repeat): %v", err)
	}
	seederData2, _ := os.ReadFile(filepath.Join(dir, "db/seeds/gorm/seeder.go"))
	seederStr2 := string(seederData2)
	if strings.Count(seederStr2, `{"products", SeedProductsUp}`) > 1 {
		t.Error("seeder.go should not have duplicate entries for Product")
	}
}

// ---------------------------------------------------------------------------
// goValueForField tests
// ---------------------------------------------------------------------------

func TestGoValueForField_String(t *testing.T) {
	v := goValueForField(FieldInfo{
		Name:       "Name",
		ColumnName: "name",
		GoType:     "string",
	}, 0)
	if !strings.HasPrefix(v, `"`) || !strings.HasSuffix(v, `"`) {
		t.Errorf("expected quoted string, got: %s", v)
	}
}

func TestGoValueForField_UUID(t *testing.T) {
	v := goValueForField(FieldInfo{
		Name:       "ID",
		ColumnName: "id",
		GoType:     "uuid.UUID",
	}, 3)
	if !strings.Contains(v, "uuid.MustParse") {
		t.Errorf("expected uuid.MustParse, got: %s", v)
	}
}

func TestGoValueForField_Int(t *testing.T) {
	v := goValueForField(FieldInfo{GoType: "int"}, 0)
	if v[0] == '"' {
		t.Errorf("int should not be quoted: %s", v)
	}
}

func TestGoValueForField_Bool(t *testing.T) {
	v := goValueForField(FieldInfo{GoType: "bool"}, 0)
	if v != "true" && v != "false" {
		t.Errorf("expected true/false, got: %s", v)
	}
}

func TestGoValueForField_Time(t *testing.T) {
	v := goValueForField(FieldInfo{GoType: "time.Time"}, 0)
	if !strings.Contains(v, "time.Date") {
		t.Errorf("expected time.Date, got: %s", v)
	}
}

func TestGoValueForField_Enum(t *testing.T) {
	v := goValueForField(FieldInfo{
		Name:       "Status",
		ColumnName: "status",
		GoType:     "Status",
		IsEnum:     true,
		EnumValues: []string{"active", "inactive"},
	}, 0)
	quoted := strings.Trim(v, `"`)
	if quoted != "active" && quoted != "inactive" {
		t.Errorf("expected 'active' or 'inactive', got: %s", v)
	}
}

// ---------------------------------------------------------------------------
// generateMain / generateSeeder tests
// ---------------------------------------------------------------------------

func TestGenerateMain(t *testing.T) {
	content := generateMain(Options{
		ModulePath: "example.com/myapp",
	})
	if !strings.Contains(content, "package main") {
		t.Error("missing package main")
	}
	if !strings.Contains(content, `"example.com/myapp/db/seeds/gorm"`) {
		t.Error("missing seed import")
	}
	if !strings.Contains(content, `"example.com/myapp/internal/config"`) {
		t.Error("missing config import")
	}
	if !strings.Contains(content, "config.Load()") {
		t.Error("missing config.Load()")
	}
}

func TestGenerateSeeder(t *testing.T) {
	content := generateSeeder(Options{})
	if !strings.Contains(content, "package gorm") {
		t.Error("missing package gorm")
	}
	if !strings.Contains(content, "crank:seed-up-begin") {
		t.Error("missing up marker")
	}
	if !strings.Contains(content, "crank:seed-down-end") {
		t.Error("missing down marker")
	}
}

// ---------------------------------------------------------------------------
// updateSeeder tests
// ---------------------------------------------------------------------------

func TestUpdateSeeder(t *testing.T) {
	dir := t.TempDir()
	seederPath := filepath.Join(dir, "seeder.go")

	initial := generateSeeder(Options{})
	if err := os.WriteFile(seederPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	info := &StructInfo{
		Name:      "Product",
		TableName: "products",
	}
	opts := Options{}

	if err := updateSeeder(seederPath, info, opts); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(seederPath)
	content := string(data)

	if !strings.Contains(content, `{"products", SeedProductsUp}`) {
		t.Error("missing SeedProductsUp entry")
	}
	if !strings.Contains(content, `{"products", SeedProductsDown}`) {
		t.Error("missing SeedProductsDown entry")
	}

	// Second call should be idempotent.
	if err := updateSeeder(seederPath, info, opts); err != nil {
		t.Fatal(err)
	}
	data2, _ := os.ReadFile(seederPath)
	if strings.Count(string(data2), `{"products", SeedProductsUp}`) > 1 {
		t.Error("updateSeeder should be idempotent")
	}
}

// ---------------------------------------------------------------------------
// Legacy SQL value tests (kept for test coverage)
// ---------------------------------------------------------------------------

func TestFakeValue_UUID(t *testing.T) {
	v := fakeValue(FieldInfo{
		Name:       "ID",
		ColumnName: "id",
		GoType:     "uuid.UUID",
	})
	if !strings.HasPrefix(v, "'") || !strings.HasSuffix(v, "'") {
		t.Errorf("uuid value should be quoted: %s", v)
	}
	inner := strings.Trim(v, "'")
	if len(inner) != 36 {
		t.Errorf("uuid length = %d, want 36: %s", len(inner), inner)
	}
}

func TestFakeValue_ORMTypeUUID(t *testing.T) {
	v := fakeValue(FieldInfo{
		Name:       "ID",
		ColumnName: "id",
		GoType:     "string",
		ORMType:    "uuid",
	})
	if !strings.HasPrefix(v, "'") {
		t.Errorf("expected quoted uuid, got: %s", v)
	}
	inner := strings.Trim(v, "'")
	if len(inner) != 36 {
		t.Errorf("expected 36-char uuid, got %q (len=%d)", inner, len(inner))
	}
}

func TestFakeValue_Enum(t *testing.T) {
	v := fakeValue(FieldInfo{
		Name:       "Status",
		ColumnName: "status",
		GoType:     "Status",
		IsEnum:     true,
		EnumValues: []string{"active", "inactive", "pending"},
	})
	quoted := strings.Trim(v, "'")
	found := false
	for _, e := range []string{"active", "inactive", "pending"} {
		if quoted == e {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("enum value %q not in expected set", quoted)
	}
}

func TestFakeValue_NumericTypes(t *testing.T) {
	for _, typ := range []string{"int", "int8", "int32", "int64", "uint", "uint32"} {
		v := fakeValue(FieldInfo{GoType: typ})
		if len(v) == 0 || v[0] == '\'' {
			t.Errorf("numeric type %q produced quoted value: %s", typ, v)
		}
	}
}

func TestFakeValue_Time(t *testing.T) {
	v := fakeValue(FieldInfo{GoType: "time.Time"})
	if !strings.HasPrefix(v, "'") || !strings.HasSuffix(v, "'") {
		t.Errorf("time value should be quoted: %s", v)
	}
	inner := strings.Trim(v, "'")
	if len(inner) != 19 {
		t.Errorf("expected 19-char timestamp, got %q (len=%d)", inner, len(inner))
	}
}

func TestEscapeSQLString(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello", "hello"},
		{"it's", "it''s"},
		{"no 'single' quotes", "no ''single'' quotes"},
		{"", ""},
	}
	for _, c := range cases {
		got := escapeSQLString(c.in)
		if got != c.want {
			t.Errorf("escapeSQLString(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFakeStringForField_IDColumn(t *testing.T) {
	v := fakeStringForField("ID", "id")
	if len(v) != 36 {
		t.Errorf("expected uuid for id column, got %q (len=%d)", v, len(v))
	}
}

func TestFakeStringForField_Email(t *testing.T) {
	v := fakeStringForField("Email", "email")
	if !strings.Contains(v, "@") {
		t.Errorf("expected email, got: %s", v)
	}
}

func TestFakeStringForField_FullName(t *testing.T) {
	v := fakeStringForField("FullName", "full_name")
	if !strings.Contains(v, " ") {
		t.Errorf("expected full name with space, got: %s", v)
	}
}

func TestFakeStringForField_Status(t *testing.T) {
	v := fakeStringForField("Status", "status")
	valid := map[string]bool{"active": true, "inactive": true, "pending": true, "archived": true, "draft": true}
	if !valid[v] {
		t.Errorf("unexpected status value: %s", v)
	}
}

func TestRdPick(t *testing.T) {
	items := []string{"a", "b", "c"}
	seen := make(map[string]int)
	for i := 0; i < 100; i++ {
		pick := rdPick(items...)
		seen[pick]++
	}
	if len(seen) != 3 {
		t.Errorf("expected all 3 items to be picked at least once, got %d: %v", len(seen), seen)
	}
}
