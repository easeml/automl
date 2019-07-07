package middleware

import (
	"github.com/ds3lab/easeml/engine/database/model"
	"net/http"

	"github.com/gorilla/context"
)

// Connection is the alias for the database connection struct.
//type Connection database.Connection

// ModelContext is the alias for the model context struct.
//type ModelContext model.Context

// Inject takes a database connection from the connection pool (performed by Copy)
// and injects it into the request context.
func (apiContext Context) Inject(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Clone mongo session. This translates to taking a connection from the connection pool.

		modelContextCopy := model.Context(apiContext.ModelContext).Clone()
		//modelContextCopy.Session = modelContextCopy.Session.Copy()
		defer modelContextCopy.Session.Close()

		context.Set(r, "modelContext", modelContextCopy)

		h.ServeHTTP(w, r)
	})
}
