//go:build linux
// +build linux

package kosmixutil

import (
	"errors"
	"os/exec"
	"syscall"
)

func PauseExec(command *exec.Cmd) error {
	if command == nil {
		return errors.New("command is nil")
	}
	return command.Process.Signal(syscall.SIGSTOP)
}

func ResumeExec(command *exec.Cmd) error {
	if command == nil {
		return errors.New("command is nil")
	}
	return command.Process.Signal(syscall.SIGCONT)
}
