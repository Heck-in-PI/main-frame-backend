package wifi

import (
	"log"

	"github.com/gorilla/mux"
)

// Wifi module routes
func WifiRoutes(baseRoute *mux.Router) {

	wifiSubRouter := baseRoute.PathPrefix("/wifi").Subrouter()

	wifiSubRouter.HandleFunc("/viewer", wifiViewer)
	log.Println("[WS] /api/v1/modules/wifi/viewer")

	wifiSubRouter.HandleFunc("/interfaces", interfacesHandler)
	log.Println("[GET] /api/v1/modules/wifi/interfaces")

	wifiSubRouter.HandleFunc("/scanAp/{interfaceName}", scanApHandler)
	log.Println("[GET] /api/v1/modules/wifi/scanAp/{interfaceName}")

	wifiSubRouter.HandleFunc("/scanClient", scanClientHandler)
	log.Println("[GET] /api/v1/modules/wifi/scanClient")

	wifiSubRouter.HandleFunc("/deauth", deauthHandler)
	log.Println("[POST] /api/v1/modules/wifi/deauth")

	wifiSubRouter.HandleFunc("/connectAp/{interfaceName}", connectApHandler)
	log.Println("[POST] /api/v1/modules/wifi/connectAp/{interfaceName}")

	wifiSubRouter.HandleFunc("/cptHandshake", cptHandshakeHandler)
	log.Println("[GET] /api/v1/modules/wifi/cptHandshake")

	wifiSubRouter.HandleFunc("/probe", probeHandler)
	log.Println("[POST] /api/v1/modules/wifi/probe")

	wifiSubRouter.HandleFunc("/beacon", beaconHandler)
	log.Println("[POST] /api/v1/modules/wifi/beacon")

	wifiSubRouter.HandleFunc("/rogueAp", rogueApHandler)
	log.Println("[POST] /api/v1/modules/wifi/rogueAp")

	wifiSubRouter.HandleFunc("/stop", stopHandler)
	log.Println("[GET] /api/v1/modules/wifi/stop")

	wifiSubRouter.HandleFunc("/stopScanClient", stopScanClientHandler)
	log.Println("[GET] /api/v1/modules/wifi/stopScanClient")

	wifiSubRouter.HandleFunc("/stopCptHandshake", stopCptHandshakeHandler)
	log.Println("[GET] /api/v1/modules/wifi/stopCptHandshake")

	wifiSubRouter.HandleFunc("/stopBeaconer", stopBeaconerHandler)
	log.Println("[GET] /api/v1/modules/wifi/stopBeaconer")

	wifiSubRouter.HandleFunc("/stopRogueAp", stopRogueApHandler)
	log.Println("[GET] /api/v1/modules/wifi/stopRogueAp")
}
