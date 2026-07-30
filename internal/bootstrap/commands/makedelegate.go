package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/anurag925/crank/internal/utils"
)

// makeTargetRe matches a Makefile target definition at the start of a line:
// a name in the first column followed by a single ':' that is NOT part of an
// assignment operator (':=', '::=', etc.). This deliberately excludes variable
// assignments like `APP_NAME := myapp`.
var makeTargetRe = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9_.-]*)[ \t]*:([^=]|$)`)

// TryMakeDelegation inspects the raw CLI args and either delegates to the
// target project's Makefile or lets cobra handle the command natively.
//
// The project's Makefile is consulted first. If it defines a target matching
// the requested command, `make <target>` is run instead of the native crank
// command. This lets projects customise or replace crank's tool wrappers
// (dev, build, test, …) with project-specific behaviour.
//
// If the Makefile does not define the target, cobra handles it natively (or
// reports "unknown command" if it is not a registered crank subcommand).
func TryMakeDelegation(args []string) (handled bool, err error) {
	if len(args) == 0 {
		return false, nil
	}

	candidate := args[0]
	if candidate == "" || strings.HasPrefix(candidate, "-") {
		return false, nil
	}
	// Built-in commands never delegate.
	if candidate == "help" || candidate == "completion" {
		return false, nil
	}

	projectDir, makeArgs := splitProjectFlag(args[1:])

	if hasMakefileTarget(projectDir, candidate) {
		return true, runMakeTarget(projectDir, candidate, makeArgs)
	}
	return false, nil
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

// hasMakefileTarget reports whether the project's Makefile in projectDir
// defines a target with the given name.
func hasMakefileTarget(projectDir, target string) bool {
	makefile := filepath.Join(projectDir, "Makefile")
	if !utils.PathExists(makefile) {
		return false
	}
	targets, err := makefileTargets(makefile)
	if err != nil {
		return false
	}
	return targets[target]
}
