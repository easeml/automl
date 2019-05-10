package workers

import (
	ctx "context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"github.com/ds3lab/easeml/engine/easeml/database/model/types"

	"github.com/cavaliercoder/grab"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

// ModuleDownloadListener periodically checks if there are any modules which have been created
// with source set to "download" but the download hasn't been successfully performed yet.
func (context Context) ModuleDownloadListener() {

	for {
		module, err := context.ModelContext.LockModule(model.F{"source": types.ModuleDownload, "status": types.ModuleCreated}, context.ProcessID, "", "")
		if err == nil {
			go context.ModuleDownloadWorker(module)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		module, err = context.ModelContext.LockModule(model.F{"source": types.ModuleLocal, "status": types.ModuleCreated}, context.ProcessID, "", "")
		if err == nil {
			go context.ModuleLocalCopyWorker(module)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		module, err = context.ModelContext.LockModule(model.F{"source": types.ModuleRegistry, "status": types.ModuleCreated}, context.ProcessID, "", "")
		if err == nil {
			go context.ModuleRegistryPullWorker(module)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		time.Sleep(context.Period)
	}

}

// ModuleDownloadWorker performs the actual module download.
func (context Context) ModuleDownloadWorker(module types.Module) {

	// Get the download target directory.
	path, err := context.StorageContext.GetModulePath(module.ID, module.Type, ".download")
	if err != nil {
		// This means that we cannot access the file system, so we need to panic.
		panic(err)
	}

	// Perform download.
	resp, err := grab.Get(filepath.Join(path, downloadFilename), module.SourceAddress)
	if err != nil {

		err = errors.WithStack(err)
		context.Logger.WithFields(
			"module-id", module.ID,
			"source", module.Source,
			"source-address", module.SourceAddress,
		).WithStack(err).WithError(err).WriteError("MODULE TRANSFER ERROR")

		context.repeatUntilSuccess(func() error {
			return context.ModelContext.UpdateModuleStatus(module.ID, types.ModuleError, err.Error())
		})

		return
	}

	// Unlock the module and update the status.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UpdateModuleStatus(module.ID, types.ModuleTransferred, "")
	})
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockModule(module.ID, context.ProcessID)
	})

	// Log task completion.
	context.Logger.WithFields(
		"module-id", module.ID,
		"source", module.Source,
		"source-address", module.SourceAddress,
		"destination-path", resp.Filename,
	).WriteInfo("MODULE TRANSFER COMPLETED")
}

// ModuleLocalCopyWorker copies the local module if it is a directory.
func (context Context) ModuleLocalCopyWorker(module types.Module) {

	// Check if the source address points to a file.
	fileInfo, err := os.Stat(module.SourceAddress)
	if err != nil || fileInfo.IsDir() {

		if err == nil {
			err = errors.New("the module source must be a file and not a directory")
		} else {
			err = errors.WithStack(err)
		}

		context.Logger.WithFields(
			"module-id", module.ID,
			"source", module.Source,
			"source-address", module.SourceAddress,
		).WithStack(err).WithError(err).WriteError("MODULE SOURCE ACCESS ERROR")

		context.repeatUntilSuccess(func() error {
			return context.ModelContext.UpdateModuleStatus(module.ID, types.ModuleError, err.Error())
		})

		return
	}

	// Get the download target directory.
	path, err := context.StorageContext.GetModulePath(module.ID, module.Type, "")
	if err != nil {
		// This means that we cannot access the file system, so we need to panic.
		panic(err)
	}

	// Do the actual copy.
	err = copy.Copy(module.SourceAddress, path)
	if err != nil {
		err = errors.WithStack(err)
		context.Logger.WithFields(
			"module-id", module.ID,
			"source", module.Source,
			"source-address", module.SourceAddress,
		).WithStack(err).WithError(err).WriteError("MODULE TRANSFER ERROR")

		context.repeatUntilSuccess(func() error {
			return context.ModelContext.UpdateModuleStatus(module.ID, types.ModuleError, err.Error())
		})

		return
	}

	// Unlock the module and update the status.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UpdateModuleStatus(module.ID, types.ModuleTransferred, "")
	})
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockModule(module.ID, context.ProcessID)
	})

	// Log task completion.
	context.Logger.WithFields(
		"module-id", module.ID,
		"source", module.Source,
		"source-address", module.SourceAddress,
		"destination-path", path,
	).WriteInfo("MODULE TRANSFER COMPLETED")
}

// ModuleRegistryPullWorker copies the local module if it is a directory.
func (context Context) ModuleRegistryPullWorker(module types.Module) {

	// TODO: Get API version automatically.
	// See: https://stackoverflow.com/a/48638182
	cli, err := client.NewClientWithOpts(client.WithVersion("1.37"))
	if err != nil {
		panic(err)
	}

	// Pull image from registry.
	resp, err := cli.ImagePull(ctx.Background(), module.SourceAddress, dockertypes.ImagePullOptions{})
	defer resp.Close()
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp)
	if err != nil {
		panic(err)
	}
	// TODO: Maybe this is not needed. Maybe save it as log. Or maybe just discard.
	log.Printf(string(body))

	// Get the download target directory.
	path, err := context.StorageContext.GetModulePath(module.ID, module.Type, "")
	if err != nil {
		// This means that we cannot access the file system, so we need to panic.
		panic(err)
	}

	// Save the image as a TAR.
	reader, err := cli.ImageSave(ctx.Background(), []string{module.SourceAddress})
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	f, err := os.Create(filepath.Join(path, "module.tar"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	if err != nil {
		panic(err)
	}

	// Unlock the module and update the status.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UpdateModuleStatus(module.ID, types.ModuleTransferred, "")
	})
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockModule(module.ID, context.ProcessID)
	})

	// Log task completion.
	context.Logger.WithFields(
		"module-id", module.ID,
		"source", module.Source,
		"source-address", module.SourceAddress,
	).WriteInfo("MODULE TRANSFER COMPLETED")
}
