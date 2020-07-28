package workers

import (
	"log"
	"time"

	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"

	"github.com/pkg/errors"
)

// TaskStatusMaintainerListener periodically checks if there are any tasks whose status has changed to
// pausing or terminating and handles them.
func (context Context) TaskStatusMaintainerListener() {

	for {
		var task types.Task
		var err error

		task, err = context.ModelContext.LockTask(model.F{"status": types.TaskPausing}, context.ProcessID, "", "")
		if err == nil {
			log.Printf("TASK FOUND IN THE PAUSING STATE")
			go context.TaskPausingWorker(task)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		task, err = context.ModelContext.LockTask(model.F{"status": types.TaskTerminating}, context.ProcessID, "", "")
		if err == nil {
			log.Printf("TASK FOUND IN THE TERMINATING STATE")
			go context.TaskTerminatingWorker(task)
		} else if errors.Cause(err) != model.ErrNotFound {
			panic(err)
		}

		time.Sleep(context.Period)
	}
}

// TaskPausingWorker handles all pausing tasks by pausing their tasks.
func (context Context) TaskPausingWorker(task types.Task) {

	// Mark task as paused.
	context.repeatUntilSuccess(func() (err error) {
		_, err = context.ModelContext.UpdateTask(task.ID, model.F{"status": types.TaskPaused})
		return
	})

	// Unlock the task.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockTask(task.ID, context.ProcessID)
	})

}

// TaskTarminatingWorker handles all terminating tasks.
func (context Context) TaskTerminatingWorker(task types.Task) {

	// Mark task as paused.
	context.repeatUntilSuccess(func() (err error) {
		_, err = context.ModelContext.UpdateTask(task.ID, model.F{"status": types.TaskTerminated})
		return
	})

	// Unlock the task.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockTask(task.ID, context.ProcessID)
	})

}
