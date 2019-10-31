package workers

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

		// If the file source is a gitlfs it should pull the dataset
		dataset, err = context.ModelContext.LockDataset(model.F{"source": types.DatasetGit, "status": types.DatasetCreated}, context.ProcessID, "", "")
		if err == nil {
			go context.DatasetGitWorker(dataset)
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
	).WriteInfo("DATASET TRANSFER STARTED")

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

func ExecExternal(dir string,name string, arg ...string)  (outStr string, errStr string, err error) {
	cmd := exec.Command(name, arg...)
	if dir != ""{
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	//Debug output
	outStr, errStr = string(stdout.Bytes()), string(stderr.Bytes())
	//log.Println("out:\n%s\nerr:\n%s\n", outStr, errStr)
	return outStr, errStr, err
}

// DatasetGitWorker performs the actual dataset fetch from a git lfs repository.
func (context Context) DatasetGitWorker(dataset types.Dataset) {

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
	).WriteInfo("DATASET GIT TRANSFER STARTED")

	splitedAddress := strings.Split(dataset.SourceAddress, "::")
	if len(splitedAddress) != 2{
		context.Logger.WithFields(
			"dataset-id", dataset.ID,
			"source", dataset.Source,
			"source-address", dataset.SourceAddress,
		).WriteError("DATASET GIT ADDRESS::FILE ERROR")
	}
	if dataset.Secret == "" || dataset.Secret=="expired"{
		context.Logger.WithFields(
			"dataset-id", dataset.ID,
			"source", dataset.Source,
			"source-address", dataset.SourceAddress,
			"destination-path", path,
		).WriteInfo("DATASET GIT EMPTY Secret")
	}else{
		if !strings.Contains(splitedAddress[0], "https://"){
			context.Logger.WithFields(
				"dataset-id", dataset.ID,
				"source", dataset.Source,
				"source-address", splitedAddress[0],
			).WriteError("DATASET GIT REPO ADDRESS ERROR")
		}else{
			splitedAddress[0]=strings.ReplaceAll(splitedAddress[0],"https://","")
			splitedAddress[0]=strings.Join([]string{"https://",dataset.Secret,"@",splitedAddress[0]},"")
		}
	}

	wdir:=filepath.Join(path, "temp.git")
	ExecExternal("","git","clone", "--depth=1", "--no-checkout", "--filter=blob:none",splitedAddress[0], wdir)
	ExecExternal(wdir,"git", "checkout", "master", "--", splitedAddress[1])
	ExecExternal(wdir, "git", "lfs", "pull", "-I", splitedAddress[1])

	//Flush used secret
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.FlushDatasetSecret(dataset.ID)
	})
	dataset.Secret=""
	splitedAddress[0]=""

	// Copy the file.
	err = copy.Copy(filepath.Join(wdir,splitedAddress[1]), filepath.Join(path, downloadFilename))
	if err != nil {
		err = errors.WithStack(err)
		context.Logger.WithFields(
			"dataset-id", dataset.ID,
			"source", dataset.Source,
			"source-address", dataset.SourceAddress,
		).WithStack(err).WithError(err).WriteError("DATASET GIT TRANSFER ERROR")

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
	).WriteInfo("DATASET GIT TRANSFER COMPLETED")
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
