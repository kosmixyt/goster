//go:build windows
// +build windows

package kosmixutil

import (
	"errors"
	"os/exec"
)

func PauseExec(command *exec.Cmd) error {
	if command == nil {
		return errors.New("command is nil")
	}
	return errors.New("cannot pause command on windows")
	return nil
}
func ResumeExec(command *exec.Cmd) error {
	if command == nil {
		return errors.New("command is nil")
	}
	return errors.New("cannot resume command on windows")
	return nil
}
