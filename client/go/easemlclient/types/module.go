package types

import (
	"time"
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
	ID            string        `json:"id"`
	User          string        `json:"user"`
	Type          string        `json:"type"`
	Label         string        `json:"label"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	SchemaIn      string        `json:"schema-in"`
	SchemaOut     string        `json:"schema-out"`
	ConfigSpace   string        `json:"config-space"`
	Source        string        `json:"source"`
	SourceAddress string        `json:"source-address"`
	CreationTime  time.Time     `json:"creation-time"`
	Status        string        `json:"status"`
	StatusMessage string        `json:"status-message"`
	Process       string        `json:"process"`
}
