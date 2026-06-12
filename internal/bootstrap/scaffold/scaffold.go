// Package scaffold implements crank's in-project code generators (the `crank
// make` family). Given a resource name and optional field specs it renders
// layered Go code — models, repositories, services and HTTP handlers — into an
// existing crank-generated project, mirroring the conventions of the base and
// postgres features. Generated handlers are automatically wired into the
// project's Echo router so the resulting endpoints work out of the box.
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
)

// Kinds returns the supported generator kinds in display order.
func Kinds() []string {
	return []string{KindModel, KindRepository, KindService, KindHandler, KindScaffold}
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

// tmplData is the value passed to every template during rendering.
type tmplData struct {
	Module       string
	Postgres     bool
	Auth         bool
	R            Resource
	Fields       []Field
	HasTime      bool
	StorePkg     string // "repository" | "service"
	StoreType    string // "Repository" | "Service"
	StoreCtorArg string // "deps.DB" | ""
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
// manifest to decide between the postgres (Bun repository + migration) and
// in-memory (service) variants, renders the relevant templates, writes the
// files and — for handler/scaffold kinds — wires the new handler into the Echo
// router.
func Generate(opts Options) (*Result, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("a resource name is required (e.g. `crank make %s Order`)", opts.Kind)
	}

	res := NewResource(opts.Name)
	if res.Pascal == "" {
		return nil, fmt.Errorf("resource name %q does not resolve to a valid identifier", opts.Name)
	}

	fields, err := ParseFields(opts.Fields)
	if err != nil {
		return nil, err
	}

	info, err := bootstrap.LoadProjectInfo(opts.ProjectDir)
	if err != nil {
		return nil, err
	}

	data := tmplData{
		Module:   info.ModulePath,
		Postgres: info.Has("postgres"),
		Auth:     info.Has("auth"),
		R:        res,
		Fields:   fields,
		HasTime:  hasTimeField(fields),
	}
	if data.Postgres {
		data.StorePkg, data.StoreType, data.StoreCtorArg = "repository", "Repository", "deps.DB"
	} else {
		data.StorePkg, data.StoreType, data.StoreCtorArg = "service", "Service", ""
	}

	plan, wantMigration, wantWire, err := buildPlan(opts, data)
	if err != nil {
		return nil, err
	}

	result := &Result{Resource: res}

	for _, a := range plan {
		dest := filepath.Join(opts.ProjectDir, a.out)
		if utils.PathExists(dest) {
			if a.primary && !opts.Force {
				return nil, fmt.Errorf("%s already exists (use --force to overwrite)", a.out)
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

	if wantWire {
		wr, err := wireHandler(opts.ProjectDir, res)
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

// buildPlan turns the requested kind into the concrete set of artifacts plus
// flags for migration generation and handler wiring.
func buildPlan(opts Options, data tmplData) (plan []artifact, wantMigration, wantWire bool, err error) {
	model := artifact{out: "internal/model/" + data.R.Snake + ".go", tmpl: "model.go.tmpl", testTmpl: "model_test.go.tmpl", goFile: true}
	repo := artifact{out: "internal/repository/" + data.R.Snake + ".go", tmpl: "repository.go.tmpl", testTmpl: "repository_test.go.tmpl", goFile: true}
	svc := artifact{out: "internal/service/" + data.R.Snake + ".go", tmpl: "service.go.tmpl", testTmpl: "service_test.go.tmpl", goFile: true}
	handler := artifact{out: "internal/handler/" + data.R.Snake + ".go", tmpl: "handler.go.tmpl", testTmpl: "handler_test.go.tmpl", goFile: true}

	migration := data.Postgres && !opts.SkipMigration

	switch opts.Kind {
	case KindModel:
		model.primary = true
		plan = []artifact{model}
		wantMigration = migration

	case KindRepository:
		repo.primary = true
		plan = []artifact{repo}
		if !opts.Only {
			plan = append(plan, model)
		}
		wantMigration = migration

	case KindService:
		svc.primary = true
		plan = []artifact{svc}
		if !opts.Only {
			plan = append(plan, model)
		}
		// In-memory services do not need a migration.
		wantMigration = false

	case KindHandler, KindScaffold:
		// scaffold always pulls in the full stack.
		only := opts.Only && opts.Kind == KindHandler
		handler.primary = true
		plan = []artifact{handler}
		if !only {
			plan = append(plan, model)
			if data.Postgres {
				plan = append(plan, repo)
			} else {
				plan = append(plan, svc)
			}
		}
		wantMigration = migration
		wantWire = true

	default:
		return nil, false, false, fmt.Errorf("unknown kind %q (supported: %s)", opts.Kind, strings.Join(Kinds(), ", "))
	}

	if opts.Tests {
		plan = withTestArtifacts(plan)
	}

	return plan, wantMigration, wantWire, nil
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

// generateMigration writes a timestamped create-table migration pair for the
// resource. It is a no-op (returns the files as skipped) when a create
// migration for the same table already exists.
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
