package handlers

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ds3lab/easeml/api"
	"github.com/ds3lab/easeml/api/responses"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

// Context is the placeholder for the common API context struct.
type Context api.Context

// ServeLocalResource is a utility function that accepts a path to a local resource (file or directory) and serves it.
// The path is given in two parts - as a base path (dataPath) and the relative path (relativePath). A file is served
// directly. A directory is served depending on the way it is specified. If the resource path has
// a suffix (.tar, .tar.gz or .zip) then the directory is packed into an archive. Otherwise, the we return
// a JSON response with fields "path" for relative path, "files" for file names and "directories" for
// directory names. We must also specify the modtime argument which is required for caching purposes.
func (apiContext Context) ServeLocalResource(dataPath, relativePath string, modtime time.Time, w http.ResponseWriter, r *http.Request) {

	// Check if the request URL points to a file.
	resourcePath := filepath.Join(dataPath, relativePath)
	fileInfo, err := os.Stat(resourcePath)
	if err != nil && os.IsNotExist(err) == false {
		// We ingore the not found error for now.
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	} else if os.IsNotExist(err) == false && fileInfo.IsDir() == false {
		// If the request URL points to a file, then we simply serve that file.
		http.ServeFile(w, r, resourcePath)
		return
	}

	// If the resource path is not a file, we try to see if it is a directory.
	// This works only if the request suffix is ".zip", ".tar" or ".tar.gz"
	var extension string
	var arch archiver.Archiver
	if strings.HasSuffix(resourcePath, ".tar") {
		arch = archiver.Zip
		extension = ".tar"
	} else if strings.HasSuffix(resourcePath, ".zip") {
		arch = archiver.Tar
		extension = ".zip"
	} else if strings.HasSuffix(resourcePath, ".tar.gz") {
		arch = archiver.TarGz
		extension = ".tar.gz"
	}

	if arch != nil {
		directoryPath := resourcePath[0 : len(resourcePath)-len(extension)]
		fileInfo, err := os.Stat(directoryPath)
		if err != nil && os.IsNotExist(err) == false {
			// We ingore the not found error for now.
			responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
			return
		} else if os.IsNotExist(err) == false && fileInfo.IsDir() == true {

			// We build a list of all files/directories in the dataset directory. The reason for this
			// is so that the archiver would not include the parent direcory in the archive.
			files, err := ioutil.ReadDir(directoryPath)
			if err != nil {
				responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
				return
			}
			sources := make([]string, len(files))
			for i := range files {
				sources[i] = filepath.Join(directoryPath, files[i].Name())
			}

			// If the request URL points to a directory, then we build an appropriate reader.
			var b bytes.Buffer
			arch.Write(&b, sources)
			http.ServeContent(w, r, relativePath, modtime, bytes.NewReader(b.Bytes()))
			return
		}
	}

	// Finally check if the resource is a directory. In that case, we simply return a list of files.
	fileInfo, err = os.Stat(resourcePath)
	if err != nil && os.IsNotExist(err) == false {
		// We ingore the not found error for now.
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	} else if os.IsNotExist(err) == false && fileInfo.IsDir() == true {
		var dirResponse struct {
			Path        string   `json:"path"`
			Files       []string `json:"files"`
			Directories []string `json:"directories"`
		}
		dirResponse.Path = relativePath
		files, err := ioutil.ReadDir(resourcePath)
		if err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
			return
		}
		for i := range files {
			if files[i].IsDir() {
				dirResponse.Directories = append(dirResponse.Directories, files[i].Name())
			} else {
				dirResponse.Files = append(dirResponse.Files, files[i].Name())
			}
		}
		responses.RespondWithJSON(w, http.StatusOK, dirResponse)
		return
	}

	// Fall back to 404 Not Found.
	responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), nil)
	return
}
