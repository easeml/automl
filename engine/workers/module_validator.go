package workers

import (
	"encoding/json"
	"os"
	"time"

	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"
	"github.com/ds3lab/easeml/engine/modules"

	sch "github.com/ds3lab/easeml/schema/go/easemlschema/schema"

	"github.com/pkg/errors"
)

// ModuleValidateListener periodically checks if there are any modules which have been transferred
// but have not yet been validated. It performs various checks to make sure the model is ready to
// become activated.
func (context Context) ModuleValidateListener() {

	for {
		module, err := context.ModelContext.LockModule(model.F{"status": types.ModuleTransferred}, context.ProcessID, "", "")
		if err == nil {
			// Log task completion.
			context.Logger.WithFields(
				"module-id", module.ID,
				"source", module.Source,
				"source-address", module.SourceAddress,
			).WriteInfo("MODULE FOUND FOR VALIDATION")
			go context.ModuleValidateWorker(module)
		} else if errors.Cause(err) == model.ErrNotFound {
			time.Sleep(context.Period)
		} else {
			panic(err)
		}
	}

}

// ModuleValidateWorker performs the actual module validation. It makes sure the model has a defined
// config space and input and output schema.
func (context Context) ModuleValidateWorker(module types.Module) {

	// Get the module directory.
	modulePath, err := context.StorageContext.GetModulePath(module.ID, module.Type, "")
	if err != nil {
		panic(err) // This means that we cannot access the file system, so we need to panic.
	}

	// If the module was uploaded, we need to rename the uploaded file into "module.tar".
	if module.Source == types.ModuleUpload {
		// Find all uploaded files and their target paths.
		sourceFilePaths, destinationFilePaths := getUploadedFilesDestinationPaths(modulePath, "module.tar")

		// Copy all files to the target path.
		for i := range sourceFilePaths {
			err := os.Rename(sourceFilePaths[i], destinationFilePaths[i])
			if err != nil {
				panic(err)
			}
		}
	}

	// Load image and get name.
	imageFilePath := context.getModuleImagePath(module.ID, module.Type)
	imageName, err := modules.LoadImage(imageFilePath)
	if err != nil {
		err = errors.WithStack(err)
		context.moduleValidationError(err, module)
		return
	}

	// Extract image information.
	_, name, description, jsonSchemaIn, jsonSchemaOut, configSpace, err := modules.InferModuleProperties(imageName)
	if err != nil {
		err = errors.WithStack(err)
		context.moduleValidationError(err, module)
		return
	}
	var schemaIn, schemaOut *sch.Schema

	// Unmarshal image schemas if they were found.
	if jsonSchemaIn != "" {

		var structSchemaIn map[string]interface{}
		err = json.Unmarshal([]byte(jsonSchemaIn), &structSchemaIn)
		if err != nil {
			panic(err) // This should never happen because InferModuleProperties encodes this JSON.
		}

		schemaIn, err = sch.Load(structSchemaIn)
		if err != nil {
			err = errors.WithStack(err)
			context.moduleValidationError(err, module)
			return
		}
		module.SchemaIn = jsonSchemaIn

	}
	if jsonSchemaOut != "" {
		var structSchemaOut map[string]interface{}
		err = json.Unmarshal([]byte(jsonSchemaOut), &structSchemaOut)
		if err != nil {
			panic(err) // This should never happen because InferModuleProperties encodes this JSON.
		}

		schemaOut, err = sch.Load(structSchemaOut)
		if err != nil {
			err = errors.WithStack(err)
			context.moduleValidationError(err, module)
			return
		}
		module.SchemaOut = jsonSchemaOut
	}

	if module.Type == types.ModuleModel && (schemaIn == nil || schemaOut == nil) {
		err = errors.New("module of type model must have an input and output schema")
		err = errors.WithStack(err)
		context.moduleValidationError(err, module)
		return

	} else if module.Type == types.ModuleObjective && schemaIn == nil {
		err = errors.New("module of type objective must have an input schema")
		err = errors.WithStack(err)
		context.moduleValidationError(err, module)
		return
	}

	// Update the module.
	var updates = map[string]interface{}{
		"schema-in":    jsonSchemaIn,
		"schema-out":   jsonSchemaOut,
		"config-space": configSpace,
	}
	if module.Name == "" {
		updates["name"] = name
	}
	if module.Description == "" {
		updates["description"] = description
	}
	_, err = context.ModelContext.UpdateModule(module.ID, updates)
	if err != nil {
		err = errors.WithStack(err)
		context.moduleValidationError(err, module)
		return
	}

	// Update the status.
	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UpdateModuleStatus(module.ID, types.ModuleActive, "")
	})

	// If we are dealing with a new model, get all running jobs, look at their datasets and find ones
	// to which the model can be applied.
	if module.Type == types.ModuleModel {
		err = context.ModelContext.AddModelToApplicableJobs(module)
		if err != nil {
			err = errors.WithStack(err)
			context.moduleValidationError(err, module)
			return
		}
	}

	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UnlockModule(module.ID, context.ProcessID)
	})

	// Log task completion.
	context.Logger.WithFields(
		"module-id", module.ID,
		"source", module.Source,
		"source-address", module.SourceAddress,
	).WriteInfo("MODULE VALIDATION COMPLETED")
}

func (context Context) moduleValidationError(err error, module types.Module) {
	context.Logger.WithFields(
		"module-id", module.ID,
		"source", module.Source,
		"source-address", module.SourceAddress,
	).WithStack(err).WithError(err).WriteError("MODULE VALIDATION ERROR")

	context.repeatUntilSuccess(func() error {
		return context.ModelContext.UpdateModuleStatus(module.ID, types.ModuleError, err.Error())
	})
}
