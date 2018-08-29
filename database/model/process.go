package model

import (
	"encoding/binary"
	"encoding/hex"
	"net"
	"os"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

const (
	// ProcController is the type of process that serves as the interface between
	// the users and the data model, as well as controlling the operation of the system.
	ProcController = "controller"

	// ProcWorker is the type of process that trains models and evaluates them.
	ProcWorker = "worker"

	// ProcScheduler is the type of process that handles scheduling of tasks.
	ProcScheduler = "scheduler"

	// ProcIdle is the status of the process when it is running but not doing any work.
	ProcIdle = "idle"

	// ProcWorking is the status of the process when it is running and doing work.
	ProcWorking = "working"

	// ProcTerminated is the status of the process that is not running anymore.
	ProcTerminated = "terminated"
)

// Process contains information about processes.
type Process struct {
	ID            bson.ObjectId `bson:"_id" json:"id"`
	ProcessID     uint64        `bson:"process-id" json:"process-id"`
	HostID        string        `bson:"host-id" json:"host-id"`
	HostAddress   string        `bson:"host-address" json:"host-address"`
	StartTime     time.Time     `bson:"start-time" json:"start-time"`
	LastKeepalive time.Time     `bson:"last-keepalive" json:"last-keepalive"`
	Type          string        `bson:"type" json:"type"`
	Resource      string        `bson:"resource" json:"resource"`
	Status        string        `bson:"status" json:"status"`
	RunningOrinal int           `bson:"running-ordinal" json:"running-ordinal"`
}

// GetProcessByID returns a process given its id.
func (context Context) GetProcessByID(id bson.ObjectId) (result Process, err error) {

	// Currently there are no restrictions for non-root users here.

	c := context.Session.DB(context.DBName).C("processes")
	var allResults []Process
	err = c.Find(bson.M{"_id": id}).All(&allResults)
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

// GetProcesses lists all processes given some filter criteria.
func (context Context) GetProcesses(
	filters F,
	limit int,
	cursor string,
	sortBy string,
	order string,
) (result []Process, cm CollectionMetadata, err error) {

	c := context.Session.DB(context.DBName).C("processes")

	// Validate the parameters.
	if sortBy != "" &&
		sortBy != "id" &&
		sortBy != "process-id" &&
		sortBy != "host-id" &&
		sortBy != "host-address" &&
		sortBy != "start-time" &&
		sortBy != "type" &&
		sortBy != "resource" &&
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

	// We currently don't limit access to this collection. Everyone can see it.

	// Build a query given the parameters.
	query := bson.M{}
	for k, v := range filters {
		switch k {
		case "id":
			setDefault(&query, "_id", bson.M{})
			query["_id"].(bson.M)["$in"] = v.([]bson.ObjectId)
		case "process-id":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(uint64)
		case "host-id", "host-address", "type", "resource", "status":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
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
		if sortBy != "" && sortBy != "id" {
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
			case "host-id", "host-address", "type", "resource", "status":
				otherCursor = string(decoded)
			case "process-id":
				otherCursor = binary.BigEndian.Uint64(decoded)
			case "start-time":
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
	if sortBy == "" || sortBy == "id" {
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
	var allResults []Process
	err = q.All(&allResults)
	if err != nil {
		err = errors.Wrap(err, "mongo find failed")
		return
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
			case "id":
				b = []byte(lastResult.ID)
			case "process-id":
				b = make([]byte, 4)
				binary.BigEndian.PutUint64(b, lastResult.ProcessID)
			case "host-id":
				b = []byte(lastResult.HostID)
			case "host-address":
				b = []byte(lastResult.HostAddress)
			case "start-time":
				b, err = lastResult.StartTime.GobEncode()
			case "type":
				b = []byte(lastResult.Type)
			case "resource":
				b = []byte(lastResult.Resource)
			case "status":
				b = []byte(lastResult.Status)
			}
			encoded = hex.EncodeToString(b)
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

// CountProcesses is the same as GetProcesses but returns only the count, not the actual processes.
func (context Context) CountProcesses(filters F) (count int, err error) {

	c := context.Session.DB(context.DBName).C("processes")

	// Build a query given the parameters.
	query := bson.M{}
	for k, v := range filters {
		switch k {
		case "id":
			setDefault(&query, "_id", bson.M{})
			query["_id"].(bson.M)["$in"] = v.([]bson.ObjectId)
		case "process-id":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(uint64)
		case "host-id", "host-address", "type", "resource", "status":
			setDefault(&query, k, bson.M{})
			query[k].(bson.M)["$eq"] = v.(string)
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

// CreateProcess adds a given process to the database.
func (context Context) CreateProcess(proc Process) (result Process, err error) {

	// This action is only permitted for the root user.
	if context.User.IsRoot() == false {
		err = ErrUnauthorized
		return
	}
	if proc.Type != ProcController && proc.Type != ProcWorker && proc.Type != ProcScheduler {
		err = errors.Wrapf(ErrBadInput,
			"value of type can be \"%s\", \"%s\" or \"%s\", but found \"%s\"",
			ProcController, ProcWorker, ProcScheduler, proc.Type)
		return
	}

	// Find the first candidate ordinal. We do this optimistically assuming there is no race conditions.
	proc.RunningOrinal, err = context.findCandidateOrdinal(proc.Type)
	if err != nil {
		err = errors.Wrap(err, "find candidate ordinal failed")
		return
	}

	// Give default values to some fields.
	proc.ID = bson.NewObjectId()
	proc.Status = ProcIdle
	proc.StartTime = time.Now()

	c := context.Session.DB(context.DBName).C("processes")
	err = c.Insert(proc)
	if err != nil {
		lastError := err.(*mgo.LastError)
		if lastError.Code == 11000 {
			err = ErrIdentifierTaken
			return
		}
		err = errors.Wrap(err, "mongo insert failed")
		return
	}

	// Check if the ordinal was accepted. If not, then there was a race condition so we need to handle it.
	var accepted bool
	accepted, err = context.isOrdinalAccepted(proc.Type, proc.ID, proc.RunningOrinal)
	if err != nil {
		err = errors.Wrap(err, "is ordinal accepted failed")
		return
	}
	for accepted == false {
		// As long as the ordinal isn't accepted we need to try again.
		proc.RunningOrinal, err = context.findCandidateOrdinal(proc.Type)
		if err != nil {
			err = errors.Wrap(err, "find candidate ordinal failed")
			return
		}

		// Update the process with the new ordinal.
		c := context.Session.DB(context.DBName).C("processes")
		err = c.Update(bson.M{"_id": proc.ID}, bson.M{"$set": bson.M{"running-ordinal": proc.RunningOrinal}})
		if err != nil {
			err = errors.Wrap(err, "mongo update failed")
			return
		}

		// Check if the ordinal was accepted.
		accepted, err = context.isOrdinalAccepted(proc.Type, proc.ID, proc.RunningOrinal)
		if err != nil {
			err = errors.Wrap(err, "is ordinal accepted failed")
			return
		}
	}

	return proc, nil

}

func (context Context) findCandidateOrdinal(processType string) (ordinal int, err error) {

	c := context.Session.DB(context.DBName).C("processes")
	query := bson.M{"type": processType, "status": bson.M{"$ne": ProcTerminated}}
	var ordinals []struct {
		Ordinal int `bson:"running-ordinal"`
	}
	q := c.Find(query).Select(bson.M{"running-ordinal": 1}).Sort("running-ordinal")
	err = q.All(&ordinals)
	if err != nil {
		return -1, err
	}

	ordinal = 1
	for i := range ordinals {
		if ordinal == ordinals[i].Ordinal {
			ordinal++
		}
	}

	return
}

func (context Context) isOrdinalAccepted(processType string, processID bson.ObjectId, ordinal int) (result bool, err error) {
	c := context.Session.DB(context.DBName).C("processes")
	query := bson.M{"type": processType, "status": bson.M{"$ne": ProcTerminated}, "running-ordinal": ordinal}
	var ordinals []struct {
		ObjectID bson.ObjectId `bson:"_id"`
		Ordinal  int           `bson:"running-ordinal"`
	}
	q := c.Find(query).Select(bson.M{"_id": 1, "running-ordinal": 1})
	err = q.All(&ordinals)
	if err != nil {
		return false, err
	}

	result = true
	for i := range ordinals {
		if processID > ordinals[i].ObjectID {
			result = false
			break
		}
	}
	return
}

// UpdateProcess updates the information about a given process.
func (context Context) UpdateProcess(id bson.ObjectId, updates map[string]interface{}) (result Process, err error) {

	// This action is only permitted for the root user.
	if context.User.IsRoot() == false {
		err = ErrUnauthorized
		return
	}

	// Build the update document. Validate values.
	valueUpdates := bson.M{}
	for k, v := range updates {
		switch k {
		case "status":
			status := v.(string)

			if status != ProcIdle && status != ProcWorking && status != ProcTerminated {
				err = errors.Wrapf(ErrBadInput,
					"value of status can be \"%s\", \"%s\" or \"%s\", but found \"%s\"",
					ProcIdle, ProcWorking, ProcTerminated, status)
				return
			}
			valueUpdates["status"] = status
		case "last-keepalive":
			valueUpdates["last-keepalive"] = v.(time.Time)
		default:
			err = errors.Wrap(ErrBadInput, "invalid value of parameter updates")
			return
		}
	}

	// If there were no updates, then we can skip this step.
	if len(valueUpdates) > 0 {
		c := context.Session.DB(context.DBName).C("processes")
		err = c.Update(bson.M{"_id": id}, bson.M{"$set": valueUpdates})
		if err != nil {
			err = errors.Wrap(err, "mongo update failed")
			return
		}
	}

	// Get the updated process and update cache if needed.
	result, err = context.GetProcessByID(id)
	if err != nil {
		err = errors.Wrap(err, "process get by ID failed")
		return
	}

	return

}

// StartProcess starts a process of a given type and initializes all other fields automatically.
func (context Context) StartProcess(processType string) (result Process, err error) {

	if processType != ProcController && processType != ProcScheduler && processType != ProcWorker {
		panic("invalid processType")
	}

	var hostID string
	hostID, err = os.Hostname()
	if err != nil {
		err = errors.Wrap(err, "get hostname from os failed")
		return
	}

	var hostAddress string
	hostAddress, err = getOutboundIP()
	if err != nil {
		err = errors.Wrap(err, "get outbound ip failed")
		return
	}

	process := Process{
		HostID:      hostID,
		HostAddress: hostAddress,
		ProcessID:   uint64(os.Getpid()),
		Resource:    "cpu", // TODO: Change this later.
		Type:        processType,
	}

	return context.CreateProcess(process)
}

// getOutboundIP returns the preferred outbound ip of this machine
func getOutboundIP() (ip string, err error) {
	var conn net.Conn
	conn, err = net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip = localAddr.IP.String()

	return
}

// SetProcessStatus updates the processes state to terminated.
func (context Context) SetProcessStatus(id bson.ObjectId, status string) (result Process, err error) {

	if status != ProcIdle && status != ProcWorking && status != ProcTerminated {
		panic("invalid status")
	}

	return context.UpdateProcess(id, F{"status": status})
}

// ProcessKeepalive updates the keepalive-time field of the process
// thus notifying the system that it is still running.
func (context Context) ProcessKeepalive(id bson.ObjectId) (err error) {
	_, err = context.UpdateProcess(id, F{"last-keepalive": time.Now()})
	return
}

// TerminateDeadProcesses goes through all processes that have stopped making keepalive updates
// and sets their status to terminated.
func (context Context) TerminateDeadProcesses(cutoffTime time.Time) (err error) {

	// Terminated all idle processes and release all locks that they held.
	selector := bson.M{
		"last-keepalive": bson.M{"$lt": cutoffTime},
		"status":         bson.M{"$ne": ProcTerminated},
	}
	update := bson.M{
		"$set": bson.M{"status": ProcTerminated},
	}
	change := mgo.Change{Update: update, ReturnNew: true}
	found := true

	for found {
		var process Process
		c := context.Session.DB(context.DBName).C("processes")
		changeInfo, err := c.Find(selector).Apply(change, &process)
		if err != nil && err != mgo.ErrNotFound {
			return err
		}

		if changeInfo != nil && changeInfo.Updated > 0 {

			// Release any locks that the terminated process held.
			context.ReleaseDatasetLockByProcess(process.ID)
			if err != nil {
				return err
			}
			context.ReleaseModuleLockByProcess(process.ID)
			if err != nil {
				return err
			}
			context.ReleaseJobLockByProcess(process.ID)
			if err != nil {
				return err
			}
			context.ReleaseTaskLockByProcess(process.ID)
			if err != nil {
				return err
			}

		} else {
			found = false
		}
	}

	return nil
}
