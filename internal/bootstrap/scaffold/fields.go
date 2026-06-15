package scaffold

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Field describes a single column/attribute parsed from a "name:type" spec
// passed on the command line (e.g. "title:string", "price:float", "active:bool").
type Field struct {
	Name     string // snake_case column / json name
	GoName   string // PascalCase struct field name
	Camel    string // camelCase parameter / accessor name
	GoType   string // Go type (string, int64, float64, bool, time.Time)
	SQLType  string // PostgreSQL column type
	Validate string // go-playground/validator tag, may be empty
	IsTime   bool   // whether the field uses time.Time
	IsUUID   bool   // whether the field is a typed identifier (uuid type)
	Sample   string // a valid Go literal used in generated tests
}

// typeMapping maps a user-facing type keyword to the corresponding Go and SQL
// types plus a sensible default validation rule.
type typeMapping struct {
	goType   string
	sqlType  string
	validate string
	isTime   bool
	sample   string // valid Go literal satisfying the validation rule
}

var fieldTypes = map[string]typeMapping{
	"string":  {goType: "string", sqlType: "TEXT", validate: "required", sample: `"sample"`},
	"text":    {goType: "string", sqlType: "TEXT", validate: "required", sample: `"sample text"`},
	"int":     {goType: "int", sqlType: "INTEGER", validate: "required", sample: "1"},
	"int64":   {goType: "int64", sqlType: "BIGINT", validate: "required", sample: "1"},
	"float":   {goType: "float64", sqlType: "DOUBLE PRECISION", validate: "required", sample: "1.5"},
	"float64": {goType: "float64", sqlType: "DOUBLE PRECISION", validate: "required", sample: "1.5"},
	"bool":    {goType: "bool", sqlType: "BOOLEAN", validate: "", sample: "true"},
	"time":    {goType: "time.Time", sqlType: "TIMESTAMPTZ", validate: "required", isTime: true, sample: "time.Now()"},
	"uuid":    {goType: "string", sqlType: "UUID", validate: "required,uuid", sample: `"123e4567-e89b-12d3-a456-426614174000"`},
	"email":   {goType: "string", sqlType: "TEXT", validate: "required,email", sample: `"sample@example.com"`},
}

// ParseFields converts a slice of "name:type" specs into Fields. A spec without
// a type defaults to "string". Unknown types return an error listing the
// supported keywords.
func ParseFields(specs []string) ([]Field, error) {
	var fields []Field
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		name := spec
		typ := "string"
		if idx := strings.Index(spec, ":"); idx >= 0 {
			name = spec[:idx]
			typ = strings.ToLower(strings.TrimSpace(spec[idx+1:]))
			if typ == "" {
				typ = "string"
			}
		}

		words := splitWords(name)
		if len(words) == 0 {
			return nil, fmt.Errorf("invalid field spec %q: empty name\n\nEach field must follow the format 'name:type' or just 'name'.\nExample: crank make scaffold Order title:string price:float", spec)
		}
		mapping, ok := fieldTypes[typ]
		if !ok {
			return nil, fmt.Errorf("unknown field type %q in %q\n\nSupported types: %s\n\nExample: crank make scaffold Order title:string price:float paid:bool", typ, spec, supportedTypes())
		}

		fields = append(fields, Field{
			Name:     strings.Join(words, "_"),
			GoName:   pascalCase(words),
			Camel:    camelCase(words),
			GoType:   mapping.goType,
			SQLType:  mapping.sqlType,
			Validate: mapping.validate,
			IsTime:   mapping.isTime,
			IsUUID:   typ == "uuid",
			Sample:   mapping.sample,
		})
	}
	return fields, nil
}

// hasTimeField reports whether any field uses time.Time, so model templates can
// decide whether to import the time package.
func hasTimeField(fields []Field) bool {
	for _, f := range fields {
		if f.IsTime {
			return true
		}
	}
	return false
}

// hasUUIDField reports whether any field has the "uuid" type. The scaffold
// uses this to decide whether to emit a typed-ID value object for the
// resource. The DDD convention is that the first uuid field (if any) becomes
// the aggregate's primary identifier.
func hasUUIDField(fields []Field) bool {
	return uuidFieldOrNil(fields) != nil
}

