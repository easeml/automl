package workers

import (
	"bufio"
	"encoding/json"
	"github.com/ds3lab/easeml/engine/database/model/types"
	"github.com/ds3lab/easeml/engine/modules"
	"github.com/ds3lab/easeml/engine/storage"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/otiai10/copy" // Surprised that there is no standard library for this
	"github.com/pkg/errors"
)

// OptimizerRunSuggestCreateTask runs the optimization sequence.
func (context Context) OptimizerRunSuggestCreateTask(optimizerID string, numProcesses, numTasks int,jobs []*types.Job) {
	// Get optimizer image.
	imageFilePath := context.getModuleImagePath(optimizerID, types.ModuleOptimizer)
	imageName, err := modules.LoadImage(imageFilePath)
	if err != nil {
		err = errors.WithStack(err)
		context.Logger.WithFields(
			"module-id", optimizerID,
		).WithStack(err).WithError(err).WriteError("OPTIMIZER LOAD ERROR")
	}

	// Write all job config spaces to directory.
	confPath, err := context.StorageContext.GetSchedulingInputPath("config")
	if err != nil {
		panic(err) // This means that we cannot access the file system, so we need to panic.
	}
	// Delete everything from the config path directory.
	err = storage.ClearDirectory(confPath)
	if err != nil {
		panic(err) // This means that we cannot access the file system, so we need to panic.
	}

	jobsDict := map[string]*types.Job{}
	for i := range jobs {
		jobsDict[jobs[i].ID.Hex()] = jobs[i]
		filename := filepath.Join(confPath, jobs[i].ID.Hex()+".json")
		ioutil.WriteFile(filename, []byte(jobs[i].ConfigSpace), storage.DefaultFilePerm)
	}

	histPath, err := context.StorageContext.GetSchedulingInputPath("history")
	if err != nil {
		// This means that we cannot access the file system, so we need to panic.
		panic(err)
	}

	err = copy.Copy(confPath,histPath)
	if err != nil {
		// This means that we cannot access the file system, so we need to panic.
		panic(err)
	}

	// Call optimizer.
	numNewTasks := numProcesses*2 - numTasks
	command := []string{
		"suggest",
		"--space", modules.MntPrefix + confPath,
		"--history", modules.MntPrefix + histPath,
		"--num-tasks", strconv.Itoa(numNewTasks),
	}
	outReader, err := modules.RunContainerAndCollectOutput(imageName, nil, command, nil)
	if err != nil {
		err = errors.Wrap(err, "docker container start error")
		context.Logger.WithFields(
			"module-id", optimizerID,
		).WithStack(err).WithError(err).WriteError("OPTIMIZER START ERROR")
		return
	}
	defer outReader.Close()

	// Define the item type.
	type modelConf struct {
		ID     string      `json:"id"`
		Config interface{} `json:"config"`
	}
	type jobConf struct {
		ID    string    `json:"id"`
		Model modelConf `json:"model"`
	}

	// Parse result and generate new tasks.
	scanner := bufio.NewScanner(outReader)
	for scanner.Scan() {
		line := scanner.Text()

		var conf jobConf
		err := json.Unmarshal([]byte(line), &conf)
		if err != nil {
			panic(err)
		}
		job := jobsDict[conf.ID]
		modelID := conf.Model.ID

		modelConfig, err := json.Marshal(conf.Model.Config)
		if err != nil {
			panic(err)
		}

		// Define new task.
		task := types.Task{
			Job:    job.ID,
			Model:  modelID,
			Pipeline: job.Pipeline,
			Config: string(modelConfig),
		}

		task, err = context.ModelContext.CreateTask(task)
		if err != nil {
			panic(err)
		}

		context.Logger.WithFields(
			"task-id", task.ID,
			"model", task.Model,
			"dataset", task.Dataset,
			"objective", task.Objective,
		).WriteInfo("SCHEDULED NEW TASK")
	}
}
