package handlers

import (
	"github.com/ds3lab/easeml/api/responses"
	"github.com/ds3lab/easeml/database/model"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/gorilla/mux"

	"github.com/gorilla/context"
)

// UsersGet returns all (visible) users.
func (apiContext Context) UsersGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Extract query parameters.
	query := r.URL.Query()
	idStr := query.Get("id")
	status := query.Get("status")
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
	if status != "" {
		filters["status"] = status
	}

	// Access model.
	result, cm, err := modelContext.GetUsers(filters, limit, cursor, orderBy, order)
	if errors.Cause(err) == model.ErrBadInput {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Parameters were wrong.", errors.WithStack(err))
		return
	} else if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// We never return the user's password.
	for i := 0; i < len(result); i++ {
		result[i].PasswordHash = ""
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
		query.Set("status", status)
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

// UsersPost creates a new user specified in the request body.
func (apiContext Context) UsersPost(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Parse body.
	var user model.User
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Invalid request payload.", errors.WithStack(err))
		return
	}
	defer r.Body.Close()

	// Access model.
	user, err := modelContext.CreateUser(user)
	if errors.Cause(err) == model.ErrUnauthorized {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusForbidden, "Unauthorized access.", errors.WithStack(err))
		return
	}
	if errors.Cause(err) == model.ErrIdentifierTaken {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusConflict, "Identifier taken.", errors.WithStack(err))
		return
	}
	if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Return response.
	var resourceURL = "http://" + r.Host + "/users/" + user.ID
	w.Header().Set("Location", resourceURL)
	w.WriteHeader(http.StatusCreated)
}

// UsersByIDGet returns a specific user by ID.
func (apiContext Context) UsersByIDGet(w http.ResponseWriter, r *http.Request) {

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

	// If the provided IS is "this" then replace it with the currently logged in user.
	if id == model.UserThis {
		id = modelContext.User.ID
	}

	// Access model.
	user, err := modelContext.GetUserByID(id)
	if errors.Cause(err) == model.ErrNotFound {
		// TODO: Go through all known errors such as "not found" and remove the stack trace. It is useless.
		responses.Context(apiContext).RespondWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.WithStack(err))
		return
	} else if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// We never return the user's password.
	user.PasswordHash = ""

	// Build the response.
	var response = map[string]interface{}{}
	response["data"] = user
	responses.RespondWithJSON(w, http.StatusOK, response)
}

// UsersByIDPatch updates fields of a specific user by ID.
func (apiContext Context) UsersByIDPatch(w http.ResponseWriter, r *http.Request) {

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
	if rawStatus, ok := patchBody["status"]; ok {
		var status string
		if err := json.Unmarshal(*rawStatus, &status); err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Body is not properly formatted JSON.", errors.WithStack(err))
			return
		}
		updates["status"] = status
	}
	if rawPassword, ok := patchBody["password"]; ok {
		var password string
		if err := json.Unmarshal(*rawPassword, &password); err != nil {
			responses.Context(apiContext).RespondWithError(w, r, http.StatusBadRequest, "Body is not properly formatted JSON.", errors.WithStack(err))
			return
		}
		updates["password"] = password
	}

	// Access model.
	_, err := modelContext.UpdateUser(id, updates)
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

// UsersLoginGet logs a user in given their credentials and returns an API key.
func (apiContext Context) UsersLoginGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Access model. We assume the user is authenticated with the authentication middleware.
	user, err := modelContext.UserLogin()
	if errors.Cause(err) == model.ErrNotPermitedForRoot {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), errors.WithStack(err))
		return
	}
	if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Return response.
	w.Header().Set("X-API-KEY", user.APIKey)
	w.WriteHeader(http.StatusOK)
}

// UsersLogoutGet logs a user out and erases their API key from the database.
func (apiContext Context) UsersLogoutGet(w http.ResponseWriter, r *http.Request) {

	// Get context variables.
	modelContext := context.Get(r, "modelContext").(model.Context)

	// Access model.
	err := modelContext.UserLogout()
	if errors.Cause(err) == model.ErrNotPermitedForRoot {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), errors.WithStack(err))
		return
	}
	if err != nil {
		responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Something went wrong.", errors.WithStack(err))
		return
	}

	// Return response.
	w.WriteHeader(http.StatusOK)
}
