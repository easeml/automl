package model

import (
	"github.com/ds3lab/easeml/database"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGetTaskByID(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	jobID := bson.NewObjectId()
	var task = Task{
		ObjectID: bson.NewObjectId(),
		ID:       string(jobID) + "/1",
		Job:      jobID,
		//Process:       jobID,
		User:          "root",
		Dataset:       "root/dataset1",
		Model:         "root/model1",
		Objective:     "root/objective1",
		AltObjectives: []string{},
		CreationTime:  time.Now(),
		Status:        "running",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: User{ID: UserRoot}}

	// Add a test task to the test database.
	c := connection.Session.DB(TestDBName).C("tasks")
	err = c.Insert(task)
	assert.Nil(err)

	// Get the task by ID.
	result, err := context.GetTaskByID(task.ID)
	assert.Nil(err)
	assert.Equal(task.ObjectID, result.ObjectID)
	assert.Equal(task.ID, result.ID)

	// Ensure the we can't find the wrong task.
	result, err = context.GetTaskByID(string(jobID) + "/2")
	assert.Equal(ErrNotFound, err)
	assert.Empty(result.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestGetTasks(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	jobID := bson.NewObjectId()
	var task1 = Task{
		ObjectID:      bson.NewObjectId(),
		ID:            string(jobID) + "/1",
		Job:           jobID,
		User:          "root",
		Dataset:       "root/dataset1",
		Model:         "root/model1",
		Objective:     "root/objective1",
		AltObjectives: []string{},
		AltQualities:  []float64{},
		CreationTime:  time.Now().Round(time.Millisecond).UTC(),
		Status:        "running",
	}
	var task2 = Task{
		ObjectID:      bson.NewObjectId(),
		ID:            string(jobID) + "/2",
		Job:           jobID,
		User:          "user1", //TODO: This is an impossible situation. The user is always the same for both the task and job.
		Dataset:       "root/dataset1",
		Model:         "root/model1",
		Objective:     "root/objective1",
		AltObjectives: []string{},
		AltQualities:  []float64{},
		CreationTime:  time.Now().Round(time.Millisecond).Add(time.Second).UTC(),
		Status:        "completed",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: User{ID: UserRoot}}

	// Add a test tasks to the test database.
	c := connection.Session.DB(TestDBName).C("tasks")
	err = c.Insert(task1, task2)
	assert.Nil(err)

	// Get all tasks.
	result, cm, err := context.GetTasks(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.Equal(task1, result[0])
	assert.ElementsMatch([]Task{task1, task2}, result)
	assert.True(result[0].ID < result[1].ID)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter tasks by ID.
	result, cm, err = context.GetTasks(map[string]interface{}{"id": []string{task1.ID, task1.ID}}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]Task{task1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter tasks by status.
	result, cm, err = context.GetTasks(map[string]interface{}{"status": "completed"}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]Task{task2}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order tasks by ID.
	result, cm, err = context.GetTasks(map[string]interface{}{}, 0, "", "creation-time", "desc")
	assert.Nil(err)
	assert.Equal([]Task{task2, task1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order tasks by status.
	result, cm, err = context.GetTasks(map[string]interface{}{}, 0, "", "status", "asc")
	assert.Nil(err)
	assert.Equal([]Task{task2, task1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Pagination with cursors. Natual order (by ObjectID).
	result1, cm, err := context.GetTasks(map[string]interface{}{}, 1, "", "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err := context.GetTasks(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err := context.GetTasks(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].ID < result2[0].ID)

	// Pagination with cursors. Ordered by status.
	result1, cm, err = context.GetTasks(map[string]interface{}{}, 1, "", "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err = context.GetTasks(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err = context.GetTasks(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].Status < result2[0].Status)

	// Try to order by a non-existant field.
	_, _, err = context.GetTasks(map[string]interface{}{}, 1, "", "wrong-field", "")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Try an unsupported order keyword.
	_, _, err = context.GetTasks(map[string]interface{}{}, 1, "", "status", "wrong")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Ensure access is limited for non-root tasks.
	context.User.ID = "user2"
	result, cm, err = context.GetTasks(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]Task{task1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestCreateTask(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	jobID := bson.NewObjectId()
	var task = Task{
		ObjectID:      bson.NewObjectId(),
		ID:            string(jobID) + "/1",
		Job:           jobID,
		User:          "root",
		Dataset:       "root/dataset1",
		Model:         "root/model1",
		Objective:     "root/objective1",
		AltObjectives: []string{},
		CreationTime:  time.Now(),
		Status:        "running",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: User{ID: UserRoot}}

	// Create the task.
	newTask, err := context.CreateTask(task)
	assert.Nil(err)
	assert.Equal("scheduled", newTask.Status)
	assert.True(newTask.CreationTime.Nanosecond() > task.CreationTime.Nanosecond())
	assert.NotEqual(task.ID, newTask.ID)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("tasks").Find(bson.M{"id": newTask.ID}).One(&task)
	assert.Nil(err)
	assert.Equal("scheduled", task.Status)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestPatchTask(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	jobID := bson.NewObjectId()
	var task = Task{
		ObjectID:      bson.NewObjectId(),
		ID:            string(jobID) + "/1",
		Job:           jobID,
		User:          "root",
		Dataset:       "root/dataset1",
		Model:         "root/model1",
		Objective:     "root/objective1",
		AltObjectives: []string{},
		CreationTime:  time.Now(),
		Status:        "running",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: User{ID: UserRoot}}

	// Add a test task to the test database.
	c := connection.Session.DB(TestDBName).C("tasks")
	err = c.Insert(task)
	assert.Nil(err)

	// Update the task.
	updates := map[string]interface{}{
		"status": "terminating",
	}
	task, err = context.UpdateTask(task.ID, updates)
	assert.Nil(err)
	assert.Equal("terminating", task.Status)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("tasks").Find(bson.M{"id": task.ID}).One(&task)
	assert.Nil(err)
	assert.Equal("terminating", task.Status)

	// Verify that we cannot add unsupported update fields.
	task, err = context.UpdateTask(task.ID, map[string]interface{}{"wrong-field": "terminating"})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Verify that we cannot update tasks that don't belong to us.
	context.User.ID = "user11"
	task, err = context.UpdateTask(task.ID, map[string]interface{}{"status": "terminating"})
	assert.Equal(ErrNotFound, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestLockTasks(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	jobID := bson.NewObjectId()
	var task1 = Task{
		ObjectID:      bson.NewObjectId(),
		ID:            jobID.Hex() + "/1",
		Job:           jobID,
		User:          "root",
		Dataset:       "root/dataset1",
		Model:         "root/model1",
		Objective:     "root/objective1",
		AltObjectives: []string{},
		AltQualities:  []float64{},
		CreationTime:  time.Now().Round(time.Millisecond).UTC(),
		Status:        "running",
	}
	var task2 = Task{
		ObjectID:      bson.NewObjectId(),
		ID:            jobID.Hex() + "/2",
		Job:           jobID,
		User:          "user1", //TODO: This is an impossible situation. The user is always the same for both the task and job.
		Dataset:       "root/dataset1",
		Model:         "root/model1",
		Objective:     "root/objective1",
		AltObjectives: []string{},
		AltQualities:  []float64{},
		CreationTime:  time.Now().Round(time.Millisecond).Add(time.Second).UTC(),
		Status:        "completed",
	}
	taskIDs := []string{task1.ID, task2.ID}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: User{ID: UserRoot}}

	// Add test tasks to the test database.
	c := connection.Session.DB(TestDBName).C("tasks")
	err = c.Insert(task1, task2)
	assert.Nil(err)

	// Mock process ID that we will be using for locking.
	processID := bson.NewObjectId()

	// Assert basic lock with filter works.
	var task Task
	task, err = context.LockTask(F{"id": taskIDs, "status": "running"}, processID, "", "")
	assert.Nil(err)
	assert.Equal(task1.ID, task.ID)

	// Assert we cannot lock any more tasks with the same filter.
	task, err = context.LockTask(F{"id": taskIDs, "status": "running"}, processID, "", "")
	assert.Equal(ErrNotFound, errors.Cause(err))

	// Unlock the task.
	err = context.UnlockTask(task1.ID, processID)
	assert.Nil(err)

	// Assert that we can now lock that task again.
	task, err = context.LockTask(F{"id": taskIDs, "status": "running"}, processID, "", "")
	assert.Nil(err)
	assert.Equal(task1.ID, task.ID)

	// Unlock the task.
	err = context.UnlockTask(task1.ID, processID)
	assert.Nil(err)

	// Assert that sorting works properly when locking.
	task, err = context.LockTask(F{"id": []string{task1.ID, task2.ID}}, processID, "creation-time", "desc")
	assert.Nil(err)
	assert.Equal(task2.ID, task.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}
