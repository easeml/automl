package types

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

const (
	// ModuleModel is the module type that represent machine learning models.
	ModuleModel = "model"

	// ModuleObjective is the module type that represents objective functions.
	ModuleObjective = "objective"

	// ModuleOptimizer is the module type representing optimizers.
	ModuleOptimizer = "optimizer"

	// ModuleUpload is a module that has veen uploaded to the system.
	ModuleUpload = "upload"

	// ModuleDownload is a module that has been downloaded from a remote location.
	ModuleDownload = "download"

	// ModuleLocal is a module that resides on a file system that is local to the easeml service.
	ModuleLocal = "local"

	// ModuleRegistry is a module that is obtained from a Docker registry.
	ModuleRegistry = "registry"

	// ModuleCreated is the status of a module that is recorded in the system but not yet transferred.
	ModuleCreated = "created"

	// ModuleTransferred is the status of a module that is transferred but not yet validated.
	ModuleTransferred = "transferred"

	// ModuleActive is the status of a module that is transferred and ready to use.
	ModuleActive = "active"

	// ModuleArchived is the status of a module that is no longer usable.
	ModuleArchived = "archived"

	// ModuleError is the status of a mofule when something goes wrong. The details will be logged.
	ModuleError = "error"
)

// Module contains information about modules which are stateless Docker images.
type Module struct {
	ObjectID      bson.ObjectId `bson:"_id"`
	ID            string        `bson:"id" json:"id"`
	User          string        `bson:"user" json:"user"`
	Type          string        `bson:"type" json:"type"`
	Label         string        `bson:"label" json:"label"`
	Name          string        `bson:"name" json:"name"`
	Description   string        `bson:"description" json:"description"`
	SchemaIn      string        `bson:"schema-in" json:"schema-in"`
	SchemaOut     string        `bson:"schema-out" json:"schema-out"`
	ConfigSpace   string        `bson:"config-space" json:"config-space"`
	Source        string        `bson:"source" json:"source"`
	SourceAddress string        `bson:"source-address" json:"source-address"`
	CreationTime  time.Time     `bson:"creation-time" json:"creation-time"`
	Status        string        `bson:"status" json:"status"`
	StatusMessage string        `bson:"status-message" json:"status-message"`
	Process       bson.ObjectId `bson:"process,omitempty" json:"process"`
	AccessKey	  string     	`bson:"access-key,omitempty" json:"access-key"`
}