// uuidFieldOrNil returns the first uuid Field in the list, or nil when there
// is none. Templates use it to pull the typed-ID column name.
func uuidFieldOrNil(fields []Field) *Field {
	for i, f := range fields {
		if f.IsUUID {
			return &fields[i]
		}
	}
	return nil
}

// reservedAggregateFields are the lifecycle fields every generated aggregate
// carries; they are not user data and must be excluded when inferring the
// domain's field list from a previously generated aggregate file.
var reservedAggregateFields = map[string]bool{
	"id":        true,
	"createdat": true,
	"updatedat": true,
	"events":    true,
}

// InferFieldsFromDomain reads the existing domain aggregate file for `res` in
// `projectDir` and returns the field list it declares. The aggregate is
// expected to be in the format produced by `domain_aggregate.go.tmpl`: a
// struct with unexported, lower-camel fields preceded by `id` and followed
// by `createdAt`, `updatedAt` and `events`. Returns (nil, nil) when the
// file is missing or its body cannot be parsed — callers fall back to
// the user-supplied field list in that case.
func InferFieldsFromDomain(projectDir string, res Resource) ([]Field, error) {
	path := filepath.Join(projectDir, res.DDDDomainPath(), res.Snake+".go")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil //nolint:nilerr // missing model is a normal "no inference" case
	}
	content := string(data)

	inStruct := false
	braceDepth := 0
	var fields []Field
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !inStruct {
			if strings.HasPrefix(line, "type "+res.Pascal+" struct") {
				inStruct = true
				braceDepth = 1
			}
			continue
		}
		// Track brace depth to know when the struct ends.
		braceDepth += strings.Count(line, "{")
		braceDepth -= strings.Count(line, "}")
		if braceDepth <= 0 {
			break
		}
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		// Strip an inline trailing comment.
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		// A field line is `<name> <type>`. We only need the first two
		// whitespace-separated tokens; the type may be a complex form
		// like `[]shared.DomainEvent` or a pointer.
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		// Skip reserved aggregate fields (id, createdAt, updatedAt, events).
		if reservedAggregateFields[strings.ToLower(name)] {
			continue
		}
		goType := parts[1]
		// Map the Go type back to a Field. We don't know the SQL type or
		// validator from the aggregate file (those live in the DTO), so we
		// synthesise minimal metadata. ParseFields sets the rest.
		fields = append(fields, fieldFromGoType(name, goType))
	}
	if err := scanner.Err(); err != nil {
		return nil, nil
	}
	return fields, nil
}

// fieldFromGoType re-derives a Field from a name + Go type pair. The
// validator/SQL/IsUUID fields are best-effort: only types we recognise are
// tagged, everything else is passed through untouched. The IsTime and
// IsUUID booleans drive template import decisions and generated tests.
func fieldFromGoType(name, goType string) Field {
	mapping, ok := goTypeToMapping(goType)
	if !ok {
		// Unknown type — emit a minimal Field and let the user fix the
		// generated validator if needed. This keeps the generator from
		// erroring on a hand-edited model with a foreign type.
		return Field{
			Name:   name,
			GoName: pascalCase(splitWords(name)),
			Camel:  name,
			GoType: goType,
		}
	}
	return Field{
		Name:     name,
		GoName:   pascalCase(splitWords(name)),
		Camel:    name,
		GoType:   mapping.goType,
		SQLType:  mapping.sqlType,
		Validate: mapping.validate,
		IsTime:   mapping.isTime,
		IsUUID:   goType == "string" && mapping.validate == "required,uuid",
		Sample:   mapping.sample,
	}
}

// goTypeToMapping maps a Go type back to the typeMapping that produced it.
// It's the inverse of the keyword→type table; the validator flag drives
// IsUUID detection (only the `uuid` keyword produces a `required,uuid`
// validator, so the round-trip is unambiguous in practice).
func goTypeToMapping(goType string) (typeMapping, bool) {
	if mapping, ok := fieldTypes[goType]; ok {
		return mapping, true
	}
	return typeMapping{}, false
}

func supportedTypes() string {
	return "string, text, int, int64, float, bool, time, uuid, email"
}
