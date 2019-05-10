package model

import (
	"testing"
	"time"

	"github.com/ds3lab/easeml/engine/easeml/database"
	"github.com/ds3lab/easeml/engine/easeml/database/model/types"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGetProcessByID(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var process = types.Process{
		ID:          bson.NewObjectId(),
		ProcessID:   1,
		HostID:      "123456",
		HostAddress: "123.123.123.123",
		StartTime:   time.Now().UTC(),
		Type:        "controller",
		Resource:    "cpu",
		Status:      "idle",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test process to the test database.
	c := connection.Session.DB(TestDBName).C("processes")
	err = c.Insert(process)
	assert.Nil(err)

	// Get the process by ID.
	result, err := context.GetProcessByID(process.ID)
	assert.Nil(err)
	assert.Equal(process.ID, result.ID)

	// Ensure the we can't find the wrong process.
	result, err = context.GetProcessByID(bson.NewObjectId())
	assert.Equal(ErrNotFound, err)
	assert.Empty(result.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestGetProcesses(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var process1 = types.Process{
		ID:          bson.NewObjectId(),
		ProcessID:   1,
		HostID:      "123456",
		HostAddress: "123.123.123.123",
		StartTime:   time.Now().Round(time.Millisecond).UTC(),
		Type:        "controller",
		Resource:    "cpu",
		Status:      "idle",
	}
	var process2 = types.Process{
		ID:          bson.NewObjectId(),
		ProcessID:   1,
		HostID:      "123456",
		HostAddress: "123.123.123.123",
		StartTime:   time.Now().Round(time.Millisecond).Add(time.Second).UTC(),
		Type:        "worker",
		Resource:    "cpu",
		Status:      "terminated",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test processes to the test database.
	c := connection.Session.DB(TestDBName).C("processes")
	err = c.Insert(process1, process2)
	assert.Nil(err)

	// Get all processes.
	result, cm, err := context.GetProcesses(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Process{process1, process2}, result)
	assert.True(result[0].ID < result[1].ID)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter processes by ID.
	result, cm, err = context.GetProcesses(map[string]interface{}{"id": []bson.ObjectId{process1.ID, process1.ID}}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Process{process1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter processes by status.
	result, cm, err = context.GetProcesses(map[string]interface{}{"status": "terminated"}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Process{process2}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order processes by ID.
	result, cm, err = context.GetProcesses(map[string]interface{}{}, 0, "", "id", "desc")
	assert.Nil(err)
	assert.Equal([]types.Process{process2, process1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order processes by status.
	result, cm, err = context.GetProcesses(map[string]interface{}{}, 0, "", "status", "asc")
	assert.Nil(err)
	assert.Equal([]types.Process{process1, process2}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Pagination with cursors. Natual order (by ObjectID).
	result1, cm, err := context.GetProcesses(map[string]interface{}{}, 1, "", "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err := context.GetProcesses(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err := context.GetProcesses(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].ID < result2[0].ID)

	// Pagination with cursors. Ordered by status.
	result1, cm, err = context.GetProcesses(map[string]interface{}{}, 1, "", "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err = context.GetProcesses(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err = context.GetProcesses(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].Status < result2[0].Status)

	// Try to order by a non-existant field.
	_, _, err = context.GetProcesses(map[string]interface{}{}, 1, "", "wrong-field", "")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Try an unsupported order keyword.
	_, _, err = context.GetProcesses(map[string]interface{}{}, 1, "", "status", "wrong")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Currently there is not access control for non-root users, so we won't test it.

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestCreateProcess(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var process = types.Process{
		ID:          bson.NewObjectId(),
		ProcessID:   1,
		HostID:      "123456",
		HostAddress: "123.123.123.123",
		StartTime:   time.Now(),
		Type:        "controller",
		Resource:    "cpu",
		Status:      "running",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Create the process.
	newProcess, err := context.CreateProcess(process)
	assert.Nil(err)
	assert.Equal("idle", newProcess.Status)
	assert.True(newProcess.StartTime.Nanosecond() > process.StartTime.Nanosecond())
	assert.NotEqual(process.ID, newProcess.ID)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("processes").Find(bson.M{"_id": newProcess.ID}).One(&process)
	assert.Nil(err)
	assert.Equal("idle", process.Status)

	// Try to create a process with a name that is already taken.
	newProcess, err = context.CreateProcess(types.Process{ID: process.ID})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestPatchProcess(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var process = types.Process{
		ID:          bson.NewObjectId(),
		ProcessID:   1,
		HostID:      "123456",
		HostAddress: "123.123.123.123",
		StartTime:   time.Now(),
		Type:        "controller",
		Resource:    "cpu",
		Status:      "idle",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test process to the test database.
	c := connection.Session.DB(TestDBName).C("processes")
	err = c.Insert(process)
	assert.Nil(err)

	// Update the process.
	updates := map[string]interface{}{
		"status": "terminated",
	}
	process, err = context.UpdateProcess(process.ID, updates)
	assert.Nil(err)
	assert.Equal("terminated", process.Status)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("processes").Find(bson.M{"_id": process.ID}).One(&process)
	assert.Nil(err)
	assert.Equal("terminated", process.Status)

	// Verify that we cannot add unsupported update fields.
	process, err = context.UpdateProcess(process.ID, map[string]interface{}{"wrong-field": "terminated"})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Verify that we cannot update processes that don't belong to us.
	context.User.ID = "user11"
	process, err = context.UpdateProcess("root/process1", map[string]interface{}{"status": "terminated"})
	assert.Equal(types.ErrUnauthorized, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}
