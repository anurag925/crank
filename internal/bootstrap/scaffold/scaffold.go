// Package scaffold implements crank's in-project code generators (the `crank
// make` family). Given a resource name and optional field specs it renders
// Domain-Driven Go code — domain aggregates + value objects + events +
// repository ports, application command/query handlers, persistence adapters
// (bun-backed when the bun feature is enabled, in-memory otherwise) and an
// HTTP handler — into an existing crank-generated project. Generated
// handlers are automatically wired into the project's
// `internal/adapters/http/web/routes.go` so the resulting endpoints work out
// of the box.
package scaffold

import (
	"embed"
	"fmt"
	"go/format"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/utils"
)

//go:embed templates/*.tmpl
var templates embed.FS

// Recognized generator kinds.
const (
	KindModel      = "model"
	KindRepository = "repository"
	KindService    = "service"
	KindHandler    = "handler"
	KindScaffold   = "scaffold"
	KindWorkflow   = "workflow"
	KindActivity   = "activity"
)

// Kinds returns the supported generator kinds in display order.
func Kinds() []string {
	return []string{KindModel, KindRepository, KindService, KindHandler, KindScaffold, KindWorkflow, KindActivity}
}

// Options controls a single generation run.
type Options struct {
	ProjectDir    string   // path to the target project (contains .crank.yaml)
	Kind          string   // one of the Kind* constants
	Name          string   // resource name in any casing (e.g. "OrderItem")
	Fields        []string // "name:type" specs
	Only          bool     // generate only the primary artifact, skipping dependencies
	Force         bool     // overwrite the primary artifact if it already exists
	SkipMigration bool     // do not generate a table migration even when postgres is enabled
	Tests         bool     // also generate _test.go files alongside each generated artifact
}

// Result reports what a generation run produced.
type Result struct {
	Resource Resource
	Created  []string
	Skipped  []string
	Wired    bool
	WireHint string
}

// tmplData is the value passed to every template during rendering. It carries
// the per-run context (module path, enabled features, the resource) plus a
// handful of derived flags templates consult to decide which sections to emit.
type tmplData struct {
	Module string
	// Bun is true when the project has the bun ORM feature enabled. When
	// true, the scaffold emits a Bun-backed repository.
	Bun bool
	// Gorm is true when the project has the gorm ORM feature enabled. When
	// true, the scaffold emits a GORM-backed repository. Only one of Bun
	// and Gorm can be true at a time.
	Gorm     bool
	Auth     bool
	Temporal bool
	R        Resource
	Fields   []Field
	HasTime  bool
	HasUUID  bool
	// IDField is the first uuid-typed field, if any. Templates use it to wire
	// the aggregate's primary identifier. It is nil when no field is uuid.
	IDField *Field
}

// artifact is a single file to render and write.
type artifact struct {
	out      string // path relative to the project root
	tmpl     string // template filename inside templates/
	testTmpl string // optional companion test template (rendered when --tests is set)
	primary  bool   // the explicitly requested artifact (errors if it exists without --force)
	goFile   bool   // whether to gofmt the rendered output
}

