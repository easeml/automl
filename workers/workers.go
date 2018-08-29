package workers

import (
	"github.com/ds3lab/easeml/database/model"
	"github.com/ds3lab/easeml/logger"
	"github.com/ds3lab/easeml/storage"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/tus/tusd/filestore"
)

// Context contains all information needed to do work.
type Context struct {
	ModelContext   model.Context
	StorageContext storage.Context
	ProcessID      bson.ObjectId
	Period         time.Duration
	Logger         logger.Logger
}

// Clone makes a copy of the mongo session.
func (context Context) Clone() (clonedContext Context) {
	clonedContext = context
	clonedContext.ModelContext.Session = context.ModelContext.Session.Copy()
	return
}

func (context Context) repeatUntilSuccess(function func() error) {

	timeout := time.Second

	for err := function(); err != nil; {

		// If not found then the operation had bad parameters.
		if errors.Cause(err) == model.ErrNotFound {
			panic(err)
		}

		context.Logger.WithFields(
			"timeout", timeout,
		).WithStack(err).WithError(err).WriteError("DATABASE OPERATION FAILED")
		time.Sleep(timeout)
		timeout *= 2
	}

}

func (context Context) getModuleImagePath(moduleID, moduleType string) string {
	// Get the module directory.
	path, err := context.StorageContext.GetModulePath(moduleID, moduleType, "")
	if err != nil {
		// This means that we cannot access the file system, so we need to panic.
		panic(err)
	}

	// Find first tar file.
	matches, err := filepath.Glob(filepath.Join(path, "*.tar"))
	if err != nil {
		// This can only happen if the pattern is bad.
		panic(err)
	}
	if len(matches) != 1 {
		panic("there should be only one image tar file")
	}
	return matches[0]
}

func getUploadedFilesDestinationPaths(modulePath string, defaultFilename string) (sourceFilePaths []string, destinationFilePaths []string) {

	sourceFilePaths = []string{}
	destinationFilePaths = []string{}

	// Find all uploaded files and their target paths.
	uploadPath := filepath.Join(modulePath, ".upload")

	store := filestore.FileStore{
		Path: uploadPath,
	}
	files, err := ioutil.ReadDir(uploadPath)
	if err != nil {
		panic(err)
	}
	for i := range files {
		filename := files[i].Name()
		extension := filepath.Ext(filename)
		if extension != ".bin" {
			continue
		}
		name := filename[0 : len(filename)-len(extension)]

		fileInfo, err := store.GetInfo(name)
		if err != nil {
			panic(err)
		}

		// Check if metadata has a target filepath field.
		destRelPath, ok := fileInfo.MetaData["filepath"]
		if ok {
			os.MkdirAll(filepath.Join(modulePath, destRelPath), storage.DefaultFilePerm)
		} else {
			destRelPath = defaultFilename
		}

		sourceFilePaths = append(sourceFilePaths, filepath.Join(uploadPath, filename))
		destinationFilePaths = append(destinationFilePaths, filepath.Join(modulePath, destRelPath))
	}

	return
}
