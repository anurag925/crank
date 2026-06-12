package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GoGet runs `go get` for each dependency in the generated project directory,
// then runs `go mod tidy` to clean up. This replaces hard-coded go.mod require
// blocks with actual dependency resolution.
func GoGet(projectDir string, deps []string) error {
	if len(deps) == 0 {
		return nil
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go is required to install dependencies: %w", err)
	}

	// Run `go get` with all deps in a single invocation.
	args := append([]string{"get"}, deps...)
	argv := append([]string{goBin}, args...)
	c := exec.Cmd{
		Path:   goBin,
		Args:   argv,
		Dir:    projectDir,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	fmt.Printf("→ go get %s\n", joinDeps(deps))
	if err := c.Run(); err != nil {
		return fmt.Errorf("go get failed: %w", err)
	}

	return Tidy(projectDir)
}

// Tidy runs `go mod tidy` in the project directory.
func Tidy(projectDir string) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go is required: %w", err)
	}

	c := exec.Cmd{
		Path:   goBin,
		Args:   []string{goBin, "mod", "tidy"},
		Dir:    projectDir,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	fmt.Println("→ go mod tidy")
	if err := c.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	return nil
}

func joinDeps(deps []string) string {
	if len(deps) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(deps[0])
	for _, d := range deps[1:] {
		out.WriteString(" " + d)
	}
	return out.String()
}