// Generate runs a code generator according to opts. It reads the project's
// manifest to decide between the postgres (Bun-backed adapter + migration)
// and in-memory (always shipped) variants, renders the relevant templates,
// writes the files and — for handler/scaffold kinds — wires the new handler
// into the Echo router through `internal/adapters/http/web/routes.go`.
func Generate(opts Options) (*Result, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("a resource name is required\n\nUsage: crank make %s <Name> [field:type ...]\n\nExample: crank make %s Order", opts.Kind, opts.Kind)
	}

	res := NewResource(opts.Name)
	if res.Pascal == "" {
		return nil, fmt.Errorf("invalid resource name %q: could not derive a Go identifier from it\n\nNames must start with a letter and contain at least one alphanumeric character.\nExamples: Order, OrderItem, order_item", opts.Name)
	}

	fields, err := ParseFields(opts.Fields)
	if err != nil {
		return nil, err
	}
	// When the user invokes `crank make handler|repository|service Foo`
	// against a resource whose domain aggregate already exists, we
	// reuse the existing field list so the generated handler/service/
	// persistence signatures line up with the domain. Without this, the
	// generated code calls e.g. `receipt.NewReceipt(id)` while the
	// domain expects `NewReceipt(id, amount)`, breaking the build.
	if len(fields) == 0 && opts.Kind != KindModel {
		if inferred, ferr := InferFieldsFromDomain(opts.ProjectDir, res); ferr == nil && len(inferred) > 0 {
			fields = inferred
		}
	}

	info, err := bootstrap.LoadProjectInfo(opts.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("cannot load project: %w", err)
	}

	if opts.Kind == KindWorkflow || opts.Kind == KindActivity {
		if !info.Has("temporal") {
			return nil, fmt.Errorf("the %s generator requires the temporal feature\n\nTo add it, run:\n  crank add temporal --project %s", opts.Kind, opts.ProjectDir)
		}
	}

	idField := uuidFieldOrNil(fields)
	data := tmplData{
		Module:   info.ModulePath,
		Bun:      info.Has("bun"),
		Gorm:     info.Has("gorm"),
		Auth:     info.Has("auth"),
		Temporal: info.Has("temporal"),
		R:        res,
		Fields:   fields,
		HasTime:  hasTimeField(fields),
		HasUUID:  idField != nil,
		IDField:  idField,
	}

	plan, wantMigration, wire, err := buildPlan(opts, data)
	if err != nil {
		return nil, err
	}

	result := &Result{Resource: res}

	for _, a := range plan {
		dest := filepath.Join(opts.ProjectDir, a.out)
		if utils.PathExists(dest) {
			if a.primary && !opts.Force {
				return nil, fmt.Errorf("file %s already exists\n\nTo overwrite it, run:\n  crank make %s %s --force", a.out, opts.Kind, opts.Name)
			}
			if !a.primary {
				result.Skipped = append(result.Skipped, a.out)
				continue
			}
		}
		rendered, err := renderArtifact(a, data)
		if err != nil {
			return nil, err
		}
		if err := utils.WriteFile(dest, rendered); err != nil {
			return nil, err
		}
		result.Created = append(result.Created, a.out)
	}

	if wantMigration {
		created, skipped, err := generateMigration(opts.ProjectDir, data)
		if err != nil {
			return nil, err
		}
		result.Created = append(result.Created, created...)
		result.Skipped = append(result.Skipped, skipped...)
	}

	if wire != wireNone {
		var wr wireResult
		switch wire {
		case wireHandlerTarget:
			wr, err = wireHandler(opts.ProjectDir, res)
		case wireWorkflowTarget:
			wr, err = wireWorkflow(opts.ProjectDir, res)
		case wireActivityTarget:
			wr, err = wireActivity(opts.ProjectDir, res)
		}
		if err != nil {
			return nil, err
		}
		result.Wired = wr.Wired
		result.WireHint = wr.Hint
	}

	sort.Strings(result.Created)
	sort.Strings(result.Skipped)
	return result, nil
}

// wire targets identify which aggregator a generated artifact must be
// registered with (if any).
const (
	wireNone           = ""
	wireHandlerTarget  = "handler"
	wireWorkflowTarget = "workflow"
	wireActivityTarget = "activity"
)

