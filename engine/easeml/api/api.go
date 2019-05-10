package api

import (
	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"github.com/ds3lab/easeml/engine/easeml/logger"
	"github.com/ds3lab/easeml/engine/easeml/storage"
)

// Context contains all information needed to use the api functionality.
type Context struct {
	ModelContext   model.Context
	StorageContext storage.Context
	Logger         logger.Logger
}
