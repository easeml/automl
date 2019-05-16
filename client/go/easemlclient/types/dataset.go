package types

import (
	"time"
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
	ID            string    `json:"id"`
	User          string    `json:"user"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	SchemaIn      string    `json:"schema-in"`
	SchemaOut     string    `json:"schema-out"`
	Source        string    `json:"source"`
	SourceAddress string    `json:"source-address"`
	CreationTime  time.Time `json:"creation-time"`
	Status        string    `json:"status"`
	StatusMessage string    `json:"status-message"`
	Process       string    `json:"process"`
}
