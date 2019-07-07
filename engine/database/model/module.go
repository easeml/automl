package model

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ds3lab/easeml/engine/database/model/types"
	sch "github.com/ds3lab/easeml/schema/go/easemlschema/schema"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

// GetModuleByID returns the module given its id. The id is given as "user-id/module-id".
func (context Context) GetModuleByID(id string) (result types.Module, err error) {

	// If the id is not given as user-id/module-id we assume user-id is the current user.
	ids := strings.Split(id, "/")
	if len(ids) == 1 {
		id = fmt.Sprintf("%s/%s", context.User.ID, id)
	}

	c := context.Session.DB(context.DBName).C("modules")
	var allResults []types.Module

	// Only the root user can look up modules other than their own.
	if context.User.IsRoot() {
		err = c.Find(bson.M{"id": id}).All(&allResults)
	} else {
		err = c.Find(bson.M{"id": id, "user": bson.M{"$in": []string{context.User.ID, types.UserRoot}}}).All(&allResults)
	}

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

// GetModules lists all modules given some filter criteria.
func (context Context) GetModules(
	filters F,
	limit int,
	cursor string,
	sortBy string,
	order string,
) (result []types.Module, cm types.CollectionMetadata, err error) {

	c := context.Session.DB(context.DBName).C("modules")

	// Validate the parameters.
	if sortBy != "" &&
		sortBy != "id" &&
		sortBy != "user" &&
		sortBy != "type" &&
		sortBy != "label" &&
		sortBy != "source" &&
		sortBy != "source-address" &&
		sortBy != "creation-time" &&
		sortBy != "status" {
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
		query = bson.M{"user": bson.M{"$in": []string{context.User.ID, types.UserRoot}}}
	}

	// Schema placeholders in case we want to do schema matching.
	var schInput, schOutput *sch.Schema

	// Build a query given the parameters.
	for k, v := range filters {
		switch k {
		case "id":
			setDefault(&query, "id", bson.M{})
			query["id"].(bson.M)["$in"] = v.([]string)
		case "user", "type", "label", "status", "source", "source-address":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
		case "schema-in":
			schInput, err = deserializeSchema(v.(string))
			if err != nil {
				err = errors.Wrap(ErrBadInput, "the given input schema definition is invalid")
				return
			}
		case "schema-out":
			schOutput, err = deserializeSchema(v.(string))
			if err != nil {
				err = errors.Wrap(ErrBadInput, "the given output schema definition is invalid")
				return
			}
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
			var decoded []byte
			decoded, err = hex.DecodeString(splits[0])
			if err != nil {
				err = errors.Wrap(err, "hex decode string failed")
				return
			}
			var otherCursor interface{}
			switch sortBy {
			case "id", "user", "type", "label", "source", "source-address", "status":
				otherCursor = string(decoded)
			case "creation-time":
				var t time.Time
				t.GobDecode(decoded)
				otherCursor = t
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

	if limit > 0 && schInput == nil && schOutput == nil {
		q = q.Limit(limit)
	}

	// Collect the results.
	var allResults []types.Module
	err = q.All(&allResults)
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	// If there is a schema filter, then we do manual shema matching here.
	if schInput != nil || schOutput != nil {
		var allResultsFiltered []types.Module
		for i := range allResults {

			schInputDst, err := deserializeSchema(allResults[i].SchemaIn)
			if err != nil {
				panic(err)
			}
			schOutputDst, err := deserializeSchema(allResults[i].SchemaOut)
			if err != nil {
				panic(err)
			}

			var match = true
			if schInput != nil {
				match, _ = schInputDst.Match(schInput, false)
			}
			if match && schOutput != nil {
				match, _ = schOutputDst.Match(schOutput, false)
			}
			if match {
				allResultsFiltered = append(allResultsFiltered, allResults[i])
			}
			if limit > 0 && len(allResultsFiltered) >= limit {
				break
			}
		}
		allResults = allResultsFiltered
	}

	// Compute the next cursor.
	nextCursor := ""
	if limit > 0 && len(allResults) == limit {
		lastResult := allResults[len(allResults)-1]
		nextCursor = lastResult.ObjectID.Hex()

		if sortBy != "" {
			var encoded string
			var b []byte
			switch sortBy {
			case "id":
				b = []byte(lastResult.ID)
			case "user":
				b = []byte(lastResult.User)
			case "type":
				b = []byte(lastResult.Type)
			case "label":
				b = []byte(lastResult.Label)
			case "source":
				b = []byte(lastResult.Source)
			case "source-address":
				b = []byte(lastResult.SourceAddress)
			case "creation-time":
				b, err = lastResult.CreationTime.GobEncode()
			case "status":
				b = []byte(lastResult.Status)
			}
			encoded = hex.EncodeToString(b)
			nextCursor = encoded + "-" + nextCursor
		}
	}

	// Assemble the results.
	result = allResults
	cm = types.CollectionMetadata{
		TotalResultSize:    resultSize,
		ReturnedResultSize: len(result),
		NextPageCursor:     nextCursor,
	}
	return

}

// CreateModule adds a given module to the database.
func (context Context) CreateModule(module types.Module) (result types.Module, err error) {

	// Perform validation of fields.
	ids := strings.Split(module.ID, "/")
	if len(ids) == 1 {
		module.ID = fmt.Sprintf("%s/%s", context.User.ID, module.ID)
	} else if len(ids) != 2 || ids[0] != context.User.ID {
		err = errors.Wrap(ErrBadInput, "the id must be of the format module-id or user-id/module-id")
		return
	}
	if module.Source != types.ModuleUpload &&
		module.Source != types.ModuleLocal &&
		module.Source != types.ModuleRegistry &&
		module.Source != types.ModuleDownload {
		err = errors.Wrapf(ErrBadInput,
			"value of source can be \"%s\", \"%s\", \"%s\" or \"%s\", but found \"%s\"",
			types.ModuleUpload, types.ModuleLocal, types.ModuleDownload, types.ModuleRegistry, module.Source)
		return
	}
	if module.Type != types.ModuleModel &&
		module.Type != types.ModuleObjective &&
		module.Type != types.ModuleOptimizer {
		err = errors.Wrapf(ErrBadInput,
			"value of type can be \"%s\", \"%s\" or \"%s\", but found \"%s\"",
			types.ModuleModel, types.ModuleObjective, types.ModuleOptimizer, module.Type)
		return
	}
	// Validate the schemas.
	if module.SchemaIn != "" {
		_, err = deserializeSchema(module.SchemaIn)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "the given input schema definition is invalid")
			return
		}
		module.SchemaIn, err = jsonCompact(module.SchemaIn)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "json schema input compact error")
			return
		}
	}
	if module.SchemaOut != "" {
		_, err = deserializeSchema(module.SchemaOut)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "the given output schema definition is invalid")
			return
		}
		module.SchemaOut, err = jsonCompact(module.SchemaOut)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "json schema output compact error")
			return
		}
	}

	// Give default values to some fields.
	module.ObjectID = bson.NewObjectId()
	module.User = context.User.ID
	module.CreationTime = time.Now()
	module.Status = types.ModuleCreated

	c := context.Session.DB(context.DBName).C("modules")
	err = c.Insert(module)
	if err != nil {
		lastError := err.(*mgo.LastError)
		if lastError.Code == 11000 {
			err = types.ErrIdentifierTaken
			return
		}
		err = errors.Wrap(err, "mongo insert failed")
		return
	}

	return module, nil

}

