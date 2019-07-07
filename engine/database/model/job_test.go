package model

import (
	"testing"
	"time"

	"github.com/ds3lab/easeml/engine/database"
	"github.com/ds3lab/easeml/engine/database/model/types"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGetJobByID(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var job = types.Job{
		ID:           bson.NewObjectId(),
		User:         "root",
		Dataset:      "root/dataset1",
		Models:       []string{"root/model1"},
		Objective:    "root/objective1",
		CreationTime: time.Now(),
		Status:       "running",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test job to the test database.
	c := connection.Session.DB(TestDBName).C("jobs")
	err = c.Insert(job)
	assert.Nil(err)

	// Get the job by ID.
	result, err := context.GetJobByID(job.ID)
	assert.Nil(err)
	assert.Equal(job.ID, result.ID)
	assert.Equal(job.ID, result.ID)

	// Ensure the we can't find the wrong job.
	result, err = context.GetJobByID(bson.NewObjectId())
	assert.Equal(ErrNotFound, err)
	assert.Empty(result.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestGetJobs(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var job1 = types.Job{
		ID:            bson.NewObjectId(),
		User:          "root",
		Dataset:       "root/dataset1",
		Models:        []string{"root/model1"},
		Objective:     "root/objective1",
		AltObjectives: []string{},
		CreationTime:  time.Now().Round(time.Millisecond).UTC(),
		Status:        "error",
	}
	var job2 = types.Job{
		ID:            bson.NewObjectId(),
		User:          "user1",
		Dataset:       "root/dataset2",
		Models:        []string{"root/model1"},
		Objective:     "root/objective1",
		AltObjectives: []string{},
		CreationTime:  time.Now().Round(time.Millisecond).Add(time.Second).UTC(),
		Status:        "completed",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test jobs to the test database.
	c := connection.Session.DB(TestDBName).C("jobs")
	err = c.Insert(job1, job2)
	assert.Nil(err)

	// Get all jobs.
	result, cm, err := context.GetJobs(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Job{job1, job2}, result)
	assert.True(result[0].ID < result[1].ID)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter jobs by ID.
	result, cm, err = context.GetJobs(map[string]interface{}{"id": []bson.ObjectId{job1.ID, job1.ID}}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Job{job1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter jobs by status.
	result, cm, err = context.GetJobs(map[string]interface{}{"status": "completed"}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Job{job2}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order jobs by ID.
	result, cm, err = context.GetJobs(map[string]interface{}{}, 0, "", "creation-time", "desc")
	assert.Nil(err)
	assert.Equal([]types.Job{job2, job1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order jobs by status.
	result, cm, err = context.GetJobs(map[string]interface{}{}, 0, "", "status", "asc")
	assert.Nil(err)
	assert.Equal([]types.Job{job2, job1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Pagination with cursors. Natual order (by ObjectID).
	result1, cm, err := context.GetJobs(map[string]interface{}{}, 1, "", "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err := context.GetJobs(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err := context.GetJobs(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].ID < result2[0].ID)

	// Pagination with cursors. Ordered by status.
	result1, cm, err = context.GetJobs(map[string]interface{}{}, 1, "", "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err = context.GetJobs(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err = context.GetJobs(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].Status < result2[0].Status)

	// Try to order by a non-existant field.
	_, _, err = context.GetJobs(map[string]interface{}{}, 1, "", "wrong-field", "")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Try an unsupported order keyword.
	_, _, err = context.GetJobs(map[string]interface{}{}, 1, "", "status", "wrong")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Ensure access is limited for non-root jobs.
	context.User.ID = "user2"
	result, cm, err = context.GetJobs(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Job{job1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestCreateJob(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var job = types.Job{
		ID:           bson.NewObjectId(),
		User:         "root",
		Dataset:      "root/dataset1",
		Models:       []string{"root/model1"},
		Objective:    "root/objective1",
		CreationTime: time.Now(),
		Status:       "running",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Create the job.
	newJob, err := context.CreateJob(job)
	assert.Nil(err)
	assert.Equal("running", newJob.Status)
	assert.True(newJob.CreationTime.Nanosecond() > job.CreationTime.Nanosecond())
	assert.NotEqual(job.ID, newJob.ID)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("jobs").Find(bson.M{"_id": newJob.ID}).One(&job)
	assert.Nil(err)
	assert.Equal("running", job.Status)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestPatchJob(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	jobID := bson.NewObjectId()
	var job = types.Job{
		ID:           jobID,
		User:         "root",
		Dataset:      "root/dataset1",
		Models:       []string{"root/model1"},
		Objective:    "root/objective1",
		CreationTime: time.Now(),
		Status:       "running",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test job to the test database.
	c := connection.Session.DB(TestDBName).C("jobs")
	err = c.Insert(job)
	assert.Nil(err)

	// Update the job.
	updates := map[string]interface{}{
		"status": "terminating",
	}
	job, err = context.UpdateJob(jobID, updates)
	assert.Nil(err)
	assert.Equal("terminating", job.Status)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("jobs").Find(bson.M{"_id": job.ID}).One(&job)
	assert.Nil(err)
	assert.Equal("terminating", job.Status)

	// Verify that we cannot add unsupported update fields.
	job, err = context.UpdateJob(jobID, map[string]interface{}{"wrong-field": "terminating"})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Verify that we cannot update jobs that don't belong to us.
	context.User.ID = "user11"
	job, err = context.UpdateJob(jobID, map[string]interface{}{"status": "terminating"})
	assert.Equal(types.ErrUnauthorized, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestLockJobs(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var job1 = types.Job{
		ID:            bson.NewObjectId(),
		User:          "root",
		Dataset:       "root/dataset1",
		Models:        []string{"root/model1"},
		Objective:     "root/objective1",
		AltObjectives: []string{},
		CreationTime:  time.Now().Round(time.Millisecond).UTC(),
		Status:        "error",
	}
	var job2 = types.Job{
		ID:            bson.NewObjectId(),
		User:          "user1",
		Dataset:       "root/dataset2",
		Models:        []string{"root/model1"},
		Objective:     "root/objective1",
		AltObjectives: []string{},
		CreationTime:  time.Now().Round(time.Millisecond).Add(time.Second).UTC(),
		Status:        "completed",
	}
	jobIDs := []bson.ObjectId{job1.ID, job2.ID}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add test jobs to the test database.
	c := connection.Session.DB(TestDBName).C("jobs")
	err = c.Insert(job1, job2)
	assert.Nil(err)

	// Mock process ID that we will be using for locking.
	processID := bson.NewObjectId()

	// Assert basic lock with filter works.
	var job types.Job
	job, err = context.LockJob(F{"id": jobIDs, "status": "error"}, processID, "", "")
	assert.Nil(err)
	assert.Equal(job1.ID, job.ID)

	// Assert we cannot lock any more jobs with the same filter.
	job, err = context.LockJob(F{"id": jobIDs, "status": "error"}, processID, "", "")
	assert.Equal(ErrNotFound, errors.Cause(err))

	// Unlock the job.
	err = context.UnlockJob(job1.ID, processID)
	assert.Nil(err)

	// Assert that we can now lock that job again.
	job, err = context.LockJob(F{"id": jobIDs, "status": "error"}, processID, "", "")
	assert.Nil(err)
	assert.Equal(job1.ID, job.ID)

	// Unlock the job.
	err = context.UnlockJob(job1.ID, processID)
	assert.Nil(err)

	// Assert that sorting works properly when locking.
	job, err = context.LockJob(F{"id": jobIDs}, processID, "creation-time", "desc")
	assert.Nil(err)
	assert.Equal(job2.ID, job.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}
