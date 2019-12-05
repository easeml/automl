package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ds3lab/easeml/engine/api/responses"
	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/tus/tusd"
	"github.com/tus/tusd/filestore"
)

// ModulesGet returns all (visible) modules.
func (apiContext Context) ModulesGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Extract query parameters.
	query := r.URL.Query()
	idStr := query.Get("id")
	user := query.Get("user")
	status := query.Get("status")
	moduleType := query.Get("type")
	label := query.Get("label")
	source := query.Get("source")
	schemaIn := query.Get("schema-in")
	schemaOut := query.Get("schema-out")
	cursor := query.Get("cursor")
	limitStr := query.Get("limit")
	orderBy := query.Get("order-by")
	order := query.Get("order")

	// Parse non-string parametes.
	id := []string{}
	if idStr != "" {
		id = strings.Split(idStr, ",")
	}
	limit := 20
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "The 'limit' parameter is not a valid integer.", errors.WithStack(err))
			return
		}
		if limit < 1 {
			limit = 1
		}
		if limit > 100 {
			limit = 100
		}
	}

	// Build the filters map.
	var filters = map[string]interface{}{}
	if len(id) > 0 {
		filters["id"] = id
	}
	if user != "" {
		filters["user"] = user
	}
	if status != "" {
		filters["status"] = status
	}
	if moduleType != "" {
		filters["type"] = moduleType
	}
	if label != "" {
		filters["label"] = label
	}
	if source != "" {
		filters["source"] = source
	}
	if schemaIn != "" {
		filters["schema-in"] = schemaIn
	}
	if schemaOut != "" {
		filters["schema-out"] = schemaOut
	}

	// Access model.
	result, cm, err := modelContext.GetModules(filters, limit, cursor, orderBy, order)
	if errors.Cause(err) == model.ErrBadInput {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Parameters were wrong.", errors.WithStack(err))
		return
	} else if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Build the response.
	var response = map[string]interface{}{}
	response["data"] = result
	response["metadata"] = cm

	// Add the next link.
	response["links"] = map[string]interface{}{
		"next": nil,
	}
	if cm.NextPageCursor != "" {
		query = url.Values{}
		query.Set("id", strings.Join(id, ","))
		query.Set("user", user)
		query.Set("status", status)
		query.Set("type", moduleType)
		query.Set("label", label)
		query.Set("source", source)
		query.Set("schema-in", schemaIn)
		query.Set("schema-out", schemaOut)
		query.Set("limit", strconv.Itoa(limit))
		query.Set("order-by", orderBy)
		query.Set("order", order)
		query.Set("cursor", cm.NextPageCursor)
		for key := range query {
			if query.Get(key) == "" {
				query.Del(key)
			}
		}
		nextURL := url.URL{
			Scheme:   "http",
			Host:     r.Host,
			Path:     r.URL.Path,
			RawQuery: query.Encode(),
		}
		response["links"].(map[string]interface{})["next"] = nextURL.String()
	}

	// Return response.
	responses.RespondWithJSON(w, http.StatusOK, response)
}

// ModulesPost creates a new module specified in the request body.
func (apiContext Context) ModulesPost(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Parse body.
	var module types.Module
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&module); err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Invalid request payload.", errors.WithStack(err))
		return
	}
	defer r.Body.Close()

	// Access model.
	module, err := modelContext.CreateModule(module)
	if errors.Cause(err) == types.ErrUnauthorized {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusForbidden, "Unauthorized access.", errors.WithStack(err))
		return
	}
	if errors.Cause(err) == types.ErrIdentifierTaken {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusConflict, "Identifier taken.", errors.WithStack(err))
		return
	}
	if errors.Cause(err) == model.ErrBadInput {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Bad input parameters.", errors.WithStack(err))
		return
	}
	if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Return response.
	var resourceURL = "http://" + r.Host + "/modules/" + module.ID
	w.Header().Set("Location", resourceURL)
	w.WriteHeader(http.StatusCreated)
}

