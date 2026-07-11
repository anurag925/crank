package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/utils"
)

// makeTargetRe matches a Makefile target definition at the start of a line:
// a name in the first column followed by a single ':' that is NOT part of an
// assignment operator (':=', '::=', etc.). This deliberately excludes variable
// assignments like `APP_NAME := myapp`.
var makeTargetRe = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9_.-]*)[ \t]*:([^=]|$)`)

// TryMakeDelegation inspects the raw CLI args and, when the first argument is
// not a known crank subcommand but matches a target in the target project's
// Makefile, runs `make <target>` in that project.
//
// It returns handled=true only when it takes responsibility for the command
// (i.e. a matching Makefile target was found and make was invoked). In every
// other case it returns handled=false so that cobra can run the command
// natively or report an unknown command as usual.
//
// Native crank commands always take precedence: the fallback is consulted only
// for names cobra does not already recognize. This means a Makefile can extend
// crank with project-specific targets, and (in the future) be used to override
// behavior for names crank does not ship natively.
func TryMakeDelegation(root *cobra.Command, args []string) (handled bool, err error) {
	if len(args) == 0 {
		return false, nil
	}

	candidate := args[0]
	// Root flags (-h, --help, --version, ...) are never make targets.
	if candidate == "" || strings.HasPrefix(candidate, "-") {
		return false, nil
	}
	// Defer to cobra for anything it already knows about.
	if isKnownCommand(root, candidate) {
		return false, nil
	}

	projectDir, makeArgs := splitProjectFlag(args[1:])

	makefile := filepath.Join(projectDir, "Makefile")
	if !utils.PathExists(makefile) {
		// No Makefile to delegate to; let cobra report the unknown command.
		return false, nil
	}

	targets, terr := makefileTargets(makefile)
	if terr != nil {
		return false, fmt.Errorf("read Makefile in %s: %w", projectDir, terr)
	}
	if !targets[candidate] {
		// Not a make target either; let cobra report the unknown command.
		return false, nil
	}

	return true, runMakeTarget(projectDir, candidate, makeArgs)
}

// isKnownCommand reports whether name matches a registered crank subcommand
// (including the built-in help/completion commands) or one of its aliases.
func isKnownCommand(root *cobra.Command, name string) bool {
	if name == "help" || name == "completion" {
		return true
	}
	for _, c := range root.Commands() {
		if c.Name() == name {
			return true
		}
		for _, a := range c.Aliases {
			if a == name {
				return true
			}
		}
	}
	return false
}

// splitProjectFlag extracts a --project value (supporting both `--project dir`
// and `--project=dir`) from args, returning the project directory (defaulting
// to ".") and the remaining args with the project flag removed. The remaining
// args are forwarded to make verbatim.
func splitProjectFlag(args []string) (projectDir string, rest []string) {
	projectDir = "."
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--project":
			if i+1 < len(args) {
				projectDir = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--project="):
			projectDir = strings.TrimPrefix(a, "--project=")
		default:
			rest = append(rest, a)
		}
	}
	return projectDir, rest
}

// makefileTargets parses a Makefile and returns the set of user-facing target
// names. Special targets (those starting with '.', e.g. .PHONY) and variable
// assignments are excluded.
func makefileTargets(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	targets := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Recipe lines, comments, and blank lines are not target definitions.
		if line == "" || line[0] == '\t' || line[0] == ' ' || line[0] == '#' {
			continue
		}
		m := makeTargetRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		if strings.HasPrefix(name, ".") {
			continue
		}
		targets[name] = true
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}

// runMakeTarget invokes `make <target> [args...]` in projectDir, streaming I/O.
func runMakeTarget(projectDir, target string, args []string) error {
	bin, err := utils.FindBinary("make", "install make via your system package manager (e.g. xcode-select --install, apt install make)")
	if err != nil {
		return err
	}
	makeArgs := append([]string{target}, args...)
	fmt.Printf("→ make %s\n", strings.Join(makeArgs, " "))
	return utils.RunExternal(&utils.ExecConfig{
		Binary: bin,
		Args:   makeArgs,
		Dir:    projectDir,
	})
}
