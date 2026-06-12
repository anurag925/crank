package scaffold

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

// handlerFile is the path (relative to the project root) of the central handler
// aggregator that wires individual handlers into the Echo router.
const handlerFile = "internal/handler/handler.go"

// Marker comments emitted by the base feature template. When present, new
// handlers are spliced in at these anchors.
const (
	markerFields   = "// crank:handler-fields"
	markerInit     = "// crank:handler-init"
	markerRegister = "// crank:handler-register"
)

// wireResult reports the outcome of attempting to register a handler in
// handler.go.
type wireResult struct {
	// Wired is true when handler.go was edited successfully.
	Wired bool
	// Hint, when non-empty, contains manual instructions the user should apply
	// because automatic wiring was not possible.
	Hint string
}

// wireHandler registers a generated handler with the project's central Handler
// aggregator (internal/handler/handler.go) so its routes are served without any
// manual edits. It is best-effort and never corrupts the file: if the resulting
// source does not compile/format, the edit is discarded and a manual hint is
// returned instead.
func wireHandler(projectDir string, r Resource) (wireResult, error) {
	path := filepath.Join(projectDir, handlerFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return wireResult{Hint: manualWireHint(r)}, nil
	}
	if err != nil {
		return wireResult{}, fmt.Errorf("read %s: %w", handlerFile, err)
	}

	content := string(data)

	// Already wired? Avoid creating duplicate registrations.
	ctor := fmt.Sprintf("New%sHandler(deps)", r.Pascal)
	if strings.Contains(content, ctor) {
		return wireResult{Wired: true}, nil
	}

	fieldLine := fmt.Sprintf("\t%s *%sHandler\n", r.CamelPlural, r.Pascal)
	initLine := fmt.Sprintf("\t\t%s: New%sHandler(deps),\n", r.CamelPlural, r.Pascal)
	registerLine := fmt.Sprintf("\th.%s.Register(e)\n", r.CamelPlural)

	updated, ok := spliceAtMarkers(content, fieldLine, initLine, registerLine)
	if !ok {
		updated, ok = spliceAtBraces(content, fieldLine, initLine, registerLine)
	}
	if !ok {
		return wireResult{Hint: manualWireHint(r)}, nil
	}

	formatted, err := format.Source([]byte(updated))
	if err != nil {
		// The edit produced invalid Go; do not write a broken file.
		return wireResult{Hint: manualWireHint(r)}, nil
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return wireResult{}, fmt.Errorf("write %s: %w", handlerFile, err)
	}
	return wireResult{Wired: true}, nil
}

// spliceAtMarkers inserts the snippets immediately before the marker comments.
// It returns false if any marker is missing.
func spliceAtMarkers(content, fieldLine, initLine, registerLine string) (string, bool) {
	if !strings.Contains(content, markerFields) ||
		!strings.Contains(content, markerInit) ||
		!strings.Contains(content, markerRegister) {
		return "", false
	}
	content = strings.Replace(content, markerFields, strings.TrimPrefix(fieldLine, "\t")+"\t"+markerFields, 1)
	content = strings.Replace(content, markerInit, strings.TrimPrefix(initLine, "\t\t")+"\t\t"+markerInit, 1)
	content = strings.Replace(content, markerRegister, strings.TrimPrefix(registerLine, "\t")+"\t"+markerRegister, 1)
	return content, true
}

// spliceAtBraces is the fallback used for projects generated before markers
// existed. It locates the Handler struct, the New() composite literal and the
// Register method body and inserts the snippets before their closing braces.
func spliceAtBraces(content, fieldLine, initLine, registerLine string) (string, bool) {
	var ok bool
	content, ok = insertBeforeClosing(content, "type Handler struct {", "\n}", fieldLine)
	if !ok {
		return "", false
	}
	content, ok = insertBeforeClosing(content, "return &Handler{", "\n\t}", initLine)
	if !ok {
		return "", false
	}
	content, ok = insertBeforeClosing(content, "func (h *Handler) Register(", "\n}", registerLine)
	if !ok {
		return "", false
	}
	return content, true
}

// insertBeforeClosing finds anchor in content, then the first occurrence of
// closing after it, and inserts snippet just before that closing token.
func insertBeforeClosing(content, anchor, closing, snippet string) (string, bool) {
	start := strings.Index(content, anchor)
	if start < 0 {
		return content, false
	}
	rel := strings.Index(content[start:], closing)
	if rel < 0 {
		return content, false
	}
	at := start + rel
	return content[:at] + "\n" + strings.TrimRight(snippet, "\n") + content[at:], true
}

// manualWireHint returns copy-pasteable instructions for registering a handler
// by hand when automatic wiring is not possible.
func manualWireHint(r Resource) string {
	return fmt.Sprintf(`could not auto-register the handler in %s. Add these manually:

  • in the Handler struct:        %s *%sHandler
  • in New(), the &Handler{} body: %s: New%sHandler(deps),
  • in Register():                h.%s.Register(e)`,
		handlerFile,
		r.CamelPlural, r.Pascal,
		r.CamelPlural, r.Pascal,
		r.CamelPlural)
}
