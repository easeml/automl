---
title: "Types"
---

# types
--
    import "."


## Usage

```go
const (
	// DatasetUpload is a data set that has been uploaded to the system.
	DatasetUpload = "upload"

	// DatasetLocal is a data set that resides on a file system that is local to the easeml service.
	DatasetLocal = "local"

	// DatasetDownload is a data set that has been downloaded from a remote location.
	DatasetDownload = "download"

	// DatasetCreated is the status of a dataset when it is recorded in the system but the data is not yet transferred.
	DatasetCreated = "created"

	// DatasetTransferred is the status of a dataset when it is transferred but hasn't been unpacked yet.
	DatasetTransferred = "transferred"

	// DatasetUnpacked is the status of a dataset when all its files have been extracted and is ready for validation.
	DatasetUnpacked = "unpacked"

	// DatasetValidated is the status of a dataset when it has been validated and is ready to be used.
	DatasetValidated = "validated"

	// DatasetArchived is the status of a dataset when it is no longer usable.
	DatasetArchived = "archived"

	// DatasetError is the status of a dataset when something goes wrong. The details will be logged.
	DatasetError = "error"
)
```

```go
const (
	// JobScheduled is a job that is scheduled but not running yet.
	JobScheduled = "scheduled"

	// JobRunning is a job that is picked up by the scheduler and started running.
	JobRunning = "running"

	// JobPausing is a job that is in a pausing state. All its tasks are being paused.
	JobPausing = "pausing"

	// JobPaused is a job that is paused but may be resumed.
	JobPaused = "paused"

	// JobResuming is a job that is leaving the paused state. All its paused tasks are being resumed.
	JobResuming = "resuming"

	// JobCompleted is a job that has been completed.
	JobCompleted = "completed"

	// JobTerminating is a job that is in a terminating state. All its tasks are being terminated.
	JobTerminating = "terminating"

	// JobTerminated is a job that was terminated before completion.
	JobTerminated = "terminated"

	// JobError is a job that is in an error state. The error information is logged.
	JobError = "error"

	// DefaultMaxTasks is the default number of tasks per job.
	DefaultMaxTasks = 100
)
```

```go
const (
	// ModuleModel is the module type that represent machine learning models.
	ModuleModel = "model"

	// ModuleObjective is the module type that represents objective functions.
	ModuleObjective = "objective"

	// ModuleOptimizer is the module type representing optimizers.
	ModuleOptimizer = "optimizer"

	// ModuleUpload is a module that has veen uploaded to the system.
	ModuleUpload = "upload"

	// ModuleDownload is a module that has been downloaded from a remote location.
	ModuleDownload = "download"

	// ModuleLocal is a module that resides on a file system that is local to the easeml service.
	ModuleLocal = "local"

	// ModuleRegistry is a module that is obtained from a Docker registry.
	ModuleRegistry = "registry"

	// ModuleCreated is the status of a module that is recorded in the system but not yet transferred.
	ModuleCreated = "created"

	// ModuleTransferred is the status of a module that is transferred but not yet validated.
	ModuleTransferred = "transferred"

	// ModuleActive is the status of a module that is transferred and ready to use.
	ModuleActive = "active"

	// ModuleArchived is the status of a module that is no longer usable.
	ModuleArchived = "archived"

	// ModuleError is the status of a mofule when something goes wrong. The details will be logged.
	ModuleError = "error"
)
```

```go
const (
	// ProcController is the type of process that serves as the interface between
	// the users and the data model, as well as controlling the operation of the system.
	ProcController = "controller"

	// ProcWorker is the type of process that trains models and evaluates them.
	ProcWorker = "worker"

	// ProcScheduler is the type of process that handles scheduling of tasks.
	ProcScheduler = "scheduler"

	// ProcIdle is the status of the process when it is running but not doing any work.
	ProcIdle = "idle"

	// ProcWorking is the status of the process when it is running and doing work.
	ProcWorking = "working"

	// ProcTerminated is the status of the process that is not running anymore.
	ProcTerminated = "terminated"
)
```

```go
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
```

```go
const (
	// UserRoot is the name of the root user. This user has no password, cannot log in or
	// log out and can only be authenticated with an API key.
	UserRoot = "root"

	// UserAnon is the user id assigned to unauthenticated users.
	UserAnon = "anonymous"

	// UserThis is the user id of the currently logged in user.
	UserThis = "this"
)
```

#### type CollectionMetadata

```go
type CollectionMetadata struct {

	// The total size of the result after applying query filters but before pagination.
	TotalResultSize int `json:"total-result-size"`

	// The size of the current page of results that is being returned.
	ReturnedResultSize int `json:"returned-result-size"`

	// The string to pass as a cursor to obtain the next page of results.
	NextPageCursor string `json:"next-page-cursor"`
}
```

