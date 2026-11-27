package export

import (
	"os/exec"
	"time"
)

// killWithChildren stops waiting for the output pipes shortly after the
// command's context ends; os/exec kills the draw.io process itself.
func killWithChildren(cmd *exec.Cmd) {
	cmd.WaitDelay = 2 * time.Second
}
