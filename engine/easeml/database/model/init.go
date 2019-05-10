package model

import (
	"reflect"
	"strings"

	"github.com/ds3lab/easeml/engine/easeml/database/model/types"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

// Clear drops the specified database if it exists.
func (context Context) Clear(databaseName string) (err error) {

	// To be thread safe, always create a copy of the session.
	//sess := database.Session.Copy()
	//defer sess.Close()

	// Check if the database exists and if yes, then drop it.
	names, err := context.Session.DatabaseNames()
	for i := range names {
		if names[i] == databaseName {
			err = context.Session.DB(databaseName).DropDatabase()
			if err != nil {
				err = errors.Wrap(err, "mongo drop database failed")
			}
		}
	}

	return
}

// Initialize ensures that the target database is properly initalized.
func (context Context) Initialize(databaseName string) (err error) {

	// Databases and collections are created implicitly upon usage.
	db := context.Session.DB(databaseName)

	// All indexes to create.
	var requiredIndexes = map[string][]mgo.Index{
		"users": []mgo.Index{
			mgo.Index{Key: []string{"id"}, Unique: true},
			mgo.Index{Key: []string{"status"}},
		},
		"processes": []mgo.Index{
			mgo.Index{Key: []string{"type"}},
			mgo.Index{Key: []string{"status"}},
			mgo.Index{Key: []string{"last-keepalive"}},
		},
		"datasets": []mgo.Index{
			mgo.Index{Key: []string{"id"}, Unique: true},
			mgo.Index{Key: []string{"user"}},
			mgo.Index{Key: []string{"source"}},
			mgo.Index{Key: []string{"status"}},
			mgo.Index{Key: []string{"process"}},
			mgo.Index{Key: []string{"creation-time"}},
		},
		"modules": []mgo.Index{
			mgo.Index{Key: []string{"id"}, Unique: true},
			mgo.Index{Key: []string{"user"}},
			mgo.Index{Key: []string{"type"}},
			mgo.Index{Key: []string{"status"}},
			mgo.Index{Key: []string{"process"}},
		},
		"jobs": []mgo.Index{
			mgo.Index{Key: []string{"user"}},
			mgo.Index{Key: []string{"dataset"}},
			mgo.Index{Key: []string{"models"}},
			mgo.Index{Key: []string{"objective"}},
			mgo.Index{Key: []string{"status"}},
			mgo.Index{Key: []string{"process"}},
		},
		"tasks": []mgo.Index{
			mgo.Index{Key: []string{"id"}, Unique: true},
			mgo.Index{Key: []string{"job"}},
			mgo.Index{Key: []string{"process"}},
			mgo.Index{Key: []string{"model"}},
			mgo.Index{Key: []string{"objective"}},
			mgo.Index{Key: []string{"dataset"}},
			mgo.Index{Key: []string{"user"}},
			mgo.Index{Key: []string{"status"}},
			mgo.Index{Key: []string{"stage"}},
		},
	}

	// Get list of all collections to see which ones we need to create.
	var collectionNames []string
	collectionNames, err = db.CollectionNames()
	if err != nil {
		return errors.Wrap(err, "mongo get collection names failed")
	}

	// Ensure all indices exist.
	for collection := range requiredIndexes {

		// Ensure the collection exists.
		var found bool
		for i := range collectionNames {
			if collection == collectionNames[i] {
				found = true
				break
			}
		}
		if found == false {
			db.C(collection).Create(&mgo.CollectionInfo{})
		}

		// Get list of indexes to see which ones we need to create.
		var encounteredIndexes []mgo.Index
		encounteredIndexes, err = db.C(collection).Indexes()
		if err != nil {
			return errors.Wrap(err, "mongo get indices failed")
		}

		collectionRequiredIndexes := requiredIndexes[collection]

		for i := range collectionRequiredIndexes {
			var found bool
			for j := range encounteredIndexes {
				if reflect.DeepEqual(collectionRequiredIndexes[i].Key, encounteredIndexes[j].Key) {

					// If the indexes are not the same, we will have to drop the existing one.
					if collectionRequiredIndexes[i].Unique != encounteredIndexes[j].Unique {
						db.C(collection).DropIndexName(encounteredIndexes[j].Name)
					} else {
						found = true
					}

					break

				}
			}

			// If this index wasn't found or was dropped, we will have to create it.
			if found == false {
				err = db.C(collection).EnsureIndex(collectionRequiredIndexes[i])
				if err != nil {
					err = errors.Wrapf(err, "mongo ensure index for key %s in collection %s failed",
						strings.Join(collectionRequiredIndexes[i].Key, ","), collection)
					return err
				}
			}
		}
	}

	// Create the root user if needed.
	_, err = context.GetUserByID(types.UserRoot)
	if err == ErrNotFound {
		c := context.Session.DB(databaseName).C("users")
		err = c.Insert(types.User{ObjectID: bson.NewObjectId(), ID: types.UserRoot, Name: "Admin User", Status: "active"})
		if err != nil {
			// If it is a duplicate, then this was a simple race condition and a user was created in the meantime.
			// Since the same code is creating the user, we can simply ignore.
			if mgo.IsDup(err) == false {
				return errors.Wrap(err, "mongo insert failed")
			}
		}
	} else if err != nil {
		return errors.Wrap(err, "user get by ID failed")
	}

	return nil
}
