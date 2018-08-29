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

func loadDirectory(root string, relPath string, name string, opener Opener, metadataOnly bool) (*Directory, error) {
	path := path.Join(relPath, name)
	files, err := opener.GetDir(root, path, true)
	if err != nil {
		return nil, err
	}

	children := map[string]File{}
	for i := range files {
		if strings.HasSuffix(files[i], TypeExtensions["tensor"]) {

			fileName := files[i][:len(files[i])-len(TypeExtensions["tensor"])]
			tensor, err := loadTensor(root, path, fileName, opener, metadataOnly)
			if err != nil {
				return nil, err
			}
			children[fileName] = tensor

		} else if strings.HasSuffix(files[i], TypeExtensions["category"]) {

			fileName := files[i][:len(files[i])-len(TypeExtensions["category"])]
			category, err := loadCategory(root, path, fileName, opener, metadataOnly)
			if err != nil {
				return nil, err
			}
			children[fileName] = category

		} else if strings.HasSuffix(files[i], TypeExtensions["links"]) {

			fileName := files[i][:len(files[i])-len(TypeExtensions["links"])]
			links, err := loadLinks(root, path, fileName, opener, metadataOnly)
			if err != nil {
				return nil, err
			}
			children[fileName] = links

		} else if strings.HasSuffix(files[i], TypeExtensions["class"]) {

			fileName := files[i][:len(files[i])-len(TypeExtensions["class"])]
			class, err := loadClass(root, path, fileName, opener, metadataOnly)
			if err != nil {
				return nil, err
			}
			children[fileName] = class

		} else {

			fileName := files[i]
			directory, err := loadDirectory(root, path, fileName, opener, metadataOnly)
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
