package api

import (
	v1 "mf-backend/api/v1"

	"github.com/gorilla/mux"
)

// api routes
func ApiRoutes(baseRoute *mux.Router) {

	apiSubRouter := baseRoute.PathPrefix("/api").Subrouter()

	v1.V1Routes(apiSubRouter)
}
