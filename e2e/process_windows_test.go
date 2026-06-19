//go:build windows

package e2e

import "os/exec"

func configureProcessGroup(cmd *exec.Cmd) {}

func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
