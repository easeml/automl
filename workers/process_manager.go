package workers

import "time"

// ProcessKeepaliveWorker updates the keepalive-time field of the process
// thus notifying the system that it is still running.
func (context Context) ProcessKeepaliveWorker(period time.Duration) {
	for {
		err := context.ModelContext.ProcessKeepalive(context.ProcessID)
		if err != nil {
			panic(err)
		}
		time.Sleep(period)
	}
}

// TerminateDeadProcessesWorker goes through all processes that have stopped making keepalive updates
// and sets their status to terminated.
func (context Context) TerminateDeadProcessesWorker(period time.Duration) {

	for {
		time.Sleep(period)
		cutoffTime := time.Now().Add(-5 * period)
		err := context.ModelContext.TerminateDeadProcesses(cutoffTime)
		if err != nil {
			panic(err)
		}
	}

}
