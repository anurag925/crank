package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExecConfig holds the configuration for running an external command.
type ExecConfig struct {
	Binary string   // full path to the binary
	Args   []string // arguments (binary is prepended automatically)
	Dir    string   // working directory
	Env    []string // additional KEY=VALUE env vars to merge with os.Environ
}

// RunExternal executes an external command, streaming stdout/stderr.
func RunExternal(cfg *ExecConfig) error {
	argv := append([]string{cfg.Binary}, cfg.Args...)
	env := os.Environ()
	if len(cfg.Env) > 0 {
		env = append(env, cfg.Env...)
	}

	c := exec.Cmd{
		Path:   cfg.Binary,
		Args:   argv,
		Dir:    cfg.Dir,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    env,
	}
	return c.Run()
}

// FindBinary looks up a binary on PATH, returning the full path or an error
// with the provided install hint.
func FindBinary(name, installHint string) (string, error) {
	bin, err := exec.LookPath(name)
	if err != nil {
		if installHint != "" {
			return "", fmt.Errorf("%s is not installed or not on PATH.\n  Install with: %s", name, installHint)
		}
		return "", fmt.Errorf("%s is not installed or not on PATH", name)
	}
	return bin, nil
}

// ShellJoin joins an argv slice into a human-readable command string.
func ShellJoin(argv []string) string {
	return strings.Join(argv, " ")
}
