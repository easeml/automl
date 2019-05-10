package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ds3lab/easeml/engine/easeml/api"
	"github.com/ds3lab/easeml/engine/easeml/api/router"
	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"github.com/ds3lab/easeml/engine/easeml/database/model/types"
	"github.com/ds3lab/easeml/engine/easeml/logger"
	"github.com/ds3lab/easeml/engine/easeml/process"
	"github.com/ds3lab/easeml/engine/easeml/storage"
	"github.com/ds3lab/easeml/engine/easeml/workers"

	"github.com/gobuffalo/packr"
)

// Start is the entry point.
func Start(context process.Context) {

	log := logger.NewProcessLogger(context.DebugLog)

	// Initialize the storage context.
	storageContext := storage.Context{WorkingDir: context.WorkingDir}

	// TODO: Move all this code to the server.

	modelContext, err := model.Connect(context.DatabaseAddress, context.DatabaseName, false)
	if err != nil {
		log.WriteFatal(fmt.Sprintf("fatal: %+v", err))
	}
	defer modelContext.Session.Close()

	// Initialize the database.
	err = modelContext.Initialize(context.DatabaseName)
	if err != nil {
		log.WriteFatal(fmt.Sprintf("fatal: %+v", err))
	}

	// Register the new process.
	var process types.Process
	process, err = modelContext.StartProcess(types.ProcController)
	if err != nil {
		log.WriteFatal(fmt.Sprintf("fatal: %+v", err))
	}
	defer modelContext.SetProcessStatus(process.ID, types.ProcTerminated)
	log.WithFields("process-id", process.ID.Hex(), "PID", process.ProcessID).WriteInfo("CONTROLLER PROCESS STARTED")
	log.ProcessID = process.ID.Hex()
	//log.Prefix = fmt.Sprintf("CTL%02d", process.RunningOrinal)

	// Create log file.
	processPath, err := storageContext.GetProcessPath(process.ID.Hex(), "")
	if err != nil {
		panic(err)
	}
	logFilePath := filepath.Join(processPath, process.Type+".log")
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE, storage.DefaultFilePerm)
	if err != nil {
		panic(err)
	}
	log.AddJSONWriter(logFile)
	defer logFile.Close()

	// Log the root user in and generate their API key.
	// TODO: Log out later (if no other controllers are alive).
	user, err := modelContext.UserLogin()
	if err != nil {
		log.WriteFatal(fmt.Sprintf("fatal: %+v", err))
	}
	log.WithFields("api-key", user.APIKey).WriteInfo("ROOT USER LOGGED IN")

	// Report the root API key to the API key channel.
	context.RootAPIKey <- user.APIKey

	// Run the downloader.
	workersContext := workers.Context{
		ModelContext:   modelContext,
		StorageContext: storageContext,
		ProcessID:      process.ID,
		Period:         context.ListenerPeriod,
		Logger:         log,
	}

	// Process keepalive goroutine.
	go func() {
		workersContextCopy := workersContext.Clone()
		defer workersContextCopy.ModelContext.Session.Close()
		workersContextCopy.ProcessKeepaliveWorker(context.KeepAlivePeriod)
	}()

	// Terminate dead processes.
	go func() {
		workersContextCopy := workersContext.Clone()
		defer workersContextCopy.ModelContext.Session.Close()
		workersContextCopy.TerminateDeadProcessesWorker(context.KeepAlivePeriod)
	}()

	// Data download worker.
	go func() {
		workersContextCopy := workersContext.Clone()
		defer workersContextCopy.ModelContext.Session.Close()
		workersContextCopy.DatasetDownloadListener()
	}()

	// Data unpack worker.
	go func() {
		workersContextCopy := workersContext.Clone()
		defer workersContextCopy.ModelContext.Session.Close()
		workersContextCopy.DatasetUnpackListener()
	}()

	// Data validate worker.
	go func() {
		workersContextCopy := workersContext.Clone()
		defer workersContextCopy.ModelContext.Session.Close()
		workersContextCopy.DatasetValidatorListener()
	}()

	// Module download worker.
	go func() {
		workersContextCopy := workersContext.Clone()
		defer workersContextCopy.ModelContext.Session.Close()
		workersContextCopy.ModuleDownloadListener()
	}()

	// Module validate worker.
	go func() {
		workersContextCopy := workersContext.Clone()
		defer workersContextCopy.ModelContext.Session.Close()
		workersContextCopy.ModuleValidateListener()
	}()

	// Job status maintainer worker.
	go func() {
		workersContextCopy := workersContext.Clone()
		defer workersContextCopy.ModelContext.Session.Close()
		workersContextCopy.JobStatusMaintainerListener()
	}()

	// Task status maintainer worker.
	go func() {
		workersContextCopy := workersContext.Clone()
		defer workersContextCopy.ModelContext.Session.Close()
		workersContextCopy.TaskStatusMaintainerListener()
	}()

	// Start the HTTP server. We need to reconnect as an anonimous user.
	// TODO: Start actual server and handle graceful shutdown.
	anonContext, err := model.Connect(context.DatabaseAddress, context.DatabaseName, true)
	if err != nil {
		log.WriteFatal(fmt.Sprintf("fatal: %+v", err))
	}
	defer anonContext.Session.Close()

	// Initialize the API context and API router.
	apiContext := api.Context{ModelContext: anonContext, StorageContext: storageContext, Logger: log}
	apiRouter := router.New(apiContext)
	http.Handle("/api/v1/", apiRouter)

	// Initialize the WEB router.
	box := packr.NewBox("../../web/dist")
	webRouter := http.FileServer(box)
	http.Handle("/", webRouter)

	log.WriteFatal(http.ListenAndServe(context.ServerAddress, nil).Error())
	log.WriteInfo("After log")
}
