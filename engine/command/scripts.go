package command

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

func runScriptAsync(script string, args []string, outputWriter, errorWriter io.Writer) (output string, err error) {
	cmdArgs := []string{"-s", "--"}
	if args != nil {
		cmdArgs = append(cmdArgs, args...)
	}

	scriptCommand := exec.Command("/bin/sh", cmdArgs...)
	scriptCommand.Stdin = strings.NewReader(script)
	stdoutIn, _ := scriptCommand.StdoutPipe()
	stderrIn, _ := scriptCommand.StderrPipe()

	// By default we will copy standard output and standard error to a collection buffer.
	var stdoutBuf, stderrBuf bytes.Buffer
	var stdout io.Writer = &stdoutBuf
	var stderr io.Writer = &stderrBuf

	// If additional writers were passed, we will output to them as well.
	if outputWriter != nil {
		stdout = io.MultiWriter(outputWriter, &stdoutBuf)
	}
	if errorWriter != nil {
		stderr = io.MultiWriter(errorWriter, &stderrBuf)
	}

	err = scriptCommand.Start()
	if err != nil {
		return "", errors.Wrap(err, "script command start error")
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var errStdout error
	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
		wg.Done()
	}()

	_, errStderr := io.Copy(stderr, stderrIn)
	wg.Wait()
	if err != nil {
		return "", errors.Wrap(err, "script command start error")
	}

	err = scriptCommand.Wait()
	if err != nil {
		return "", errors.Wrap(err, "script command run error")
	}
	if errStdout != nil || errStderr != nil {
		return "", errors.Errorf("failed to capture stdout or stderr")
	}

	output = string(stdoutBuf.Bytes())
	errorString := string(stderrBuf.Bytes())
	if errorString != "" {
		err = errors.Errorf("command produced error output: %s", errorString)
	}

	return
}

func runScript(script string, args []string) (output string, err error) {
	return runScriptAsync(script, args, nil, nil)
}

func runEmbeddedScriptAsync(scriptName string, args []string, outputWriter, errorWriter io.Writer) (output string, err error) {
	script, err := scriptBox.FindString(scriptName)
	if err != nil {
		return "", errors.Wrap(err, "error while trying to find script")
	}
	return runScriptAsync(script, args, outputWriter, errorWriter)
}

func runEmbeddedScript(scriptName string, args []string) (output string, err error) {
	return runEmbeddedScriptAsync(scriptName, args, nil, nil)
}
