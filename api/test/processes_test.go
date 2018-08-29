package test

import (
	"crypto/sha256"
	"github.com/ds3lab/easeml/database/model"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/forest"
	"github.com/stretchr/testify/assert"
)

func createProcess(process model.Process) (result model.Process, err error) {
	context, err := model.Connect(testDbAddr, testDbName, false)
	if err != nil {
		log.Fatalf("fatal: %+v", err)
	}
	defer context.Session.Close()
	context.User.APIKey = rootAPIKey

	return context.CreateProcess(process)
}

func TestProcessesGet(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create user_pg_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_pg_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Create processes.
	var processes = []model.Process{
		model.Process{ProcessID: 1, HostID: "localhost", HostAddress: "1.1.1.1", Type: "controller"},
		model.Process{ProcessID: 2, HostID: "localhost", HostAddress: "2.2.2.2", Type: "controller"},
		model.Process{ProcessID: 2, HostID: "localhost", HostAddress: "0.0.0.0", Type: "worker"},
	}
	for i := range processes {
		processes[i], err = createProcess(processes[i])
		assert.Nil(t, err)
	}

	// Don't authenticate. Get processes. Should return 403.
	config = forest.NewConfig("/processes")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Get processes. Should return 401.
	config = forest.NewConfig("/processes").Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get all processes. Should return 200.
	config = forest.NewConfig("/processes").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.total-result-size"), "Field total-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.returned-result-size"), "Field returned-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.next-page-cursor"), "Field next-page-cursor empty.")

	// Authenticate as root. Get all processes. Apply limit. Should return 200.
	config = forest.NewConfig("/processes").Header("X-API-KEY", rootAPIKey).Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Get all processes. Order by and order. Should return 200.
	config = forest.NewConfig("/processes").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "host-address").Query("order", "desc")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, processes[1].ID.Hex(), forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as root. Get all processes. Apply filter. Should return 200.
	config = forest.NewConfig("/processes").Header("X-API-KEY", rootAPIKey)
	config = config.Query("type", "worker")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, processes[2].ID.Hex(), forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Pick two processes. Should return 200.
	config = forest.NewConfig("/processes").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", fmt.Sprintf("%s,%s", processes[1].ID.Hex(), processes[2].ID.Hex()))
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, processes[1].ID.Hex(), forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, processes[2].ID.Hex(), forest.JSONPath(t, r, ".data.1.id"))

	// Authenticate as root. Pick three processes. Verify cursor.
	config = forest.NewConfig("/processes").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", fmt.Sprintf("%s,%s,%s", processes[0].ID.Hex(), processes[1].ID.Hex(), processes[2].ID.Hex())).Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	cursor := forest.JSONPath(t, r, ".metadata.next-page-cursor").(string)
	config.Query("cursor", cursor)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Pick three processes. Verify cursor with ordering.
	config = forest.NewConfig("/processes").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "host-address").Query("order", "asc")
	config = config.Query("id", fmt.Sprintf("%s,%s,%s", processes[0].ID.Hex(), processes[1].ID.Hex(), processes[2].ID.Hex())).Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	cursor = forest.JSONPath(t, r, ".metadata.next-page-cursor").(string)
	config.Query("cursor", cursor)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, processes[1].ID.Hex(), forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as user_pg_1. Get all processes. Should return 200 and all processes.
	config = forest.NewConfig("/processes").BasicAuth("user_pg_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.returned-result-size"))
}

func TestProcessesGetById(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error
	var process model.Process

	// Create the process which we will use in the test.
	process, err = createProcess(model.Process{ProcessID: 1, HostID: "localhost", HostAddress: "1.1.1.1", Type: "controller"})
	assert.Nil(t, err)

	// Create user_pbid_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_pbid_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Don't authenticate. Get root process. Should return 404.
	config = forest.NewConfig(fmt.Sprintf("/processes/%s", process.ID.Hex()))
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)

	// Use wrong API key. Get root process. Should return 401.
	config = forest.NewConfig(fmt.Sprintf("/processes/%s", process.ID.Hex())).Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Should return 200.
	config = forest.NewConfig(fmt.Sprintf("/processes/%s", process.ID.Hex())).Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, process.ID.Hex(), forest.JSONPath(t, r, ".data.id"))
	assert.Equal(t, "controller", forest.JSONPath(t, r, ".data.type"))

	// Authenticate as user_pbid_1. Get process of user2. Should return 404.
	config = forest.NewConfig(fmt.Sprintf("/processes/%s", process.ID.Hex())).BasicAuth("user_pbid_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
}
