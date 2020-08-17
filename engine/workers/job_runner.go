package workers

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"
	"github.com/otiai10/copy" // Surprised that there is no standard library for this
	"github.com/pkg/errors"
)

// JobRunListener periodically checks if there are jobs ready to make tasks from
func (context Context) JobRunListener(optimizerID string) {

	for {
		// Task scheduling is triggered when the number of tasks that are scheduled but not running is below
		// the number of running workers. Of course, we need to have running jobs to even consider scheduling tasks.

		numJobs, err := context.ModelContext.CountJobs(model.F{"status": types.JobRunning})
		if err != nil {
			panic(err)
		}

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

				context.ScheduleMoreTasks(optimizerID, idleCount+workingCount, numTasks)

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
func (context Context) ScheduleMoreTasks(optimizerID string, numProcesses, numTasks int) {
	// Get all running jobs.
	jobs, _, err := context.ModelContext.GetJobs(model.F{"status": types.JobRunning}, 0, "", "", "")
	if err != nil {
		panic(err)
	}
	// If there are no jobs to run, simply return.
	if len(jobs) == 0 {
		return
	}

	var trainableJobs []*types.Job
	var preTrainedJobs []*types.Job
	for i := range jobs {
		if len(jobs[i].TaskIds) > 0{
			preTrainedJobs = append(preTrainedJobs,&jobs[i])
		}else{
			trainableJobs = append(trainableJobs,&jobs[i])
		}
	}
	if len(trainableJobs) > 0{
		context.OptimizerRunSuggestCreateTask(optimizerID, numProcesses, numTasks,trainableJobs)
	}

	if len(preTrainedJobs) > 0{
		context.CreatePreTrainedTask(numProcesses, numTasks,preTrainedJobs)
	}

}

func IsEmptyDirectory(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

// OptimizerRunSuggestCreateTask runs the optimization sequence.
func (context Context) CreatePreTrainedTask(numProcesses, numTasks int,jobs []*types.Job) {
	for i := range jobs {
		for j := range jobs[i].TaskIds {
			taskId := jobs[i].TaskIds[j]

			// Access model.
			oldTask, err := context.ModelContext.GetTaskByID(taskId)
			if err!=nil {
				panic(err)
			}

			if oldTask.Status != types.TaskCompleted {
				context.Logger.WithFields(
					"job-id", jobs[i].ID,
					"old-task-id", taskId,
					"dataset", jobs[i].Dataset,
				).WithError(errors.Errorf("PRE-TRAINED TASK NOT COMPLETED"))
				break
			}

			oldPath, err:=context.StorageContext.GetTaskPath(taskId, "")

			if err != nil {
				// This means that we cannot access the file system, so we need to panic.
				panic(err)
			}

			empty,err:=IsEmptyDirectory(filepath.Join(oldPath, "config"))
			if err!=nil || empty {
				context.Logger.WithFields(
					"job-id", jobs[i].ID,
					"old-task-id", taskId,
					"dataset", jobs[i].Dataset,
				).WithError(errors.Errorf("PRE-TRAINED TASK WITHOUT CONFIG DATA"))
				break
			}

			empty,err =IsEmptyDirectory(filepath.Join(oldPath, "parameters"))
			if err!=nil || empty {
				context.Logger.WithFields(
					"job-id", jobs[i].ID,
					"old-task-id", taskId,
					"dataset", jobs[i].Dataset,
				).WithError(errors.Errorf("PRE-TRAINED TASK WITHOUT PARAMETERS DATA"))
				break
			}

			// Add task model to the list of job models
			var found bool
			for i := range jobs[i].Models {
				if oldTask.Model == jobs[i].Models[i] {
					found = true
					break
				}
			}
			if found == false {
				// Add model to job
				jobs[i].Models = append(jobs[i].Models,oldTask.Model)
				// Update the job.
				_, err = context.ModelContext.UpdateJob(jobs[i].ID, model.F{"models": jobs[i].Models})
				if err != nil {
					err = errors.Wrap(err, "job update failed")
					return
				}
			}
			// Define new task.
			task := types.Task{
				Job:      jobs[i].ID,
				Model:    oldTask.Model,
				Pipeline: jobs[i].Pipeline,
				Config:   oldTask.Config,
			}
			task, err = context.ModelContext.CreateTask(task)
			if err != nil {
				panic(err)
			}

			newPath, err := context.StorageContext.GetTaskPath(task.ID,"")

			err = copy.Copy(filepath.Join(oldPath, "parameters"), filepath.Join(newPath, "parameters"))
			err = copy.Copy(filepath.Join(oldPath, "config"), filepath.Join(newPath, "config"))
			if err != nil {
				// This means that we cannot access the file system, so we need to panic.
				panic(err)
			}

			context.Logger.WithFields(
				"task-id", task.ID,
				"model", task.Model,
				"dataset", task.Dataset,
				"objective", task.Objective,
			).WriteInfo("SCHEDULED NEW PRE-TRAINED TASK")
		}
	}
}

