package workers

import (
	"log"
	"time"

	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"

	"github.com/pkg/errors"
)

// JobStatusMaintainerListener periodically checks if there are any jobs whose status has changed to
// pausing, resuming or terminating and handles them.
func (context Context) JobStatusMaintainerListener() {

	for {
		var job types.Job
		var err error

		job, err = context.ModelContext.LockJob(model.F{"status": types.JobPausing}, context.ProcessID, "", "")
		if err == nil {
			log.Printf("JOB FOUND IN THE PAUSING STATE")
			go context.JobPausingWorker(job)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		job, err = context.ModelContext.LockJob(model.F{"status": types.JobResuming}, context.ProcessID, "", "")
		if err == nil {
			log.Printf("JOB FOUND IN THE RESUMING STATE")
			go context.JobResumingWorker(job)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		job, err = context.ModelContext.LockJob(model.F{"status": types.JobTerminating}, context.ProcessID, "", "")
		if err == nil {
			log.Printf("JOB FOUND IN THE TERMINATING STATE")
			go context.JobTerminatingWorker(job)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		time.Sleep(context.Period)
	}
}

// JobPausingWorker handles all pausing jobs by pausing their tasks.
func (context Context) JobPausingWorker(job types.Job) {

	// Mark all tasks as pausing.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.PauseRunningTasks(job.ID)
	})

	// Mark job as paused.
	context.repeatUntilSuccess(func() (err error) {
		_, err = context.ModelContext.UpdateJob(job.ID, model.F{"status": types.JobPaused})
		return
	})

	// Unlock the job.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockJob(job.ID, context.ProcessID)
	})

}

// JobResumingWorker handles all resuming jobs by pausing their tasks.
func (context Context) JobResumingWorker(job types.Job) {

	// Mark all tasks as scheduled.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.ResumePausedTasks(job.ID)
	})

	// Mark job as running.
	context.repeatUntilSuccess(func() (err error) {
		_, err = context.ModelContext.UpdateJob(job.ID, model.F{"status": types.JobRunning})
		return
	})

	// Unlock the job.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockJob(job.ID, context.ProcessID)
	})

}

// JobTerminatingWorker handles all terminating jobs by pausing their tasks.
func (context Context) JobTerminatingWorker(job types.Job) {

	// Mark all tasks as terminating.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.TerminateRunningTasks(job.ID)
	})

	// Mark job as terminated.
	context.repeatUntilSuccess(func() (err error) {
		_, err = context.ModelContext.UpdateJob(job.ID, model.F{"status": types.JobTerminated})
		return
	})

	// Unlock the job.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockJob(job.ID, context.ProcessID)
	})

}
