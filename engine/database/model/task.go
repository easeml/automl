package model

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ds3lab/easeml/engine/database/model/types"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

// GetTaskByID returns the task given its id. The id is given as "user-id/task-id".
func (context Context) GetTaskByID(id string) (result types.Task, err error) {

	c := context.Session.DB(context.DBName).C("tasks")
	var allResults []types.Task

	// Only the root user can look up tasks other than their own.
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

	result = allResults[0]

	// Update computed fields.
	result.StageDurations = result.GetStageDurations()
	result.RunningDuration = result.GetRunningDuration()

	return result, nil
}

// GetTasks lists all tasks given some filter criteria.
func (context Context) GetTasks(
	filters F,
	limit int,
	cursor string,
	sortBy string,
	order string,
) (result []types.Task, cm types.CollectionMetadata, err error) {

	c := context.Session.DB(context.DBName).C("tasks")

	// Validate the parameters.
	if sortBy != "" &&
		sortBy != "id" &&
		sortBy != "process" &&
		sortBy != "job" &&
		sortBy != "user" &&
		sortBy != "dataset" &&
		sortBy != "objective" &&
		sortBy != "model" &&
		sortBy != "quality" &&
		sortBy != "quality-train" &&
		sortBy != "quality-expected" &&
		sortBy != "creation-time" &&
		sortBy != "status" &&
		sortBy != "stage" {
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
			setDefault(&query, "id", bson.M{})
			query["id"].(bson.M)["$in"] = v.([]string)
		case "user", "dataset", "model", "objective", "status", "stage":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
		case "process", "job":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(bson.ObjectId)
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
			case "id", "user", "process", "job", "dataset", "model", "objective", "status", "stage":
				otherCursor = string(decoded)
			case "creation-time":
				var t time.Time
				t.GobDecode(decoded)
				otherCursor = t
			case "quality", "quality-train", "quality-expected":
				otherCursor = math.Float64frombits(binary.BigEndian.Uint64(decoded))
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
	var allResults []types.Task
	err = q.All(&allResults)
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}

	// Update computed fields.
	for i := range allResults {
		allResults[i].StageDurations = allResults[i].GetStageDurations()
		allResults[i].RunningDuration = allResults[i].GetRunningDuration()
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
			case "process":
				b = []byte(lastResult.Process)
			case "job":
				b = []byte(lastResult.Job)
			case "dataset":
				b = []byte(lastResult.Dataset)
			case "model":
				b = []byte(lastResult.Model)
			case "objective":
				b = []byte(lastResult.Objective)
			case "creation-time":
				b, err = lastResult.CreationTime.GobEncode()
			case "quality":
				b = make([]byte, 8)
				binary.BigEndian.PutUint64(b, math.Float64bits(lastResult.Quality))
			case "quality-train":
				b = make([]byte, 8)
				binary.BigEndian.PutUint64(b, math.Float64bits(lastResult.QualityTrain))
			case "quality-expected":
				b = make([]byte, 8)
				binary.BigEndian.PutUint64(b, math.Float64bits(lastResult.QualityTrain))
			case "status":
				b = []byte(lastResult.Status)
			case "stage":
				b = []byte(lastResult.Stage)
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

// CountTasks is the same as GetTasks but returns only the count, not the actual tasks.
func (context Context) CountTasks(filters F) (count int, err error) {

	c := context.Session.DB(context.DBName).C("tasks")

	// If the user is not root then we need to limit access.
	query := bson.M{}
	if context.User.IsRoot() == false {
		query = bson.M{"user": bson.M{"$in": []string{context.User.ID, types.UserRoot}}}
	}

	// Build a query given the parameters.
	for k, v := range filters {
		switch k {
		case "id":
			setDefault(&query, "id", bson.M{})
			query["id"].(bson.M)["$in"] = v.([]string)
		case "user", "dataset", "model", "objective", "status", "stage":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
		case "process", "job":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(bson.ObjectId)
		case "alt-objective":
			setDefault(&query, "alt-objectives", bson.M{})
			query["alt-objectives"].(bson.M)["$elemMatch"] = bson.M{"$eq": v.(string)}
		default:
			err = errors.Wrap(ErrBadInput, "invalid value of argument filters")
			return
		}
	}

	// We count the number of tasks that satisfy the filter criteria.
	count, err = c.Find(query).Count()
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
	}

	return
}

// CreateTask adds a given task to the database.
func (context Context) CreateTask(task types.Task) (result types.Task, err error) {

	// Validate that the job exists and is running.
	var job types.Job
	job, err = context.GetJobByID(task.Job)
	if err != nil && err != ErrNotFound {
		err = errors.Wrap(err, "error while trying to access the referenced job")
		return
	} else if err == ErrNotFound || job.Status != types.JobRunning {
		err = errors.Wrapf(ErrBadInput,
			"the referenced objective \"%s\" does not exist or is running", task.Job)
	}

	// Validate that the models exist and are active.
	var found bool
	for i := range job.Models {
		if task.Model == job.Models[i] {
			found = true
			break
		}
	}
	if found == false {
		err = errors.Wrapf(ErrBadInput,
			"the referenced model \"%s\" does not appear in the models list of the parent job \"%s\"",
			task.Model, job.ID)
	}

	// Give default values to some fields. Copy some from the job.
	task.ObjectID = bson.NewObjectId()
	task.User = job.User
	task.Dataset = job.Dataset
	task.Objective = job.Objective
	task.AltObjectives = job.AltObjectives
	task.CreationTime = time.Now()
	task.Status = types.TaskScheduled
	task.Stage = types.TaskStageBegin
	task.StageTimes = types.TaskStageIntervals{}
	task.StageDurations = types.TaskStageDurations{}
	task.RunningDuration = 0
	task.Quality = 0.0
	task.AltQualities = make([]float64, len(task.AltObjectives))

	// Get next ID.
	c := context.Session.DB(context.DBName).C("tasks")
	query := bson.M{"job": bson.M{"$eq": task.Job}}
	var resultSize int
	resultSize, err = c.Find(query).Count()
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
	}
	task.ID = fmt.Sprintf("%s/%010d", task.Job.Hex(), resultSize+1)

	err = c.Insert(task)
	if err != nil {
		lastError := err.(*mgo.LastError)
		if lastError.Code == 11000 {
			err = types.ErrIdentifierTaken
			return
		}
		err = errors.Wrap(err, "mongo insert failed")
		return
	}

	return task, nil

}

