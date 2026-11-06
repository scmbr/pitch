//go:build windows

package slides

import (
	"os/exec"
	"strconv"
	"syscall"
)

func configureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	// Terminate only this conversion's process tree on timeout/application exit.
	cmd.Cancel = func() error {
		kill := exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
		kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return kill.Run()
	}
}
