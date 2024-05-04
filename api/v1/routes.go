package v1

import (
	modules "mf-backend/api/v1/modules"

	"github.com/gorilla/mux"
)

// v1 version routes
func V1Routes(baseRoute *mux.Router) {

	v1SubRouter := baseRoute.PathPrefix("/v1").Subrouter()

	modules.ModulesRoutes(v1SubRouter)
}
