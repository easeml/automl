package middleware

import (
	"net/http"
	"time"

	"github.com/ds3lab/easeml/engine/easeml/database/model"

	"github.com/gorilla/context"
)

// Logging records all incoming requests to the log. Only in debug trace mode.
func (apiContext Context) Logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		startTime := time.Now()

		h.ServeHTTP(w, r)

		// Try to extract the request ID.
		var requestID string
		if value, ok := context.GetOk(r, "request-id"); ok {
			requestID = value.(string)
		}
		var userID string
		if value, ok := context.GetOk(r, "modelContext"); ok {
			userID = value.(model.Context).User.ID
		}

		duration := time.Now().Sub(startTime)

		apiContext.Logger.WithFields(
			"request-id", requestID,
			"user-id", userID,
			"duration", duration,
			"request-url", r.URL.String(),
			"host", r.Host,
			"method", r.Method,
		).WriteDebug("API REQUEST COMPLETED")

	})
}
