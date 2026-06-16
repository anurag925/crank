package scaffold

import (
	"strings"
	"unicode"
)

// Resource holds the various name forms derived from a user-supplied resource
// name (e.g. "OrderItem", "order_items", "orders"). All forms are computed from
// the singular base so generated code is internally consistent.
type Resource struct {
	Pascal       string // OrderItem
	PascalPlural string // OrderItems
	Camel        string // orderItem
	CamelPlural  string // orderItems
	Snake        string // order_item
	SnakePlural  string // order_items (table name)
	Kebab        string // order-item
	KebabPlural  string // order-items (route segment)
	Receiver     string // o (lowercase first letter of Pascal)
	ContextParam string // c, or ctx when the receiver is c to avoid parameter shadowing
}

// NewResource builds a Resource from any reasonable input casing. The last word
// of the input is singularized so that "orders", "order_items" and "OrderItem"
// all resolve to the same canonical resource.
func NewResource(input string) Resource {
	words := splitWords(input)
	if len(words) == 0 {
		return Resource{}
	}
	// Singularize the final word so plural inputs are normalized.
	words[len(words)-1] = singularize(words[len(words)-1])

	plural := make([]string, len(words))
	copy(plural, words)
	plural[len(plural)-1] = pluralize(plural[len(plural)-1])

	pascal := pascalCase(words)
	r := Resource{
		Pascal:       pascal,
		PascalPlural: pascalCase(plural),
		Camel:        camelCase(words),
		CamelPlural:  camelCase(plural),
		Snake:        strings.Join(words, "_"),
		SnakePlural:  strings.Join(plural, "_"),
		Kebab:        strings.Join(words, "-"),
		KebabPlural:  strings.Join(plural, "-"),
	}
	if pascal != "" {
		r.Receiver = strings.ToLower(pascal[:1])
	}
	// The conventional name for an echo.Context parameter is `c`, but that
	// shadows the receiver on any resource whose Pascal form starts with `C`
	// (e.g. Customer, Category, Comment). Fall back to `ctx` in that case
	// so the generated handler methods don't fail to compile.
	if r.Receiver == "c" {
		r.ContextParam = "ctx"
	} else {
		r.ContextParam = "c"
	}
	return r
}

// splitWords breaks an identifier into lowercase words, handling camelCase,
// PascalCase, snake_case, kebab-case and space-separated inputs.
func splitWords(s string) []string {
	var words []string
	var cur strings.Builder

	flush := func() {
		if cur.Len() > 0 {
			words = append(words, strings.ToLower(cur.String()))
			cur.Reset()
		}
	}

	runes := []rune(s)
	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || r == ' ':
			flush()
		case unicode.IsUpper(r):
			// Boundary before an uppercase letter that follows a lowercase
			// letter or digit (camelCase / PascalCase transition).
			if i > 0 {
				prev := runes[i-1]
				if unicode.IsLower(prev) || unicode.IsDigit(prev) {
					flush()
				}
			}
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return words
}

func pascalCase(words []string) string {
	var b strings.Builder
	for _, w := range words {
		b.WriteString(title(w))
	}
	return b.String()
}

func camelCase(words []string) string {
	var b strings.Builder
	for i, w := range words {
		if i == 0 {
			b.WriteString(w)
			continue
		}
		b.WriteString(title(w))
	}
	return b.String()
}

func title(w string) string {
	if w == "" {
		return ""
	}
	r := []rune(w)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// pluralize applies a small set of English pluralization rules. It is not
// exhaustive but covers the common cases for generated resource names.
func pluralize(w string) string {
	if w == "" {
		return w
	}
	switch {
	case endsWithAny(w, "s", "x", "z", "ch", "sh"):
		return w + "es"
	case strings.HasSuffix(w, "y") && len(w) > 1 && !isVowel(rune(w[len(w)-2])):
		return w[:len(w)-1] + "ies"
	default:
		return w + "s"
	}
}

// singularize reverses the common pluralization rules.
func singularize(w string) string {
	if w == "" {
		return w
	}
	switch {
	case strings.HasSuffix(w, "ies") && len(w) > 3:
		return w[:len(w)-3] + "y"
	case endsWithAny(w, "ses", "xes", "zes", "ches", "shes"):
		return w[:len(w)-2]
	case strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") && len(w) > 1:
		return w[:len(w)-1]
	default:
		return w
	}
}

func endsWithAny(s string, suffixes ...string) bool {
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}

func isVowel(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

// DDDPath returns the project-relative path of the domain layer directory for
// a resource, e.g. "internal/domain/order". It is purely a string helper so
// templates and generator code can stay in sync.
func (r Resource) DDDDomainPath() string { return "internal/domain/" + r.Snake }

// DDDAppPath returns the project-relative path of the application layer
// directory, e.g. "internal/application/order".
func (r Resource) DDDAppPath() string { return "internal/application/" + r.Snake }

// DDDBunAdapterPath returns the project-relative path of the bun-backed
// adapter file, e.g. "internal/adapters/persistence/bun/order_repository.go".
func (r Resource) DDDBunAdapterPath() string {
	return "internal/adapters/persistence/bun/" + r.Snake + "_repository.go"
}

// DDDGormAdapterPath returns the project-relative path of the gorm-backed
// adapter file, e.g. "internal/adapters/persistence/gorm/order_repository.go".
func (r Resource) DDDGormAdapterPath() string {
	return "internal/adapters/persistence/gorm/" + r.Snake + "_repository.go"
}

// DDDPostgresAdapterPath is an alias for DDDBunAdapterPath kept for
// backwards compatibility with any external callers. Prefer DDDBunAdapterPath
// in new code.
func (r Resource) DDDPostgresAdapterPath() string {
	return r.DDDBunAdapterPath()
}

// DDDMemoryAdapterPath returns the project-relative path of the in-memory
// adapter file.
func (r Resource) DDDMemoryAdapterPath() string {
	return "internal/adapters/persistence/memory/" + r.Snake + "_repository.go"
}

// DDDHTTPHandlerPath returns the project-relative path of the HTTP adapter.
func (r Resource) DDDHTTPHandlerPath() string {
	return "internal/adapters/http/web/" + r.Snake + "_handler.go"
}

// DDDWorkflowPath returns the project-relative path of the temporal workflow
// file.
func (r Resource) DDDWorkflowPath() string {
	return "internal/adapters/temporal/workflow/" + r.Snake + ".go"
}

// DDDActivityPath returns the project-relative path of the temporal activity
// file.
func (r Resource) DDDActivityPath() string {
	return "internal/adapters/temporal/activity/" + r.Snake + ".go"
}
