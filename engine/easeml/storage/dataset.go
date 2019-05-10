package storage

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ds3lab/easeml/engine/easeml/database/model"

	ds "github.com/ds3lab/easeml/schema/go/easemlschema/dataset"
	sch "github.com/ds3lab/easeml/schema/go/easemlschema/schema"

	"github.com/pkg/errors"
)

// InferDatasetProperties takes a dataset available on the file system and tries to infer
// its basic properties such as id, name and description.
func InferDatasetProperties(sourcePath string) (id, name, description string, err error) {

	// First check if the dataset exists.
	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(sourcePath)
	if err != nil {
		err = errors.Wrap(err, "dataset access error")
		return
	}

	// ID could be the name of the last path element.
	_, id = filepath.Split(sourcePath)

	// Remove all extensions. There could be multiple.
	for {

		ext := filepath.Ext(id)
		if ext == "" {
			break
		}
		id = strings.TrimSuffix(id, ext)
	}

	// Remove all unwanted characters.
	id = model.IDRegexpNegative.ReplaceAllString(id, "")

	// Name and description can be extracted from the README, if available.
	if fileInfo.IsDir() {
		// If the file is a directory, try to find a README file in its root.
		var matches []string
		matches, err = filepath.Glob(filepath.Join(sourcePath, "README*"))
		if err != nil {
			// The only possible returned error is ErrBadPattern, when pattern is malformed.
			panic(err)
		}
		if matches != nil && len(matches) > 0 {
			var file *os.File
			file, err = os.Open(matches[0])
			if err != nil {
				err = errors.Wrap(err, "error accessing readme file at "+matches[0])
			}
			name, description = ScanReadme(file)
		}
	} else {
		size := fileInfo.Size()
		var datasetFile *os.File
		datasetFile, err = os.Open(sourcePath)
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return
		}

		file := findFileInArchive(datasetFile, "README*", size)
		if file != nil {
			name, description = ScanReadme(file)
		}
	}

	return
}

func findFileInArchive(file *os.File, namePattern string, size int64) io.Reader {

	var archiveReader io.Reader = file

	// Check if it is gzip.
	if gzipReader, err := gzip.NewReader(file); err == nil {
		archiveReader = gzipReader
		defer gzipReader.Close()
	} else {
		// We seek to the beginning of the file. If we fail, then we musth abort.
		_, err := file.Seek(0, 0)
		if err != nil {
			return nil
		}
	}

	// Check if it is tar. If yes, try to find the readme.
	tarReader := tar.NewReader(archiveReader)
	for {
		header, err := tarReader.Next()

		if err != nil {
			// If get an EOF then we have reached the end of the tar.
			if err == io.EOF {
				return nil
			}
			// Otherwise we should try the zip.
			break
		}

		match, err := path.Match(namePattern, header.Name)
		if err != nil {
			// This can only happen if the pattern is bad.
			panic(err)
		}

		if match {
			// We read the file.
			buf := bytes.Buffer{}
			if _, err := io.Copy(&buf, tarReader); err != nil {
				return nil
			}
			return bytes.NewReader(buf.Bytes())
		}
	}

	// Check if it is zip.
	zipReader, err := zip.NewReader(file, size)
	if err != nil {
		return nil
	}
	for i := range zipReader.File {

		match, err := path.Match(namePattern, zipReader.File[i].Name)
		if err != nil {
			// This can only happen if the pattern is bad.
			panic(err)
		}
		if match {
			reader, err := zipReader.File[i].Open()
			if err != nil {
				return nil
			}
			defer reader.Close()
			buf := bytes.Buffer{}
			if _, err := io.Copy(&buf, reader); err != nil {
				return nil
			}
			return bytes.NewReader(buf.Bytes())
		}
	}

	return nil
}

