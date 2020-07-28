package database

import (
	"github.com/ds3lab/easeml/engine/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	assert := assert.New(t)
	var MongoInstance = utils.GetEnvVariableOrDefault("EASEML_DATABASE_ADDRESS","localhost")
	connection, err := Connect(MongoInstance, "testdb")

	assert.Nil(err)
	assert.Equal("testdb", connection.DBName)

	_, err = connection.Session.BuildInfo()
	assert.Nil(err)
}
