package bootstrap

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Tool represents an external CLI tool that the crank CLI wraps.
// Each tool is exposed as a subcommand (e.g. "crank migrate", "crank swag").
type Tool interface {
	// Name returns the subcommand name (e.g. "migrate", "swag", "build").
	Name() string
	// Description is a short one-line description shown in help text.
	Description() string
	// LongDescription is the detailed help text for the subcommand.
	LongDescription() string
	// BinaryName is the executable name to look for on PATH.
	BinaryName() string
	// InstallCmd returns a human-readable install instruction shown on errors.
	InstallCmd() string
	// RequiresFeatures returns feature names that must be enabled in the project
	// for this tool to be useful. Empty means the tool is always available.
	RequiresFeatures() []string
	// AddFlags lets the tool register custom CLI flags on the cobra command.
	AddFlags(cmd *cobra.Command)
	// Prepare resolves the binary path and builds the final invocation.
	// It receives the project directory and the cobra command so tools can
	// read their custom flags via cmd.Flags().
	Prepare(projectDir string, cmd *cobra.Command) (*ToolInvocation, error)
	// Install downloads and installs the tool. Returns nil on success.
	Install() error
}

// CheckResult is the outcome of a single in-process check. OK is true when the
// project passed the check; Summary is the human-readable label (e.g. "manifest
// parses"). Detail is appended on failure to help the user find the problem.
type CheckResult struct {
	OK      bool
	Summary string
	Detail  string
}

// InProcessTool is an optional interface a Tool can implement to run its work
// inside the crank binary (no external binary required). The harness calls
// RunInProcess instead of looking up a binary and exec'ing it. This is the
// natural home for tools that operate on project files (linters, health
// checks) rather than wrapping an external CLI.
type InProcessTool interface {
	Tool
	// RunInProcess executes the tool's checks against projectDir and writes
	// human-readable results to out. The returned error is reserved for
	// unrecoverable failures (e.g. the directory is not a crank project);
	// per-check failures should be reported via CheckResult, not as errors.
	RunInProcess(projectDir string, out io.Writer) ([]CheckResult, error)
}

// HasInProcess reports whether t implements InProcessTool.
func HasInProcess(t Tool) bool {
	_, ok := t.(InProcessTool)
	return ok
}

// ToolInvocation holds the resolved command to execute.
type ToolInvocation struct {
	Binary string   // full path to the binary
	Args   []string // arguments (without the binary name)
	Dir    string   // working directory
	Env    []string // additional KEY=VALUE env vars (merged with os.Environ)
	Stdin  bool     // whether to pass os.Stdin through
}

// ToolRegistry holds registered tools, analogous to the Feature Registry.
type ToolRegistry struct {
	tools map[string]Tool
	order []string
}

// NewToolRegistry creates an empty tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry. Names must be unique.
func (r *ToolRegistry) Register(t Tool) error {
	if _, exists := r.tools[t.Name()]; exists {
		return fmt.Errorf("tool %q already registered", t.Name())
	}
	r.tools[t.Name()] = t
	r.order = append(r.order, t.Name())
	return nil
}

// MustRegister panics if Register returns an error.
func (r *ToolRegistry) MustRegister(t Tool) {
	if err := r.Register(t); err != nil {
		panic(err)
	}
}

// Get returns the tool with the given name, or false if unknown.
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// All returns every registered tool in registration order.
func (r *ToolRegistry) All() []Tool {
	out := make([]Tool, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.tools[n])
	}
	return out
}

// ForFeatures returns tools whose feature requirements are satisfied by the
// given feature list. Tools with no requirements are always included.
func (r *ToolRegistry) ForFeatures(features []string) []Tool {
	featureSet := make(map[string]bool, len(features))
	for _, f := range features {
		featureSet[f] = true
	}
	var out []Tool
	for _, t := range r.All() {
		reqs := t.RequiresFeatures()
		if len(reqs) == 0 {
			out = append(out, t)
			continue
		}
		for _, req := range reqs {
			if featureSet[req] {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

// ForFeature returns tools that specifically require the named feature.
func (r *ToolRegistry) ForFeature(feature string) []Tool {
	var out []Tool
	for _, t := range r.All() {
		for _, req := range t.RequiresFeatures() {
			if req == feature {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

// ValidateToolRequirements checks that the project's manifest includes the
// features required by the tool.
func ValidateToolRequirements(projectDir string, t Tool) error {
	reqs := t.RequiresFeatures()
	if len(reqs) == 0 {
		return nil
	}
	m, err := readManifest(projectDir)
	if err != nil {
		return nil // no manifest — let Prepare handle it
	}
	for _, req := range reqs {
		if !contains(m.Features, req) {
			return fmt.Errorf("tool %q requires feature %q — install it with: crank add %s --project %s",
				t.Name(), req, req, projectDir)
		}
	}
	return nil
}
