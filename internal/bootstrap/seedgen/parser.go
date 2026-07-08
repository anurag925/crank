// Package seedgen parses Go source files in a crank-generated project to
// discover domain structs and their exported fields, then generates timestamped
// SQL seed files with type-appropriate fake data.
package seedgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// FieldInfo describes a single exported struct field for seed generation.
type FieldInfo struct {
	// Name is the Go field name (exported, PascalCase).
	Name string
	// ColumnName is the SQL column name (snake_case), derived from the ORM
	// tag (bun/gorm) or the field name.
	ColumnName string
	// GoType is the unqualified Go type name (e.g. "string", "int",
	// "uuid.UUID", "Status").
	GoType string
	// ORMType is the type hint from the ORM tag (e.g. "uuid", "TEXT",
	// "DOUBLE PRECISION"), or empty if not specified.
	ORMType string
	// IsEnum is true when the field's type is a named type in the same
	// package that has const values.
	IsEnum bool
	// EnumValues holds the string representation of each const value
	// when IsEnum is true.
	EnumValues []string
}

// StructInfo holds the parsed struct metadata for seed generation.
type StructInfo struct {
	// Name is the PascalCase struct name (e.g. "User", "Order").
	Name string
	// TableName is the snake_plural table name (e.g. "users", "orders").
	TableName string
	// ExportedFields are the struct's exported fields (public Go fields).
	ExportedFields []FieldInfo
}

// FindStruct looks for a struct matching modelName in the project's domain
// layer (internal/domain/<snake>/). If the domain struct exists but has no
// exported fields (as is typical for DDD aggregates), it additionally searches
// the bun and gorm adapter directories for a corresponding Row DTO struct
// (which always has exported fields with ORM tags).
//
// Returns (nil, nil) when no matching struct is found.
func FindStruct(projectDir, modelName string) (*StructInfo, error) {
	snake := pascalToSnake(modelName)
	domainDir := filepath.Join(projectDir, "internal/domain", snake)

	// Parse the domain directory first.
	pkgInfo, parseErr := parsePackage(domainDir)
	if parseErr == nil && pkgInfo != nil {
		// Look for the struct by exact name (e.g. "User") in the domain.
		st := pkgInfo.findStruct(modelName)
		if st != nil {
			fields, enums := exportedFields(st, pkgInfo)
			if len(fields) > 0 {
				return &StructInfo{
					Name:           modelName,
					TableName:      snakePlural(snake),
					ExportedFields: resolveColumns(fields, pkgInfo, enums),
				}, nil
			}
			// Domain struct exists but has no exported fields (DDD aggregate
			// with all private fields). Fall through to check adapter dirs.
		}
	}

	// No exported fields on the domain aggregate (or domain dir didn't exist).
	// Look for a Row DTO in the bun or gorm adapter directory.
	for _, adapterDir := range []string{
		filepath.Join(projectDir, "internal/adapters/persistence/bun"),
		filepath.Join(projectDir, "internal/adapters/persistence/gorm"),
	} {
		rowPkg, rowErr := parsePackage(adapterDir)
		if rowErr != nil || rowPkg == nil {
			continue
		}
		// Search for <Model>Row or <model>Row (the DTO naming convention).
		for _, candidate := range []string{modelName + "Row", strings.ToLower(string(modelName[0])) + modelName[1:] + "Row"} {
			rowSt := rowPkg.findStruct(candidate)
			if rowSt == nil {
				continue
			}
			fields, enums := exportedFields(rowSt, rowPkg)
			if len(fields) > 0 {
				return &StructInfo{
					Name:           modelName,
					TableName:      snakePlural(snake),
					ExportedFields: resolveColumns(fields, rowPkg, enums),
				}, nil
			}
		}
	}

	// No exported fields found anywhere.
	return nil, nil
}

// packageInfo holds parsed AST data for a single Go package.
type packageInfo struct {
	astFile   *ast.File
	typeSpecs map[string]*ast.TypeSpec // name -> TypeSpec
	consts    map[string][]constValue  // typeName -> const values
	imports   map[string]string        // local name -> full import path
}

type constValue struct {
	name  string
	value string
}

