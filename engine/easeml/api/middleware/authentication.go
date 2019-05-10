package middleware

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/ds3lab/easeml/engine/easeml/database/model"
	"github.com/ds3lab/easeml/engine/easeml/database/model/types"

	"github.com/gorilla/context"
)

const (
	basicSchema         string = "Basic "
	bearerSchema        string = "Bearer "
	authorizationHeader string = "Authorization"
	apiKeyHeader        string = "X-API-KEY"
	apiKeyQuery         string = "api-key"
)

// Authenticate reads the authentication headers and attempts to authenticate the user.
// If no credentials are specified, the session is continued assuming an anonimus user.
// If the credentials are wrong, then a HTTP 401 error response is returned.
func (apiContext Context) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// The default user is anonimus.
		user := types.GetAnonUser()

		// Get connection from context.
		//var connection Connection
		//if value := context.Get(r, "connection"); value == nil {
		// If a connection was not defined then we cannot continue.
		//	panic("context has no connection")
		//} else {
		//	connection = value.(Connection)
		//}

		// Build model context which is used to access the database.
		//modelContext := model.Context{Session: connection.Session, DBName: connection.DBName, User: user}
		modelContext := context.Get(r, "modelContext").(model.Context)

		// We first check if there is an API key.
		// We support three possible places to store it: header, query or bearer token.
		user.APIKey = r.Header.Get(apiKeyHeader)

		if user.APIKey == "" {
			query := r.URL.Query()
			user.APIKey = query.Get(apiKeyQuery)
		}

		if user.APIKey == "" {

			authHeader := r.Header.Get(authorizationHeader)

			if authHeader != "" {

				if strings.HasPrefix(authHeader, basicSchema) {

					decodedString, err := base64.StdEncoding.DecodeString(authHeader[len(basicSchema):])

					if err != nil {
						// There was a problem in decoding the base64 string.
						http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
						return
					}

					// We extract the credentials which are in the form id:raw_password and manually hash the password.
					credentials := strings.Split(string(decodedString), ":")
					user.ID = credentials[0]
					hasher := sha256.New()
					hasher.Write([]byte(credentials[1]))
					user.PasswordHash = hex.EncodeToString(hasher.Sum(nil))

				} else if strings.HasPrefix(authHeader, bearerSchema) {

					user.APIKey = authHeader[len(bearerSchema):]

				} else {

					// The Authorization header is invalid.
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return

				}
			}
		}

		// If we have found an API key we will attempt to authenticate with it.
		if user.APIKey != "" || user.PasswordHash != "" {
			var err error
			user, err = modelContext.UserAuthenticate(user)
			if err == types.ErrWrongAPIKey || err == types.ErrWrongCredentials {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			} else if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}

		// We add the user to the context and continue the the handler chain.
		modelContext.User = user
		context.Set(r, "modelContext", modelContext)
		next.ServeHTTP(w, r)
		return
	})
}
