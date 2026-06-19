//go:build !unix

package e2e

import (
	"os"
	"os/exec"
)

func configureCommandProcessGroup(cmd *exec.Cmd) {}

func signalCommandProcessGroup(cmd *exec.Cmd, sig os.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Signal(sig)
}

func processGroupTerminateSignal() os.Signal { return os.Interrupt }

func processGroupKillSignal() os.Signal { return os.Kill }
