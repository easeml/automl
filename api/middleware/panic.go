package middleware

import (
	"github.com/ds3lab/easeml/api/responses"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// PanicRecovery catches all panics that happen downstream, returns
// an HTTP 500 Internal Server Error and log the state without crashing the server.
func (apiContext Context) PanicRecovery(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			if err := recover(); err != nil {

				if e, ok := err.(error); ok == true {
					responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Internal server error.", e)
				} else {
					log.Printf("fatal: %+v", err)
					responses.Context(apiContext).RespondWithError(w, r, http.StatusInternalServerError, "Internal server error.", nil)
				}

			}
		}()

		h.ServeHTTP(w, r)
	})
}