// ModulesByIDGet returns a specific module by ID.
func (apiContext Context) ModulesByIDGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Get path parameters. Format the ID as user-id/module-id.
	vars := mux.Vars(r)
	userID := vars["user-id"]
	id := vars["id"]

	// Validate parameters.
	if id == "" {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "The 'id' parameter is required.", nil)
		return
	}
	id = fmt.Sprintf("%s/%s", userID, id)

	// Access model.
	module, err := modelContext.GetModuleByID(id)
	if errors.Cause(err) == model.ErrNotFound {
		// TODO: It is common for clients to query if a module exists, this is not always an error, i.e. before adding a module, perhaps add HEAD request
		responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), err)
		return
	} else if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Build the response.
	var response = map[string]interface{}{}
	response["data"] = module
	responses.RespondWithJSON(w, http.StatusOK, response)
}

// ModulesByIDPatch updates fields of a specific module by ID.
func (apiContext Context) ModulesByIDPatch(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Get path parameters. Format the ID as user-id/module-id.
	vars := mux.Vars(r)
	userID := vars["user-id"]
	id := vars["id"]

	// Validate parameters.
	if id == "" {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "The 'id' parameter is required.", nil)
		return
	}
	id = fmt.Sprintf("%s/%s", userID, id)

	// Parse body.
	var patchBody map[string]*json.RawMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&patchBody); err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Invalid request payload", errors.WithStack(err))
		return
	}
	defer r.Body.Close()

	// Parse specific fields from the body.
	var updates = map[string]interface{}{}
	if rawName, ok := patchBody["name"]; ok {
		var name string
		if err := json.Unmarshal(*rawName, &name); err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Body is not properly formatted JSON.", errors.WithStack(err))
			return
		}
		updates["name"] = name
	}
	if rawDescription, ok := patchBody["description"]; ok {
		var description string
		if err := json.Unmarshal(*rawDescription, &description); err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Body is not properly formatted JSON.", errors.WithStack(err))
			return
		}
		updates["description"] = description
	}
	if rawStatus, ok := patchBody["status"]; ok {
		var status string
		if err := json.Unmarshal(*rawStatus, &status); err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Body is not properly formatted JSON.", errors.WithStack(err))
			return
		}
		updates["status"] = status
	}

	// Access model.
	_, err := modelContext.UpdateModule(id, updates)
	if errors.Cause(err) == model.ErrNotFound {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.WithStack(err))
		return
	} else if errors.Cause(err) == model.ErrBadInput {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), errors.WithStack(err))
		return
	} else if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Return response.
	w.WriteHeader(http.StatusOK)
}

// ModulesUploadHandler handles all data upload related requests.
func (apiContext Context) ModulesUploadHandler(basePath string) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Get context variables.
		modelContext := context.Get(r, "modelContext").(model.Context)

		// Get path parameters. Format the ID as user-id/module-id.
		vars := mux.Vars(r)
		userID := vars["user-id"]
		moduleID := vars["module-id"]

		// Build base path.
		myBasePath := basePath
		myBasePath = strings.Replace(myBasePath, "{user-id}", userID, 1)
		myBasePath = strings.Replace(myBasePath, "{module-id}", moduleID, 1)

		// Validate parameters.
		if moduleID == "" {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "The 'module-id' parameter is required.", nil)
			return
		}
		id := fmt.Sprintf("%s/%s", userID, moduleID)

		// Access model.
		module, err := modelContext.GetModuleByID(id)
		if errors.Cause(err) == model.ErrNotFound {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.WithStack(err))
			return
		} else if err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
			return
		}

		// Uploading is permitted only if the module source is "upload".
		if module.Source != types.ModuleUpload {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), nil)
			return
		}

		// Changes to the module are permitted only if it is in the "created" state.
		if module.Status != types.ModuleCreated {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusForbidden, "The module is read-only.", nil)
			return
		}

		// Get the data directory of the module while ensuring it exists.
		dataPath, err := apiContext.StorageContext.GetModulePath(id, module.Type, ".upload")
		if err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
			return
		}

		store := filestore.FileStore{
			Path: dataPath,
		}
		composer := tusd.NewStoreComposer()
		store.UseIn(composer)

		config := tusd.Config{
			BasePath:      myBasePath,
			StoreComposer: composer,
		}

		handler, err := tusd.NewUnroutedHandler(config)
		if err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
			return
		}

		switch r.Method {
		case "PATCH":
			handler.PatchFile(w, r)
		case "HEAD":
			handler.HeadFile(w, r)
		case "POST":
			handler.PostFile(w, r)
			//http.StripPrefix(myBasePath, http.HandlerFunc(handler.PostFile)).ServeHTTP(w, r)
		}
	})
}
