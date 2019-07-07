package model

import (
	"testing"

	"github.com/ds3lab/easeml/engine/database"
	"github.com/ds3lab/easeml/engine/database/model/types"

	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/assert"
)

func TestGetGetConfigCaches(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	jobID := bson.NewObjectId()
	var task1 = types.Task{
		ObjectID:      bson.NewObjectId(),
		ID:            string(jobID) + "/1",
		Job:           jobID,
		User:          "root",
		Dataset:       "root/dataset1",
		Objective:     "root/objective1",
		Model:         "root/model1",
		Config:        "config1",
		AltObjectives: []string{},
		Quality:       1.0,
	}
	var task2 = types.Task{
		ObjectID:      bson.NewObjectId(),
		ID:            string(jobID) + "/2",
		Job:           jobID,
		User:          "root",
		Dataset:       "root/dataset1",
		Objective:     "root/objective2",
		Model:         "root/model1",
		Config:        "config1",
		AltObjectives: []string{},
		Quality:       2.0,
	}
	var task3 = types.Task{
		ObjectID:      bson.NewObjectId(),
		ID:            string(jobID) + "/3",
		Job:           jobID,
		User:          "root",
		Dataset:       "root/dataset1",
		Objective:     "root/objective2",
		Model:         "root/model1",
		Config:        "config1",
		AltObjectives: []string{},
		Quality:       3.0,
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test tasks to the test database.
	c := connection.Session.DB(TestDBName).C("tasks")
	err = c.Insert(task1, task2, task3)
	assert.Nil(err)

	// Get the config cache objects.
	result, err := context.GetConfigCaches(nil)
	assert.Nil(err)
	assert.Equal(1, len(result))
	assert.Equal(2.0, result[0].AvgQuality)
	assert.Equal(1.0, result[0].Quality["root/dataset1"]["root/objective1"])
	assert.Equal(2.5, result[0].Quality["root/dataset1"]["root/objective2"])
}
