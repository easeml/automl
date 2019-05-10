package types

import (
	"time"

	"github.com/globalsign/mgo/bson"
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
	ID            bson.ObjectId `bson:"_id" json:"id"`
	ProcessID     uint64        `bson:"process-id" json:"process-id"`
	HostID        string        `bson:"host-id" json:"host-id"`
	HostAddress   string        `bson:"host-address" json:"host-address"`
	StartTime     time.Time     `bson:"start-time" json:"start-time"`
	LastKeepalive time.Time     `bson:"last-keepalive" json:"last-keepalive"`
	Type          string        `bson:"type" json:"type"`
	Resource      string        `bson:"resource" json:"resource"`
	Status        string        `bson:"status" json:"status"`
	RunningOrinal int           `bson:"running-ordinal" json:"running-ordinal"`
}