// buildPlan turns the requested kind into the concrete set of artifacts plus
// flags for migration generation and the aggregator (if any) that generated
// code must be wired into.
//
// The mapping mirrors the spec:
//
//	KindModel      → domain files only (aggregate, value objects, events, port, errors)
//	KindRepository → persistence adapters only (port is always re-emitted)
//	KindService    → application command/query files only
//	KindHandler    → HTTP adapter only (+ wire into routes.go)
//	KindScaffold   → everything above
//	KindWorkflow   → temporal workflow under internal/adapters/temporal/workflow/
//	KindActivity   → temporal activity under internal/adapters/temporal/activity/
func buildPlan(opts Options, data tmplData) (plan []artifact, wantMigration bool, wire string, err error) {
	r := data.R

	// Domain files. Always emitted; the test companion is also always available
	// when --tests is set. The templates are parameterised by Resource, so
	// the same file produces the right output for any resource name.
	aggregate := artifact{out: r.DDDDomainPath() + "/" + r.Snake + ".go", tmpl: "domain_aggregate.go.tmpl", testTmpl: "domain_aggregate_test.go.tmpl", goFile: true}
	idVO := artifact{out: r.DDDDomainPath() + "/" + r.Snake + "_id.go", tmpl: "domain_id.go.tmpl", testTmpl: "domain_id_test.go.tmpl", goFile: true}
	events := artifact{out: r.DDDDomainPath() + "/events.go", tmpl: "domain_events.go.tmpl", testTmpl: "domain_events_test.go.tmpl", goFile: true}
	derrors := artifact{out: r.DDDDomainPath() + "/errors.go", tmpl: "domain_errors.go.tmpl", goFile: true}
	repoPort := artifact{out: r.DDDDomainPath() + "/repository.go", tmpl: "domain_repository.go.tmpl", goFile: true}

	// Application files.
	commands := artifact{out: r.DDDAppPath() + "/commands.go", tmpl: "application_commands.go.tmpl", testTmpl: "application_commands_test.go.tmpl", goFile: true}
	cmdHandler := artifact{out: r.DDDAppPath() + "/command_handler.go", tmpl: "application_command_handler.go.tmpl", testTmpl: "application_command_handler_test.go.tmpl", goFile: true}
	queries := artifact{out: r.DDDAppPath() + "/queries.go", tmpl: "application_queries.go.tmpl", testTmpl: "application_queries_test.go.tmpl", goFile: true}
	qryHandler := artifact{out: r.DDDAppPath() + "/query_handler.go", tmpl: "application_query_handler.go.tmpl", testTmpl: "application_query_handler_test.go.tmpl", goFile: true}

	// Persistence adapters.
	bunAdapter := artifact{out: r.DDDBunAdapterPath(), tmpl: "adapter_persistence_postgres_repository.go.tmpl", testTmpl: "adapter_persistence_postgres_repository_test.go.tmpl", goFile: true}
	gormAdapter := artifact{out: r.DDDGormAdapterPath(), tmpl: "adapter_persistence_gorm_repository.go.tmpl", testTmpl: "adapter_persistence_gorm_repository_test.go.tmpl", goFile: true}
	memAdapter := artifact{out: r.DDDMemoryAdapterPath(), tmpl: "adapter_persistence_memory_repository.go.tmpl", testTmpl: "adapter_persistence_memory_repository_test.go.tmpl", goFile: true}

	// HTTP adapter.
	httpAdapter := artifact{out: r.DDDHTTPHandlerPath(), tmpl: "adapter_http_handler.go.tmpl", testTmpl: "adapter_http_handler_test.go.tmpl", goFile: true}

	migration := (data.Bun || data.Gorm) && !opts.SkipMigration

	switch opts.Kind {
	case KindModel:
		aggregate.primary = true
		plan = []artifact{aggregate, idVO, events, derrors, repoPort}
		wantMigration = migration

	case KindRepository:
		// ORM adapter is primary when either bun or gorm is enabled;
		// otherwise the in-memory adapter is the primary artifact.
		if data.Bun || data.Gorm {
			if data.Bun {
				bunAdapter.primary = true
				plan = []artifact{bunAdapter, memAdapter, repoPort}
			} else {
				gormAdapter.primary = true
				plan = []artifact{gormAdapter, memAdapter, repoPort}
			}
		} else {
			memAdapter.primary = true
			plan = []artifact{memAdapter, repoPort}
		}
		if !opts.Only {
			plan = append(plan, aggregate, idVO, events, derrors)
		}
		wantMigration = migration

	case KindService:
		commands.primary = true
		plan = []artifact{commands, cmdHandler, queries, qryHandler, memAdapter}
		if !opts.Only {
			plan = append(plan, aggregate, idVO, events, derrors, repoPort)
		}
		// Application layer has no migration of its own.
		wantMigration = false

	case KindHandler:
		// HTTP adapter is always primary; the spec says --only produces just it.
		only := opts.Only
		httpAdapter.primary = true
		plan = []artifact{httpAdapter}
		if !only {
			plan = append(plan, aggregate, idVO, events, derrors, repoPort,
				commands, cmdHandler, queries, qryHandler,
				memAdapter)
			if data.Bun {
				plan = append(plan, bunAdapter)
			} else if data.Gorm {
				plan = append(plan, gormAdapter)
			}
		}
		wantMigration = migration
		wire = wireHandlerTarget

	case KindScaffold:
		httpAdapter.primary = true
		plan = []artifact{httpAdapter,
			aggregate, idVO, events, derrors, repoPort,
			commands, cmdHandler, queries, qryHandler,
			memAdapter}
		if data.Bun {
			plan = append(plan, bunAdapter)
		} else if data.Gorm {
			plan = append(plan, gormAdapter)
		}
		wantMigration = migration
		wire = wireHandlerTarget

	case KindWorkflow:
		wf := artifact{out: r.DDDWorkflowPath(), tmpl: "workflow.go.tmpl", testTmpl: "workflow_test.go.tmpl", goFile: true, primary: true}
		plan = []artifact{wf}
		wire = wireWorkflowTarget

	case KindActivity:
		act := artifact{out: r.DDDActivityPath(), tmpl: "activity.go.tmpl", testTmpl: "activity_test.go.tmpl", goFile: true, primary: true}
		plan = []artifact{act}
		wire = wireActivityTarget

	default:
		return nil, false, wireNone, fmt.Errorf("unknown generator kind %q\n\nSupported kinds: %s\nRun 'crank make --help' for details.", opts.Kind, strings.Join(Kinds(), ", "))
	}

	if opts.Tests {
		plan = withTestArtifacts(plan)
	}

	return plan, wantMigration, wire, nil
}

