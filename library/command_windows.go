//go:build windows

package library

import (
	"os"
	"os/exec"
	"syscall"
)

func RunCmd(prog string, args ...string) error {
	var err error
	cmd := exec.Command(prog, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err = cmd.Start(); err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}
