package test

import (
	"crypto/sha256"
	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/emicklei/forest"
	"github.com/stretchr/testify/assert"
)

func createTask(task model.Task) (result model.Task, err error) {
	context, err := model.Connect(testDbAddr, testDbName, false)
	if err != nil {
		log.Fatalf("fatal: %+v", err)
	}
	defer context.Session.Close()
	context.User.ID = task.User

	return context.CreateTask(task)
}

func TestTasksGet(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create user_tg_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_tg_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	var job model.Job
	job, err = createJob(model.Job{
		User:      "user1",
		Dataset:   "root/dataset1",
		Models:    []string{"root/model1", "root/model2", "root/model3"},
		Objective: "root/objective1",
	}, "running")
	assert.Nil(t, err)

	// Create tasks.
	var tasks = []model.Task{
		model.Task{Job: job.ID, Model: "root/model1"},
		model.Task{Job: job.ID, Model: "root/model3"},
		model.Task{Job: job.ID, Model: "root/model2"},
	}
	for i := range tasks {
		tasks[i], err = createTask(tasks[i])
		assert.Nil(t, err)
	}
	fmt.Println(job.ID.Hex())

	// Don't authenticate. Get tasks. Should return 403.
	config = forest.NewConfig("/tasks")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Get tasks. Should return 401.
	config = forest.NewConfig("/tasks").Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get all tasks. Should return 200.
	config = forest.NewConfig("/tasks").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.total-result-size"), "Field total-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.returned-result-size"), "Field returned-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.next-page-cursor"), "Field next-page-cursor empty.")

	// Authenticate as root. Get all tasks. Apply limit. Should return 200.
	config = forest.NewConfig("/tasks").Header("X-API-KEY", rootAPIKey).Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Get all tasks. Order by and order. Should return 200.
	config = forest.NewConfig("/tasks").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "model").Query("order", "desc")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, tasks[1].ID, forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as root. Get all tasks. Apply filter. Should return 200.
	config = forest.NewConfig("/tasks").Header("X-API-KEY", rootAPIKey)
	config = config.Query("model", "root/model3")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, tasks[1].ID, forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Pick two tasks. Should return 200.
	config = forest.NewConfig("/tasks").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", fmt.Sprintf("%s,%s", tasks[1].ID, tasks[2].ID))
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, tasks[1].ID, forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, tasks[2].ID, forest.JSONPath(t, r, ".data.1.id"))

	// Authenticate as root. Pick three tasks. Verify cursor.
	config = forest.NewConfig("/tasks").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", fmt.Sprintf("%s,%s,%s", tasks[0].ID, tasks[1].ID, tasks[2].ID)).Query("limit", 2)
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

	// Authenticate as root. Pick three tasks. Verify cursor with ordering.
	config = forest.NewConfig("/tasks").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "id").Query("order", "desc")
	config = config.Query("id", fmt.Sprintf("%s,%s,%s", tasks[0].ID, tasks[1].ID, tasks[2].ID)).Query("limit", 2)
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
	assert.EqualValues(t, tasks[0].ID, forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as user_tg_1. Get all tasks. Should return 200 and only tasks by root.
	config = forest.NewConfig("/tasks").BasicAuth("user_tg_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 0, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 0, forest.JSONPath(t, r, ".metadata.returned-result-size"))
}

func TestTasksGetById(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error
	var task model.Task

	// Create the task which we will use in the test.
	var job model.Job
	job, err = createJob(model.Job{
		User:      "user1",
		Dataset:   "root/dataset1",
		Models:    []string{"root/model1", "root/model2", "root/model3"},
		Objective: "root/objective1",
	}, "running")
	assert.Nil(t, err)
	task, err = createTask(model.Task{Job: job.ID, Model: "root/model1"})
	assert.Nil(t, err)

	// Create user_tbid_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_tbid_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Don't authenticate. Get root task. Should return 404.
	config = forest.NewConfig("/tasks/" + task.ID)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)

	// Use wrong API key. Get root task. Should return 401.
	config = forest.NewConfig("/tasks/"+task.ID).Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get root task. Should return 200.
	config = forest.NewConfig("/tasks/"+task.ID).Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, task.ID, forest.JSONPath(t, r, ".data.id"))
	assert.Equal(t, "root/model1", forest.JSONPath(t, r, ".data.model"))

	// Authenticate as user_tbid_1. Get task of user1. Should return 404.
	config = forest.NewConfig("/tasks/"+task.ID).BasicAuth("user_tbid_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)
}
