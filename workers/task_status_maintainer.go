package workers

import (
	"github.com/ds3lab/easeml/database/model"
	"log"
	"time"

	"github.com/pkg/errors"
)

// TaskStatusMaintainerListener periodically checks if there are any tasks whose status has changed to
// pausing or terminating and handles them.
func (context Context) TaskStatusMaintainerListener() {

	for {
		var task model.Task
		var err error

		task, err = context.ModelContext.LockTask(model.F{"status": model.TaskPausing}, context.ProcessID, "", "")
		if err == nil {
			log.Printf("TASK FOUND IN THE PAUSING STATE")
			go context.TaskPausingWorker(task)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		task, err = context.ModelContext.LockTask(model.F{"status": model.TaskTerminating}, context.ProcessID, "", "")
		if err == nil {
			log.Printf("TASK FOUND IN THE TERMINATING STATE")
			go context.TaskTarminatingWorker(task)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		time.Sleep(context.Period)
	}
}

// TaskPausingWorker handles all pausing tasks by pausing their tasks.
func (context Context) TaskPausingWorker(task model.Task) {

	// Mark task as paused.
	context.repeatUntilSuccess(func() (err error) {
		_, err = context.ModelContext.UpdateTask(task.ID, model.F{"status": model.TaskPaused})
		return
	})

	// Unlock the task.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockTask(task.ID, context.ProcessID)
	})

}

// TaskTarminatingWorker handles all terminating tasks.
func (context Context) TaskTarminatingWorker(task model.Task) {

	// Mark task as paused.
	context.repeatUntilSuccess(func() (err error) {
		_, err = context.ModelContext.UpdateTask(task.ID, model.F{"status": model.TaskTerminated})
		return
	})

	// Unlock the task.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockTask(task.ID, context.ProcessID)
	})

}
