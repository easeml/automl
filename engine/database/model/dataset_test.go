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

const testSchemaInSrc1 string = `{
	"nodes":{
		"c1_src":{"singleton":true,"type":"category","class":"class1_src"}
	},"classes":{
		"class1_src":{"dim":16}
	}
}`

const testSchemaOutSrc1 string = `{
	"nodes":{
		"node1_src":{"singleton":true,"type":"tensor","dim":[16]}
	}
}`

const testSchemaInSrc2 string = `{
	"nodes":{
		"c1_src":{"singleton":true,"type":"category","class":"class1_src"}
	},"classes":{
		"class1_src":{"dim":16}
	}
}`

const testSchemaOutSrc2 string = `{
	"nodes":{
		"node1_src":{"singleton":true,"type":"tensor","dim":[32, 32]}
	}
}`

const testSchemaInDst1 string = `{
	"nodes":{
		"c1_src":{"singleton":true,"type":"category","class":"class1_src"}
	},"classes":{
		"class1_src":{"dim":"a"}
	}
}`

const testSchemaOutDst1 string = `{
	"nodes":{
		"node1_src":{"singleton":true,"type":"tensor","dim":["a"]}
	}
}`

const testSchemaInDst2 string = `{
	"nodes":{
		"c1_src":{"singleton":true,"type":"category","class":"class1_src"}
	},"classes":{
		"class1_src":{"dim":"a"}
	}
}`

const testSchemaOutDst2 string = `{
	"nodes":{
		"node1_src":{"singleton":true,"type":"tensor","dim":["a", "a"]}
	}
}`

