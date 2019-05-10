package test

import (
	"github.com/ds3lab/easeml/engine/easeml/api"
	"github.com/ds3lab/easeml/engine/easeml/api/router"
	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"github.com/ds3lab/easeml/engine/easeml/logger"
	"github.com/ds3lab/easeml/engine/easeml/storage"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/emicklei/forest"
)

var server *httptest.Server
var client *forest.APITesting
var rootAPIKey string

const testDbAddr = "localhost"
const testDbName = "easeml_test"
const apiBasepath = "/api/v1"
const testWorkDir = ""

func TestMain(m *testing.M) {
	context, err := model.Connect(testDbAddr, testDbName, false)
	if err != nil {
		log.Fatalf("fatal: %+v", err)
	}
	defer context.Session.Close()

	// Initialize the database.
	err = context.Initialize(testDbName)
	if err != nil {
		log.Fatalf("fatal: %+v", err)
	}
	defer context.Clear(testDbName)

	// Log the root user in and generate their API key.
	user, err := context.UserLogin()
	if err != nil {
		log.Fatalf("fatal: %+v", err)
	}
	rootAPIKey = user.APIKey

	// Start the HTTP server. We need to reconnect as an anonimous user.
	// TODO: Start actual server and handle graceful shutdown.
	context, err = model.Connect(testDbAddr, testDbName, true)
	if err != nil {
		log.Fatalf("fatal: %+v", err)
	}
	defer context.Session.Close()

	// Initialize the storage context.
	storageContext := storage.Context{WorkingDir: testWorkDir}

	// Initialize the API context.
	// TODO: Maybe replace this logger.
	apiContext := api.Context{ModelContext: context, StorageContext: storageContext, Logger: &logger.EmptyLogger{}}

	router := router.New(apiContext)
	server = httptest.NewServer(router)
	client = forest.NewClient(server.URL+apiBasepath, new(http.Client))
	log.Println("Users test main")

	code := m.Run()
	server.Close()
	context.Clear(testDbName)
	os.Exit(code)
}
