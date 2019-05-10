package dataset

import (
	"bufio"
	"fmt"
	"path"
	"strings"
)

// Class is.
type Class struct {
	Name       string
	Categories []string
}

// Type is.
func (f Class) Type() string { return "class" }

// Subtype is.
func (f Class) Subtype() string { return "default" }

func loadClass(root string, relPath string, name string, opener Opener, metadataOnly bool, subtype string) (File, error) {
	path := path.Join(relPath, name+TypeExtensions["class"][subtype])
	file, err := opener.GetFile(root, path, true, false)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	categoriesSet := map[string]interface{}{}
	categories := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if _, ok := categoriesSet[line]; ok {
			return nil, &datasetError{err: "Class file contains duplicate entries.", path: path}
		}
		categories = append(categories, line)
		categoriesSet[line] = nil
	}

	return &Class{Name: name, Categories: categories}, nil
}

func (f *Class) dump(root string, relPath string, name string, opener Opener) error {
	path := path.Join(relPath, name) + TypeExtensions["category"][f.Subtype()]
	file, err := opener.GetFile(root, path, false, false)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for i := range f.Categories {
		fmt.Fprintln(writer, f.Categories[i])
	}
	return writer.Flush()
}
