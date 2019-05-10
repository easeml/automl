package handlers

import (
	"github.com/ds3lab/easeml/engine/easeml/api/responses"
	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// ProcessesGet returns all (visible) processes.
func (apiContext Context) ProcessesGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Extract query parameters.
	query := r.URL.Query()
	idStr := query.Get("id")
	status := query.Get("status")
	procType := query.Get("type")
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
	if procType != "" {
		filters["type"] = procType
	}
	if status != "" {
		filters["status"] = status
	}

	// Access model.
	result, cm, err := modelContext.GetProcesses(filters, limit, cursor, orderBy, order)
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
		query.Set("status", status)
		query.Set("type", procType)
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

// ProcesssesByIDGet returns a specific process by ID.
func (apiContext Context) ProcesssesByIDGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Get path parameters.
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
	process, err := modelContext.GetProcessByID(bson.ObjectIdHex(id))
	if errors.Cause(err) == model.ErrNotFound {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.WithStack(err))
		return
	} else if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Build the response.
	var response = map[string]interface{}{}
	response["data"] = process
	responses.RespondWithJSON(w, http.StatusOK, response)
}
