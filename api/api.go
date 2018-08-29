package api

import (
	"github.com/ds3lab/easeml/database/model"
	"github.com/ds3lab/easeml/logger"
	"github.com/ds3lab/easeml/storage"
)

// Context contains all information needed to use the api functionality.
type Context struct {
	ModelContext   model.Context
	StorageContext storage.Context
	Logger         logger.Logger
}
