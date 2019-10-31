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

// GetDatasetByID returns the dataset given its id. The id is given as "user-id/dataset-id".
func (context Context) GetDatasetByID(id string) (result types.Dataset, err error) {

	// If the id is not given as user-id/dataset-id we assume user-id is the current user.
	ids := strings.Split(id, "/")
	if len(ids) == 1 {
		id = fmt.Sprintf("%s/%s", context.User.ID, id)
	}

	c := context.Session.DB(context.DBName).C("datasets")
	var allResults []types.Dataset

	// Only the root user can look up datasets other than their own.
	if context.User.IsRoot() {
		fmt.Println("### ROOT GETTING DATASETS ="+ id)
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
		fmt.Println("### HERE3")
		return
	}

	return allResults[0], nil
}

// GetDatasets lists all datasets given some filter criteria.
func (context Context) GetDatasets(
	filters F,
	limit int,
	cursor string,
	sortBy string,
	order string,
) (result []types.Dataset, cm types.CollectionMetadata, err error) {

	c := context.Session.DB(context.DBName).C("datasets")

	// Validate the parameters.
	if sortBy != "" &&
		sortBy != "id" &&
		sortBy != "user" &&
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
		case "user", "status", "source", "source-address":
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
			if bson.IsObjectIdHex(cursor) == false {
				err = errors.Wrap(ErrBadInput, "invalid cursor")
				return
			}

			var otherCursor interface{}
			switch sortBy {
			case "id", "user", "source", "source-address", "status":
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

	// We only apply limit to the database query if we don't need to do schema-based filtering.
	if limit > 0 && schInput == nil && schOutput == nil {
		q = q.Limit(limit)
	}

	// Collect the results.
	var allResults []types.Dataset
	err = q.All(&allResults)
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	// If there is a schema filter, then we do manual shema matching here.
	if schInput != nil || schOutput != nil {
		var allResultsFiltered []types.Dataset
		for i := range allResults {

			schInputSrc, err := deserializeSchema(allResults[i].SchemaIn)
			if err != nil {
				panic(err)
			}
			schOutputSrc, err := deserializeSchema(allResults[i].SchemaOut)
			if err != nil {
				panic(err)
			}

			var match = true
			if schInput != nil {
				match, _ = schInput.Match(schInputSrc, false)
			}
			if match && schOutput != nil {
				match, _ = schOutput.Match(schOutputSrc, false)
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

// CreateDataset adds a given dataset to the database.
func (context Context) CreateDataset(dataset types.Dataset) (result types.Dataset, err error) {

	// Perform validation of fields.
	ids := strings.Split(dataset.ID, "/")
	if len(ids) == 1 {
		dataset.ID = fmt.Sprintf("%s/%s", context.User.ID, dataset.ID)
	} else if len(ids) != 2 || ids[0] != context.User.ID {
		err = errors.Wrap(ErrBadInput, "the id must be of the format dataset-id or user-id/dataset-id")
		return
	}
	if dataset.Source != types.DatasetUpload && dataset.Source != types.DatasetLocal && dataset.Source != types.DatasetDownload && dataset.Source != types.DatasetGit {
		err = errors.Wrapf(ErrBadInput,
			"value of source can be \"%s\", \"%s\" or \"%s\", but found \"%s\"",
			types.DatasetUpload, types.DatasetLocal, types.DatasetDownload,types.DatasetGit, dataset.Source)
		return
	}
	// Validate the schemas.
	if dataset.SchemaIn != "" {
		_, err = deserializeSchema(dataset.SchemaIn)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "the given input schema definition is invalid")
			return
		}
		dataset.SchemaIn, err = jsonCompact(dataset.SchemaIn)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "json schema input compact error")
			return
		}
	}
	if dataset.SchemaOut != "" {
		_, err = deserializeSchema(dataset.SchemaOut)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "the given output schema definition is invalid")
			return
		}
		dataset.SchemaOut, err = jsonCompact(dataset.SchemaOut)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "json schema output compact error")
			return
		}
	}

	// Load schema structure.
	// If there are no errors, then it is valid.

	// Give default values to some fields.
	dataset.ObjectID = bson.NewObjectId()
	dataset.User = context.User.ID
	dataset.CreationTime = time.Now()
	dataset.Status = types.DatasetCreated

	c := context.Session.DB(context.DBName).C("datasets")
	err = c.Insert(dataset)
	if err != nil {
		lastError := err.(*mgo.LastError)
		if lastError.Code == 11000 {
			err = types.ErrIdentifierTaken
			return
		}
		err = errors.Wrap(err, "mongo insert failed")
		return
	}

	return dataset, nil

}

