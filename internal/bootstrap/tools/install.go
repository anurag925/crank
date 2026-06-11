package tools

import (
	"fmt"
	"os"
	"os/exec"
)

// InstallGoTool runs `go install` for a Go module path.
// tags is a comma-separated list of build tags (e.g. "postgres"), or empty.
func InstallGoTool(modulePath, name, tags string) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go is required to install %s: %w", name, err)
	}

	args := []string{"install"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, modulePath)

	fmt.Printf("  → Installing %s via go install %s...\n", name, modulePath)
	c := exec.Cmd{
		Path:   goBin,
		Args:   append([]string{goBin}, args...),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w\n  Manual install: go install -tags '%s' %s", name, err, tags, modulePath)
	}
	fmt.Printf("  ✔ %s installed successfully\n", name)
	return nil
}
