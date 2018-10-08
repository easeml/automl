package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ds3lab/easeml/api/responses"
	"github.com/ds3lab/easeml/database/model"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// TasksGet returns all (visible) tasks.
func (apiContext Context) TasksGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Extract query parameters.
	query := r.URL.Query()
	idStr := query.Get("id")
	job := query.Get("job")
	user := query.Get("user")
	process := query.Get("process")
	dataset := query.Get("dataset")
	taskModel := query.Get("model")
	objective := query.Get("objective")
	status := query.Get("status")
	schema := query.Get("schema")
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
	if job != "" {
		filters["job"] = bson.ObjectIdHex(job)
	}
	if process != "" {
		filters["process"] = process
	}
	if dataset != "" {
		filters["dataset"] = dataset
	}
	if taskModel != "" {
		filters["model"] = taskModel
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
	result, cm, err := modelContext.GetTasks(filters, limit, cursor, orderBy, order)
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
		query.Set("job", job)
		query.Set("dataset", dataset)
		query.Set("process", process)
		query.Set("model", taskModel)
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

// TasksByIDGet returns a specific task by ID.
func (apiContext Context) TasksByIDGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Get path parameters. Format the ID as user-id/module-id.
	vars := mux.Vars(r)
	taskID := vars["job-id"]
	id := vars["id"]

	// Validate parameters.
	if id == "" {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "The 'id' parameter is required.", nil)
		return
	}
	id = fmt.Sprintf("%s/%s", taskID, id)

	// Access model.
	task, err := modelContext.GetTaskByID(id)
	if errors.Cause(err) == model.ErrNotFound {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.WithStack(err))
		return
	} else if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Build the response.
	var response = map[string]interface{}{}
	response["data"] = task
	responses.RespondWithJSON(w, http.StatusOK, response)
}

// TaskPredictionsDownloadHandler handles all task prediction download requests.
func (apiContext Context) TaskPredictionsDownloadHandler(basePath string) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Get context variables.
		modelContext := context.Get(r, "modelContext").(model.Context)

		// Get path parameters. Format the ID as job-id/task-id.
		vars := mux.Vars(r)
		jobID := vars["job-id"]
		taskID := vars["task-id"]

		// Build base path.
		myBasePath := basePath
		myBasePath = strings.Replace(myBasePath, "{job-id}", jobID, 1)
		myBasePath = strings.Replace(myBasePath, "{task-id}", taskID, 1)

		// Validate parameters.
		if taskID == "" {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "The 'task-id' parameter is required.", nil)
			return
		}
		id := fmt.Sprintf("%s/%s", jobID, taskID)

		// Extract the relative path.
		if strings.HasPrefix(r.RequestURI, myBasePath) == false {
			panic(fmt.Sprintf("the request URI '%s' does not match basePath '%s'", r.RequestURI, myBasePath))
		}
		relativePath := r.RequestURI[len(myBasePath):]

		// Access model.
		task, err := modelContext.GetTaskByID(id)
		if errors.Cause(err) == model.ErrNotFound {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.WithStack(err))
			return
		} else if err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
			return
		}

		// Predictions access is permitted only if the task status is "completed".
		if task.Status != model.TaskCompleted {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), nil)
			return
		}

		// Get the data directory of the task while ensuring it exists.
		taskPaths, err := apiContext.StorageContext.GetAllTaskPaths(id)
		if err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
			return
		}
		apiContext.ServeLocalResource(taskPaths.Predictions, relativePath, task.StageTimes.Predicting.End, w, r)

	})

}
