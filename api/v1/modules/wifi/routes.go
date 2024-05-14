package wifi

import (
	"log"

	"github.com/gorilla/mux"
)

// Wifi module routes
func WifiRoutes(baseRoute *mux.Router) {

	wifiSubRouter := baseRoute.PathPrefix("/wifi").Subrouter()

	wifiSubRouter.HandleFunc("/interfaces", interfacesHandler)
	log.Println("[GET] /api/v1/modules/wifi/interface")

	wifiSubRouter.HandleFunc("/scanap/{interfaceName}", scanApHandler)
	log.Println("[POST] /api/v1/modules/wifi/scanap/{interfaceName}")

	wifiSubRouter.HandleFunc("/deauth", deauthHandler)
	log.Println("[POST] /api/v1/modules/wifi/deauth")

	wifiSubRouter.HandleFunc("/connectAp/{interfaceName}", connectApHandler)
	log.Println("[POST] /api/v1/modules/wifi/connectAp/{interfaceName}")
}