// UpdateModule updates the information about a given module.
func (context Context) UpdateModule(id string, updates map[string]interface{}) (result types.Module, err error) {

	// Perform validation of fields.
	ids := strings.Split(id, "/")
	if len(ids) != 2 {
		err = errors.Wrap(ErrBadInput, "the id must be of the format user-id/module-id")
		return
	} else if context.User.IsRoot() == false && ids[0] != context.User.ID {
		err = ErrNotFound
		return
	}

	// Build the update document. Validate values.
	valueUpdates := bson.M{}
	for k, v := range updates {
		switch k {
		case "name":
			valueUpdates["name"] = v.(string)
		case "description":
			valueUpdates["description"] = v.(string)
		case "schema-in":
			schemaString := v.(string)
			if schemaString != "" {
				_, err = deserializeSchema(schemaString)
				if err != nil {
					err = errors.Wrap(ErrBadInput, "the given input schema definition is invalid")
					return
				}
				schemaString, err = jsonCompact(schemaString)
				if err != nil {
					err = errors.Wrap(ErrBadInput, "json schema input compact error")
					return
				}
			}
			valueUpdates["schema-in"] = schemaString
		case "schema-out":
			schemaString := v.(string)
			if schemaString != "" {
				_, err = deserializeSchema(schemaString)
				if err != nil {
					err = errors.Wrap(ErrBadInput, "the given output schema definition is invalid")
					return
				}
				schemaString, err = jsonCompact(schemaString)
				if err != nil {
					err = errors.Wrap(ErrBadInput, "json schema output compact error")
					return
				}
				valueUpdates["schema-out"] = schemaString
			}
		case "config-space":
			valueUpdates["config-space"] = v.(string)
		case "status":
			status := v.(string)

			if status != types.ModuleCreated &&
				status != types.ModuleTransferred &&
				status != types.ModuleActive &&
				status != types.ModuleArchived &&
				status != types.ModuleError {
				err = errors.Wrapf(ErrBadInput,
					"value of status can be \"%s\", \"%s\", \"%s\", \"%s\" or \"%s\", but found \"%s\"",
					types.ModuleCreated, types.ModuleTransferred, types.ModuleActive, types.ModuleArchived, types.ModuleError, status)
				return
			}
			valueUpdates["status"] = status

		case "status-message":
			valueUpdates["status-message"] = v.(string)

		default:
			err = errors.Wrap(ErrBadInput, "invalid value of parameter updates")
			return
		}
	}

	// If there were no updates, then we can skip this step.
	if len(valueUpdates) > 0 {
		c := context.Session.DB(context.DBName).C("modules")
		err = c.Update(bson.M{"id": id}, bson.M{"$set": valueUpdates})
		if err != nil {
			err = errors.Wrap(err, "mongo update failed")
			return
		}
	}

	// Get the updated module and update cache if needed.
	result, err = context.GetModuleByID(id)
	if err != nil {
		err = errors.Wrap(err, "module get by ID failed")
		return
	}

	return

}

