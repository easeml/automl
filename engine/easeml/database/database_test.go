package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	assert := assert.New(t)
	connection, err := Connect("localhost", "testdb")

	assert.Nil(err)
	assert.Equal("testdb", connection.DBName)

	_, err = connection.Session.BuildInfo()
	assert.Nil(err)
}
