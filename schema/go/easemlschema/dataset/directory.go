package dataset

import (
	"path"
	"strings"
)

// Directory is.
type Directory struct {
	Name     string
	Children map[string]File
}

// Type is.
func (f Directory) Type() string { return "directory" }

// Subtype is.
func (f Directory) Subtype() string { return "default" }

func loadDirectory(root string, relPath string, name string, opener Opener, metadataOnly bool, subtype string) (File, error) {
	path := path.Join(relPath, name)
	files, err := opener.GetDir(root, path, true)
	if err != nil {
		return nil, err
	}

	children := map[string]File{}
	for i := range files {

		matched := false
		for fileType := range TypeExtensions {
			for fileSubtype, ext := range TypeExtensions[fileType] {
				if strings.HasSuffix(files[i], ext) {

					fileName := files[i][:len(files[i])-len(ext)]
					loader := LoaderFunctions[fileType]
					file, err := loader(root, path, fileName, opener, metadataOnly, fileSubtype)
					if err != nil {
						return nil, err
					}
					children[fileName] = file
					matched = true

				}
			}
		}

		if matched == false {

			fileName := files[i]
			directory, err := loadDirectory(root, path, fileName, opener, metadataOnly, "default")
			if err != nil {
				return nil, err
			}
			children[fileName] = directory

		}

	}

	return &Directory{Name: name, Children: children}, nil
}

func (f *Directory) dump(root string, relPath string, name string, opener Opener) (err error) {
	path := path.Join(relPath, name)
	_, err = opener.GetDir(root, path, false)
	if err != nil {
		return
	}

	for k, v := range f.Children {
		if tensor, ok := v.(*Tensor); ok {
			err = tensor.dump(root, path, k, opener)
		} else if category, ok := v.(*Category); ok {
			err = category.dump(root, path, k, opener)
		} else if links, ok := v.(*Links); ok {
			err = links.dump(root, path, k, opener)
		} else if class, ok := v.(*Class); ok {
			err = class.dump(root, path, k, opener)
		} else if directory, ok := v.(*Directory); ok {
			err = directory.dump(root, path, k, opener)
		} else {
			panic("unexpected child type")
		}
	}

	return
}
