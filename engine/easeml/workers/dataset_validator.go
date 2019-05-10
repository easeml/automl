package workers

import (
	"encoding/json"
	"log"
	"time"

	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"github.com/ds3lab/easeml/engine/easeml/database/model/types"
	"github.com/ds3lab/easeml/engine/easeml/storage"

	"github.com/pkg/errors"
)

// DatasetValidatorListener periodically checks if there are any datasets which have been unpacked
// in order to validate them.
func (context Context) DatasetValidatorListener() {

	for {
		dataset, err := context.ModelContext.LockDataset(model.F{"status": types.DatasetUnpacked}, context.ProcessID, "", "")
		if err == nil {
			log.Printf("DATASET FOUND FOR VALIDATION")
			go context.DatasetValidatorkWorker(dataset)
		} else if errors.Cause(err) == model.ErrNotFound {
			time.Sleep(context.Period)
		} else {
			panic(err)
		}
	}

}

// DatasetValidatorkWorker performs the unpacking operation.
func (context Context) DatasetValidatorkWorker(dataset types.Dataset) {

	// Get the dataset directory.
	datasetPath, err := context.StorageContext.GetDatasetPath(dataset.ID, "")
	if err != nil {
		panic(err) // This means that we cannot access the file system, so we need to panic.
	}

	// Check if we can infer the schema.
	schemaIn, schemaOut, err := storage.InferDatasetSchema(datasetPath)
	if err != nil || schemaIn == nil || schemaOut == nil {

		if err == nil {
			err = errors.New("every dataset must have an input and output schema")
		}

		err = errors.WithStack(err)
		context.Logger.WithFields(
			"dataset-id", dataset.ID,
			"source", dataset.Source,
			"source-address", dataset.SourceAddress,
		).WithStack(err).WithError(err).WriteError("DATASET VALIDATION ERROR")

		context.repeatUntilSuccess(func() error {
			return context.ModelContext.UpdateDatasetStatus(dataset.ID, types.DatasetError, err.Error())
		})
		return
	}

	// Generate schema strings.
	jsonSchemaIn, err := json.Marshal(schemaIn.Dump())
	if err != nil {
		panic(err) // This should never happen.
	}
	jsonSchemaOut, err := json.Marshal(schemaOut.Dump())
	if err != nil {
		panic(err) // This should never happen.
	}

	// Update the dataset with the schema.
	context.repeatUntilSuccess(func() error {
		updates := model.F{"schema-in": string(jsonSchemaIn), "schema-out": string(jsonSchemaOut)}
		_, err := context.ModelContext.UpdateDataset(dataset.ID, updates)
		return err
	})

	// Unlock the dataset and update the status.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UpdateDatasetStatus(dataset.ID, types.DatasetValidated, "")
	})
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockDataset(dataset.ID, context.ProcessID)
	})

	// Log task completion.
	context.Logger.WithFields(
		"dataset-id", dataset.ID,
		"source", dataset.Source,
		"source-address", dataset.SourceAddress,
	).WriteInfo("DATASET VALIDATION COMPLETED")

}
