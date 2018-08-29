package model

import (
	"github.com/ds3lab/easeml/database"
	e "errors"
	"regexp"
	"time"

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
	User    User
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
	context.User = User{ID: UserRoot}
	return context
}

// CollectionMetadata contains additional information about a query that returns
// arrays as results. This information aids the caller with navigating the partial results.
type CollectionMetadata struct {
	// The total number of items in a collection that is being accessed. - PROBABLY USELESS
	// TotalCollectionSize int

	// The total size of the result after applying query filters but before pagination.
	TotalResultSize int `json:"total-result-size"`

	// The size of the current page of results that is being returned.
	ReturnedResultSize int `json:"returned-result-size"`

	// The string to pass as a cursor to obtain the next page of results.
	NextPageCursor string `json:"next-page-cursor"`
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
		user := User{ID: UserAnon}
		context = Context{Session: connection.Session, DBName: connection.DBName, User: user}
	} else {
		user := User{ID: UserRoot}
		context = Context{Session: connection.Session, DBName: connection.DBName, User: user}
		user, err = context.GetUserByID(UserRoot)
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

// TimeInterval represents a time interval with specific start and end times.
type TimeInterval struct {
	Start time.Time `bson:"start" json:"start"`
	End   time.Time `bson:"end" json:"end"`
}

// IDRegexp defines the pattern of all acceptable identifiers.
var IDRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// IDRegexpNegative defines all unacceptable characters in an identifier.
var IDRegexpNegative = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
