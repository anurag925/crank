package views

import (
	"embed"

	"github.com/anurag925/crank/internal/bootstrap"
)

//go:embed templates/*.tmpl
var tmpls embed.FS

type feature struct{}

func init() {
	if err := bootstrap.GlobalRegistry.Register(feature{}); err != nil {
		panic(err)
	}
}

func (feature) Name() string { return "views" }
func (feature) Description() string {
	return "React SPA with Vite: embedded frontend served by the Go binary with dev-mode hot reload"
}
func (feature) Templates() embed.FS { return tmpls }

func (feature) Dependencies() []string { return nil }

func (feature) Requirements() []string { return nil }

func (feature) Files() []bootstrap.FileMapping {
	return []bootstrap.FileMapping{
		// Frontend project scaffolding — React + Vite
		{TemplatePath: "templates/package.json.tmpl", OutputPath: "views/package.json"},
		{TemplatePath: "templates/vite.config.js.tmpl", OutputPath: "views/vite.config.js"},
		{TemplatePath: "templates/index.html.tmpl", OutputPath: "views/index.html"},
		{TemplatePath: "templates/src_main.jsx.tmpl", OutputPath: "views/src/main.jsx"},
		{TemplatePath: "templates/src_App.jsx.tmpl", OutputPath: "views/src/App.jsx"},
		{TemplatePath: "templates/src_App.css.tmpl", OutputPath: "views/src/App.css"},
		{TemplatePath: "templates/src_api.js.tmpl", OutputPath: "views/src/api.js"},

		// Placeholder so Go embed doesn't fail on empty dist/.
		{TemplatePath: "templates/static_dist_index.html.tmpl", OutputPath: "static/dist/index.html"},

		// Go-side embed + serving
		{TemplatePath: "templates/static_embed.go.tmpl", OutputPath: "static/embed.go", SkipIfExists: true},
		{TemplatePath: "templates/internal_adapters_http_web_views.go.tmpl", OutputPath: "internal/adapters/http/web/views.go"},
	}
}