func TestGetDatasetByID(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var dataset = types.Dataset{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/dataset1",
		User:          "root",
		Name:          "Dataset1",
		Description:   "Description of Dataset1",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "upload",
		SourceAddress: "http://dataset1",
		CreationTime:  time.Now(),
		Status:        "validated",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test dataset to the test database.
	c := connection.Session.DB(TestDBName).C("datasets")
	err = c.Insert(dataset)
	assert.Nil(err)

	// Get the dataset by ID.
	result, err := context.GetDatasetByID("root/dataset1")
	assert.Nil(err)
	assert.Equal(dataset.ObjectID, result.ObjectID)
	assert.Equal(dataset.ID, result.ID)

	// Ensure the we can't find the wrong dataset.
	result, err = context.GetDatasetByID("root/dataset11")
	assert.Equal(ErrNotFound, err)
	assert.Empty(result.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestGetDatasets(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var dataset1 = types.Dataset{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/dataset1",
		User:          "root",
		Name:          "Dataset1",
		Description:   "Description of Dataset1",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "upload",
		SourceAddress: "http://dataset1",
		CreationTime:  time.Now().Round(time.Millisecond).UTC(),
		Status:        "validated",
	}
	var dataset2 = types.Dataset{
		ObjectID:      bson.NewObjectId(),
		ID:            "user1/dataset2",
		User:          "user1",
		Name:          "Dataset2",
		Description:   "Description of Dataset2",
		SchemaIn:      testSchemaInSrc2,
		SchemaOut:     testSchemaOutSrc2,
		Source:        "download",
		SourceAddress: "http://dataset2",
		CreationTime:  time.Now().Round(time.Millisecond).Add(time.Second).UTC(),
		Status:        "archived",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add test datasets to the test database.
	c := connection.Session.DB(TestDBName).C("datasets")
	err = c.Insert(dataset1, dataset2)
	assert.Nil(err)

	// Get all datasets.
	result, cm, err := context.GetDatasets(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Dataset{dataset1, dataset2}, result)
	assert.True(result[0].ObjectID < result[1].ObjectID)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter datasets by ID.
	result, cm, err = context.GetDatasets(map[string]interface{}{"id": []string{"root/dataset1", "root/dataset1"}}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Dataset{dataset1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter datasets by status.
	result, cm, err = context.GetDatasets(map[string]interface{}{"status": "archived"}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Dataset{dataset2}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter datasets by schema.
	filter := F{"schema-in": testSchemaInDst1, "schema-out": testSchemaOutDst1}
	result, cm, err = context.GetDatasets(filter, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Dataset{dataset1}, result)
	assert.Equal(2, cm.TotalResultSize) // This is not correct, but cannot be fixed given the current design.
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order datasets by ID.
	result, cm, err = context.GetDatasets(map[string]interface{}{}, 0, "", "id", "desc")
	assert.Nil(err)
	assert.Equal([]types.Dataset{dataset2, dataset1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order datasets by status.
	result, cm, err = context.GetDatasets(map[string]interface{}{}, 0, "", "status", "asc")
	assert.Nil(err)
	assert.Equal([]types.Dataset{dataset2, dataset1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Pagination with cursors. Natual order (by ObjectID).
	result1, cm, err := context.GetDatasets(map[string]interface{}{}, 1, "", "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err := context.GetDatasets(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err := context.GetDatasets(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].ObjectID < result2[0].ObjectID)

	// Pagination with cursors. Ordered by status.
	result1, cm, err = context.GetDatasets(map[string]interface{}{}, 1, "", "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err = context.GetDatasets(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err = context.GetDatasets(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].Status < result2[0].Status)

	// Try to order by a non-existant field.
	_, _, err = context.GetDatasets(map[string]interface{}{}, 1, "", "wrong-field", "")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Try an unsupported order keyword.
	_, _, err = context.GetDatasets(map[string]interface{}{}, 1, "", "status", "wrong")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Ensure access is limited for non-root datasets.
	context.User.ID = "user2"
	result, cm, err = context.GetDatasets(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Dataset{dataset1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestCreateDataset(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var dataset = types.Dataset{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/dataset1",
		User:          "root",
		Name:          "Dataset1",
		Description:   "Description of Dataset1",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "upload",
		SourceAddress: "http://dataset1",
		CreationTime:  time.Now(),
		Status:        "validated",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Create the dataset.
	newDataset, err := context.CreateDataset(dataset)
	assert.Nil(err)
	assert.Equal("created", newDataset.Status)
	assert.True(newDataset.CreationTime.Nanosecond() > dataset.CreationTime.Nanosecond())
	assert.NotEqual(dataset.ObjectID, newDataset.ObjectID)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("datasets").Find(bson.M{"id": "root/dataset1"}).One(&dataset)
	assert.Nil(err)
	assert.Equal("created", dataset.Status)

	// Try to create a dataset with a name that is already taken.
	newDataset, err = context.CreateDataset(types.Dataset{ID: "root/dataset1"})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestPatchDataset(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var dataset = types.Dataset{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/dataset1",
		User:          "root",
		Name:          "Dataset1",
		Description:   "Description of Dataset1",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "upload",
		SourceAddress: "http://dataset1",
		CreationTime:  time.Now(),
		Status:        "validated",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test dataset to the test database.
	c := connection.Session.DB(TestDBName).C("datasets")
	err = c.Insert(dataset)
	assert.Nil(err)

	// Update the dataset.
	updates := map[string]interface{}{
		"status": "archived",
		"name":   "Dataset One",
	}
	dataset, err = context.UpdateDataset("root/dataset1", updates)
	assert.Nil(err)
	assert.Equal("Dataset One", dataset.Name)
	assert.Equal("archived", dataset.Status)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("datasets").Find(bson.M{"id": "root/dataset1"}).One(&dataset)
	assert.Nil(err)
	assert.Equal("Dataset One", dataset.Name)
	assert.Equal("archived", dataset.Status)

	// Verify that we cannot add unsupported update fields.
	dataset, err = context.UpdateDataset("root/dataset1", map[string]interface{}{"wrong-field": "archived"})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Verify that we cannot update datasets that don't belong to us.
	context.User.ID = "user11"
	dataset, err = context.UpdateDataset("root/dataset1", map[string]interface{}{"status": "archived"})
	assert.Equal(ErrNotFound, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestLockDatasets(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var dataset1 = types.Dataset{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/lock_dataset1",
		User:          "root",
		Name:          "Dataset1",
		Description:   "Description of Dataset1",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "upload",
		SourceAddress: "http://dataset1",
		CreationTime:  time.Now().Round(time.Millisecond).UTC(),
		Status:        "validated",
	}
	var dataset2 = types.Dataset{
		ObjectID:      bson.NewObjectId(),
		ID:            "user1/lock_dataset2",
		User:          "user1",
		Name:          "Dataset2",
		Description:   "Description of Dataset2",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "download",
		SourceAddress: "http://dataset2",
		CreationTime:  time.Now().Round(time.Millisecond).Add(time.Second).UTC(),
		Status:        "archived",
	}
	datasetIDs := []string{dataset1.ID, dataset2.ID}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add test datasets to the test database.
	c := connection.Session.DB(TestDBName).C("datasets")
	err = c.Insert(dataset1, dataset2)
	assert.Nil(err)

	// Mock process ID that we will be using for locking.
	processID := bson.NewObjectId()

	// Assert basic lock with filter works.
	var dataset types.Dataset
	dataset, err = context.LockDataset(F{"id": datasetIDs, "source": "upload"}, processID, "", "")
	assert.Nil(err)
	assert.Equal(dataset1.ID, dataset.ID)

	// Assert we cannot lock any more datasets with the same filter.
	dataset, err = context.LockDataset(
		F{
			"id":     datasetIDs,
			"source": "upload"},
		processID, "", "")
	assert.Equal(ErrNotFound, errors.Cause(err))

	// Unlock the dataset.
	err = context.UnlockDataset(dataset1.ID, processID)
	assert.Nil(err)

	// Assert that we can now lock that dataset again.
	dataset, err = context.LockDataset(F{"id": datasetIDs, "source": "upload"}, processID, "", "")
	assert.Nil(err)
	assert.Equal(dataset1.ID, dataset.ID)

	// Unlock the dataset.
	err = context.UnlockDataset(dataset1.ID, processID)
	assert.Nil(err)

	// Assert that sorting works properly when locking.
	dataset, err = context.LockDataset(F{"id": datasetIDs}, processID, "creation-time", "desc")
	assert.Nil(err)
	assert.Equal(dataset2.ID, dataset.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}
