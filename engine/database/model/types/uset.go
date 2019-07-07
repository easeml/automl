package types

import (
	e "errors"

	"github.com/globalsign/mgo/bson"
)

var (
	// ErrWrongCredentials is returned when the wrong username or password are provided.
	ErrWrongCredentials = e.New("wrong username or password provided")

	// ErrWrongAPIKey is returned when the wrong API key is provided.
	ErrWrongAPIKey = e.New("wrong API key provided")

	// ErrNotPermitedForRoot is returned for actions that are not permitted to root users.
	ErrNotPermitedForRoot = e.New("this action is not permitted for the root user")

	// ErrNotPermitedForAnon is returned for actions that are not permitted to root users.
	ErrNotPermitedForAnon = e.New("this action is not permitted for anonimous users")

	// ErrUnauthorized is returned when a user tries to perform a forbidden action.
	ErrUnauthorized = e.New("this action is not permitted for the given user")

	// ErrIdentifierTaken is returned when we attempt to insert an item with an identifier that already exists.
	ErrIdentifierTaken = e.New("the specified identifier is already taken")
)

const (
	// UserRoot is the name of the root user. This user has no password, cannot log in or
	// log out and can only be authenticated with an API key.
	UserRoot = "root"

	// UserAnon is the user id assigned to unauthenticated users.
	UserAnon = "anonymous"

	// UserThis is the user id of the currently logged in user.
	UserThis = "this"
)

// User contains information about users.
type User struct {
	ObjectID     bson.ObjectId `bson:"_id"`
	ID           string        `bson:"id" json:"id"`
	Name         string        `bson:"name" json:"name"`
	Status       string        `bson:"status" json:"status"`
	PasswordHash string        `bson:"password-hash" json:"password,omitempty"`
	APIKey       string        `bson:"api-key" json:"-"`
}

// IsRoot returns true if the given user is the root user.
func (user User) IsRoot() bool {
	return user.ID == UserRoot
}

// IsAnon returns true if the given user is the anonymous user.
func (user User) IsAnon() bool {
	return user.ID == UserAnon
}

// GetAnonUser returns an anonymous user instance.
func GetAnonUser() User {
	return User{ID: UserAnon}
}
