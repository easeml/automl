package model

import (
	"testing"

	"github.com/ds3lab/easeml/engine/database"
	"github.com/ds3lab/easeml/engine/database/model/types"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/assert"
)

var MongoInstance = "localhost"
var TestDBName = "testdb"

func TestUserAuthenticate(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test user to the test database.
	c := connection.Session.DB(TestDBName).C("users")
	err = c.Insert(types.User{ObjectID: bson.NewObjectId(), ID: "user1", PasswordHash: "hash1", Status: "active", APIKey: "apikey1"})
	assert.Nil(err)

	// Authenticate with API Key.
	user, err := context.UserAuthenticate(types.User{APIKey: "apikey1"})
	assert.Nil(err)
	assert.Equal("user1", user.ID)

	// If the API Key is wrong, we should get an ErrWrongAPIKey error.
	user, err = context.UserAuthenticate(types.User{APIKey: "apikey11"})
	assert.Equal(types.ErrWrongAPIKey, err)

	// Authenticate with user ID and password hash.
	user, err = context.UserAuthenticate(types.User{ID: "user1", PasswordHash: "hash1"})
	assert.Nil(err)
	assert.Equal("apikey1", user.APIKey)

	// If the user ID or password are wrong, we should get an ErrWrongCredentials error.
	user, err = context.UserAuthenticate(types.User{ID: "user11", PasswordHash: "hash1"})
	assert.Equal(types.ErrWrongCredentials, err)
	assert.Empty(user.ID)
	user, err = context.UserAuthenticate(types.User{ID: "user1", PasswordHash: "hash11"})
	assert.Equal(types.ErrWrongCredentials, err)
	assert.Empty(user.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestUserGenerateAPIKey(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	var user = types.User{ObjectID: bson.NewObjectId(), ID: "user1", PasswordHash: "hash1", Status: "active"}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: user}

	// Add a test user to the test database.
	c := connection.Session.DB(TestDBName).C("users")
	err = c.Insert(user)
	assert.Nil(err)

	// Generate the API key.
	var result string
	result, err = context.UserGenerateAPIKey()
	assert.Nil(err)
	apiKey, err := uuid.FromString(result)
	assert.Nil(err)
	assert.Equal(uuid.V4, apiKey.Version())

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("users").Find(bson.M{"id": "user1"}).One(&user)
	assert.Nil(err)
	assert.Equal(apiKey.String(), user.APIKey)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestUserDeleteAPIKey(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	var user = types.User{ObjectID: bson.NewObjectId(), ID: "user1", PasswordHash: "hash1", Status: "active"}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: user}

	// Add a test user to the test database.
	c := connection.Session.DB(TestDBName).C("users")
	err = c.Insert(user)
	assert.Nil(err)

	// Generate the API key.
	var apiKey string
	apiKey, err = context.UserGenerateAPIKey()
	assert.Nil(err)
	context.User.APIKey = apiKey

	// Delete the API key.
	err = context.UserDeleteAPIKey()
	assert.Nil(err)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("users").Find(bson.M{"id": "user1"}).One(&user)
	assert.Nil(err)
	assert.Empty(user.APIKey)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestGetUserByID(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	var user = types.User{ObjectID: bson.NewObjectId(), ID: "user1", PasswordHash: "hash1", Status: "active", APIKey: "apikey1"}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test user to the test database.
	c := connection.Session.DB(TestDBName).C("users")
	err = c.Insert(user)
	assert.Nil(err)

	// Get the user by ID.
	result, err := context.GetUserByID("user1")
	assert.Nil(err)
	assert.Equal(user, result)

	// Ensure the we can't find the wrong user.
	result, err = context.GetUserByID("user11")
	assert.Equal(ErrNotFound, err)
	assert.Empty(result.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestGetUserByAPIKey(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	var user = types.User{ObjectID: bson.NewObjectId(), ID: "user1", PasswordHash: "hash1", Status: "active", APIKey: "apikey1"}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test user to the test database.
	c := connection.Session.DB(TestDBName).C("users")
	err = c.Insert(user)
	assert.Nil(err)

	// Get the user by ID.
	result, err := context.GetUserByAPIKey("apikey1")
	assert.Nil(err)
	assert.Equal(user, result)

	// Ensure the we can't find the wrong user.
	result, err = context.GetUserByAPIKey("apikey11")
	assert.Equal(ErrNotFound, err)
	assert.Empty(result.ID)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestGetUsers(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	var user1 = types.User{ObjectID: bson.NewObjectId(), ID: "user1", PasswordHash: "hash1", Status: "active", APIKey: "apikey1"}
	var user2 = types.User{ObjectID: bson.NewObjectId(), ID: "user2", PasswordHash: "hash2", Status: "archived", APIKey: "apikey2"}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Add a test users to the test database.
	c := connection.Session.DB(TestDBName).C("users")
	err = c.Insert(user1, user2)
	assert.Nil(err)

	// Get all users.
	result, cm, err := context.GetUsers(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.User{user1, user2}, result)
	assert.True(result[0].ObjectID < result[1].ObjectID)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter users by ID.
	result, cm, err = context.GetUsers(map[string]interface{}{"id": []string{"user1", "user1"}}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.User{user1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Filter users by status.
	result, cm, err = context.GetUsers(map[string]interface{}{"status": "archived"}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.User{user2}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order users by ID.
	result, cm, err = context.GetUsers(map[string]interface{}{}, 0, "", "id", "desc")
	assert.Nil(err)
	assert.Equal([]types.User{user2, user1}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Order users by status.
	result, cm, err = context.GetUsers(map[string]interface{}{}, 0, "", "status", "asc")
	assert.Nil(err)
	assert.Equal([]types.User{user1, user2}, result)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(2, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Pagination with cursors. Natual order (by ObjectID).
	result1, cm, err := context.GetUsers(map[string]interface{}{}, 1, "", "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err := context.GetUsers(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err := context.GetUsers(map[string]interface{}{}, 1, cm.NextPageCursor, "", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].ObjectID < result2[0].ObjectID)

	// Pagination with cursors. Ordered by status.
	result1, cm, err = context.GetUsers(map[string]interface{}{}, 1, "", "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result2, cm, err = context.GetUsers(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.NotEmpty(cm.NextPageCursor)

	result3, cm, err = context.GetUsers(map[string]interface{}{}, 1, cm.NextPageCursor, "status", "")
	assert.Nil(err)
	assert.Equal(2, cm.TotalResultSize)
	assert.Equal(0, cm.ReturnedResultSize)
	assert.Empty(cm.NextPageCursor)
	assert.Empty(result3)

	assert.True(result1[0].Status < result2[0].Status)

	// Try to order by a non-existant field.
	_, _, err = context.GetUsers(map[string]interface{}{}, 1, "", "wrong-field", "")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Try an unsupported order keyword.
	_, _, err = context.GetUsers(map[string]interface{}{}, 1, "", "status", "wrong")
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Ensure access is limited for non-root users.
	context.User = user1
	result, cm, err = context.GetUsers(map[string]interface{}{}, 0, "", "", "")
	assert.Nil(err)
	assert.ElementsMatch([]types.User{user1}, result)
	assert.Equal(1, cm.TotalResultSize)
	assert.Equal(1, cm.ReturnedResultSize)
	assert.Equal("", cm.NextPageCursor)

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestCreateUser(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	var user = types.User{ObjectID: bson.NewObjectId(), ID: "user1", PasswordHash: "hash1", Status: "active", APIKey: "apikey1"}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: types.User{ID: types.UserRoot}}

	// Create the user.
	newUser, err := context.CreateUser(user)
	assert.Nil(err)
	assert.Empty(newUser.APIKey)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("users").Find(bson.M{"id": "user1"}).One(&user)
	assert.Nil(err)
	assert.Empty(user.APIKey)

	// Try to create a root user.
	newUser, err = context.CreateUser(types.User{ID: types.UserRoot})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestPatchUser(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	var user = types.User{ObjectID: bson.NewObjectId(), ID: "user1", PasswordHash: "hash1", Status: "active", APIKey: "apikey1"}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: user}

	// Add a test user to the test database.
	c := connection.Session.DB(TestDBName).C("users")
	err = c.Insert(user)
	assert.Nil(err)

	// Update the user.
	user, err = context.UpdateUser("user1", map[string]interface{}{"status": "archived", "name": "User One"})
	assert.Nil(err)
	assert.Equal("User One", user.Name)
	assert.Equal("archived", user.Status)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("users").Find(bson.M{"id": "user1"}).One(&user)
	assert.Nil(err)
	assert.Equal("User One", user.Name)
	assert.Equal("archived", user.Status)

	// Verify that we cannot add unsupported update fields.
	user, err = context.UpdateUser("user1", map[string]interface{}{"wrong-field": "archived"})
	assert.Equal(ErrBadInput, errors.Cause(err))

	// Verify that we cannot update other users.
	context.User.ID = "user11"
	user, err = context.UpdateUser("user1", map[string]interface{}{"status": "archived"})
	assert.Equal(ErrNotFound, errors.Cause(err))

	// Drop the test database.
	err = connection.Session.DB(TestDBName).DropDatabase()
	assert.Nil(err)
}

func TestUserLoginAndOut(t *testing.T) {
	assert := assert.New(t)

	// Establish a connection.
	connection, err := database.Connect(MongoInstance, TestDBName)
	assert.Nil(err)
	var user = types.User{ObjectID: bson.NewObjectId(), ID: "user1", PasswordHash: "hash1", Status: "active"}
	var context = Context{Session: connection.Session, DBName: connection.DBName, User: user}

	// Add a test user to the test database.
	c := connection.Session.DB(TestDBName).C("users")
	err = c.Insert(user)
	assert.Nil(err)

	// Log the user in.
	user, err = context.UserLogin()
	assert.Nil(err)
	assert.NotEmpty(user.APIKey)
	context.User.APIKey = user.APIKey

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("users").Find(bson.M{"id": "user1"}).One(&user)
	assert.Nil(err)
	assert.NotEmpty(user.APIKey)

	// Log the user out.
	err = context.UserLogout()
	assert.Nil(err)

	// Verify the database has been updated.
	err = connection.Session.DB(TestDBName).C("users").Find(bson.M{"id": "user1"}).One(&user)
	assert.Nil(err)
	assert.Empty(user.APIKey)
}
