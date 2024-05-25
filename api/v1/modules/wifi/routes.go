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

	wifiSubRouter.HandleFunc("/scanAp/{interfaceName}", scanApHandler)
	log.Println("[GET] /api/v1/modules/wifi/scanAp/{interfaceName}")

	wifiSubRouter.HandleFunc("/scanClient", scanClientHandler)
	log.Println("[GET] /api/v1/modules/wifi/scanClient")

	wifiSubRouter.HandleFunc("/deauth/{interfaceName}", deauthHandler)
	log.Println("[POST] /api/v1/modules/wifi/deauth/{interfaceName}")

	wifiSubRouter.HandleFunc("/connectAp/{interfaceName}", connectApHandler)
	log.Println("[POST] /api/v1/modules/wifi/connectAp/{interfaceName}")

	wifiSubRouter.HandleFunc("/cptHandshake/{interfaceName}", cptHandshakeHandler)
	log.Println("[GET] /api/v1/modules/wifi/cptHandshake/{interfaceName}")
}
