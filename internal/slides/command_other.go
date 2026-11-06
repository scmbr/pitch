//go:build !windows

package slides

import "os/exec"

func configureCommand(_ *exec.Cmd) {}
