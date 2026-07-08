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
// Parser tests
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

	bunDir := filepath.Join(dir, "internal/adapters/persistence/bun")
	if err := os.MkdirAll(bunDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bunDir, "product_repository.go"), []byte(`package bun

import "github.com/google/uuid"

type productRow struct {
	ID    uuid.UUID `+"`"+`bun:"id,pk,type:uuid"`+"`"+`
	Name  string    `+"`"+`bun:"name,notnull"`+"`"+`
	Price float64   `+"`"+`bun:"price"`+"`"+`
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
	// Only exported fields should be picked up.
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
	// Create domain directory but with no matching struct name.
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
	bunDir := filepath.Join(dir, "internal/adapters/persistence/gorm")
	if err := os.MkdirAll(bunDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bunDir, "user_repository.go"), []byte(`package gorm

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
	// GORM column tag should be used.
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

	// The Status field should be detected as an enum.
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
	// All enum values should be present.
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
		{`bun:"id,pk"`, "bun:", "id,pk"},
		{`bun:"name,notnull" gorm:"column:name"`, "bun:", "name,notnull"},
		{`bun:"name,notnull" gorm:"column:name"`, "gorm:", "column:name"},
		{`gorm:"column:user_id;primaryKey"`, "gorm:", "column:user_id;primaryKey"},
		{`json:"name"`, "bun:", ""},
		{``, "bun:", ""},
		{`bun:""`, "bun:", ""},
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
		{"UserID", makeTag(`bun:"user_id,pk"`), "user_id"},
		{"UserName", makeTag(`gorm:"column:user_name"`), "user_name"},
		{"Email", makeTag(`bun:"-"`), "email"}, // skip tag → fallback
		{"CreatedAt", nil, "created_at"},       // no tag → snake_case
		{"ID", nil, "id"},
		{"MyField", makeTag(`bun:"my_field,notnull"`), "my_field"},
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
		{makeTag(`bun:"id,pk,type:uuid"`), "uuid"},
		{makeTag(`gorm:"column:id;type:uuid;primaryKey"`), "uuid"},
		{makeTag(`bun:"name,notnull,type:TEXT"`), "TEXT"},
		{makeTag(`bun:"name,notnull"`), ""},
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
		{"github.com/uptrace/bun", "bun"},
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
// typeString tests (AST type -> Go string)
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
// GenerateSeed tests
// ---------------------------------------------------------------------------

func TestGenerateSeed_WithFields(t *testing.T) {
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

	files, err := GenerateSeed(Options{
		ProjectDir: dir,
		ModelName:  "Product",
		Count:      5,
	})
	if err != nil {
		t.Fatalf("GenerateSeed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (up + down), got %d", len(files))
	}

	upPath := filepath.Join(dir, "db/seeds", files[0].Path)
	data, err := os.ReadFile(upPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "INSERT INTO products") {
		t.Errorf("missing INSERT INTO products:\n%s", content)
	}
	if !strings.Contains(content, "VALUES") {
		t.Errorf("missing VALUES:\n%s", content)
	}
	// Each row has 2 commas (3 fields), so 5 rows = 10 commas.
	if strings.Count(content, ",") < 9 {
		t.Errorf("expected ~10 commas for 5 rows of 3 fields, got %d", strings.Count(content, ","))
	}

	// Verify down file content.
	downPath := filepath.Join(dir, "db/seeds", files[1].Path)
	downData, err := os.ReadFile(downPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(downData), "TRUNCATE TABLE products") {
		t.Errorf("down file should TRUNCATE products:\n%s", downData)
	}
}

func TestGenerateSeed_Empty(t *testing.T) {
	dir := t.TempDir()
	files, err := GenerateSeed(Options{
		ProjectDir: dir,
		ModelName:  "",
		Count:      0,
	})
	if err != nil {
		t.Fatalf("GenerateSeed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file for empty seed, got %d", len(files))
	}
	if !strings.HasSuffix(files[0].Path, "_empty_seed.up.sql") {
		t.Errorf("unexpected filename: %s", files[0].Path)
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

func TestGenerateSeed_ModelNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := GenerateSeed(Options{
		ProjectDir: dir,
		ModelName:  "NonExistent",
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

	// Count=0 should default to 10.
	files, err := GenerateSeed(Options{
		ProjectDir: dir,
		ModelName:  "Item",
		Count:      0,
	})
	if err != nil {
		t.Fatalf("GenerateSeed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	data, err := os.ReadFile(filepath.Join(dir, "db/seeds", files[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	// 10 rows of 2 fields: each row has 1 comma between values, plus
	// 9 `),` separators between rows = 19 commas total.
	got := strings.Count(string(data), ",")
	if got < 15 || got > 25 {
		t.Errorf("expected ~19 commas for 10 rows of 2 fields, got %d", got)
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

	// First generation.
	files1, err := GenerateSeed(Options{ProjectDir: dir, ModelName: "Item", Count: 1})
	if err != nil {
		t.Fatal(err)
	}

	// Second generation without force should skip.
	files2, err := GenerateSeed(Options{ProjectDir: dir, ModelName: "Item", Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !files2[0].Skipped {
		t.Error("expected second call to skip existing file without --force")
	}

	// Third generation with force should overwrite.
	files3, err := GenerateSeed(Options{ProjectDir: dir, ModelName: "Item", Count: 1, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if files3[0].Skipped {
		t.Error("expected --force to overwrite, but file was skipped")
	}
	_ = files1
}

func TestGenerateSeed_EmptyFileSkip(t *testing.T) {
	dir := t.TempDir()

	// First empty generation.
	files1, err := GenerateSeed(Options{ProjectDir: dir, ModelName: ""})
	if err != nil {
		t.Fatal(err)
	}

	// Second without force skips.
	files2, err := GenerateSeed(Options{ProjectDir: dir, ModelName: ""})
	if err != nil {
		t.Fatal(err)
	}
	if !files2[0].Skipped {
		t.Error("expected empty seed generation to skip when file exists")
	}

	// Third with force writes.
	files3, err := GenerateSeed(Options{ProjectDir: dir, ModelName: "", Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if files3[0].Skipped {
		t.Error("expected --force to overwrite empty seed")
	}
	_ = files1
}

func TestGenerateInsert_Content(t *testing.T) {
	info := &StructInfo{
		Name:      "Widget",
		TableName: "widgets",
		ExportedFields: []FieldInfo{
			{Name: "ID", ColumnName: "id", GoType: "string"},
			{Name: "Label", ColumnName: "label", GoType: "string"},
		},
	}
	sql := generateInsert(info, 3, "20250101000000")
	if !strings.Contains(sql, "-- Seed data for Widget") {
		t.Errorf("missing header: %s", sql)
	}
	if !strings.Contains(sql, "INSERT INTO widgets (id, label) VALUES") {
		t.Errorf("missing INSERT: %s", sql)
	}
	// Should have 3 rows.
	rows := strings.Count(sql, "),")
	if rows != 2 { // 2 commas between 3 rows, last row ends with ");"
		t.Errorf("expected 2 row separators for 3 rows, got %d", rows)
	}
	if !strings.HasSuffix(strings.TrimSpace(sql), ");") {
		t.Errorf("should end with ');'")
	}
}

// ---------------------------------------------------------------------------
// Fake value generation tests
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
	// ORM type hint should take precedence over string Go type.
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

func TestFakeValue_FloatTypes(t *testing.T) {
	for _, typ := range []string{"float32", "float64"} {
		v := fakeValue(FieldInfo{GoType: typ})
		if !strings.Contains(v, ".") {
			t.Errorf("float type %q produced value without decimal: %s", typ, v)
		}
	}
}

func TestFakeValue_Bool(t *testing.T) {
	v := fakeValue(FieldInfo{GoType: "bool"})
	if v != "true" && v != "false" {
		t.Errorf("expected true/false, got: %s", v)
	}
}

func TestFakeValue_Time(t *testing.T) {
	v := fakeValue(FieldInfo{GoType: "time.Time"})
	if !strings.HasPrefix(v, "'") || !strings.HasSuffix(v, "'") {
		t.Errorf("time value should be quoted: %s", v)
	}
	inner := strings.Trim(v, "'")
	// Should match YYYY-MM-DD HH:MM:SS format.
	if len(inner) != 19 {
		t.Errorf("expected 19-char timestamp, got %q (len=%d)", inner, len(inner))
	}
}

func TestFakeValue_NullablePointer(t *testing.T) {
	// Nullable types (*string, *int) should be handled by Stripping the '*' prefix.
	v := fakeValue(FieldInfo{Name: "Name", ColumnName: "name", GoType: "*string"})
	if !strings.HasPrefix(v, "'") {
		t.Errorf("expected quoted value for *string, got: %s", v)
	}
}

func TestFakeValue_UnknownType(t *testing.T) {
	v := fakeValue(FieldInfo{
		Name:       "Widget",
		ColumnName: "widget",
		GoType:     "WidgetType",
	})
	// Should produce a quoted placeholder.
	if !strings.HasPrefix(v, "'") || !strings.HasSuffix(v, "'") {
		t.Errorf("expected quoted fallback value, got: %s", v)
	}
}

func TestFakeValue_EnumNoValues(t *testing.T) {
	// IsEnum with no values should not panic and fall through to type-based generation.
	v := fakeValue(FieldInfo{
		Name:       "Status",
		ColumnName: "status",
		GoType:     "Status",
		IsEnum:     true,
		EnumValues: nil,
	})
	_ = v
}

// ---------------------------------------------------------------------------
// fakeStringForField heuristic tests
// ---------------------------------------------------------------------------

func TestFakeStringForField_IDColumn(t *testing.T) {
	v := fakeStringForField("ID", "id")
	if len(v) != 36 {
		t.Errorf("expected uuid for id column, got %q (len=%d)", v, len(v))
	}
}

func TestFakeStringForField_ForeignKeyColumn(t *testing.T) {
	v := fakeStringForField("UserID", "user_id")
	if len(v) != 36 {
		t.Errorf("expected uuid for user_id column, got %q", v)
	}
}

func TestFakeStringForField_Email(t *testing.T) {
	v := fakeStringForField("Email", "email")
	if !strings.Contains(v, "@") {
		t.Errorf("expected email, got: %s", v)
	}
}

func TestFakeStringForField_Phone(t *testing.T) {
	v := fakeStringForField("PhoneNumber", "phone_number")
	if !strings.HasPrefix(v, "+1-") {
		t.Errorf("expected phone format (+1-xxx-xxx-xxxx), got: %s", v)
	}
}

func TestFakeStringForField_Website(t *testing.T) {
	v := fakeStringForField("Website", "website")
	if !strings.HasPrefix(v, "https://") {
		t.Errorf("expected URL, got: %s", v)
	}
}

func TestFakeStringForField_Address(t *testing.T) {
	v := fakeStringForField("Address", "address")
	if len(v) < 5 {
		t.Errorf("expected address string, got: %s", v)
	}
}

func TestFakeStringForField_FirstName(t *testing.T) {
	v := fakeStringForField("FirstName", "first_name")
	if v == "" {
		t.Errorf("expected non-empty first name")
	}
}

func TestFakeStringForField_LastName(t *testing.T) {
	v := fakeStringForField("LastName", "last_name")
	if v == "" {
		t.Errorf("expected non-empty last name")
	}
}

func TestFakeStringForField_FullName(t *testing.T) {
	v := fakeStringForField("FullName", "full_name")
	if !strings.Contains(v, " ") {
		t.Errorf("expected full name with space, got: %s", v)
	}
}

func TestFakeStringForField_Password(t *testing.T) {
	v := fakeStringForField("Password", "password")
	if v != "password123" {
		t.Errorf("expected 'password123', got: %s", v)
	}
}

func TestFakeStringForField_Token(t *testing.T) {
	v := fakeStringForField("Token", "token")
	if !strings.HasPrefix(v, "tok_") {
		t.Errorf("expected token starting with tok_, got: %s", v)
	}
}

func TestFakeStringForField_Status(t *testing.T) {
	v := fakeStringForField("Status", "status")
	valid := map[string]bool{"active": true, "inactive": true, "pending": true, "archived": true, "draft": true}
	if !valid[v] {
		t.Errorf("unexpected status value: %s", v)
	}
}

func TestFakeStringForField_Code(t *testing.T) {
	v := fakeStringForField("Sku", "sku")
	if len(v) < 4 {
		t.Errorf("expected sku code, got: %s", v)
	}
}

func TestFakeStringForField_Color(t *testing.T) {
	v := fakeStringForField("Color", "color")
	if v == "" {
		t.Errorf("expected non-empty color")
	}
}

func TestFakeStringForField_Default(t *testing.T) {
	v := fakeStringForField("UnknownField", "unknown_field")
	if v == "" {
		t.Errorf("expected fallback value, got empty")
	}
}

func TestFakeStringForField_ColumnNamePrecedence(t *testing.T) {
	// Column name should take precedence over field name for ID detection.
	v := fakeStringForField("UserUUID", "user_uuid")
	if len(v) != 36 {
		t.Errorf("expected uuid from column name, got %q", v)
	}
}

// ---------------------------------------------------------------------------
// escapeSQLString tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// emptySeedContent tests
// ---------------------------------------------------------------------------

func TestEmptySeedContent(t *testing.T) {
	content := emptySeedContent("20250101000000")
	if !strings.Contains(content, "Seed data") {
		t.Errorf("missing title")
	}
	if !strings.Contains(content, "20250101000000") {
		t.Errorf("missing timestamp")
	}
	if !strings.Contains(content, "TODO") {
		t.Errorf("missing TODO hint")
	}
	if !strings.Contains(content, "+migrate Up") {
		t.Errorf("missing migrate directive")
	}
}

// ---------------------------------------------------------------------------
// rdPick tests
// ---------------------------------------------------------------------------

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
