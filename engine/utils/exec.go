package utils

import (
	"bytes"
	"os/exec"
	"syscall"
)

// ExecExternal is a helper function to run an external command
func ExecExternal(dir string,name string, arg ...string)  (outStr string, errStr string, err error) {
	cmd := exec.Command(name, arg...)
	// Creates a new session without a controlling terminal
	// It has the effect of letting the executed command fail if it needs external input
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if dir != ""{
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	outStr, errStr = string(stdout.Bytes()), string(stderr.Bytes())

	return outStr, errStr, err
}
