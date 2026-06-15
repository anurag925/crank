//go:build unix

package e2e

import (
	"os"
	"os/exec"
	"syscall"
)

func configureCommandProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalCommandProcessGroup(cmd *exec.Cmd, sig os.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	sysSig, ok := sig.(syscall.Signal)
	if !ok {
		return cmd.Process.Signal(sig)
	}
	return syscall.Kill(-cmd.Process.Pid, sysSig)
}

func processGroupTerminateSignal() os.Signal { return syscall.SIGTERM }

func processGroupKillSignal() os.Signal { return syscall.SIGKILL }