// withTestArtifacts expands a plan so each artifact that declares a test
// template is followed by its companion _test.go artifact. Test files are never
// treated as primary, so they are skipped (not errored) when they already exist.
func withTestArtifacts(plan []artifact) []artifact {
	out := make([]artifact, 0, len(plan)*2)
	for _, a := range plan {
		out = append(out, a)
		if a.testTmpl == "" {
			continue
		}
		out = append(out, artifact{
			out:    testPath(a.out),
			tmpl:   a.testTmpl,
			goFile: true,
		})
	}
	return out
}

// testPath converts a Go source path into its _test.go counterpart.
func testPath(p string) string {
	return strings.TrimSuffix(p, ".go") + "_test.go"
}

// renderArtifact renders a single template and, for Go files, runs gofmt.
func renderArtifact(a artifact, data tmplData) (string, error) {
	out, err := renderTemplate(a.tmpl, data)
	if err != nil {
		return "", err
	}
	if a.goFile {
		formatted, ferr := format.Source([]byte(out))
		if ferr != nil {
			return "", fmt.Errorf("format %s: %w", a.out, ferr)
		}
		return string(formatted), nil
	}
	return out, nil
}

// renderTemplate parses and executes a named template against data.
func renderTemplate(name string, data tmplData) (string, error) {
	body, err := templates.ReadFile("templates/" + name)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", name, err)
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(body))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return sb.String(), nil
}

// TmplDataFor builds a tmplData value for a single scaffold run. It is
// exposed so that external callers (e.g. the smoke test under cmd/_smoke)
// can drive the same template-rendering pipeline that Generate uses.
func TmplDataFor(module string, Bun, Gorm, auth bool, r Resource, fields []Field) tmplData {
	idField := uuidFieldOrNil(fields)
	return tmplData{
		Module:  module,
		Bun:     Bun,
		Gorm:    Gorm,
		Auth:    auth,
		R:       r,
		Fields:  fields,
		HasTime: hasTimeField(fields),
		HasUUID: idField != nil,
		IDField: idField,
	}
}

// RenderTemplateForTest renders a named scaffold template against the given
// tmplData without any post-processing. It is intended for diagnostic use.
func RenderTemplateForTest(name string, data tmplData) (string, error) {
	return renderTemplate(name, data)
}

// generateMigration writes a timestamped create-table migration pair for the
// resource. It is a no-op (returns the files as skipped) when a create
// migration for the same table already exists.
//
// Columns mirror the postgres row DTO emitted by the adapter template:
// text for primitive strings, UUID for the typed identifier.
func generateMigration(projectDir string, data tmplData) (created, skipped []string, err error) {
	dir := filepath.Join(projectDir, "migrations")
	name := "create_" + data.R.SnakePlural

	existing, _ := filepath.Glob(filepath.Join(dir, "*_"+name+".up.sql"))
	if len(existing) > 0 {
		rel, _ := filepath.Rel(projectDir, existing[0])
		return nil, []string{rel}, nil
	}

	if err := utils.EnsureDir(dir); err != nil {
		return nil, nil, err
	}

	stamp := time.Now().UTC().Format("20060102150405")
	upRel := filepath.Join("migrations", fmt.Sprintf("%s_%s.up.sql", stamp, name))
	downRel := filepath.Join("migrations", fmt.Sprintf("%s_%s.down.sql", stamp, name))

	up, err := renderTemplate("migration.up.sql.tmpl", data)
	if err != nil {
		return nil, nil, err
	}
	down, err := renderTemplate("migration.down.sql.tmpl", data)
	if err != nil {
		return nil, nil, err
	}
	if err := utils.WriteFile(filepath.Join(projectDir, upRel), up); err != nil {
		return nil, nil, err
	}
	if err := utils.WriteFile(filepath.Join(projectDir, downRel), down); err != nil {
		return nil, nil, err
	}
	return []string{upRel, downRel}, nil, nil
}
