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

func TestGetModuleByID(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var module = types.Module{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/module1",
		User:          "root",
		Name:          "Module1",
		Description:   "Description of Module1",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "upload",
		SourceAddress: "http://module1",
		CreationTime:  time.Now(),
		Status:        "validated",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test module to the test database.
	c := connection.Session.DB(TestDBName).C("modules")
	err = c.Insert(module)
	assert.Nil(err)

	// Get the module by ID.
	result, err := context.GetModuleByID("root/module1")
	assert.Nil(err)
	assert.Equal(module.ObjectID, result.ObjectID)
	assert.Equal(module.ID, result.ID)

	// Ensure the we can't find the wrong module.
	result, err = context.GetModuleByID("root/module11")
	assert.Equal(ErrNotFound, err)
	assert.Empty(result.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestGetModules(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var module1 = types.Module{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/module1",
		User:          "root",
		Name:          "Module1",
		Description:   "Description of Module1",
		SchemaIn:      testSchemaInDst1,
		SchemaOut:     testSchemaOutDst1,
		Source:        "upload",
		SourceAddress: "http://module1",
		CreationTime:  time.Now().Round(time.Millisecond).UTC(),
		Status:        "validated",
	}
	var module2 = types.Module{
		ObjectID:      bson.NewObjectId(),
		ID:            "user1/module2",
		User:          "user1",
		Name:          "Module2",
		Description:   "Description of Module2",
		SchemaIn:      testSchemaInDst2,
		SchemaOut:     testSchemaOutDst2,
		Source:        "download",
		SourceAddress: "http://module2",
		CreationTime:  time.Now().Round(time.Millisecond).Add(time.Second).UTC(),
		Status:        "archived",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test modules to the test database.
	c := connection.Session.DB(TestDBName).C("modules")
	err = c.Insert(module1, module2)
	assert.Nil(err)

	// Get all modules.
	result, cm, err := context.GetModules(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Module{module1, module2}, result)
	assert.True(result[0].ObjectID < result[1].ObjectID)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter modules by ID.
	result, cm, err = context.GetModules(map[string]interface{}{"id": []string{"root/module1", "root/module1"}}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Module{module1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter modules by status.
	result, cm, err = context.GetModules(map[string]interface{}{"status": "archived"}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Module{module2}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter datasets by schema.
	filter := F{"schema-in": testSchemaInSrc1, "schema-out": testSchemaOutSrc1}
	result, cm, err = context.GetModules(filter, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Module{module1}, result)
	assert.Equal(2, cm.TotalResultSize) // This is not correct, but cannot be fixed given the current design.
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order modules by ID.
	result, cm, err = context.GetModules(map[string]interface{}{}, 0, "", "id", "desc")
	assert.Nil(err)
	assert.Equal([]types.Module{module2, module1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order modules by status.
	result, cm, err = context.GetModules(map[string]interface{}{}, 0, "", "status", "asc")
	assert.Nil(err)
	assert.Equal([]types.Module{module2, module1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Pagination with cursors. Natual order (by ObjectID).
	result1, cm, err := context.GetModules(map[string]interface{}{}, 1, "", "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err := context.GetModules(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err := context.GetModules(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].ObjectID < result2[0].ObjectID)

	// Pagination with cursors. Ordered by status.
	result1, cm, err = context.GetModules(map[string]interface{}{}, 1, "", "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err = context.GetModules(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err = context.GetModules(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].Status < result2[0].Status)

	// Try to order by a non-existant field.
	_, _, err = context.GetModules(map[string]interface{}{}, 1, "", "wrong-field", "")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Try an unsupported order keyword.
	_, _, err = context.GetModules(map[string]interface{}{}, 1, "", "status", "wrong")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Ensure access is limited for non-root modules.
	context.User.ID = "user2"
	result, cm, err = context.GetModules(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.Module{module1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestCreateModule(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var module = types.Module{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/module1",
		User:          "root",
		Name:          "Module1",
		Description:   "Description of Module1",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "upload",
		Type:          "model",
		SourceAddress: "http://module1",
		CreationTime:  time.Now(),
		Status:        "validated",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Create the module.
	newModule, err := context.CreateModule(module)
	assert.Nil(err)
	assert.Equal("created", newModule.Status)
	assert.True(newModule.CreationTime.Nanosecond() > module.CreationTime.Nanosecond())
	assert.NotEqual(module.ObjectID, newModule.ObjectID)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("modules").Find(bson.M{"id": "root/module1"}).One(&module)
	assert.Nil(err)
	assert.Equal("created", module.Status)

	// Try to create a module with a name that is already taken.
	newModule, err = context.CreateModule(types.Module{ID: "root/module1"})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestPatchModule(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var module = types.Module{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/module1",
		User:          "root",
		Name:          "Module1",
		Description:   "Description of Module1",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "upload",
		SourceAddress: "http://module1",
		CreationTime:  time.Now(),
		Status:        "validated",
	}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test module to the test database.
	c := connection.Session.DB(TestDBName).C("modules")
	err = c.Insert(module)
	assert.Nil(err)

	// Update the module.
	updates := map[string]interface{}{
		"status": "archived",
		"name":   "Module One",
	}
	module, err = context.UpdateModule("root/module1", updates)
	assert.Nil(err)
	assert.Equal("Module One", module.Name)
	assert.Equal("archived", module.Status)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("modules").Find(bson.M{"id": "root/module1"}).One(&module)
	assert.Nil(err)
	assert.Equal("Module One", module.Name)
	assert.Equal("archived", module.Status)

	// Verify that we cannot add unsupported update fields.
	module, err = context.UpdateModule("root/module1", map[string]interface{}{"wrong-field": "archived"})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Verify that we cannot update modules that don't belong to us.
	context.User.ID = "user11"
	module, err = context.UpdateModule("root/module1", map[string]interface{}{"status": "archived"})
	assert.Equal(ErrNotFound, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestLockModules(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
	var module1 = types.Module{
		ObjectID:      bson.NewObjectId(),
		ID:            "root/lock_module1",
		User:          "root",
		Name:          "Module1",
		Description:   "Description of Module1",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "upload",
		SourceAddress: "http://module1",
		CreationTime:  time.Now().Round(time.Millisecond).UTC(),
		Status:        "validated",
	}
	var module2 = types.Module{
		ObjectID:      bson.NewObjectId(),
		ID:            "user1/lock_module2",
		User:          "user1",
		Name:          "Module2",
		Description:   "Description of Module2",
		SchemaIn:      testSchemaInSrc1,
		SchemaOut:     testSchemaOutSrc1,
		Source:        "download",
		SourceAddress: "http://module2",
		CreationTime:  time.Now().Round(time.Millisecond).Add(time.Second).UTC(),
		Status:        "archived",
	}
	moduleIDs := []string{module1.ID, module2.ID}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add test modules to the test database.
	c := connection.Session.DB(TestDBName).C("modules")
	err = c.Insert(module1, module2)
	assert.Nil(err)

	// Mock process ID that we will be using for locking.
	processID := bson.NewObjectId()

	// Assert basic lock with filter works.
	var module types.Module
	module, err = context.LockModule(F{"id": moduleIDs, "source": "upload"}, processID, "", "")
	assert.Nil(err)
	assert.Equal(module1.ID, module.ID)

	// Assert we cannot lock any more modules with the same filter.
	module, err = context.LockModule(
		F{
			"id":     moduleIDs,
			"source": "upload"},
		processID, "", "")
	assert.Equal(ErrNotFound, errors.Cause(err))

	// Unlock the module.
	err = context.UnlockModule(module1.ID, processID)
	assert.Nil(err)

	// Assert that we can now lock that module again.
	module, err = context.LockModule(F{"id": moduleIDs, "source": "upload"}, processID, "", "")
	assert.Nil(err)
	assert.Equal(module1.ID, module.ID)

	// Unlock the module.
	err = context.UnlockModule(module1.ID, processID)
	assert.Nil(err)

	// Assert that sorting works properly when locking.
	module, err = context.LockModule(F{"id": moduleIDs}, processID, "creation-time", "desc")
	assert.Nil(err)
	assert.Equal(module2.ID, module.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}
