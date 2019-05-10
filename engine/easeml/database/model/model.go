package model

import (
	e "errors"
	"regexp"

	"github.com/ds3lab/easeml/engine/easeml/database"
	"github.com/ds3lab/easeml/engine/easeml/database/model/types"

	"github.com/globalsign/mgo"
	"github.com/pkg/errors"
)

var (
	// ErrNotFound can be returend when we look for a specific single resource
	// by its ID but we get no result.
	ErrNotFound = e.New("requested resource not found")

	// ErrBadInput can be returned when a model function is given invalid parameters.
	ErrBadInput = e.New("the provided set of parameters is invalid")
)

// Context contains information needed to access the data model and authorize the acessor.
type Context struct {
	Session *mgo.Session
	DBName  string
	User    types.User
}

// Clone makes a copy of the mongo session.
func (context Context) Clone() (clonedContext Context) {
	clonedContext = context
	clonedContext.Session = context.Session.Copy()
	return
}

// AsRoot elevates the current model access context to root access privileges.
// Should be used only in special cases.
func (context Context) AsRoot() (rootContext Context) {
	context.User = types.User{ID: types.UserRoot}
	return context
}

// Connect establishes a connection with the database and returns a model context instance.
func Connect(dataSourceName string, databaseName string, anonimous bool) (context Context, err error) {

	// We wrap around the database.Connect because the model is already highly database specific.
	connection, err := database.Connect(dataSourceName, databaseName)
	if err != nil {
		err = errors.Wrap(err, "database connection failed")
		return
	}

	if anonimous {
		user := types.User{ID: types.UserAnon}
		context = Context{Session: connection.Session, DBName: connection.DBName, User: user}
	} else {
		user := types.User{ID: types.UserRoot}
		context = Context{Session: connection.Session, DBName: connection.DBName, User: user}
		user, err = context.GetUserByID(types.UserRoot)
		if err == nil {
			context.User = user
		} else if err == ErrNotFound {
			// Maybe the root user hasn't yet been created.
			err = nil
		}
	}

	return
}

// F represents a map of filters where keys are field names and values are equality constraints.
type F map[string]interface{}

// IDRegexp defines the pattern of all acceptable identifiers.
var IDRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// IDRegexpNegative defines all unacceptable characters in an identifier.
var IDRegexpNegative = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
