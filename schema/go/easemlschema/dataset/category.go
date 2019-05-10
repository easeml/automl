package dataset

import (
	"bufio"
	"fmt"
	"path"
	"strings"
)

// Category is.
type Category struct {
	Name       string
	Categories []string
}

// Type is.
func (f Category) Type() string { return "category" }

// Subtype is.
func (f Category) Subtype() string { return "default" }

func loadCategory(root string, relPath string, name string, opener Opener, metadataOnly bool, subtype string) (File, error) {
	path := path.Join(relPath, name+TypeExtensions["category"][subtype])
	file, err := opener.GetFile(root, path, true, false)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	categories := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		categories = append(categories, strings.TrimSpace(line))
	}

	return &Category{Name: name, Categories: categories}, nil
}

func (f *Category) dump(root string, relPath string, name string, opener Opener) error {
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

func (f *Category) belongsToSet(categorySet map[string]interface{}) bool {
	for i := range f.Categories {
		if _, ok := categorySet[f.Categories[i]]; ok == false {
			return false
		}
	}
	return true
}
