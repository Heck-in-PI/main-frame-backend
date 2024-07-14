package wifi

import (
	"encoding/json"
	"io"
	"log"
	wifi_common "mf-backend/api/v1/modules/wifi/common"
	v1_common "mf-backend/api/v1/v1Common"
	"net"
	"strings"
	"time"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	wifi "github.com/mdlayher/wifi"
)

var WifiModule *wifi_common.WiFiModule

func wifiViewer(resp http.ResponseWriter, req *http.Request) {

	var upgrader = websocket.Upgrader{}
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(resp, req, nil)
	if err != nil {

		errorMessage := v1_common.ErrorMessage{
			Error: err.Error(),
		}

		v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

		return
	}

	defer conn.Close()
	log.Println("running")
	// Continuosly read and write message
	for {

		message, err := json.Marshal(WifiModule.GetAps())
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		log.Println(string(message))

		err = conn.WriteMessage(1, message)
		if err != nil {
			log.Println("write failed:", err)
			break
		}

		if _, _, err := conn.NextReader(); err != nil {
			conn.Close()
			break
		}

		time.Sleep(1 * time.Second)
	}
	log.Println("stopped")
}

// interfaces handler
//
//	@Summary		List wireless interfaces
//	@Description	Interfaces path will list device wirless interfaces, think of it as iwconfig.
//	@Tags			Wifi
//	@Produce		json
//	@Success		200	{array}		wifi_common.WirelessInterface
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Failure		500	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/interfaces [get]
func interfacesHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		wifiClient, err := wifi.New()
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		ifaces, err := wifiClient.Interfaces()
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		var wirelessInterfaces []wifi_common.WirelessInterface
		for _, iface := range ifaces {
			wirelessInterface := wifi_common.WirelessInterface{
				Index:        iface.Index,
				Name:         iface.Name,
				HardwareAddr: iface.HardwareAddr.String(),
				PHY:          iface.PHY,
				Device:       iface.Device,
				Type:         iface.Type,
				Frequency:    iface.Frequency,
			}

			wirelessInterfaces = append(wirelessInterfaces, wirelessInterface)
		}

		v1_common.JsonResponceHandler(resp, http.StatusOK, wirelessInterfaces)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// scan access point handler