// LockModule scans the available modules (that are not currently locked), applies the specified filters,
// sorts them if specified and locks the first one by assigning it to the specified process.
func (context Context) LockModule(
	filters F,
	processID bson.ObjectId,
	sortBy string,
	order string,
) (result types.Module, err error) {
	c := context.Session.DB(context.DBName).C("modules")

	// We are looking only for instances that are not already locked.
	query := bson.M{"process": nil}

	// If the user is not root then we need to limit access.
	if context.User.IsRoot() == false {
		query = bson.M{"user": bson.M{"$in": []string{context.User.ID, types.UserRoot}}}
	}

	// Build a query given the parameters.
	for k, v := range filters {
		switch k {
		case "id":
			setDefault(&query, "id", bson.M{})
			query["id"].(bson.M)["$in"] = v.([]string)
		case "user", "type", "status", "source", "source-address":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
		default:
			err = errors.Wrap(ErrBadInput, "invalid value of argument filters")
			return
		}
	}

	// Build the query.
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

	q = q.Limit(1)

	change := mgo.Change{Update: bson.M{"$set": bson.M{"process": processID}}, ReturnNew: false}

	var oneResult types.Module
	var changeInfo *mgo.ChangeInfo
	changeInfo, err = q.Apply(change, &oneResult)
	if err == mgo.ErrNotFound || changeInfo.Updated < 1 {
		err = ErrNotFound
		return
	} else if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	} else if changeInfo.Updated > 1 {
		// Fail safe. This should never happen.
		panic(changeInfo)
	}

	return oneResult, nil
}

// UnlockModule releases the lock on a given module.
func (context Context) UnlockModule(id string, processID bson.ObjectId) (err error) {

	// Perform validation of fields.
	ids := strings.Split(id, "/")
	if len(ids) != 2 {
		err = errors.Wrap(ErrBadInput, "the id must be of the format user-id/module-id")
		return
	} else if context.User.IsRoot() == false && ids[0] != context.User.ID {
		err = ErrNotFound
		return
	}

	c := context.Session.DB(context.DBName).C("modules")
	err = c.Update(bson.M{"id": id, "process": processID}, bson.M{"$set": bson.M{"process": nil}})
	if err == mgo.ErrNotFound {
		err = ErrNotFound
		return
	} else if err != nil {
		err = errors.Wrap(err, "mongo update failed")
		return
	}

	return
}

// UpdateModuleStatus sets the status of the module and assigns the given status message.
func (context Context) UpdateModuleStatus(id string, status string, statusMessage string) (err error) {
	_, err = context.UpdateModule(id, F{"status": status, "status-message": statusMessage})
	return
}

// ReleaseModuleLockByProcess releases all modules that have been locked by a given process and
// are not in the error state.
func (context Context) ReleaseModuleLockByProcess(processID bson.ObjectId) (numReleased int, err error) {

	c := context.Session.DB(context.DBName).C("modules")
	var changeInfo *mgo.ChangeInfo
	changeInfo, err = c.UpdateAll(
		bson.M{"process": processID, "status": bson.M{"$ne": types.ModuleError}},
		bson.M{"$set": bson.M{"process": nil}},
	)
	if err == mgo.ErrNotFound {
		err = ErrNotFound
		return
	} else if err != nil {
		err = errors.Wrap(err, "mongo update failed")
		return
	}

	return changeInfo.Updated, nil
}
