package middleware

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/gorilla/context"
)

// RequestID adds a request ID to the context so that we can track errors based on it.
func (apiContext Context) RequestID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.NewV4()
		if err != nil {
			err = errors.Wrap(err, "uuid new failed")
			panic(err)
		}
		context.Set(r, "request-id", id.String())
		h.ServeHTTP(w, r)
	})
}
