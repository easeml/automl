// Package server plugs the api and the static content routers into itself and serves them.
package server

// Server represents a HTTP server which serves the REST API and static content.
type Server struct {
	DatabaseInstance string
	DatabaseName     string
	Host             string
	Port             int
	RootUserAPIKey   string
}

// Initialize initalizes the server.
func (server Server) Initialize() (err error) {
	// Establish a connection.

	// Generate API key.

	// Build mux router.

	return
}

// Start starts the server.
func (server Server) Start() (err error) {
	return
}

// Shutdown shits the server down.
func (server Server) Shutdown() (err error) {
	return
}
