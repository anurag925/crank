package scaffold

import (
	"fmt"
	"strings"
)

// Field describes a single column/attribute parsed from a "name:type" spec
// passed on the command line (e.g. "title:string", "price:float", "active:bool").
type Field struct {
	Name     string // snake_case column / json name
	GoName   string // PascalCase struct field name
	GoType   string // Go type (string, int64, float64, bool, time.Time)
	SQLType  string // PostgreSQL column type
	Validate string // go-playground/validator tag, may be empty
	IsTime   bool   // whether the field uses time.Time
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
			return nil, fmt.Errorf("invalid field spec %q: empty name", spec)
		}
		mapping, ok := fieldTypes[typ]
		if !ok {
			return nil, fmt.Errorf("unknown field type %q in %q (supported: %s)", typ, spec, supportedTypes())
		}

		fields = append(fields, Field{
			Name:     strings.Join(words, "_"),
			GoName:   pascalCase(words),
			GoType:   mapping.goType,
			SQLType:  mapping.sqlType,
			Validate: mapping.validate,
			IsTime:   mapping.isTime,
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

func supportedTypes() string {
	return "string, text, int, int64, float, bool, time, uuid, email"
}
