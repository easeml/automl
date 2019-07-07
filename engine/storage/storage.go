package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ds3lab/easeml/engine/database/model/types"
)

// Context contains the storage model context with all information needed to access the working directory.
type Context struct {
	WorkingDir string
}

const (
	// Pattern: shared/images/models/{user-id}/{module-id}
	modelPathTemplate = "/shared/images/models/%s/%s"

	// Pattern: shared/images/objectives/{user-id}/{module-id}
	objectivePathTemplate = "/shared/images/objectives/%s/%s"

	// Pattern: shared/images/optimizers/{user-id}/{module-id}
	optimizerPathTemplate = "/shared/images/optimizers/%s/%s"

	// Pattern: shared/data/stable/{user-id}/{datased-id}
	datasetPathTemplate = "/shared/data/stable/%s/%s"

	// Pattern: /shared/jobs/{job-id}/{task-id}
	taskPathTemplate = "/shared/jobs/%s/%s"

	// Pattern: scheduling/input
	schedulingInputPathTemplate = "/shared/scheduling/input"

	// Pattern: individual/{host-id}-{process-id}
	processDirPathTemplate = "/individual/%s-%s"

	// Pattern: individual/{host-id}-{process-id}/{start-time}.log
	processLogPathTemplate = processDirPathTemplate + "/%s.log"

	// Pattern: individual/{host-id}-{process-id}/{job-id}/{task-id}
	taskDirPathTemplate = processDirPathTemplate + "/%s/%s"

	// Pattern: /shared/processes/{process-id}
	processPathTemplate = "/shared/processes/%s"
)

// DefaultFilePerm is the default file mode to be used when creating directories.
const DefaultFilePerm = os.FileMode(0755)

// ClearDirectory removes all files in a directory.
func ClearDirectory(dirpath string) (err error) {
	err = os.RemoveAll(dirpath)
	if err != nil {
		return
	}
	err = os.MkdirAll(dirpath, DefaultFilePerm)
	return
}

// GetProcessPath constructs the path for a given process, ensures it exists and returns the path string.
func (context Context) GetProcessPath(id string, subdir string) (path string, err error) {
	path = filepath.FromSlash(context.WorkingDir + fmt.Sprintf(processPathTemplate, id))
	if subdir != "" {
		path = filepath.Join(path, subdir)
	}

	err = os.MkdirAll(path, DefaultFilePerm)
	return
}

// GetDatasetPath constructs the path for a given dataset, ensures it exists and returns the path string.
func (context Context) GetDatasetPath(id string, subdir string) (path string, err error) {
	ids := strings.Split(id, "/")
	userID := ids[0]
	datasetID := ids[1]

	path = filepath.FromSlash(context.WorkingDir + fmt.Sprintf(datasetPathTemplate, userID, datasetID))
	if subdir != "" {
		path = filepath.Join(path, subdir)
	}

	err = os.MkdirAll(path, DefaultFilePerm)
	return
}

// GetModulePath constructs the path for a given module, ensures it exists and returns the path string.
func (context Context) GetModulePath(id string, moduleType string, subdir string) (path string, err error) {

	ids := strings.Split(id, "/")
	userID := ids[0]
	moduleID := ids[1]

	modulePathTemplate := ""
	switch moduleType {
	case types.ModuleModel:
		modulePathTemplate = modelPathTemplate
	case types.ModuleObjective:
		modulePathTemplate = objectivePathTemplate
	case types.ModuleOptimizer:
		modulePathTemplate = optimizerPathTemplate
	}

	path = filepath.FromSlash(context.WorkingDir + fmt.Sprintf(modulePathTemplate, userID, moduleID))
	if subdir != "" {
		path = filepath.Join(path, subdir)
	}

	err = os.MkdirAll(path, DefaultFilePerm)
	return
}

// GetSchedulingInputPath returns the path which holds temp data
func (context Context) GetSchedulingInputPath(subdir string) (path string, err error) {
	path = filepath.FromSlash(context.WorkingDir + schedulingInputPathTemplate)
	if subdir != "" {
		path = filepath.Join(path, subdir)
	}
	err = os.MkdirAll(path, DefaultFilePerm)
	return
}

// GetTaskPath constructs the path for a given dataset, ensures it exists and returns the path string.
func (context Context) GetTaskPath(id string, subdir string) (path string, err error) {

	ids := strings.Split(id, "/")
	jobID := ids[0]
	taskID := ids[1]

	path = filepath.FromSlash(context.WorkingDir + fmt.Sprintf(taskPathTemplate, jobID, taskID))
	if subdir != "" {
		path = filepath.Join(path, subdir)
	}

	err = os.MkdirAll(path, DefaultFilePerm)
	return
}

// GetAllTaskPaths constructs and returns all paths during storage execution.
func (context Context) GetAllTaskPaths(id string) (paths TaskPaths, err error) {
	if paths.Parameters, err = context.GetTaskPath(id, "parameters"); err != nil {
		return
	}
	if paths.Predictions, err = context.GetTaskPath(id, "predictions"); err != nil {
		return
	}

	// Make sure predictions contains two subdirs: train and val.
	if _, err = context.GetTaskPath(id, filepath.Join("predictions", "train")); err != nil {
		return
	}
	if _, err = context.GetTaskPath(id, filepath.Join("predictions", "val")); err != nil {
		return
	}

	if paths.Evaluations, err = context.GetTaskPath(id, "evaluations"); err != nil {
		return
	}
	if paths.Logs, err = context.GetTaskPath(id, "logs"); err != nil {
		return
	}
	if paths.Config, err = context.GetTaskPath(id, "config"); err != nil {
		return
	}
	if paths.Debug, err = context.GetTaskPath(id, "debug"); err != nil {
		return
	}
	return
}

// TaskPaths is used to store all relevant storage paths used during task execution.
type TaskPaths struct {
	Parameters  string
	Logs        string
	Debug       string
	Predictions string
	Evaluations string
	Config      string
}
