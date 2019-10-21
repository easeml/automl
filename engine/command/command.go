package command // import "github.com/ds3lab/easeml/engine/command"

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

func loadStream(source string) (string, error) {

	var file *os.File

	if source == "-" {
		file = os.Stdin
	} else {
		var err error
		file, err = os.Open(source)
		if err != nil {
			return "", errors.Wrap(err, "file open error")
		}
	}

	result, err := ioutil.ReadAll(file)
	if err != nil {
		return "", errors.Wrap(err, "read error")
	}

	return string(result), nil
}

func readLine(prompt string, result *string) error {
	if prompt != "" {
		fmt.Print(prompt)
	}
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		*result = scanner.Text()
	}
	return scanner.Err()
}