// TODO fix assumed transitions
// UpdateTask updates the information about a given task.
func (context Context) UpdateTask(id string, updates map[string]interface{}) (result types.Task, err error) {

	// Try to find the task so that we can read its state and correctly handle state transitions.
	var currentTask types.Task
	currentTask, err = context.GetTaskByID(id)
	if err != nil {
		err = errors.Wrap(err, "error while doing resource lookup")
		return
	}

	// Build the update document. Validate values.
	valueUpdates := bson.M{}
	for k, v := range updates {
		switch k {
		case "quality":
			valueUpdates["quality"] = v.(float64)
		case "quality-train":
			valueUpdates["quality-train"] = v.(float64)
		case "quality-expected":
			valueUpdates["quality-expected"] = v.(float64)
		case "alt-qualities":
			valueUpdates["alt-qualities"] = v.([]float64)
		case "status":
			status := v.(string)

			// Perform state transition validations.
			switch status {
			case types.TaskScheduled:
				if currentTask.Status != types.TaskScheduled {
					err = errors.Wrap(ErrBadInput, "transition to the scheduled state is not allowed")
					return
				}

			case types.TaskRunning:
				if currentTask.Status != types.TaskScheduled &&
					currentTask.Status != types.TaskPausing &&
					currentTask.Status != types.TaskPaused {
					err = errors.Wrap(ErrBadInput,
						"transition to the running state only allowed from the scheduled, pausing and paused state")
					return
				}

			case types.TaskPausing:
				if currentTask.Status != types.TaskRunning {
					err = errors.Wrap(ErrBadInput,
						"transition to the pausing state is only allowed from the running state")
					return
				}

			case types.TaskPaused:
				if currentTask.Status != types.TaskPausing {
					err = errors.Wrap(ErrBadInput,
						"transition to the paused state is only allowed from the pausing state")
					return
				}

			case types.TaskCompleted:
				if currentTask.Status != types.TaskRunning {
					err = errors.Wrap(ErrBadInput,
						"transition to the completed state is only allowed from the running state")
					return
				}

			case types.TaskTerminating:
				if currentTask.Status != types.TaskRunning &&
					currentTask.Status != types.TaskPausing &&
					currentTask.Status != types.TaskPaused {
					err = errors.Wrap(ErrBadInput,
						"transition to the terminating state is only allowed from the running, pausing or paused state")
					return
				}

			case types.TaskTerminated:
				if currentTask.Status != types.TaskTerminating {
					err = errors.Wrap(ErrBadInput,
						"transition to the terminated state is only allowed from the terminating state")
					return
				}

			case types.TaskCanceled:
				if currentTask.Status != types.TaskScheduled {
					err = errors.Wrap(ErrBadInput, "transition to the scheduled state is not allowed")
					return
				}

			case types.TaskError:

				// Since this can be an abrupt ending, we need to record the ending time of the stage.
				switch currentTask.Stage {
				case types.TaskStageTrain:
					valueUpdates["stage-times.training.end"] = time.Now()
				case types.TaskStagePredict:
					valueUpdates["stage-times.predicting.end"] = time.Now()
				case types.TaskStageEvaluate:
					valueUpdates["stage-times.evaluating.end"] = time.Now()
				}

			default:
				err = errors.Wrapf(ErrBadInput,
					"value of status can be \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\" or \"%s\", but found \"%s\"",
					types.TaskScheduled, types.TaskRunning, types.TaskCompleted, types.TaskTerminating, types.TaskTerminated, types.TaskPausing,
					types.TaskPaused, types.TaskError, status)
				return
			}

			// If the new status has passed validation, set it.
			valueUpdates["status"] = status

		case "stage":
			stage := v.(string)

			if currentTask.Stage != stage {
				for k, _ :=range(types.AllStages){
					if currentTask.Stage == k {
						valueUpdates["stage-times."+k+".end"] = time.Now()
					}
				}
				valueUpdates["stage-times."+stage+".start"] = time.Now()
			}

			// TODO validation through schema

			// If the new status has passed validation, set it.
			valueUpdates["stage"] = stage

		case "status-message":
			valueUpdates["status-message"] = v.(string)

		default:
			err = errors.Wrap(ErrBadInput, "invalid value of parameter updates")
			return
		}

	}

	// If there were no updates, then we can skip this step.
	if len(valueUpdates) > 0 {
		c := context.Session.DB(context.DBName).C("tasks")
		err = c.Update(bson.M{"id": id}, bson.M{"$set": valueUpdates})
		if err != nil {
			err = errors.Wrap(err, "mongo update failed")
			return
		}
	}

	// Get the updated task and update cache if needed.
	result, err = context.GetTaskByID(id)
	if err != nil {
		err = errors.Wrap(err, "task get by ID failed")
		return
	}

	return

}

