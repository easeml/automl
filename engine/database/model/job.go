package model

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ds3lab/easeml/engine/database/model/types"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

// GetJobByID returns the job given its id. The id is given as "user-id/job-id".
func (context Context) GetJobByID(id bson.ObjectId) (result types.Job, err error) {

	c := context.Session.DB(context.DBName).C("jobs")
	var allResults []types.Job

	// Only the root user can look up jobs other than their own.
	if context.User.IsRoot() {
		err = c.Find(bson.M{"_id": id}).All(&allResults)
	} else {
		err = c.Find(bson.M{"_id": id, "user": bson.M{"$in": []string{context.User.ID, types.UserRoot}}}).All(&allResults)
	}

	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	if len(allResults) == 0 {
		err = ErrNotFound
		return
	}

	result = allResults[0]

	// Update computed fields.
	result.PauseDuration = result.GetPauseDuration()
	result.RunningDuration = result.GetRunningDuration() - result.PauseDuration

	return result, nil
}

// GetJobs lists all jobs given some filter criteria.
func (context Context) GetJobs(
	filters F,
	limit int,
	cursor string,
	sortBy string,
	order string,
) (result []types.Job, cm types.CollectionMetadata, err error) {

	c := context.Session.DB(context.DBName).C("jobs")

	// Validate the parameters.
	if sortBy != "" &&
		sortBy != "user" &&
		sortBy != "dataset" &&
		sortBy != "objective" &&
		sortBy != "creation-time" &&
		sortBy != "running-time-start" &&
		sortBy != "running-time-end" &&
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

	// Build a query given the parameters.
	for k, v := range filters {
		switch k {
		case "id":
			setDefault(&query, "_id", bson.M{})
			query["_id"].(bson.M)["$in"] = v.([]bson.ObjectId)
		case "user", "dataset", "objective", "status":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
		case "accept-new-models":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(bool)
		case "model":
			setDefault(&query, "models", bson.M{})
			query["models"].(bson.M)["$elemMatch"] = bson.M{"$eq": v.(string)}
		case "alt-objective":
			setDefault(&query, "alt-objectives", bson.M{})
			query["alt-objectives"].(bson.M)["$elemMatch"] = bson.M{"$eq": v.(string)}
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
			case "user", "dataset", "objective", "status":
				otherCursor = string(decoded)
			case "creation-time", "running-time-start", "running-time-end":
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

	if limit > 0 {
		q = q.Limit(limit)
	}

	// Collect the results.
	var allResults []types.Job
	err = q.All(&allResults)
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	// Update computed fields.
	for i := range allResults {
		allResults[i].PauseDuration = allResults[i].GetPauseDuration()
		allResults[i].RunningDuration = allResults[i].GetRunningDuration() - allResults[i].PauseDuration
	}

	// Compute the next cursor.
	nextCursor := ""
	if limit > 0 && len(allResults) == limit {
		lastResult := allResults[len(allResults)-1]
		nextCursor = lastResult.ID.Hex()

		if sortBy != "" {
			var encoded string
			var b []byte
			switch sortBy {
			case "user":
				b = []byte(lastResult.User)
			case "dataset":
				b = []byte(lastResult.Dataset)
			case "objective":
				b = []byte(lastResult.Objective)
			case "creation-time":
				b, err = lastResult.CreationTime.GobEncode()
			case "running-time-start":
				b, err = lastResult.RunningTime.Start.GobEncode()
			case "running-time-end":
				b, err = lastResult.RunningTime.End.GobEncode()
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

// CountJobs is the same as GetJobs but returns only the count, not the actual tasks.
func (context Context) CountJobs(filters F) (count int, err error) {

	c := context.Session.DB(context.DBName).C("jobs")

	// If the user is not root then we need to limit access.
	query := bson.M{}
	if context.User.IsRoot() == false {
		query = bson.M{"user": bson.M{"$in": []string{context.User.ID, types.UserRoot}}}
	}

	// Build a query given the parameters.
	for k, v := range filters {
		switch k {
		case "id":
			setDefault(&query, "_id", bson.M{})
			query["_id"].(bson.M)["$in"] = v.([]bson.ObjectId)
		case "user", "dataset", "objective", "status":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
		case "accept-new-models":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(bool)
		case "model":
			setDefault(&query, "models", bson.M{})
			query["models"].(bson.M)["$elemMatch"] = bson.M{"$eq": v.(string)}
		case "alt-objective":
			setDefault(&query, "alt-objectives", bson.M{})
			query["alt-objectives"].(bson.M)["$elemMatch"] = bson.M{"$eq": v.(string)}
		default:
			err = errors.Wrap(ErrBadInput, "invalid value of argument filters")
			return
		}
	}

	// We count the number of jobs that satisfy the filter criteria.
	count, err = c.Find(query).Count()
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
	}

	return
}

// CreateJob adds a given job to the database.
func (context Context) CreateJob(job types.Job) (result types.Job, err error) {

	// Validate that the dataset exists and is active.
	var dataset types.Dataset
	dataset, err = context.GetDatasetByID(job.Dataset)
	if err != nil && err != ErrNotFound {
		err = errors.Wrap(err, "error while trying to access the referenced dataset")
		return
	} else if err == ErrNotFound || dataset.Status != types.DatasetValidated {
		err = errors.Wrapf(ErrBadInput,
			"the referenced objective \"%s\" does not exist or is not verified and active", job.Dataset)
	}

	// Validate Pipeline
	// TODO There should be no hardcoded stages
	// Validate if stages are known
	evaluate:=false
	for _,element := range job.Pipeline {
		if element == types.TaskStageEvaluate{
			evaluate=true
		}
		if !types.AllPipelineElements.Has(element) {
			err = errors.Wrapf(ErrBadInput,
				"Pipeline element \"%s\" is not recognized", element)
			return
		}
	}

	// Give default values to some fields.
	job.ID = bson.NewObjectId()
	job.User = context.User.ID
	job.CreationTime = time.Now()
	job.Status = types.JobScheduled
	job.PauseDuration = 0
	job.RunningDuration = 0 // This field will be omitted when empty.

	// TODO Super Ugly hardcoded pipelines
	if len(job.TaskIds) > 0{
		// Pre trained job
		err = nil
		if len(job.Pipeline) == 0{
			// Default trained pipeline
			job.Pipeline = []string{types.TaskStagePredict,types.TaskStageEvaluate}
		} else {
			var currentState = types.StageMapType{types.TaskStageTrain:{}}
			for _,element := range job.Pipeline {
				currentState[element]=struct{}{}
				preRequisites:=types.AllPreRequisites[element]
				for _,stage := range preRequisites{
					if !currentState.Has(stage){
						err = errors.Wrapf(ErrBadInput,
							"Pipeline does not satisfy prerequisites \"%s\" for element \"%s\"",
							preRequisites,element)
						return
					}
				}
			}
		}

	}else{
		// Untrained job
		if len(job.Pipeline) == 0{
			// Default trainable pipeline
			job.Pipeline = []string{types.TaskStageTrain,types.TaskStagePredict,types.TaskStageEvaluate}
		}else {
			var currentState = types.StageMapType{}
			for _,element := range job.Pipeline {
				currentState[element]=struct{}{}
				preRequisites:=types.AllPreRequisites[element]
				for _,stage := range preRequisites{
					if !currentState.Has(stage){
						err = errors.Wrapf(ErrBadInput,
							"Pipeline does not satisfy prerequisites \"%s\" for element \"%s\"",
							preRequisites,element)
						return
					}
				}
			}
		}

		// Validate that the models exist and are active.
		if len(job.Models) > 0 {
			var models []types.Module
			models, _, err = context.GetModules(F{"id": job.Models}, 0, "", "", "")
			if err != nil {
				err = errors.Wrap(err, "error while trying to access the referenced models")
				return
			}
			for i := range job.Models {
				var found bool
				for j := range models {
					if job.Models[i] == models[j].ID && models[j].Status == types.ModuleActive {
						found = true
						break
					}
				}
				if found == false {
					err = errors.Wrapf(ErrBadInput,
						"the referenced model \"%s\" does not exist or is not active", job.Models[i])
				}
			}
		} else {
			err = errors.Wrap(err, "error no models provided")
			return
		}

		job.ConfigSpace, err = context.GetJobConfigSpace(job)
		if err != nil {
			err = errors.Wrap(err, "error while trying to construct job config space")
			return
		}
	}



	if evaluate{
		// Validate that the objective exists and is active.
		var objective types.Module
		objective, err = context.GetModuleByID(job.Objective)
		if err != nil && err != ErrNotFound {
			err = errors.Wrap(err, "error while trying to access the referenced objective")
			return
		} else if err == ErrNotFound || objective.Status != types.ModuleActive {
			err = errors.Wrapf(ErrBadInput,
				"the referenced objective \"%s\" does not exist or is not active", job.Objective)
		}

		// Validate that the alternative objectives exist and are active.
		if len(job.AltObjectives) > 0 {
			var altObjectives []types.Module
			altObjectives, _, err = context.GetModules(F{"id": job.AltObjectives}, 0, "", "", "")
			if err != nil {
				err = errors.Wrap(err, "error while trying to access the referenced alternative objectives")
				return
			}
			for i := range job.AltObjectives {
				var found bool
				for j := range altObjectives {
					if job.AltObjectives[i] == altObjectives[j].ID && altObjectives[j].Status == types.ModuleActive {
						found = true
						break
					}
				}
				if found == false {
					err = errors.Wrapf(ErrBadInput,
						"the referenced alternative objective \"%s\" does not exist or is active", job.AltObjectives[i])
				}
			}
		}
	}

	// Immediately put the job to the running state. Is this correct?
	job.Status = types.JobRunning
	job.RunningTime.Start = time.Now()

	c := context.Session.DB(context.DBName).C("jobs")
	err = c.Insert(job)
	if err != nil {
		lastError := err.(*mgo.LastError)
		if lastError.Code == 11000 {
			err = types.ErrIdentifierTaken
			return
		}
		err = errors.Wrap(err, "mongo insert failed")
		return
	}

	return job, nil
}

// GetJobConfigSpaceByID searches for a job by ID and builds a complete config space given its models.
func (context Context) GetJobConfigSpaceByID(id bson.ObjectId) (configSpace string, err error) {

	// Get job object.
	var job types.Job
	job, err = context.GetJobByID(id)
	if err != nil {
		return
	}

	// TODO: This doesn't work as intended probably. It will just read the existing job config space.
	//       This function is never used. Reconsider rewriting before usage.
	return context.GetJobConfigSpace(job)
}

// GetJobConfigSpace takes a job and builds a complete config space given its models.
func (context Context) GetJobConfigSpace(job types.Job) (configSpace string, err error) {

	// Get all the modules of this job.
	var models []types.Module
	models, _, err = context.GetModules(F{"id": job.Models}, 0, "", "", "")
	if err != nil {
		err = errors.Wrap(err, "error while trying to access the referenced models")
		return
	}

	// If the job config space has been specified for some models then we want to take those definitions instead
	// of the original ones.
	redefinedConfigSpaces := map[string]string{}
	if job.ConfigSpace != "" {
		type modelConfigElement struct {
			ID     string      `json:"id"`
			Config interface{} `json:"config"`
		}
		var jobConfigSpace []modelConfigElement
		err = json.Unmarshal([]byte(job.ConfigSpace), &jobConfigSpace)
		if err != nil {
			err = errors.Wrap(err, "error while json decoding the job config space field")
			return
		}
		for i := range jobConfigSpace {
			var configDef []byte
			configDef, err = json.Marshal(jobConfigSpace[i])
			if err != nil {
				err = errors.Wrap(err, "error while json encoding the job config space object")
				return
			}
			redefinedConfigSpaces[jobConfigSpace[i].ID] = string(configDef)
		}
	}

	// Build the config space by building a .choice structure above the model config spaces.
	configSpaceList := make([]string, len(job.Models))
	for i := range job.Models {
		for j := range models {
			if job.Models[i] == models[j].ID {
				// If the model config space was redefined, then we use that instead of the default.
				if configDef, ok := redefinedConfigSpaces[models[j].ID]; ok {
					// TODO: Validate the redefined config space by checking that it is a subset of the default.
					configSpaceList[i] = configDef
				} else {
					configSpaceList[i] = fmt.Sprintf("{\"id\" : \"%s\", \"config\" : %s }", models[j].ID, models[j].ConfigSpace)
				}
				break
			}
		}
	}
	configSpaceJoined := strings.Join(configSpaceList, ", ")
	configSpace = fmt.Sprintf("{\"id\" : \"%s\", \"model\" : { \".choice\" : [%s] } }", job.ID.Hex(), configSpaceJoined)
	return
}

// UpdateJob updates the information about a given job.
func (context Context) UpdateJob(id bson.ObjectId, updates map[string]interface{}) (result types.Job, err error) {

	// Try to find the job so that we can read its state and correctly handle state transitions.
	var currentJob types.Job
	currentJob, err = context.GetJobByID(id)
	if err != nil {
		err = errors.Wrap(err, "error while doing resource lookup")
		return
	}
	if context.User.IsRoot() == false && currentJob.User != context.User.ID {
		err = types.ErrUnauthorized
		return
	}

	// Build the update document. Validate values.
	valueUpdates := bson.M{}
	for k, v := range updates {
		switch k {
		case "models":
			// TODO: Maybe check that no models have been removed.
			// Argument against: Maybe we want to artificially prevent some models from
			//                   being trained. This doesn't affect existing tasks.
			updateModels := v.([]string)

			// Validate that the models exist and are active.
			if len(updateModels) > 0 {
				var foundModels []types.Module
				foundModels, _, err = context.GetModules(F{"id": updateModels}, 0, "", "", "")
				if err != nil {
					err = errors.Wrap(err, "error while trying to access the referenced models")
					return
				}
				configSpaceList := []string{}
				for i := range updateModels {
					var found bool
					for j := range foundModels {
						if updateModels[i] == foundModels[j].ID && foundModels[j].Status == types.ModuleActive {
							configSpaceList = append(configSpaceList, fmt.Sprintf("{\"id\" : \"%s\", \"config\" : %s }", foundModels[j].ID, foundModels[j].ConfigSpace))
							found = true
							break
						}
					}
					if found == false {
						err = errors.Wrapf(ErrBadInput,
							"the referenced model \"%s\" does not exist or is active", updateModels)
					}
				}
				configSpaceJoined := strings.Join(configSpaceList, ", ")
				valueUpdates["config-space"] = fmt.Sprintf("{\"id\" : \"%s\", \"model\" : { \".choice\" : [%s] } }", id.Hex(), configSpaceJoined)
			}

			valueUpdates["models"] = updateModels

		case "accept-new-models":
			valueUpdates["accept-new-models"] = v.(bool)
		case "status":
			status := v.(string)

			// If the update is the same as the current state, then just skip.
			if status == currentJob.Status {
				continue
			}

			// Perform state transition validations.
			switch status {
			case types.JobScheduled:
				if currentJob.Status != types.JobScheduled {
					err = errors.Wrap(ErrBadInput, "transition to the scheduled state is not allowed")
					return
				}

			case types.JobRunning:
				if currentJob.Status != types.JobScheduled &&
					currentJob.Status != types.JobResuming {
					err = errors.Wrap(ErrBadInput,
						"transition to the running state only allowed from the scheduled and resuming state")
					return
				}
				if currentJob.Status == types.JobScheduled {
					// The job has entered the running state for the first time.
					valueUpdates["running-time.start"] = time.Now()
				}
				if currentJob.Status == types.JobPaused {
					// We are leaving the paused state and need to record the time spent in it.
					valueUpdates["prev-pause-duration"] = currentJob.PrevPauseDuration +
						uint64(time.Since(currentJob.PauseStartTime)/1000000)
				}

			case types.JobPausing:
				if currentJob.Status != types.JobRunning {
					err = errors.Wrap(ErrBadInput,
						"transition to the pausing state is only allowed from the running state")
					return
				}

			case types.JobPaused:
				if currentJob.Status != types.JobPausing {
					err = errors.Wrap(ErrBadInput,
						"transition to the paused state is only allowed from the pausing state")
					return
				}
				valueUpdates["pause-start-time"] = time.Now()

			case types.JobResuming:
				if currentJob.Status != types.JobPausing &&
					currentJob.Status != types.JobPaused {
					err = errors.Wrap(ErrBadInput,
						"transition to the resuming state is only allowed from the pausing or paused state")
					return
				}

			case types.JobCompleted:
				if currentJob.Status != types.JobRunning {
					err = errors.Wrap(ErrBadInput,
						"transition to the completed state is only allowed from the running state")
					return
				}
				valueUpdates["running-time.end"] = time.Now()

			case types.JobTerminating:
				if currentJob.Status != types.JobRunning &&
					currentJob.Status != types.JobPausing &&
					currentJob.Status != types.JobPaused {
					err = errors.Wrap(ErrBadInput,
						"transition to the terminating state is only allowed from the running, pausing or paused state")
					return
				}

			case types.JobTerminated:
				if currentJob.Status != types.JobTerminating {
					err = errors.Wrap(ErrBadInput,
						"transition to the terminated state is only allowed from the terminating state")
					return
				}
				valueUpdates["running-time.end"] = time.Now()

			case types.JobError:
				valueUpdates["running-time.end"] = time.Now()

			default:
				err = errors.Wrapf(ErrBadInput,
					"value of status can be \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\" or \"%s\", but found \"%s\"",
					types.JobScheduled, types.JobRunning, types.JobCompleted, types.JobTerminating, types.JobTerminated, types.JobPausing,
					types.JobPaused, types.JobError, status)
				return
			}

			// If the new status has passed validation, set it.
			valueUpdates["status"] = status

		case "status-message":
			valueUpdates["status-message"] = v.(string)

		case "max-tasks":
			valueUpdates["max-tasks"] = v.(uint64)

		default:
			err = errors.Wrap(ErrBadInput, "invalid value of parameter updates")
			return
		}
	}

	// If there were no updates, then we can skip this step.
	if len(valueUpdates) > 0 {
		c := context.Session.DB(context.DBName).C("jobs")
		err = c.Update(bson.M{"_id": id}, bson.M{"$set": valueUpdates})
		if err != nil {
			err = errors.Wrap(err, "mongo update failed")
			return
		}
	}

	// Get the updated job and update cache if needed.
	result, err = context.GetJobByID(id)
	if err != nil {
		err = errors.Wrap(err, "job get by ID failed")
		return
	}

	return

}

// LockJob scans the available jobs (that are not currently locked), applies the specified filters,
// sorts them if specified and locks the first one by assigning it to the specified process.
func (context Context) LockJob(
	filters F,
	processID bson.ObjectId,
	sortBy string,
	order string,
) (result types.Job, err error) {
	c := context.Session.DB(context.DBName).C("jobs")

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
			setDefault(&query, "_id", bson.M{})
			query["_id"].(bson.M)["$in"] = v.([]bson.ObjectId)
		case "user", "dataset", "objective", "status":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
		case "accept-new-models":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(bool)
		case "model":
			setDefault(&query, "models", bson.M{})
			query["models"].(bson.M)["$elemMatch"] = bson.M{"$eq": v.(string)}
		case "alt-objective":
			setDefault(&query, "alt-objectives", bson.M{})
			query["alt-objectives"].(bson.M)["$elemMatch"] = bson.M{"$eq": v.(string)}
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

	var oneResult types.Job
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

// UnlockJob releases the lock on a given job.
func (context Context) UnlockJob(id bson.ObjectId, processID bson.ObjectId) (err error) {

	// Only the root can acess this.
	if context.User.IsRoot() == false {
		err = ErrNotFound
		return
	}

	c := context.Session.DB(context.DBName).C("jobs")
	err = c.Update(bson.M{"_id": id, "process": processID}, bson.M{"$set": bson.M{"process": nil}})
	if err == mgo.ErrNotFound {
		err = ErrNotFound
		return
	} else if err != nil {
		err = errors.Wrap(err, "mongo update failed")
		return
	}

	return
}

// AddModelToApplicableJobs searches for all datasets to which a model is applicable, looks for all jobs
// running on those datasets and adds the model to them if they have "accept-new-models" set to true.
func (context Context) AddModelToApplicableJobs(module types.Module) (err error) {

	// Find all datasets to which the given model can be applied.
	datasets, _, err := context.GetDatasets(F{
		"schema-in":  module.SchemaIn,
		"schema-out": module.SchemaOut,
		"status":     types.DatasetValidated,
	}, 0, "", "", "")

	if len(datasets) == 0 {
		return nil
	}

	datasetIDs := []string{}
	for i := range datasets {
		datasetIDs = append(datasetIDs, datasets[i].ID)
	}

	// Find all jobs running on those datasets that can accept new models.
	c := context.Session.DB(context.DBName).C("jobs")
	query := bson.M{
		"dataset":           bson.M{"$in": datasetIDs},
		"status":            bson.M{"$nin": []string{types.JobTerminating, types.JobTerminated, types.JobError}},
		"accept-new-models": bson.M{"$eq": true},
	}
	var jobs []types.Job
	err = c.Find(query).All(&jobs)
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	// Add the given model to all those jobs.
	for i := range jobs {

		// Extend the list of models.
		extendedModels := append(jobs[i].Models, module.ID)

		// Update the job.
		_, err = context.UpdateJob(jobs[i].ID, F{"models": extendedModels})
		if err != nil {
			err = errors.Wrap(err, "job update failed")
			return
		}
	}

	return nil
}

// ReleaseJobLockByProcess releases all jobs that have been locked by a given process and
// are not in the error state.
func (context Context) ReleaseJobLockByProcess(processID bson.ObjectId) (numReleased int, err error) {

	c := context.Session.DB(context.DBName).C("jobs")
	var changeInfo *mgo.ChangeInfo
	changeInfo, err = c.UpdateAll(
		bson.M{"process": processID, "status": bson.M{"$ne": types.JobError}},
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
