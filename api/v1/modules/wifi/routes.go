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

	wifiSubRouter.HandleFunc("/scanap", scanApHandler)
	log.Println("[POST] /api/v1/modules/wifi/scanap")

	wifiSubRouter.HandleFunc("/deauth", deauthHandler)
	log.Println("[POST] /api/v1/modules/wifi/deauth")
}
