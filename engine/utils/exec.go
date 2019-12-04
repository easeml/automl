package utils

import (
	"bytes"
	"os/exec"
)

// ExecExternal is a helper function to run an external command
func ExecExternal(dir string,name string, arg ...string)  (outStr string, errStr string, err error) {
	cmd := exec.Command(name, arg...)
	if dir != ""{
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	outStr, errStr = string(stdout.Bytes()), string(stderr.Bytes())

	// Debug output
	// log.Println("out:\n%s\nerr:\n%s\n", outStr, errStr)
	return outStr, errStr, err
}