CollectionMetadata contains additional information about a query that returns
arrays as results. This information aids the caller with navigating the partial
results.

#### type Dataset

```go
type Dataset struct {
	ID            string    `json:"id"`
	User          string    `json:"user"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	SchemaIn      string    `json:"schema-in"`
	SchemaOut     string    `json:"schema-out"`
	Source        string    `json:"source"`
	SourceAddress string    `json:"source-address"`
	CreationTime  time.Time `json:"creation-time"`
	Status        string    `json:"status"`
	StatusMessage string    `json:"status-message"`
	Process       string    `json:"process"`
}
```

Dataset contains information about datasets.

#### type Job

```go
type Job struct {
	ID              string       `json:"id"`
	User            string       `json:"user"`
	Dataset         string       `json:"dataset"`
	Models          []string     `json:"models"`
	ConfigSpace     string       `json:"config-space"`
	AcceptNewModels bool         `json:"accept-new-models"`
	Objective       string       `json:"objective"`
	AltObjectives   []string     `json:"alt-objectives"`
	MaxTasks        uint64       `json:"max-tasks"`
	CreationTime    time.Time    `json:"creation-time"`
	RunningTime     TimeInterval `json:"running-time"`
	RunningDuration uint64       `json:"running-duration"`
	PauseDuration   uint64       `json:"pause-duration"`
	Status          string       `json:"status"`
	StatusMessage   string       `json:"status-message"`
	Process         string       `json:"process"`
}
```

Job contains information about jobs.

#### type Module

```go
type Module struct {
	ID            string    `json:"id"`
	User          string    `json:"user"`
	Type          string    `json:"type"`
	Label         string    `json:"label"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	SchemaIn      string    `json:"schema-in"`
	SchemaOut     string    `json:"schema-out"`
	ConfigSpace   string    `json:"config-space"`
	Source        string    `json:"source"`
	SourceAddress string    `json:"source-address"`
	CreationTime  time.Time `json:"creation-time"`
	Status        string    `json:"status"`
	StatusMessage string    `json:"status-message"`
	Process       string    `json:"process"`
}
```

Module contains information about modules which are stateless Docker images.

#### type Process

```go
type Process struct {
	ID            string    `json:"id"`
	ProcessID     uint64    `json:"process-id"`
	HostID        string    `json:"host-id"`
	HostAddress   string    `json:"host-address"`
	StartTime     time.Time `json:"start-time"`
	LastKeepalive time.Time `json:"last-keepalive"`
	Type          string    `json:"type"`
	Resource      string    `json:"resource"`
	Status        string    `json:"status"`
	RunningOrinal int       `json:"running-ordinal"`
}
```

Process contains information about processes.

#### type Task

```go
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
```

Task contains information about tasks.

#### func (Task) GetRunningDuration

```go
func (task Task) GetRunningDuration() uint64
```
GetRunningDuration computes the total running duration of all tasks as a sum of
training, predicting and evaluating durations.

#### func (Task) GetStageDurations

```go
func (task Task) GetStageDurations() (d TaskStageDurations)
```
GetStageDurations returns durations of all completed stages in milliseconds.
Incompleted stages are left with a zero duration.

#### func (Task) IsEnded

```go
func (task Task) IsEnded() bool
```
IsEnded returns true when the task has either completed, terminated or is in an
error state.

#### func (Task) IsPaused

```go
func (task Task) IsPaused() bool
```
IsPaused returns true when the task is in the paused state.

#### func (Task) IsStarted

```go
func (task Task) IsStarted() bool
```
IsStarted returns true when the task has passed the "scheduled" state.

#### type TaskStageDurations

```go
type TaskStageDurations struct {
	Training   uint64 `json:"training"`
	Predicting uint64 `json:"predicting"`
	Evaluating uint64 `json:"evaluating"`
}
```

TaskStageDurations contains information about the length of all task stages in
milliseconds.

#### type TaskStageIntervals

```go
type TaskStageIntervals struct {
	Training   TimeInterval `json:"training"`
	Predicting TimeInterval `json:"predicting"`
	Evaluating TimeInterval `json:"evaluating"`
}
```

TaskStageIntervals contains information about start and end times of various
task stages.

#### type TimeInterval

```go
type TimeInterval struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
```

TimeInterval represents a time interval with specific start and end times.

#### type User

```go
type User struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	PasswordHash string `json:"password,omitempty"`
}
```

User contains information about users.

#### func  GetAnonUser

```go
func GetAnonUser() User
```
GetAnonUser returns an anonymous user instance.

#### func (User) IsAnon

```go
func (user User) IsAnon() bool
```
IsAnon returns true if the given user is the anonymous user.

#### func (User) IsRoot

```go
func (user User) IsRoot() bool
```
IsRoot returns true if the given user is the root user.
