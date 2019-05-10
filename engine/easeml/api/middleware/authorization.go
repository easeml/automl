package middleware

import (
	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"net/http"

	"github.com/gorilla/context"
)

// DisallowAuth does not allow authenticated users to access the page
func (apiContext Context) DisallowAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		modelContext := context.Get(r, "modelContext").(model.Context)
		if modelContext.User.IsAnon() == false {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// DisallowAnon does not allow anonymous users to access the page
func (apiContext Context) DisallowAnon(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		modelContext := context.Get(r, "modelContext").(model.Context)
		if modelContext.User.IsAnon() {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// HideFromAnon returns a HTTP 404 Not Found response for anonymous users.
func (apiContext Context) HideFromAnon(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		modelContext := context.Get(r, "modelContext").(model.Context)
		if modelContext.User.IsAnon() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		h.ServeHTTP(w, r)
	})
}
