package workers

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"
	"github.com/ds3lab/easeml/engine/modules"
	"github.com/ds3lab/easeml/engine/storage"

	"github.com/pkg/errors"
)

// OptimizerRunListener periodically checks if there are any modules which have been transferred
// but have not yet been validated. It performs various checks to make sure the model is ready to
// become activated.
func (context Context) OptimizerRunListener(optimizerID string) {

	for {
		// Optimization is triggered when the number of tasks that are scheduled but not running is below
		// the number of running workers. Of course, we need to have running jobs to even consider scheduling tasks.

		numJobs, err := context.ModelContext.CountJobs(model.F{"status": types.JobRunning})
		if err != nil {
			panic(err)
		}
		//log.Printf("Num Jobs: %d", numJobs)
		if numJobs > 0 {

			idleCount, err := context.ModelContext.CountProcesses(model.F{"type": types.ProcWorker, "status": types.ProcIdle})
			if err != nil {
				panic(err)
			}
			workingCount, err := context.ModelContext.CountProcesses(model.F{"type": types.ProcWorker, "status": types.ProcWorking})
			if err != nil {
				panic(err)
			}
			numTasks, err := context.ModelContext.CountTasks(model.F{"status": types.TaskScheduled})
			if err != nil {
				panic(err)
			}

			if numTasks < idleCount+workingCount {

				// Mark the process as working.
				context.repeatUntilSuccess(func() (err error) {
					_, err = context.ModelContext.SetProcessStatus(context.ProcessID, types.ProcWorking)
					return
				})

				log.Printf("SCHEDULING MORE TASKS")
				context.OptimizerRunWorker(optimizerID, idleCount+workingCount, numTasks)

				// Mark the process as idle.
				context.repeatUntilSuccess(func() (err error) {
					_, err = context.ModelContext.SetProcessStatus(context.ProcessID, types.ProcIdle)
					return
				})
			}

		}

		// We always sleep for some time before trying again.
		time.Sleep(context.Period)
	}

}

// OptimizerRunWorker runs the optimization sequence.
func (context Context) OptimizerRunWorker(optimizerID string, numProcesses, numTasks int) {

	// Get optimizer image.
	imageFilePath := context.getModuleImagePath(optimizerID, types.ModuleOptimizer)
	imageName, err := modules.LoadImage(imageFilePath)
	if err != nil {
		err = errors.WithStack(err)
		context.Logger.WithFields(
			"module-id", optimizerID,
		).WithStack(err).WithError(err).WriteError("OPTIMIZER LOAD ERROR")
	}

	// Get all running jobs.
	jobs, _, err := context.ModelContext.GetJobs(model.F{"status": types.JobRunning}, 0, "", "", "")
	if err != nil {
		panic(err)
	}
	// If there are no jobs to run, simply return.
	if len(jobs) == 0 {
		return
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
		jobsDict[jobs[i].ID.Hex()] = &jobs[i]
		filename := filepath.Join(confPath, jobs[i].ID.Hex()+".json")
		ioutil.WriteFile(filename, []byte(jobs[i].ConfigSpace), storage.DefaultFilePerm)
	}

	// TODO: Dump all tasks of all jobs to history dir.
	histPath, err := context.StorageContext.GetSchedulingInputPath("history")
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
	outReader, err := modules.RunContainerAndCollectOutput(imageName, nil, command)
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