// parsePackage parses all Go files in dir and returns a merged packageInfo,
// or (nil, nil) if the directory doesn't exist or has no Go files.
func parsePackage(dir string) (*packageInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	fset := token.NewFileSet()
	var files []*ast.File

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		f, err := parser.ParseFile(fset, path, src, parser.AllErrors)
		if err != nil {
			continue
		}
		files = append(files, f)
	}

	if len(files) == 0 {
		return nil, nil
	}

	pi := &packageInfo{
		typeSpecs: make(map[string]*ast.TypeSpec),
		consts:    make(map[string][]constValue),
		imports:   make(map[string]string),
	}

	// Merge info from all files in the package.
	for _, f := range files {
		pi.astFile = f // keep the last file for reference
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			name := path
			if imp.Name != nil {
				name = imp.Name.Name
			} else {
				parts := strings.Split(path, "/")
				name = parts[len(parts)-1]
			}
			pi.imports[name] = path
		}
		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			switch genDecl.Tok {
			case token.TYPE:
				for _, spec := range genDecl.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if ok {
						pi.typeSpecs[ts.Name.Name] = ts
					}
				}
			case token.CONST:
				for _, spec := range genDecl.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Names) == 0 {
						continue
					}
					// Determine the type of the const block.
					typeName := ""
					if vs.Type != nil {
						if ident, ok := vs.Type.(*ast.Ident); ok {
							typeName = ident.Name
						}
					}
					if typeName == "" && genDecl.Tok == token.CONST {
						// Iota-based or implicit type from previous const.
						// Try iota detection by looking at the value expression.
						// For our purposes, we only care about explicitly typed consts.
						continue
					}
					for i, n := range vs.Names {
						val := ""
						if i < len(vs.Values) {
							val = exprString(vs.Values[i])
						}
						pi.consts[typeName] = append(pi.consts[typeName], constValue{
							name:  n.Name,
							value: val,
						})
					}
				}
			}
		}
	}

	return pi, nil
}

// findStruct looks for a *ast.StructType with the given name in the package.
func (pi *packageInfo) findStruct(name string) *ast.StructType {
	ts, ok := pi.typeSpecs[name]
	if !ok {
		return nil
	}
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	return st
}

// exportedFields returns the exported fields from a struct, along with any
// enum types found in the same package. A field is exported when its first
// name token starts with an uppercase letter (Go's exported rule).
func exportedFields(st *ast.StructType, pi *packageInfo) (fields []*ast.Field, enumTypes map[string]bool) {
	enumTypes = make(map[string]bool)

	// Pre-detect which named types in this package are enums (have const values).
	for typeName, vals := range pi.consts {
		if len(vals) > 0 {
			enumTypes[typeName] = true
		}
	}

	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			// Embedded field; skip.
			continue
		}
		if !f.Names[0].IsExported() {
			continue
		}
		fields = append(fields, f)
	}
	return fields, enumTypes
}

// resolveColumns converts AST fields into FieldInfo, resolving column names
// from ORM struct tags and detecting enum types.
func resolveColumns(fields []*ast.Field, pi *packageInfo, enumTypes map[string]bool) []FieldInfo {
	out := make([]FieldInfo, 0, len(fields))

	for _, f := range fields {
		name := f.Names[0].Name
		goType := typeString(f.Type, pi)

		// Extract column name from bun/gorm struct tag.
		colName := columnName(name, f.Tag)

		fi := FieldInfo{
			Name:       name,
			ColumnName: colName,
			GoType:     goType,
			ORMType:    ormType(f.Tag),
		}

		// Check if this field's type is an enum in this package.
		if enumTypes[goType] {
			fi.IsEnum = true
			fi.EnumValues = make([]string, 0, len(pi.consts[goType]))
			for _, cv := range pi.consts[goType] {
				val := cv.value
				// Strip surrounding quotes from string literals.
				val = strings.Trim(val, `"`)
				fi.EnumValues = append(fi.EnumValues, val)
			}
		}

		out = append(out, fi)
	}

	return out
}

// typeString returns the string representation of an AST expression that
// represents a Go type. It resolves SelectorExpr (e.g. "uuid.UUID") and
// star expressions (e.g. "*string") to their string forms.
func typeString(expr ast.Expr, pi *packageInfo) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		pkg, ok := t.X.(*ast.Ident)
		if !ok {
			return exprString(expr)
		}
		// Check if the package is an import; if so, fully qualify.
		if importPath, ok := pi.imports[pkg.Name]; ok {
			short := shortenImport(importPath)
			if short != "" {
				return short + "." + t.Sel.Name
			}
		}
		return pkg.Name + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X, pi)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt, pi)
	default:
		return exprString(expr)
	}
}

