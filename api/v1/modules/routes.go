package modules

import (
	wifi "mf-backend/api/v1/modules/wifi"

	"github.com/gorilla/mux"
)

// modules routes
func ModulesRoutes(baseRoute *mux.Router) {

	modulesSubRouter := baseRoute.PathPrefix("/modules").Subrouter()

	wifi.WifiRoutes(modulesSubRouter)
}
