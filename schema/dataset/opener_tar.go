package dataset

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"path"
	"strings"
)

// DataBuffer contains data that can be read and written.
type DataBuffer struct {
	Buffer   bytes.Buffer
	ReadOnly bool
}

// Read reads bytes from the buffer.
func (b *DataBuffer) Read(p []byte) (n int, err error) {
	return b.Buffer.Read(p)
}

// Write writes to the buffer.
func (b *DataBuffer) Write(p []byte) (n int, err error) {
	if b.ReadOnly {
		panic("write operation not supported")
	}

	return b.Buffer.Write(p)
}

// Close closes the reader.
func (b *DataBuffer) Close() error {
	return nil
}

// TarDir is.
type TarDir map[string]interface{}

// TarOpener is.
type TarOpener struct {
	Root TarDir
}

// NewTarOpener creates a new empty tar opener.
func NewTarOpener() *TarOpener {
	return &TarOpener{Root: TarDir{}}
}

// LoadTarOpener instantiates a new TarOpener from the given tar reader.
func LoadTarOpener(reader *tar.Reader) (*TarOpener, error) {
	result := NewTarOpener()
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Typeflag == tar.TypeReg {
			dirName, fileName := path.Split(header.Name)
			dirName = strings.TrimSuffix(dirName, "/")
			dir, err := getTarDir(result.Root, dirName, false)
			if err != nil {
				return nil, err
			}
			// We keep file data as a buffer instance.
			file := &DataBuffer{Buffer: bytes.Buffer{}, ReadOnly: true}
			size, err := io.Copy(&file.Buffer, reader)
			if err == io.EOF {
				if int64(size) != header.Size {
					return nil, errors.New("size of file and size in header missmatch for file " + header.Name)
				}
			} else if err != nil {
				return nil, err
			}

			dir[fileName] = file
		} else if header.Typeflag == tar.TypeDir {
			dirName := strings.TrimSuffix(header.Name, "/")
			_, err := getTarDir(result.Root, dirName, false)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func getTarDir(root TarDir, dirpath string, readOnly bool) (TarDir, error) {
	var err error
	parentPath, childName := path.Split(dirpath)
	tardir := root
	if parentPath != "" {
		parentPath = strings.TrimSuffix(parentPath, "/")
		tardir, err = getTarDir(root, parentPath, readOnly)
		if err != nil {
			return nil, err
		}
	}

	// Allow creation of directories only in read only mode.
	if child, ok := tardir[childName]; ok == false && readOnly == false {
		tardir[childName] = TarDir{}
		return tardir[childName].(TarDir), nil
	} else if dirChild, ok := child.(TarDir); ok {
		return dirChild, nil
	}
	if readOnly {
		return nil, errors.New("directory not found: " + dirpath)
	}
	return nil, errors.New("name conflict: " + dirpath)
}

// DumpTarOpener converts a tar opener to a bytes reader.
func DumpTarOpener(opener *TarOpener) (io.Reader, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := writeTarDir(opener.Root, tw, "")
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func writeTarDir(dir TarDir, writer *tar.Writer, dirpath string) error {

	for k, v := range dir {

		name := path.Join(dirpath, k)

		if subdir, ok := v.(TarDir); ok {

			// Create TAR directory.
			header := &tar.Header{
				Name:     name + "/",
				Mode:     0600,
				Typeflag: tar.TypeDir,
			}
			writer.WriteHeader(header)

			err := writeTarDir(subdir, writer, name)
			if err != nil {
				return err
			}
		} else if data, ok := v.(*DataBuffer); ok {

			// Create TAR file.
			header := &tar.Header{
				Name:     name,
				Size:     int64(data.Buffer.Len()),
				Mode:     0600,
				Typeflag: tar.TypeReg,
			}
			err := writer.WriteHeader(header)
			if err != nil {
				return err
			}
			_, err = writer.Write(data.Buffer.Bytes())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetFile is.
func (opener *TarOpener) GetFile(root string, relPath string, readOnly bool, binary bool) (io.ReadWriteCloser, error) {

	fullPath := path.Join(root, relPath)
	dirName, fileName := path.Split(fullPath)
	dir, err := getTarDir(opener.Root, dirName, readOnly)
	if err != nil {
		return nil, err
	}

	var result io.ReadWriteCloser

	if readOnly {

		file, ok := dir[fileName]
		if ok == false {
			return nil, errors.New("file not found: " + fullPath)
		}

		if data, ok := file.(*DataBuffer); ok {
			result = data
		} else {
			return nil, errors.New("path is not a file: " + fullPath)
		}

	} else {

		result = &DataBuffer{ReadOnly: false, Buffer: bytes.Buffer{}}
		dir[fileName] = result

	}

	return result, nil
}

// GetDir is.
func (opener *TarOpener) GetDir(root string, relPath string, readOnly bool) ([]string, error) {

	fullPath := path.Join(root, relPath)
	dir, err := getTarDir(opener.Root, fullPath, readOnly)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(dir))
	i := 0
	for k := range dir {
		result[i] = k
		i++
	}

	return result, nil
}
