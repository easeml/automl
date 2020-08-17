package types

import (
	"time"

	"github.com/globalsign/mgo/bson"
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
	ID                bson.ObjectId `bson:"_id" json:"id"`
	User              string        `bson:"user" json:"user"`
	Pipeline   		  []string      `bson:"pipeline" json:"pipeline"`
	TaskIds	   		  []string      `bson:"task-ids" json:"task-ids"`
	Dataset           string        `bson:"dataset" json:"dataset"`
	Models            []string      `bson:"models" json:"models"`
	ConfigSpace       string        `bson:"config-space" json:"config-space"`
	AcceptNewModels   bool          `bson:"accept-new-models" json:"accept-new-models"`
	Objective         string        `bson:"objective" json:"objective"`
	AltObjectives     []string      `bson:"alt-objectives" json:"alt-objectives"`
	MaxTasks          uint64        `bson:"max-tasks" json:"max-tasks"`
	CreationTime      time.Time     `bson:"creation-time" json:"creation-time"`
	RunningTime       TimeInterval  `bson:"running-time" json:"running-time"`
	RunningDuration   uint64        `bson:"running-duration,omitempty" json:"running-duration"`
	PauseStartTime    time.Time     `bson:"pause-start-time"`
	PauseDuration     uint64        `bson:"pause-duration,omitempty" json:"pause-duration"`
	PrevPauseDuration uint64        `bson:"prev-pause-duration"`
	Status            string        `bson:"status" json:"status"`
	StatusMessage     string        `bson:"status-message" json:"status-message"`
	Process           bson.ObjectId `bson:"process,omitempty" json:"process"`
}

// IsStarted returns true when the job has passed the "scheduled" state.
func (job Job) IsStarted() bool {
	return job.Status != JobScheduled
}

// IsPaused returns true when the job is in the paused state.
func (job Job) IsPaused() bool {
	return job.Status == JobPaused
}

// IsEnded returns true when the job has either completed, terminated or is in an error state.
func (job Job) IsEnded() bool {
	return job.Status == JobCompleted || job.Status == JobTerminated || job.Status == JobError
}

// GetPauseDuration computes the total time that the job has spent in the paused state.
func (job Job) GetPauseDuration() uint64 {
	pauseDuration := job.PrevPauseDuration
	if job.IsPaused() {
		pauseDuration += uint64(time.Since(job.PauseStartTime).Nanoseconds() / 1000000)
	}
	return pauseDuration
}

// GetRunningDuration computes the total time the job has spent running (including pauses).
func (job Job) GetRunningDuration() uint64 {
	var runningDuration uint64
	if job.IsEnded() {
		runningDuration = uint64(job.RunningTime.End.Sub(job.RunningTime.Start).Nanoseconds() / 1000000)

	} else if job.IsStarted() {
		runningDuration = uint64(time.Since(job.RunningTime.Start).Nanoseconds() / 1000000)
	}
	return runningDuration
}
