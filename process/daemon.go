package process

import "time"

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
}
