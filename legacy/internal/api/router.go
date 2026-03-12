package api

import "github.com/gorilla/mux"

// NewRouter creates a new gorilla/mux router with common middleware
func NewRouter() *mux.Router {
	r := mux.NewRouter()
	
	// Add common middleware here if needed
	// r.Use(loggingMiddleware)
	// r.Use(corsMiddleware)
	
	return r
}