package dataset

import (
	"io"
	"os"
	"path"
)

// Opener is.
type Opener interface {
	GetFile(root string, relPath string, readOnly bool, binary bool) (io.ReadWriteCloser, error)
	GetDir(root string, relPath string, readOnly bool) ([]string, error)
}

// DefaultOpener is.
type DefaultOpener struct{}

// GetFile is.
func (DefaultOpener) GetFile(root string, relPath string, readOnly bool, binary bool) (io.ReadWriteCloser, error) {
	path := path.Join(root, relPath)
	flag := os.O_RDWR | os.O_CREATE
	if readOnly {
		flag = os.O_RDONLY
	}
	return os.OpenFile(path, flag, 0700)
}

// GetDir is.
func (DefaultOpener) GetDir(root string, relPath string, readOnly bool) ([]string, error) {
	path := path.Join(root, relPath)
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		if readOnly == false && os.IsNotExist(err) {
			err = os.MkdirAll(path, 0700)
			if err != nil {
				return nil, err
			}
			file, err = os.Open(path)
		} else {
			return nil, err
		}
	}
	return file.Readdirnames(0)
}
