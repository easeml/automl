package middleware

import (
	"github.com/ds3lab/easeml/engine/storage"
	"net/http"

	"github.com/gorilla/context"
)

// StorageContext is the alias for the storage context struct.
type StorageContext storage.Context

// Inject takes a database connection from the connection pool (performed by Copy)
// and injects it into the request context.
func (storageContext StorageContext) Inject(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Add the storageContext variable to the request context.
		context.Set(r, "storageContext", storage.Context(storageContext))

		h.ServeHTTP(w, r)
	})
}
