package daemon

import (
	"fmt"

	"github.com/ds3lab/easeml/engine/logger"
	"github.com/ds3lab/easeml/engine/process"
)

// Start is the daemon entry point. It is used to run and maintain either the docker or the mongo daemon.
func Start(context process.Context, daemonName string) {
	log := logger.NewProcessLogger(context.DebugLog)

	if daemonName == process.ProcessTypeDocker {

	} else if daemonName == process.ProcessTypeMongo {

	} else {
		log.WriteFatal(fmt.Sprintf("fatal: unknown daemon name: %s", daemonName))
	}
}
