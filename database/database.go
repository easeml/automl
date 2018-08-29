package database

import (
	"time"

	"github.com/globalsign/mgo"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

// Connection defines dependencies required for database access.
type Connection struct {
	Session *mgo.Session
	DBName  string
}

// Session represents the MongoDB connection.
// var Session *mgo.Session

// DBName is the name of the target database.
// var DBName string

// Cache is the key/value cache store used to reduce database trafic.
var Cache = cache.New(cache.NoExpiration, cache.NoExpiration)

// Connect initializes the MongoDB session.
func Connect(dataSourceName string, databaseName string) (connection Connection, err error) {
	// Connect to MongoDB.
	if connection.Session, err = mgo.DialWithTimeout(dataSourceName, 5*time.Second); err != nil {
		err = errors.Wrap(err, "mongo dial failed")
		return
	}

	// Prevents these errors: read tcp 127.0.0.1:27017: i/o timeout.
	connection.Session.SetSocketTimeout(5 * time.Second)

	// Check if is alive.
	if err = connection.Session.Ping(); err != nil {
		err = errors.Wrap(err, "mongo dial failed")
		return
	}

	connection.DBName = databaseName

	return
}
