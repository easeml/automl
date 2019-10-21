package command

import (
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var commandScript1 = `#!/bin/sh

echo -n "Hello World!"
`

var commandScript2 = `#!/bin/sh

echo -n $1
`

var commandScriptAsync1 = `#!/bin/sh

echo -n "Hello"
sleep 1
echo -n " World!"
`

func TestRunCommand(t *testing.T) {
	assert := assert.New(t)
	output, err := runScript(commandScript1, nil)

	assert.Nil(err)
	assert.Equal("Hello World!", output)
}

func TestRunCommandArgs(t *testing.T) {
	assert := assert.New(t)
	output, err := runScript(commandScript2, []string{"Hello World!"})

	assert.Nil(err)
	assert.Equal("Hello World!", output)
}

func TestRunCommandAsync(t *testing.T) {
	assert := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(1)

	var output string
	var err error

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		output, err = runScriptAsync(commandScriptAsync1, nil, pipeWriter, nil)
		wg.Done()
	}()

	// Read first word.
	data := make([]byte, 15)
	n, readErr := pipeReader.Read(data)
	assert.Nil(readErr)
	assert.Equal(5, n)

	// Read second word.
	n, readErr = pipeReader.Read(data)
	assert.Nil(readErr)
	assert.Equal(7, n)
	pipeReader.Close()

	wg.Wait()
	assert.Nil(err)
	assert.Equal("Hello World!", output)
}
