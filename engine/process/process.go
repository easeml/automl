package process

import (
	"time"
)

// Context contains all information needed to run processes.
type Context struct {
	DatabaseAddress string
	DatabaseName    string
	ServerAddress   string
	WorkingDir      string
	KeepAlivePeriod time.Duration
	ListenerPeriod  time.Duration
	OptimizerID     string
	RootAPIKey      chan string
	DebugLog        bool
	GpuDevices      []string
}

const (
	// ProcessTypeController is the controller process.
	ProcessTypeController = "controller"

	// ProcessTypeScheduler is the scheduler process.
	ProcessTypeScheduler = "scheduler"

	// ProcessTypeWorker is the worker process.
	ProcessTypeWorker = "worker"

	// ProcessTypeDocker is the process controlling the docker daemon if it is run with the easeml engine.
	ProcessTypeDocker = "docker"

	// ProcessTypeMongo is the process controlling the mongo daemon if it is run with the easeml engine.
	ProcessTypeMongo = "mongo"
)