// ScanReadme parser a readme file from the given reader and extracts the name and description.
func ScanReadme(r io.Reader) (name, description string) {

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {

		line := scanner.Text()

		// Check if we need to extract the name and if one is available. We look for a
		// single nonempty line with letters.
		if name == "" {
			namePattern := regexp.MustCompile(`[[:blank:][:alnum:]]+`)
			name = namePattern.FindString(line)
			name = strings.TrimSpace(name)

			// If the line was consumed we can move on.
			if name != "" {
				continue
			}
		}

		// We remove consecutive blank lines and leading blank lines.
		blankPattern := regexp.MustCompile(`^[[:blank:]]*$`)
		if blankPattern.MatchString(line) && (description == "" || strings.HasSuffix(description, "\n")) {
			continue
		}

		// If we have arrived here, then we can append the line to the description.
		description += line + "\n"
	}

	return
}

// InferDatasetSchema tries to infer the schema of a dataset.
func InferDatasetSchema(sourcePath string) (schemaIn, schemaOut *sch.Schema, err error) {

	// First check if the dataset exists.
	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(sourcePath)
	if err != nil {
		err = errors.Wrap(err, "dataset access error")
		return
	}

	// If the dataset is a directory, then we read it with the default opener.
	// Otherwise we will assume it's a tar or tar.gz archive.
	var opener ds.Opener
	var basePath string
	if fileInfo.IsDir() {

		// Each dataset must have a train and val directory. Each one of them must contain
		// an input and output directory.
		var exists bool
		exists, err = directoryEsists(filepath.Join(sourcePath, "train"))
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return nil, nil, err
		} else if exists == false {
			err = errors.New("datset root must contain a \"train\" directory")
			return nil, nil, err
		}
		exists, err = directoryEsists(filepath.Join(sourcePath, "val"))
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return nil, nil, err
		} else if exists == false {
			err = errors.New("datset root must contain a \"val\" directory")
			return nil, nil, err
		}
		exists, err = directoryEsists(filepath.Join(sourcePath, "train", "input"))
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return nil, nil, err
		} else if exists == false {
			err = errors.New("datset \"train\" directory must contain an \"input\" directory")
			return nil, nil, err
		}
		exists, err = directoryEsists(filepath.Join(sourcePath, "val", "input"))
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return nil, nil, err
		} else if exists == false {
			err = errors.New("datset \"val\" directory must contain an \"input\" directory")
			return nil, nil, err
		}
		exists, err = directoryEsists(filepath.Join(sourcePath, "train", "output"))
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return nil, nil, err
		} else if exists == false {
			err = errors.New("datset \"train\" directory must contain an \"output\" directory")
			return nil, nil, err
		}
		exists, err = directoryEsists(filepath.Join(sourcePath, "val", "output"))
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return nil, nil, err
		} else if exists == false {
			err = errors.New("datset \"val\" directory must contain an \"output\" directory")
			return nil, nil, err
		}

		// If we are here, then we can set the opener and procede with reading the schemas.
		opener = ds.DefaultOpener{}
		basePath = sourcePath

	} else {

		var datasetFile *os.File
		datasetFile, err = os.Open(sourcePath)
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return nil, nil, err
		}

		// Check if it is gzip.
		var archiveReader io.Reader = datasetFile
		if gzipReader, err := gzip.NewReader(datasetFile); err == nil {
			archiveReader = gzipReader
			defer gzipReader.Close()
		} else {
			// We seek to the beginning of the file. If we fail, then we musth abort.
			_, err := datasetFile.Seek(0, 0)
			if err != nil {
				err = errors.Wrap(err, "dataset access error")
				return nil, nil, err
			}
		}

		// Try to open it as tar.
		tarReader := tar.NewReader(archiveReader)
		var tarOpener *ds.TarOpener
		tarOpener, err = ds.LoadTarOpener(tarReader)
		if err != nil {
			err = errors.Wrap(err, "error while opening the tar archive")
			return nil, nil, err
		}

		if trainChild, ok := tarOpener.Root["train"]; ok == true {
			if trainDir, ok := trainChild.(ds.TarDir); ok == true {
				if trainInChild, ok := trainDir["input"]; ok == true {
					if _, ok := trainInChild.(ds.TarDir); ok == false {
						err = errors.New("datset \"train\" directory must contain an \"input\" directory")
						return nil, nil, err
					}
				} else {
					err = errors.New("datset \"train\" directory must contain an \"input\" directory")
					return nil, nil, err
				}
			} else {
				err = errors.New("datset root must contain a \"train\" directory")
				return nil, nil, err
			}
		} else {
			err = errors.New("datset root must contain a \"train\" directory")
			return nil, nil, err
		}

		if valChild, ok := tarOpener.Root["val"]; ok == true {
			if valDir, ok := valChild.(ds.TarDir); ok == true {
				if valInChild, ok := valDir["input"]; ok == true {
					if _, ok := valInChild.(ds.TarDir); ok == false {
						err = errors.New("datset \"val\" directory must contain an \"input\" directory")
						return nil, nil, err
					}
				} else {
					err = errors.New("datset \"val\" directory must contain an \"input\" directory")
					return nil, nil, err
				}
			} else {
				err = errors.New("datset root must contain a \"val\" directory")
				return nil, nil, err
			}
		} else {
			err = errors.New("datset root must contain a \"val\" directory")
			return nil, nil, err
		}

		// If we are here, then we can set the opener and procede with reading the schemas.
		opener = tarOpener
		basePath = ""

	}

	// We infer the schema from the train directory. It must contain
	// subdirectories input and output.
	var datasetIn, datasetOut *ds.Dataset
	datasetIn, err = ds.Load(filepath.Join(basePath, "train", "input"), true, opener)
	if err != nil {
		err = errors.Wrap(err, "dataset load error")
		return nil, nil, err
	}
	schemaIn, err = datasetIn.InferSchema()
	if err != nil {
		err = errors.Wrap(err, "dataset input schema inference error")
		return nil, nil, err
	}
	datasetOut, err = ds.Load(filepath.Join(basePath, "train", "output"), true, opener)
	if err != nil {
		err = errors.Wrap(err, "dataset load error")
		return nil, nil, err
	}
	schemaOut, err = datasetOut.InferSchema()
	if err != nil {
		err = errors.Wrap(err, "dataset output schema inference error")
		return nil, nil, err
	}

	var datasetValIn, datasetValOut *ds.Dataset
	var schemaValIn, schemaValOut *sch.Schema
	datasetValIn, err = ds.Load(filepath.Join(basePath, "val", "input"), true, opener)
	if err != nil {
		err = errors.Wrap(err, "dataset load error")
		return nil, nil, err
	}
	schemaValIn, err = datasetValIn.InferSchema()
	if err != nil {
		err = errors.Wrap(err, "dataset input schema inference error")
		return nil, nil, err
	}
	datasetValOut, err = ds.Load(filepath.Join(basePath, "train", "output"), true, opener)
	if err != nil {
		err = errors.Wrap(err, "dataset load error")
		return nil, nil, err
	}
	schemaValOut, err = datasetValOut.InferSchema()
	if err != nil {
		err = errors.Wrap(err, "dataset output schema inference error")
		return nil, nil, err
	}

	matchIn, _ := schemaIn.Match(schemaValIn, false)
	if matchIn == false {
		err = errors.New("input schemas in the training and validation sets do not match")
		return nil, nil, err
	}
	matchOut, _ := schemaOut.Match(schemaValOut, false)
	if matchOut == false {
		err = errors.New("output schemas in the training and validation sets do not match")
		return nil, nil, err
	}

	return schemaIn, schemaOut, nil
}

func directoryEsists(dirpath string) (bool, error) {
	trainDir, err := os.Stat(dirpath)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return false, err
	}
	return trainDir.IsDir(), nil
}