// shortenImport returns a short familiar name for well-known import paths,
// or the last path segment.
func shortenImport(path string) string {
	// Map well-known packages to their conventional short names.
	switch {
	case strings.Contains(path, "google/uuid"):
		return "uuid"
	case strings.Contains(path, "/uuid"):
		return "uuid"
	case strings.Contains(path, "time"):
		return "time"
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// columnName extracts the SQL column name from a struct tag (bun or gorm),
// falling back to snake_case of the field name.
//
// Bun uses comma-separated options: `bun:"id,pk,type:uuid"`
// GORM uses semicolon-separated options: `gorm:"column:id;primaryKey;type:uuid"`
// Both conventions are supported here.
func columnName(fieldName string, tag *ast.BasicLit) string {
	if tag != nil {
		raw := strings.Trim(tag.Value, "`")
		for _, prefix := range []string{"bun:", "gorm:"} {
			if tagVal := extractTagValue(raw, prefix); tagVal != "" {
				// Try comma-separated (bun convention) first.
				parts := splitTagOptions(tagVal, ",")
				if len(parts) > 0 && parts[0] != "-" && !strings.HasPrefix(parts[0], "column:") {
					return parts[0]
				}
				// Try semicolon-separated (gorm convention) or look for column: prefix.
				parts = splitTagOptions(tagVal, ";")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if strings.HasPrefix(p, "column:") {
						return strings.TrimPrefix(p, "column:")
					}
					if p != "-" && !strings.Contains(p, ":") {
						return p
					}
				}
			}
		}
	}
	return pascalToSnake(fieldName)
}

// splitTagOptions splits a tag value by the given separator, trimming whitespace
// and filtering empty entries.
func splitTagOptions(val, sep string) []string {
	var out []string
	for _, p := range strings.Split(val, sep) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ormType extracts the SQL type from the ORM struct tag (e.g. "uuid" from
// `bun:"type:uuid"` or `gorm:"type:uuid"`). Returns empty string when no
// type hint is found.
func ormType(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}
	raw := strings.Trim(tag.Value, "`")
	for _, prefix := range []string{"bun:", "gorm:"} {
		if tagVal := extractTagValue(raw, prefix); tagVal != "" {
			// Try both comma and semicolon separators.
			for _, sep := range []string{",", ";"} {
				parts := splitTagOptions(tagVal, sep)
				for _, p := range parts {
					if strings.HasPrefix(p, "type:") {
						return strings.TrimPrefix(p, "type:")
					}
				}
			}
		}
	}
	return ""
}

// extractTagValue extracts the value portion of a struct tag key-value pair.
// e.g. extractTagValue(`bun:"id,pk" gorm:"column:id"`, "bun:") -> "id,pk"
func extractTagValue(tag, key string) string {
	idx := strings.Index(tag, key)
	if idx < 0 {
		return ""
	}
	rest := tag[idx+len(key):]
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// exprString is a best-effort serialization of an AST expression to a string.
func exprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.BasicLit:
		return t.Value
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.CallExpr:
		return exprString(t.Fun) + "(...)"
	case *ast.CompositeLit:
		return exprString(t.Type) + "{...}"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// pascalToSnake converts PascalCase to snake_case (e.g. "OrderItem" -> "order_item",
// "UserID" -> "user_id"). It handles consecutive uppercase letters (acronyms) by
// treating the last uppercase letter of a run as the start of a new word.
func pascalToSnake(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			// Insert underscore before uppercase that follows lowercase,
			// or before uppercase that is followed by lowercase while the
			// previous character is also uppercase (acronym boundary).
			if i > 0 {
				prevIsUpper := unicode.IsUpper(runes[i-1])
				nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if !prevIsUpper || nextIsLower {
					b.WriteRune('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// snakePlural applies basic English pluralization to a snake_case word.
func snakePlural(snake string) string {
	if snake == "" {
		return snake
	}
	last := snake[strings.LastIndex(snake, "_")+1:]
	// Basic pluralization rules.
	switch {
	case strings.HasSuffix(last, "s") || strings.HasSuffix(last, "x") || strings.HasSuffix(last, "ch") || strings.HasSuffix(last, "sh"):
		return snake + "es"
	case strings.HasSuffix(last, "z"):
		return snake + "zes"
	case strings.HasSuffix(last, "y") && len(last) > 1:
		prev := string(last[len(last)-2])
		if prev != "a" && prev != "e" && prev != "i" && prev != "o" && prev != "u" {
			return snake[:len(snake)-1] + "ies"
		}
		return snake + "s"
	default:
		return snake + "s"
	}
}
