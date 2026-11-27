//go:build !windows

package export

import (
	"os/exec"
	"syscall"
	"time"
)

// killWithChildren makes the draw.io wrapper the leader of its own process
// group and kills the whole group when the command's context ends: the wrapper
// starts dbus, xvfb-run and draw.io as children, and a child that keeps the
// output pipe open would otherwise block the wait even after the wrapper is
// gone (#716).
func killWithChildren(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 2 * time.Second
}
