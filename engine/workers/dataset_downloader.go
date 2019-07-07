package workers

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"

	"github.com/otiai10/copy"

	"github.com/cavaliercoder/grab"
	"github.com/pkg/errors"
)

const downloadFilename = "file.bin"

// DatasetDownloadListener periodically checks if there are any datasets which have been created
// with source set to "download" but the download hasn't been successfully performed yet.
func (context Context) DatasetDownloadListener() {

	for {
		dataset, err := context.ModelContext.LockDataset(model.F{"source": types.DatasetDownload, "status": types.DatasetCreated}, context.ProcessID, "", "")
		if err == nil {
			go context.DatasetDownloadWorker(dataset)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		// If the local file source is a directory, then in this step we will copy it to our repository to ensure stability.
		dataset, err = context.ModelContext.LockDataset(model.F{"source": types.DatasetLocal, "status": types.DatasetCreated}, context.ProcessID, "", "")
		if err == nil {
			go context.DatasetLocalCopyWorker(dataset)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		time.Sleep(context.Period)
	}

}

// DatasetDownloadWorker performs the actual dataset download.
func (context Context) DatasetDownloadWorker(dataset types.Dataset) {

	// Get the download target directory.
	path, err := context.StorageContext.GetDatasetPath(dataset.ID, ".download")
	if err != nil {
		// This means that we cannot access the file system, so we need to panic.
		panic(err)
	}

	context.Logger.WithFields(
		"dataset-id", dataset.ID,
		"source", dataset.Source,
		"source-address", dataset.SourceAddress,
		"destination-dir", path,
	).WriteInfo("DATASET TRANSFER COMPLETED")

	// Perform download.
	resp, err := grab.Get(filepath.Join(path, downloadFilename), dataset.SourceAddress)
	if err != nil {

		err = errors.WithStack(err)
		context.Logger.WithFields(
			"dataset-id", dataset.ID,
			"source", dataset.Source,
			"source-address", dataset.SourceAddress,
		).WithStack(err).WithError(err).WriteError("DATASET TRANSFER ERROR")

		context.repeatUntilSuccess(func() error {
			return context.ModelContext.UpdateDatasetStatus(dataset.ID, types.DatasetError, err.Error())
		})

		return
	}

	// Unlock the dataset and update the status.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UpdateDatasetStatus(dataset.ID, types.DatasetTransferred, "")
	})
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockDataset(dataset.ID, context.ProcessID)
	})

	// Log task completion.
	context.Logger.WithFields(
		"dataset-id", dataset.ID,
		"source", dataset.Source,
		"source-address", dataset.SourceAddress,
		"destination-path", resp.Filename,
	).WriteInfo("DATASET TRANSFER COMPLETED")
}

// DatasetLocalCopyWorker copies the local dataset if it is a directory.
func (context Context) DatasetLocalCopyWorker(dataset types.Dataset) {

	// Get the download target directory.
	path, err := context.StorageContext.GetDatasetPath(dataset.ID, "")
	if err != nil {
		// This means that we cannot access the file system, so we need to panic.
		panic(err)
	}

	context.Logger.WithFields(
		"dataset-id", dataset.ID,
		"source", dataset.Source,
		"source-address", dataset.SourceAddress,
		"destination-path", path,
	).WriteInfo("DATASET TRANSFER STARTED")

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

	if fileInfo.IsDir() == false {
		// Unlock the dataset and update the status.
		context.repeatUntilSuccess(func() error {
			return context.ModelContext.UpdateDatasetStatus(dataset.ID, types.DatasetTransferred, "")
		})
		context.repeatUntilSuccess(func() error {
			return context.ModelContext.UnlockDataset(dataset.ID, context.ProcessID)
		})

		// Log task completion.
		context.Logger.WithFields(
			"dataset-id", dataset.ID,
			"source", dataset.Source,
			"source-address", dataset.SourceAddress,
		).WriteInfo("DATASET TRANSFER SKIPPING, WILL UNPACK")

		return
	}

	// Do the actual copy.
	err = copy.Copy(dataset.SourceAddress, path)
	if err != nil {
		err = errors.WithStack(err)
		context.Logger.WithFields(
			"dataset-id", dataset.ID,
			"source", dataset.Source,
			"source-address", dataset.SourceAddress,
		).WithStack(err).WithError(err).WriteError("DATASET TRANSFER ERROR")

		context.repeatUntilSuccess(func() error {
			return context.ModelContext.UpdateDatasetStatus(dataset.ID, types.DatasetError, err.Error())
		})

		return
	}

	// Unlock the dataset and update the status.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UpdateDatasetStatus(dataset.ID, types.DatasetTransferred, "")
	})
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockDataset(dataset.ID, context.ProcessID)
	})

	// Log task completion.
	context.Logger.WithFields(
		"dataset-id", dataset.ID,
		"source", dataset.Source,
		"source-address", dataset.SourceAddress,
		"destination-path", path,
	).WriteInfo("DATASET TRANSFER COMPLETED")
}
