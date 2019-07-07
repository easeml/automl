package types

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

const (
	// DatasetUpload is a data set that has been uploaded to the system.
	DatasetUpload = "upload"

	// DatasetLocal is a data set that resides on a file system that is local to the easeml service.
	DatasetLocal = "local"

	// DatasetDownload is a data set that has been downloaded from a remote location.
	DatasetDownload = "download"

	// DatasetCreated is the status of a dataset when it is recorded in the system but the data is not yet transferred.
	DatasetCreated = "created"

	// DatasetTransferred is the status of a dataset when it is transferred but hasn't been unpacked yet.
	DatasetTransferred = "transferred"

	// DatasetUnpacked is the status of a dataset when all its files have been extracted and is ready for validation.
	DatasetUnpacked = "unpacked"

	// DatasetValidated is the status of a dataset when it has been validated and is ready to be used.
	DatasetValidated = "validated"

	// DatasetArchived is the status of a dataset when it is no longer usable.
	DatasetArchived = "archived"

	// DatasetError is the status of a dataset when something goes wrong. The details will be logged.
	DatasetError = "error"
)

// TODO: Schema should be a struct.

// Dataset contains information about datasets.
type Dataset struct {
	ObjectID      bson.ObjectId `bson:"_id"`
	ID            string        `bson:"id" json:"id"`
	User          string        `bson:"user" json:"user"`
	Name          string        `bson:"name" json:"name"`
	Description   string        `bson:"description" json:"description"`
	SchemaIn      string        `bson:"schema-in" json:"schema-in"`
	SchemaOut     string        `bson:"schema-out" json:"schema-out"`
	Source        string        `bson:"source" json:"source"`
	SourceAddress string        `bson:"source-address" json:"source-address"`
	CreationTime  time.Time     `bson:"creation-time" json:"creation-time"`
	Status        string        `bson:"status" json:"status"`
	StatusMessage string        `bson:"status-message" json:"status-message"`
	Process       bson.ObjectId `bson:"process,omitempty" json:"process"`
}