// LockTask scans the available tasks (that are not currently locked), applies the specified filters,
// sorts them if specified and locks the first one by assigning it to the specified process.
func (context Context) LockTask(
	filters F,
	processID bson.ObjectId,
	sortBy string,
	order string,
) (result types.Task, err error) {
	c := context.Session.DB(context.DBName).C("tasks")

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
		case "user", "process", "job", "dataset", "model", "objective", "status", "stage":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
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

	var oneResult types.Task
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

// UnlockTask releases the lock on a given task.
func (context Context) UnlockTask(id string, processID bson.ObjectId) (err error) {

	// Perform validation of fields.
	ids := strings.Split(id, "/")
	if len(ids) != 2 {
		err = errors.Wrap(ErrBadInput, "the id must be of the format job-id/task-id")
		return
	}

	c := context.Session.DB(context.DBName).C("tasks")
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

// PauseRunningTasks goes through all tasks of a job that are still running and marks them as pausing.
func (context Context) PauseRunningTasks(jobID bson.ObjectId) (err error) {

	c := context.Session.DB(context.DBName).C("tasks")
	selector := bson.M{"job": jobID, "status": bson.M{"$in": []string{types.TaskScheduled, types.TaskRunning}}}
	update := bson.M{"$set": bson.M{"status": types.TaskPausing}}

	_, err = c.UpdateAll(selector, update)
	if err != nil {
		err = errors.Wrap(err, "mongo update failed")
	}
	return
}

// ResumePausedTasks goes through all tasks of a job that are pausing or paused and marks them as scheduled.
func (context Context) ResumePausedTasks(jobID bson.ObjectId) (err error) {

	c := context.Session.DB(context.DBName).C("tasks")
	selector := bson.M{"job": jobID, "status": bson.M{"$in": []string{types.TaskPausing, types.TaskPaused}}}
	update := bson.M{"$set": bson.M{"status": types.TaskScheduled}}

	_, err = c.UpdateAll(selector, update)
	if err != nil {
		err = errors.Wrap(err, "mongo update failed")
	}
	return
}

// TerminateRunningTasks goes through all tasks of a job that are still running and marks them as terminating.
func (context Context) TerminateRunningTasks(jobID bson.ObjectId) (err error) {

	c := context.Session.DB(context.DBName).C("tasks")
	selector := bson.M{"job": jobID, "status": bson.M{"$in": []string{types.TaskScheduled, types.TaskRunning, types.TaskPausing, types.TaskPaused}}}
	update := bson.M{"$set": bson.M{"status": types.TaskTerminating}}

	_, err = c.UpdateAll(selector, update)
	if err != nil {
		err = errors.Wrap(err, "mongo update failed")
	}
	return
}

// UpdateTaskStatus sets the status of the task and assigns the given status message.
func (context Context) UpdateTaskStatus(id string, status string, statusMessage string) (err error) {
	_, err = context.UpdateTask(id, F{"status": status, "status-message": statusMessage})
	return
}

// UpdateTaskStage sets the stage of the task.
func (context Context) UpdateTaskStage(id string, stage string) (err error) {
	_, err = context.UpdateTask(id, F{"stage": stage})
	return
}

// ReleaseTaskLockByProcess releases all tasks that have been locked by a given process and
// are not in the error state.
func (context Context) ReleaseTaskLockByProcess(processID bson.ObjectId) (numReleased int, err error) {

	c := context.Session.DB(context.DBName).C("tasks")
	var changeInfo *mgo.ChangeInfo
	changeInfo, err = c.UpdateAll(
		bson.M{"process": processID, "status": bson.M{"$ne": types.TaskError}},
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
