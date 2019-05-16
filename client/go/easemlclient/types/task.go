package types

import (
	"time"
)

const (
	// TaskScheduled is a task that is scheduled but not running yet.
	TaskScheduled = "scheduled"

	// TaskRunning is a task that is picked up by the worker and started running.
	TaskRunning = "running"

	// TaskPausing is a task that is in a pausing state. It will be paused when the current stage ends.
	TaskPausing = "pausing"

	// TaskPaused is a task that is paused but may be resumed.
	TaskPaused = "paused"

	// TaskCompleted is a task that has been completed.
	TaskCompleted = "completed"

	// TaskTerminating is a task that is in a terminating state. It will be terminated when the current stage ends.
	TaskTerminating = "terminating"

	// TaskTerminated is a task that was terminated before completion.
	TaskTerminated = "terminated"

	// TaskCanceled is a task that was scheduled but the job was completed before the task got to be executed.
	TaskCanceled = "canceled"

	// TaskError is a task that is in an error state. The error information is logged.
	TaskError = "error"

	// TaskStageBegin is the stage before any other.
	TaskStageBegin = "begin"

	// TaskStageTraining is the stage when a model is being trained.
	TaskStageTraining = "training"

	// TaskStagePredicting is the stage when predictions are made.
	TaskStagePredicting = "predicting"

	// TaskStageEvaluating is the stage when the evaluation takes place.
	TaskStageEvaluating = "evaluating"

	// TaskStageEnd is entered when all stages complete.
	TaskStageEnd = "end"
)

// TaskStageIntervals contains information about start and end times of various task stages.
type TaskStageIntervals struct {
	Training   TimeInterval `json:"training"`
	Predicting TimeInterval `json:"predicting"`
	Evaluating TimeInterval `json:"evaluating"`
}

// TaskStageDurations contains information about the length of all task stages in milliseconds.
type TaskStageDurations struct {
	Training   uint64 `json:"training"`
	Predicting uint64 `json:"predicting"`
	Evaluating uint64 `json:"evaluating"`
}

// Task contains information about tasks.
type Task struct {
	ID              string             `json:"id"`
	Job             string             `json:"job"`
	Process         string             `json:"process"`
	User            string             `json:"user"`
	Dataset         string             `json:"dataset"`
	Model           string             `json:"model"`
	Objective       string             `json:"objective"`
	AltObjectives   []string           `json:"alt-objectives"`
	Config          string             `json:"config"`
	Quality         float64            `json:"quality"`
	QualityTrain    float64            `json:"quality-train"`
	QualityExpected float64            `json:"quality-expected"`
	AltQualities    []float64          `json:"alt-qualities"`
	Status          string             `json:"status"`
	StatusMessage   string             `json:"status-message"`
	Stage           string             `json:"stage"`
	StageTimes      TaskStageIntervals `json:"stage-times"`
	StageDurations  TaskStageDurations `json:"stage-durations"`
	CreationTime    time.Time          `json:"creation-time"`
	RunningDuration uint64             `json:"running-duration"`
}

// IsStarted returns true when the task has passed the "scheduled" state.
func (task Task) IsStarted() bool {
	return task.Status != TaskScheduled
}

// IsPaused returns true when the task is in the paused state.
func (task Task) IsPaused() bool {
	return task.Status != TaskPaused
}

// IsEnded returns true when the task has either completed, terminated or is in an error state.
func (task Task) IsEnded() bool {
	return task.Status == TaskCompleted || task.Status == TaskTerminated || task.Status == TaskError
}

// GetStageDurations returns durations of all completed stages in milliseconds. Incompleted stages are
// left with a zero duration.
func (task Task) GetStageDurations() (d TaskStageDurations) {
	var trainingStageEnded, predictingStageEnded, evaluatingStageEnded bool

	switch task.Stage {
	case TaskStageTraining:
		trainingStageEnded = task.Status == TaskError
	case TaskStagePredicting:
		trainingStageEnded = true
		predictingStageEnded = task.Status == TaskError
	case TaskStageEvaluating:
		trainingStageEnded = true
		predictingStageEnded = true
		evaluatingStageEnded = task.Status == TaskError
	case TaskStageEnd:
		trainingStageEnded = true
		predictingStageEnded = true
		evaluatingStageEnded = true
	}

	if trainingStageEnded {
		d.Training = uint64(task.StageTimes.Training.End.Sub(task.StageTimes.Training.Start) / 1000000)
	}
	if predictingStageEnded {
		d.Predicting = uint64(task.StageTimes.Predicting.End.Sub(task.StageTimes.Predicting.Start) / 1000000)
	}
	if evaluatingStageEnded {
		d.Evaluating = uint64(task.StageTimes.Evaluating.End.Sub(task.StageTimes.Evaluating.Start) / 1000000)
	}
	return
}

// GetRunningDuration computes the total running duration of all tasks as a sum of
// training, predicting and evaluating durations.
func (task Task) GetRunningDuration() uint64 {
	return task.StageDurations.Training + task.StageDurations.Predicting + task.StageDurations.Evaluating
}
