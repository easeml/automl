package types

import (
	"time"

	"github.com/globalsign/mgo/bson"
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
	TaskStageTrain = "train"

	// TaskStagePredicting is the stage when predictions are made.
	TaskStagePredict = "predict"

	// TaskStageEvaluating is the stage when the evaluation takes place.
	TaskStageEvaluate = "evaluate"

	// TaskStageEnd is entered when all stages complete.
	TaskStageEnd = "end"
)

// TODO UGLY HACK Maybe use go generate
type StageMapType map[string] struct{}

func (i StageMapType) Has(v string) bool {
	_, ok := i[v]
	return ok
}

var AllStages = StageMapType{
	// TaskScheduled is a task that is scheduled but not running yet.
	TaskScheduled: {},

	// TaskRunning is a task that is picked up by the worker and started running.
	TaskRunning: {},

	// TaskPausing is a task that is in a pausing state. It will be paused when the current stage ends.
	TaskPausing: {},

	// TaskPaused is a task that is paused but may be resumed.
	TaskPaused: {},

	// TaskCompleted is a task that has been completed.
   TaskCompleted: {},

	// TaskTerminating is a task that is in a terminating state. It will be terminated when the current stage ends.
	TaskTerminating: {},

	// TaskTerminated is a task that was terminated before completion.
	TaskTerminated: {},

	// TaskCanceled is a task that was scheduled but the job was completed before the task got to be executed.
	TaskCanceled: {},

	// TaskError is a task that is in an error state. The error information is logged.
	TaskError: {},

	// TaskStageBegin is the stage before any other.
	TaskStageBegin: {},

	// TaskStageTraining is the stage when a model is being trained.
	TaskStageTrain: {},

	// TaskStagePredicting is the stage when predictions are made.
	TaskStagePredict: {},

	// TaskStageEvaluating is the stage when the evaluation takes place.
	TaskStageEvaluate: {},

	// TaskStageEnd is entered when all stages complete.
	TaskStageEnd:{},
}

// TODO UGLY Should not be hardcoded

var AllPipelineElements = StageMapType{
	// TaskStageTraining is the stage when a model is being trained.
	TaskStageTrain:{},

	// TaskStagePredicting is the stage when predictions are made.
	TaskStagePredict:{},

	// TaskStageEvaluating is the stage when the evaluation takes place.
	TaskStageEvaluate:{},
}

// TODO UGLY Hack should not be hardcoded
var AllPreRequisites = map[string][]string{
	TaskStageTrain: {},
	TaskStagePredict: {TaskStageTrain},
	TaskStageEvaluate: {TaskStageTrain,TaskStagePredict},
}



// TaskStageIntervals contains information about start and end times of various task stages.
type TaskStageIntervals struct {
	Train   TimeInterval `bson:"train" json:"train"`
	Predict TimeInterval `bson:"predict" json:"predict"`
	Evaluate TimeInterval `bson:"evaluate" json:"evaluate"`
}

// TaskStageDurations contains information about the length of all task stages in milliseconds.
type TaskStageDurations struct {
	Train   uint64 `bson:"train" json:"train"`
	Predict uint64 `bson:"predict" json:"predict"`
	Evaluate uint64 `bson:"evaluate" json:"evaluate"`
}

// Task contains information about tasks.
type Task struct {
	ObjectID        bson.ObjectId      `bson:"_id"`
	ID              string             `bson:"id" json:"id"`
	Job             bson.ObjectId      `bson:"job" json:"job"`
	Process         bson.ObjectId      `bson:"process,omitempty" json:"process"`
	User            string             `bson:"user" json:"user"`
	Dataset         string             `bson:"dataset" json:"dataset"`
	Model           string             `bson:"model" json:"model"`
	Objective       string             `bson:"objective" json:"objective"`
	AltObjectives   []string           `bson:"alt-objectives" json:"alt-objectives"`
	Config          string             `bson:"config" json:"config"`
	Quality         float64            `bson:"quality" json:"quality"`
	QualityTrain    float64            `bson:"quality-train" json:"quality-train"`
	QualityExpected float64            `bson:"quality-expected" json:"quality-expected"`
	AltQualities    []float64          `bson:"alt-qualities" json:"alt-qualities"`
	Status          string             `bson:"status" json:"status"`
	Pipeline   		[]string           `bson:"pipeline" json:"pipeline"`
	StatusMessage   string             `bson:"status-message" json:"status-message"`
	Stage           string             `bson:"stage" json:"stage"`
	StageTimes      TaskStageIntervals `bson:"stage-times" json:"stage-times"`
	StageDurations  TaskStageDurations `bson:"stage-durations,omitempty" json:"stage-durations"`
	CreationTime    time.Time          `bson:"creation-time" json:"creation-time"`
	RunningDuration uint64             `bson:"running-duration,omitempty" json:"running-duration"`
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
	case TaskStageTrain:
		trainingStageEnded = task.Status == TaskError
	case TaskStagePredict:
		trainingStageEnded = true
		predictingStageEnded = task.Status == TaskError
	case TaskStageEvaluate:
		trainingStageEnded = true
		predictingStageEnded = true
		evaluatingStageEnded = task.Status == TaskError
	case TaskStageEnd:
		trainingStageEnded = true
		predictingStageEnded = true
		evaluatingStageEnded = true
	}

	if trainingStageEnded {
		d.Train = uint64(task.StageTimes.Train.End.Sub(task.StageTimes.Train.Start) / 1000000)
	}
	if predictingStageEnded {
		d.Predict = uint64(task.StageTimes.Predict.End.Sub(task.StageTimes.Predict.Start) / 1000000)
	}
	if evaluatingStageEnded {
		d.Evaluate = uint64(task.StageTimes.Evaluate.End.Sub(task.StageTimes.Evaluate.Start) / 1000000)
	}
	return
}

// GetRunningDuration computes the total running duration of all tasks as a sum of
// training, predicting and evaluating durations.
func (task Task) GetRunningDuration() uint64 {
	return task.StageDurations.Train + task.StageDurations.Predict + task.StageDurations.Evaluate
}
