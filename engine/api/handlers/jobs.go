package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ds3lab/easeml/engine/api/responses"
	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// JobsGet returns all (visible) jobs.
func (apiContext Context) JobsGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Extract query parameters.
	query := r.URL.Query()
	idStr := query.Get("id")
	user := query.Get("user")
	dataset := query.Get("dataset")
	jobModel := query.Get("model")
	objective := query.Get("objective")
	status := query.Get("status")
	schema := query.Get("schema")
	cursor := query.Get("cursor")
	limitStr := query.Get("limit")
	orderBy := query.Get("order-by")
	order := query.Get("order")

	// Parse non-string parametes.
	id := []bson.ObjectId{}
	idSlice := []string{}
	if idStr != "" {
		idSlice = strings.Split(idStr, ",")
		id = make([]bson.ObjectId, len(idSlice), len(idSlice))
		for i := range idSlice {
			id[i] = bson.ObjectIdHex(idSlice[i])
		}
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
	if dataset != "" {
		filters["dataset"] = dataset
	}
	if jobModel != "" {
		filters["model"] = jobModel
	}
	if objective != "" {
		filters["objective"] = objective
	}
	if status != "" {
		filters["status"] = status
	}
	if schema != "" {
		filters["schema"] = schema
	}

	// Access model.
	result, cm, err := modelContext.GetJobs(filters, limit, cursor, orderBy, order)
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
		query.Set("id", strings.Join(idSlice, ","))
		query.Set("user", user)
		query.Set("dataset", dataset)
		query.Set("model", jobModel)
		query.Set("objective", objective)
		query.Set("status", status)
		query.Set("schema", schema)
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

// JobsPost creates a new job specified in the request body.
func (apiContext Context) JobsPost(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Parse body.
	var job types.Job
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&job); err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Invalid request payload.", errors.WithStack(err))
		return
	}
	defer r.Body.Close()

	// Access model.
	job, err := modelContext.CreateJob(job)
	if errors.Cause(err) == types.ErrUnauthorized {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusForbidden, "Unauthorized access.", errors.WithStack(err))
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
	var resourceURL = "http://" + r.Host + "/jobs/" + string(job.ID.Hex())
	w.Header().Set("Location", resourceURL)
	w.WriteHeader(http.StatusCreated)
}

// JobsByIDGet returns a specific job by ID.
func (apiContext Context) JobsByIDGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Get path parameters. Format the ID as user-id/module-id.
	vars := mux.Vars(r)
	id := vars["id"]

	// Validate parameters.
	if id == "" {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "The 'id' parameter is required.", nil)
		return
	}
	if bson.IsObjectIdHex(id) == false {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), model.ErrNotFound)
		return
	}

	// Access model.
	job, err := modelContext.GetJobByID(bson.ObjectIdHex(id))
	if errors.Cause(err) == model.ErrNotFound {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.WithStack(err))
		return
	} else if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Build the response.
	var response = map[string]interface{}{}
	response["data"] = job
	responses.RespondWithJSON(w, http.StatusOK, response)
}

// JobsByIDPatch updates fields of a specific job by ID.
func (apiContext Context) JobsByIDPatch(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Get path parameters. Format the ID as user-id/job-id.
	vars := mux.Vars(r)
	id := vars["id"]

	// Validate parameters.
	if id == "" {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "The 'id' parameter is required.", nil)
		return
	}

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
	if rawModels, ok := patchBody["models"]; ok {
		var models []string
		if err := json.Unmarshal(*rawModels, &models); err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Body is not properly formatted JSON.", errors.WithStack(err))
			return
		}
		updates["models"] = models
	}
	if rawAcceptNewModels, ok := patchBody["accept-new-models"]; ok {
		var acceptNewModels bool
		if err := json.Unmarshal(*rawAcceptNewModels, &acceptNewModels); err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Body is not properly formatted JSON.", errors.WithStack(err))
			return
		}
		updates["accept-new-models"] = acceptNewModels
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
	_, err := modelContext.UpdateJob(bson.ObjectIdHex(id), updates)
	if errors.Cause(err) == model.ErrNotFound {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.WithStack(err))
		return
	} else if errors.Cause(err) == model.ErrBadInput {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), errors.WithStack(err))
		return
	} else if errors.Cause(err) == types.ErrUnauthorized {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), errors.WithStack(err))
		return
	} else if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Return response.
	w.WriteHeader(http.StatusOK)
}
