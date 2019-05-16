package types

import (
	"time"
)

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

// Process contains information about processes.
type Process struct {
	ID            string        `json:"id"`
	ProcessID     uint64        `json:"process-id"`
	HostID        string        `json:"host-id"`
	HostAddress   string        `json:"host-address"`
	StartTime     time.Time     `json:"start-time"`
	LastKeepalive time.Time     `json:"last-keepalive"`
	Type          string        `json:"type"`
	Resource      string        `json:"resource"`
	Status        string        `json:"status"`
	RunningOrinal int           `json:"running-ordinal"`
}
