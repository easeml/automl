package workers

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"github.com/ds3lab/easeml/engine/easeml/database/model/types"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"

	"log"
)

// DatasetUnpackListener periodically checks if there are any datasets which have been transferred (either through
// upload or download) and unpacks them.
func (context Context) DatasetUnpackListener() {

	for {
		dataset, err := context.ModelContext.LockDataset(model.F{"status": types.DatasetTransferred}, context.ProcessID, "", "")
		if err == nil {
			log.Printf("DATASET FOUND FOR UNPACK")
			go context.DatasetUnpackWorker(dataset)
		} else if errors.Cause(err) == model.ErrNotFound {
			time.Sleep(context.Period)
		} else {
			panic(err)
		}
	}

}

// DatasetUnpackWorker performs the unpacking operation.
func (context Context) DatasetUnpackWorker(dataset types.Dataset) {

	// Get the download target directory.
	datasetPath, err := context.StorageContext.GetDatasetPath(dataset.ID, "")
	if err != nil {
		// This means that we cannot access the file system, so we need to panic.
		panic(err)
	}

	sourceFilePaths := []string{}
	destinationFilePaths := []string{}

	if dataset.Source == types.DatasetDownload {

		// Build the downloaded file path and unpack destination.
		sourceFilePaths = []string{filepath.Join(datasetPath, ".download", downloadFilename)}
		destinationFilePaths = []string{datasetPath}

	} else if dataset.Source == types.DatasetUpload {

		// Find all uploaded files and their target paths.
		sourceFilePaths, destinationFilePaths = getUploadedFilesDestinationPaths(datasetPath, "")

	} else if dataset.Source == types.DatasetLocal {

		// Check if the source address points to a file.
		fileInfo, err := os.Stat(dataset.SourceAddress)
		if err != nil {
			err = errors.WithStack(err)
			context.Logger.WithFields(
				"dataset-id", dataset.ID,
				"source", dataset.Source,
				"source-address", dataset.SourceAddress,
			).WithStack(err).WithError(err).WriteError("DATASET SOURCE ACCESS ERROR")

			context.repeatUntilSuccess(func() error {
				return context.ModelContext.UpdateDatasetStatus(dataset.ID, types.DatasetError, err.Error())
			})

			return
		}

		// If the source is a file, we assume it is an archive and try to extract from it.
		if fileInfo.IsDir() == false {
			sourceFilePaths = []string{dataset.SourceAddress}
			destinationFilePaths = []string{datasetPath}
		}

	}

	// Unpack all the source files.
	for i := range sourceFilePaths {
		arch := archiver.MatchingFormat(sourceFilePaths[i])
		if arch == nil {
			err = errors.New("unknown format")
			context.Logger.WithFields(
				"dataset-id", dataset.ID,
				"source", dataset.Source,
				"source-address", dataset.SourceAddress,
			).WithStack(err).WithError(err).WriteError("DATASET UNPACK FAILED")

			_, err := context.ModelContext.UpdateDataset(dataset.ID, model.F{"status": types.DatasetError, "status-message": err.Error()})
			if err != nil {
				panic(err)
			}

			return
		}
		err = arch.Open(sourceFilePaths[i], destinationFilePaths[i])
		if err != nil {
			err = errors.WithStack(err)
			context.Logger.WithFields(
				"dataset-id", dataset.ID,
				"source", dataset.Source,
				"source-address", dataset.SourceAddress,
			).WithStack(err).WithError(err).WriteError("DATASET UNPACK FAILED")

			_, err := context.ModelContext.UpdateDataset(dataset.ID, model.F{"status": types.DatasetError, "status-message": err.Error()})
			if err != nil {
				panic(err)
			}

			return
		}
	}

	// Delete the temp directories.
	if dataset.Source == types.DatasetDownload {
		err = os.RemoveAll(filepath.Join(datasetPath, ".download"))
		if err != nil {
			panic(err)
		}
	} else if dataset.Source == types.DatasetUpload {
		err = os.RemoveAll(filepath.Join(datasetPath, ".upload"))
		if err != nil {
			panic(err)
		}
	}

	// Unlock the dataset and update the status.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UpdateDatasetStatus(dataset.ID, types.DatasetUnpacked, "")
	})
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockDataset(dataset.ID, context.ProcessID)
	})

	// Log task completion.
	context.Logger.WithFields(
		"dataset-id", dataset.ID,
		"source", dataset.Source,
		"source-address", dataset.SourceAddress,
	).WriteInfo("DATASET UNPACK COMPLETED")
}
