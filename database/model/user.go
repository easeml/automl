package model

import (
	"github.com/ds3lab/easeml/database"
	"encoding/hex"
	e "errors"
	"strings"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
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

// UserAuthenticate checks the credentials of a given user against the users in the database.
// If the APIKey field contains a non-zero value, then an attempt to authenticate is made with
// that key. Otherwise, an attempt is made with the ID and PasswordHash fields.
func (context Context) UserAuthenticate(user User) (result User, err error) {

	var dbUser User
	var found bool
	var item interface{}

	// We always authenticate with API key when possible.
	if user.APIKey != "" {

		// Check if the user is cached.
		item, found = database.Cache.Get(user.APIKey)
		if found {
			id := item.(string)
			item, found = database.Cache.Get(id)
			if found {
				dbUser = item.(User)
			}

			if found == false || user.APIKey != dbUser.APIKey {
				// If the API key points to a user that is not cached or if it is invalid,
				// we will evict the API key from the cache.
				database.Cache.Delete(user.APIKey)
			}

			if dbUser.APIKey == "" {
				database.Cache.Delete(user.ID)
				found = false
			}
		}

		if found == false {

			// The user is not cached so we have to do a database lookup.
			dbUser, err = context.AsRoot().GetUserByAPIKey(user.APIKey)

			// If there was no error, we can cache the user.
			if err == nil {
				database.Cache.SetDefault(dbUser.ID, dbUser)
				database.Cache.SetDefault(dbUser.APIKey, dbUser.ID)
			}
		}

		// Here we check the vliditiy of the provided credentials.
		if errors.Cause(err) == ErrNotFound || user.APIKey != dbUser.APIKey {
			err = ErrWrongAPIKey
			return
		} else if err != nil {
			err = errors.Wrap(err, "user get by API key failed")
			return
		}

	} else {

		if user.IsRoot() {
			err = errors.Wrap(ErrWrongAPIKey, "root user must have API key to authenticate")
			return
		}

		item, found = database.Cache.Get(user.ID)
		if found {
			dbUser = item.(User)
			if user.PasswordHash != dbUser.PasswordHash {
				found = false
			}
		}
		if found == false {
			dbUser, err = context.AsRoot().GetUserByID(user.ID)

			// If there was no error, we can cache the user.
			if err == nil {
				database.Cache.SetDefault(dbUser.ID, dbUser)
				database.Cache.SetDefault(dbUser.APIKey, dbUser.ID)
			}
		}

		if errors.Cause(err) == ErrNotFound || user.PasswordHash != dbUser.PasswordHash {
			err = ErrWrongCredentials
			return
		} else if err != nil {
			err = errors.Wrap(err, "user get by ID failed")
			return
		}
	}

	return dbUser, nil
}

// UserGenerateAPIKey creates a new API key for the user and stores it in the database.
func (context Context) UserGenerateAPIKey() (result User, err error) {

	// Generate the new API Key.
	var apiKey uuid.UUID
	if apiKey, err = uuid.NewV4(); err != nil {
		err = errors.Wrap(err, "uuid new failed")
		return
	}

	// If there was an old API key, make sure it's removed from the cache.
	if context.User.APIKey != "" {
		database.Cache.Delete(context.User.APIKey)
	}

	// Assign the new API key.
	context.User.APIKey = apiKey.String()

	// Store the API key in the database.
	c := context.Session.DB(context.DBName).C("users")
	if err = c.Update(bson.M{"id": context.User.ID}, bson.M{"$set": bson.M{"api-key": context.User.APIKey}}); err != nil {
		err = errors.Wrap(err, "mongo update failed")
		return
	}

	// Add the API key to the cache.
	database.Cache.SetDefault(context.User.APIKey, context.User.ID)

	return context.User, nil
}

// UserDeleteAPIKey deletes the API key of the user.
func (context Context) UserDeleteAPIKey() (err error) {

	c := context.Session.DB(context.DBName).C("users")
	if err = c.Update(bson.M{"id": context.User.ID}, bson.M{"$set": bson.M{"api-key": ""}}); err != nil {
		err = errors.Wrap(err, "mongo update failed")
		return
	}

	// Delete the API key from the cache.
	database.Cache.Delete(context.User.APIKey)
	database.Cache.Delete(context.User.ID)

	return
}

// GetUserByID returns a user given its id.
func (context Context) GetUserByID(id string) (result User, err error) {

	// Only the root user can look up users other than self.
	if context.User.IsRoot() == false && id != context.User.ID {
		err = ErrNotFound
		return
	}

	c := context.Session.DB(context.DBName).C("users")
	var allResults []User
	err = c.Find(bson.M{"id": id}).All(&allResults)
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	if len(allResults) == 0 {
		err = ErrNotFound
		return
	}

	return allResults[0], nil
}

// GetUserByAPIKey looks up a user by its API key.
func (context Context) GetUserByAPIKey(apiKey string) (result User, err error) {

	c := context.Session.DB(context.DBName).C("users")
	var allResults []User
	err = c.Find(bson.M{"api-key": apiKey}).All(&allResults)
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	if len(allResults) == 0 {
		err = ErrNotFound
		return
	}

	// Only the root user can look up users other than self.
	if context.User.IsRoot() == false && allResults[0].ID != context.User.ID {
		err = ErrNotFound
		return
	}

	return allResults[0], nil
}

// GetUsers lists all users given some filter criteria.
func (context Context) GetUsers(
	filters F,
	limit int,
	cursor string,
	sortBy string,
	order string,
) (result []User, cm CollectionMetadata, err error) {

	c := context.Session.DB(context.DBName).C("users")

	// Validate the parameters.
	if sortBy != "" && sortBy != "id" && sortBy != "name" && sortBy != "status" {
		err = errors.Wrapf(ErrBadInput, "cannot sort by \"%s\"", sortBy)
		return
	}
	if order != "" && order != "asc" && order != "desc" {
		err = errors.Wrapf(ErrBadInput, "order can be either \"asc\" or \"desc\", not \"%s\"", order)
		return
	}
	if order == "" {
		order = "asc"
	}

	// If the user is not root then we need to limit access.
	query := bson.M{}
	if context.User.IsRoot() == false {
		query = bson.M{"id": bson.M{"$eq": context.User.ID}}
	}

	// Build a query given the parameters.
	for k, v := range filters {
		switch k {
		case "id":
			setDefault(&query, "id", bson.M{})
			query["id"].(bson.M)["$in"] = v.([]string)
		case "status":
			setDefault(&query, "status", bson.M{})
			query["status"].(bson.M)["$eq"] = v.(string)
		default:
			err = errors.Wrap(ErrBadInput, "invalid value of argument filters")
			return
		}
	}

	// We count the result size given the filters. This is before pagination.
	var resultSize int
	resultSize, err = c.Find(query).Count()
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	// If a cursor was specified then we have to do a range query.
	if cursor != "" {
		comparer := "$gt"
		if order == "desc" {
			comparer = "$lt"
		}

		// If there is no sorting then the cursor only points to the _id field.
		if sortBy != "" {
			splits := strings.Split(cursor, "-")
			cursor = splits[1]
			var otherCursor interface{}
			switch sortBy {
			case "id", "name", "status":
				var decoded []byte
				decoded, err = hex.DecodeString(splits[0])
				if err != nil {
					err = errors.Wrap(err, "hex decode string failed")
					return
				}
				otherCursor = string(decoded)
			}

			setDefault(&query, "$or", bson.M{})
			query["$or"] = []bson.M{
				bson.M{sortBy: bson.M{comparer: otherCursor}},
				bson.M{sortBy: bson.M{"$eq": otherCursor}, "_id": bson.M{comparer: bson.ObjectIdHex(cursor)}},
			}
		} else {
			if bson.IsObjectIdHex(cursor) == false {
				err = errors.Wrap(ErrBadInput, "invalid cursor")
				return
			}
			setDefault(&query, "_id", bson.M{})
			query["_id"].(bson.M)[comparer] = bson.ObjectIdHex(cursor)
		}
	}

	// Execute the query.
	q := c.Find(query)

	// We always sort by _id, but we may also sort by a specific field.
	if sortBy == "" {
		if order == "asc" {
			q = q.Sort("_id")
		} else {
			q = q.Sort("-_id")
		}
	} else {
		if order == "asc" {
			q = q.Sort(sortBy, "_id")
		} else {
			q = q.Sort("-"+sortBy, "-_id")
		}
	}

	if limit > 0 {
		q = q.Limit(limit)
	}

	// Collect the results.
	var allResults []User
	err = q.All(&allResults)
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	// Compute the next cursor.
	nextCursor := ""
	if limit > 0 && len(allResults) == limit {
		lastResult := allResults[len(allResults)-1]
		nextCursor = lastResult.ObjectID.Hex()

		if sortBy != "" {
			var encoded string
			switch sortBy {
			case "id":
				encoded = hex.EncodeToString([]byte(lastResult.ID))
			case "name":
				encoded = hex.EncodeToString([]byte(lastResult.Name))
			case "status":
				encoded = hex.EncodeToString([]byte(lastResult.Status))
			}
			nextCursor = encoded + "-" + nextCursor
		}
	}

	// Assemble the results.
	result = allResults
	cm = CollectionMetadata{
		TotalResultSize:    resultSize,
		ReturnedResultSize: len(result),
		NextPageCursor:     nextCursor,
	}
	return
}

// CreateUser adds the given user to the database.
func (context Context) CreateUser(user User) (result User, err error) {

	// This action is only permitted for the root user.
	if context.User.IsRoot() == false {
		err = ErrUnauthorized
		return
	}

	// Check for bad inputs.
	if user.ID == UserRoot || user.ID == UserAnon || user.ID == UserThis {
		err = errors.Wrapf(ErrBadInput, "value of user ID cannot be %s, %s or %s", UserRoot, UserAnon, UserThis)
		return
	}
	if user.ID == "" {
		err = errors.Wrapf(ErrBadInput, "value of ID cannot be empty")
		return
	}
	if user.PasswordHash == "" {
		err = errors.Wrapf(ErrBadInput, "value of password hash cannot be empty")
		return
	}
	if user.Status == "" {
		user.Status = "active"
	}
	if user.Status != "active" && user.Status != "archived" {
		err = errors.Wrapf(ErrBadInput,
			"value of status can be \"active\" or \"archived\", but found \"%s\"", user.Status)
		return
	}

	user.ObjectID = bson.NewObjectId()
	user.APIKey = ""

	c := context.Session.DB(context.DBName).C("users")
	err = c.Insert(user)
	if err != nil {
		lastError := err.(*mgo.LastError)
		if lastError.Code == 11000 {
			err = ErrIdentifierTaken
			return
		}
		err = errors.Wrap(err, "mongo insert failed")
		return
	}

	return user, nil
}

// UpdateUser updates the information about a given user.
func (context Context) UpdateUser(id string, updates map[string]interface{}) (result User, err error) {

	// If the user is not root, then they can only update information about themselves.
	if context.User.IsRoot() == false && context.User.ID != id {
		err = ErrNotFound
		return
	}

	// Build the update document. Validate values.
	valueUpdates := bson.M{}
	for k, v := range updates {
		switch k {
		case "name":
			valueUpdates["name"] = v.(string)
		case "status":
			valueUpdates["status"] = v.(string)
			if valueUpdates["status"] != "active" && valueUpdates["status"] != "archived" {
				err = errors.Wrapf(ErrBadInput,
					"value of status can be \"active\" or \"archived\", but found \"%s\"", valueUpdates["status"])
				return
			}
		case "password":
			valueUpdates["password-hash"] = v.(string)
			if valueUpdates["password-hash"] == "" {
				err = errors.Wrapf(ErrBadInput, "value of password hash cannot be empty")
				return
			}
		default:
			err = errors.Wrap(ErrBadInput, "invalid value of parameter updates")
			return
		}
	}

	// If there were no updates, then we can skip this step.
	if len(valueUpdates) > 0 {
		c := context.Session.DB(context.DBName).C("users")
		err = c.Update(bson.M{"id": id}, bson.M{"$set": valueUpdates})
		if err != nil {
			err = errors.Wrap(err, "mongo update failed")
			return
		}
	}

	// Get the updated user and update cache if needed.
	result, err = context.GetUserByID(id)
	if err != nil {
		err = errors.Wrap(err, "user get by ID failed")
		return
	}
	if _, ok := database.Cache.Get(id); ok {
		database.Cache.SetDefault(id, result)
	}

	return
}

// UserLogin logs in the user from the context. It is assumed that the user is already authenticated.
func (context Context) UserLogin() (result User, err error) {

	// NOTE: Removed this. Access control for this should be implemented as middleware.
	// The root user cannot log in.
	// if context.User.IsRoot() {
	//	  err = ErrNotPermitedForRoot
	// 	  return
	// }

	if context.User.IsAnon() {
		err = ErrNotPermitedForAnon
		return
	}

	// Generate API key and save it in the database and cache.
	user := context.User
	if context.User.APIKey == "" {

		// Make sure the user hasn't logged in.
		if user, err = context.GetUserByID(user.ID); err != nil {
			err = errors.Wrap(err, "get user by ID failed")
			return
		}

		// If the user has not logged in (API key missing even in the database) then generate a new API key.
		if user.APIKey == "" {
			if user, err = context.UserGenerateAPIKey(); err != nil {
				err = errors.Wrap(err, "user generate API key failed")
				return
			}
		}
	}

	return user, nil
}

// UserLogout logs the user out and invalidates their API key.
func (context Context) UserLogout() (err error) {

	// NOTE: Removed this. Access control for this should be implemented as middleware.
	// The root user cannot log out.
	// if context.User.IsRoot() {
	// 	return ErrNotPermitedForRoot
	// }

	if context.User.IsAnon() {
		err = ErrNotPermitedForAnon
		return
	}

	return context.UserDeleteAPIKey()
}