// UpdateDataset updates the information about a given dataset.
func (context Context) UpdateDataset(id string, updates map[string]interface{}) (result types.Dataset, err error) {

	// Perform validation of fields.
	ids := strings.Split(id, "/")
	if len(ids) == 1 {
		id = fmt.Sprintf("%s/%s", context.User.ID, id)
	} else if len(ids) != 2 {
		err = errors.Wrap(ErrBadInput, "the id must be of the format user-id/dataset-id")
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
		case "status":
			status := v.(string)

			if status != types.DatasetCreated &&
				status != types.DatasetTransferred &&
				status != types.DatasetUnpacked &&
				status != types.DatasetValidated &&
				status != types.DatasetArchived &&
				status != types.DatasetError {
				err = errors.Wrapf(ErrBadInput,
					"value of status can be \"%s\", \"%s\", \"%s\", \"%s\", \"%s\" or \"%s\", but found \"%s\"",
					types.DatasetCreated, types.DatasetTransferred, types.DatasetUnpacked, types.DatasetValidated, types.DatasetArchived, types.DatasetError, status)
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
		c := context.Session.DB(context.DBName).C("datasets")
		err = c.Update(bson.M{"id": id}, bson.M{"$set": valueUpdates})
		if err != nil {
			err = errors.Wrap(err, "mongo update failed")
			return
		}
	}

	// Get the updated dataset and update cache if needed.
	result, err = context.GetDatasetByID(id)
	if err != nil {
		err = errors.Wrap(err, "dataset get by ID failed")
		return
	}

	return

}

// LockDataset scans the available datasets (that are not currently locked), applies the specified filters,
// sorts them if specified and locks the first one by assigning it to the specified process.
func (context Context) LockDataset(
	filters F,
	processID bson.ObjectId,
	sortBy string,
	order string,
) (result types.Dataset, err error) {
	c := context.Session.DB(context.DBName).C("datasets")

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
		case "user", "status", "source", "source-address":
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

	var oneResult types.Dataset
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

// UnlockDataset releases the lock on a given dataset.
func (context Context) UnlockDataset(id string, processID bson.ObjectId) (err error) {

	// Perform validation of fields.
	ids := strings.Split(id, "/")
	if len(ids) != 2 {
		err = errors.Wrap(ErrBadInput, "the id must be of the format user-id/dataset-id")
		return
	} else if context.User.IsRoot() == false && ids[0] != context.User.ID {
		err = ErrNotFound
		return
	}

	c := context.Session.DB(context.DBName).C("datasets")
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

// UpdateDatasetStatus sets the status of the dataset and assigns the given status message.
func (context Context) UpdateDatasetStatus(id string, status string, statusMessage string) (err error) {
	_, err = context.UpdateDataset(id, F{"status": status, "status-message": statusMessage})
	return
}

// ReleaseDatasetLockByProcess releases all datasets that have been locked by a given process and
// are not in the error state.
func (context Context) ReleaseDatasetLockByProcess(processID bson.ObjectId) (numReleased int, err error) {

	c := context.Session.DB(context.DBName).C("datasets")
	var changeInfo *mgo.ChangeInfo
	changeInfo, err = c.UpdateAll(
		bson.M{"process": processID, "status": bson.M{"$ne": types.DatasetError}},
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
