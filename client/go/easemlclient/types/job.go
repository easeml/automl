package types

import (
	"time"
)

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

// Job contains information about jobs.
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