//
//	@Summary		List access points
//	@Description	ScanAP path will put wireless interface in monitor mode and capture packets and filter for access point.
//	@Tags			Wifi
//	@Produce		json
//	@Param			interfaceName	path	string	true	"interface name"
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/scanAp/{interfaceName} [get]
func scanApHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		if WifiModule != nil {
			errorMessage := v1_common.ErrorMessage{
				Error: "ap scanner already running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		muxVars := mux.Vars(req)
		interfaceName := muxVars["interfaceName"]
		if interfaceName == "" {
			errorMessage := v1_common.ErrorMessage{
				Error: "interface name must be specified in path",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		var err error
		WifiModule, err = wifi_common.NewWiFiModule(interfaceName)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		err = WifiModule.Start()
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		retried := false
		for retry := 0; ; retry++ {

			if WifiModule.PktSourceChan != nil && len(WifiModule.PktSourceChan) != 0 {
				go WifiModule.AccessPointPacketAnalyzer()
				break
			} else if retried {
				err = WifiModule.ForcedStop()
				if err != nil {
					errorMessage := v1_common.ErrorMessage{
						Error: err.Error(),
					}

					v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

					return
				}

				errorMessage := v1_common.ErrorMessage{
					Error: "can't get packets",
				}

				v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

				return
			} else {
				log.Println("cant find packet retry")
				time.Sleep(1 * time.Second)
				retried = true
			}
		}

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// scan client point handler
//
//	@Summary		List clients of access points
//	@Description	ScanClient path will use the wireless interface in monitor mode and capture packets and filter for the already found access point's client.
//	@Tags			Wifi
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/scanClient [get]
func scanClientHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		if WifiModule == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "ap scanner must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		if wifi_common.ScanClientChanel != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "client scanning already running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		retried := false
		for retry := 0; ; retry++ {

			if WifiModule.PktSourceChan != nil { // && len(WifiModule.PktSourceChan) != 0 {
				wifi_common.ScanClientChanel = make(chan bool)
				go WifiModule.DiscoverClientAnalyzer()
				break
			} else if retried {

				errorMessage := v1_common.ErrorMessage{
					Error: "can't get packets",
				}

				v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

				return
			} else {
				log.Println("cant find packet retry")
				time.Sleep(1 * time.Second)
				retried = true
			}
		}

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// death handler
//
//	@Summary		Deauth client
//	@Description	Deauth path will deauthenticate user from an access point.
//	@Tags			Wifi
//	@Accept			json
//	@Produce		json
//	@Param			request	body	wifi_common.Deauther	true	"deauth info"
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Failure		500	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/deauth [post]
func deauthHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "POST" {

		var deauther wifi_common.Deauther

		body, _ := io.ReadAll(req.Body)
		err := json.Unmarshal(body, &deauther)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		if WifiModule == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "ap scanner must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		bssid, err := net.ParseMAC(deauther.ApMac)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		client, err := net.ParseMAC(deauther.ClientMac)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		// set wifi to monitor mode
		err = WifiModule.Configure()
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		log.Println("kicking out from:", bssid, ", client: ", client)
		WifiModule.SendDeauthPacket(bssid, client)

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// connect to access point handler
//
//	@Summary		Connect access point
//	@Description	ConnectAp path will connect you to an access point.
//	@Tags			Wifi
//	@Accept			json
//	@Produce		json
//	@Param			request			body	wifi_common.ConnectAp	true	"ap info"
//	@Param			interfaceName	path	string					true	"interface name"
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Failure		500	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/connectAp/{interfaceName} [post]
func connectApHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "POST" {

		muxVars := mux.Vars(req)
		interfaceName := muxVars["interfaceName"]
		if interfaceName == "" {
			errorMessage := v1_common.ErrorMessage{
				Error: "interface name must be specified in path",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		var connectAp wifi_common.ConnectAp

		body, _ := io.ReadAll(req.Body)
		err := json.Unmarshal(body, &connectAp)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		output, err := wifi_common.ConnectNetwork(interfaceName, connectAp.ApName, connectAp.ApPass)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		if strings.Contains(output, "successfully") {

			v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
			return
		} else {

			errorMessage := v1_common.ErrorMessage{
				Error: output,
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)
			return
		}

	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// capture handshake handler
//
//	@Summary		Capture handshakes
//	@Description	CaptureHandshake path will use the wireless interface in monitor mode and capture packets and filter handshakes all over the flore.
//	@Tags			Wifi
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/cptHandshake [get]
func cptHandshakeHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		if WifiModule == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "ap scanner must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		retried := false
		for retry := 0; ; retry++ {

			if WifiModule.PktSourceChan != nil { // && len(WifiModule.PktSourceChan) != 0 {
				wifi_common.CptHandshakeHandlerChanel = make(chan bool)
				go WifiModule.DiscoverHandshakeAnalyzer()
				break
			} else if retried {
				err := WifiModule.ForcedStop()
				if err != nil {
					errorMessage := v1_common.ErrorMessage{
						Error: err.Error(),
					}

					v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

					return
				}

				errorMessage := v1_common.ErrorMessage{
					Error: "can't get packets",
				}

				v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

				return
			} else {
				log.Println("cant find packet retry")
				time.Sleep(1 * time.Second)
				retried = true
			}
		}

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// probe handler
//
//	@Summary		Probe attack
//	@Description	Probe path will send a fake client probe with the given station BSSID, searching for ESSID.
//	@Tags			Wifi
//	@Accept			json
//	@Produce		json
//	@Param			request	body	wifi_common.Prober	true	"probe info"
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Failure		500	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/probe [post]
func probeHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "POST" {

		var prober wifi_common.Prober

		body, _ := io.ReadAll(req.Body)
		err := json.Unmarshal(body, &prober)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		if WifiModule == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "ap scanner must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		bssid, err := net.ParseMAC(prober.ApMac)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		// set wifi to monitor mode
		err = WifiModule.Configure()
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		WifiModule.SendProbePacket(bssid, prober.ApName)

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// beacon handler
//
//	@Summary		Beacon attack
//	@Description	Beacon path will send a fake fake management beacons in order to create N access point.
//	@Tags			Wifi
//	@Accept			json
//	@Produce		json
//	@Param			request	body	wifi_common.Beaconer	true	"beacon info"
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Failure		500	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/beacon [post]
func beaconHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "POST" {

		var beaconer wifi_common.Beaconer

		body, _ := io.ReadAll(req.Body)
		err := json.Unmarshal(body, &beaconer)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		if WifiModule == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "ap scanner must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		wifi_common.BeaconerChanel = make(chan bool)
		listAp, err := wifi_common.ApGenerator(beaconer)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		err = WifiModule.Beaconer(listAp)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// rogue ap handler
//
//	@Summary		rogue access point attack
//	@Description	RogueAP path will send a fake fake management beacons in order to create rogue access point.
//	@Tags			Wifi
//	@Accept			json
//	@Produce		json
//	@Param			request	body	wifi_common.RogueAp	true	"rogue ap info"
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Failure		500	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/rogueAp [post]
func rogueApHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "POST" {

		var rogueAp wifi_common.RogueAp

		body, _ := io.ReadAll(req.Body)
		err := json.Unmarshal(body, &rogueAp)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		if WifiModule == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "ap scanner must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		err = WifiModule.ApSettings(rogueAp)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		wifi_common.RogueApChanel = make(chan bool)
		err = WifiModule.StartAp()
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// shut down recon
//
//	@Summary		stop recon
//	@Description	Stop path will kill all process of recon.
//	@Tags			Wifi
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Failure		500	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/stop [get]
func stopHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		if WifiModule == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "ap scanner must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		err := WifiModule.ForcedStop()
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// shut down client recon
//
//	@Summary		stop client recon
//	@Description	StopScanClient path will kill process of searching for access point clients.
//	@Tags			Wifi
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/stopScanClient [get]
func stopScanClientHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		if wifi_common.ScanClientChanel == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "client scanning must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		wifi_common.ScanClientChanel <- true

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// shut down handshake recon
//
//	@Summary		stop capture handshake
//	@Description	StopCptHandshake path will kill process of searching access points.
//	@Tags			Wifi
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/stopCptHandshake [get]
func stopCptHandshakeHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		if wifi_common.CptHandshakeHandlerChanel == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "capture handshake scanning must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		wifi_common.CptHandshakeHandlerChanel <- true

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// shut down beaconer
//
//	@Summary		stop beacon attack
//	@Description	StopBeaconer path will kill process of sending beacons.
//	@Tags			Wifi
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/stopBeaconer [get]
func stopBeaconerHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		if wifi_common.BeaconerChanel == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "beaconer must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		wifi_common.BeaconerChanel <- true

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// shut down rogue ap
//
//	@Summary		stop rogue access point attack
//	@Description	StopRogueAP path will kill process of sending beacons.
//	@Tags			Wifi
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	v1_common.ErrorMessage
//	@Router			/v1/modules/wifi/stopRogueAp [get]
func stopRogueApHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		if wifi_common.RogueApChanel == nil {

			errorMessage := v1_common.ErrorMessage{
				Error: "rogue ap must be running",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		wifi_common.RogueApChanel <- true

		v1_common.JsonResponceHandler(resp, http.StatusOK, nil)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}
