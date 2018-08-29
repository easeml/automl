package responses

import (
	"github.com/ds3lab/easeml/api"
	"github.com/ds3lab/easeml/database/model"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/context"
	"github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// Context is the placeholder for the common API context struct.
type Context api.Context

// RespondWithError writes out a JSON error response. The header of the response writer
// can be initialized before calling this function.
func (apiContext Context) RespondWithError(w http.ResponseWriter, r *http.Request, code int, message string, err error) {

	// Try to extract the request ID.
	var requestID string
	if value, ok := context.GetOk(r, "request-id"); ok {
		requestID = value.(string)
	}
	var userID string
	if value, ok := context.GetOk(r, "modelContext"); ok {
		userID = value.(model.Context).User.ID
	}

	// Write error information to log (including the stack trace if available).
	apiContext.Logger.WithFields("request-id", requestID, "user-id", userID).WithError(err).WithStack(err).WriteError(fmt.Sprintf("ERROR: %s", message))
	/*if err != nil {
		syslog.Printf("Error: %+v", err)
	}*/
	/*if err, ok := err.(stackTracer); ok {
		io.WriteString(apiContext.Logger.Out, "\nStack Trace:\n")
		for _, f := range err.StackTrace() {
			io.WriteString(apiContext.Logger.Out, fmt.Sprintf("%+v\n", f))
		}
		io.WriteString(apiContext.Logger.Out, "\n")
	}*/

	RespondWithJSON(w, code, map[string]interface{}{"code": code, "error": message, "request-id": requestID})
}

// RespondWithJSON writes out a JSON response. The header of the response writer
// can be initialized before calling this function.
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {

	response, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(response)
}
