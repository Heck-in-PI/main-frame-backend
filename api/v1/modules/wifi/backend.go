package wifi

import (
	"encoding/json"
	"io"
	"log"
	wifi_common "mf-backend/api/v1/modules/wifi/common"
	v1_common "mf-backend/api/v1/v1Common"
	"net"
	"strings"

	"net/http"

	"github.com/gorilla/mux"
	wifi "github.com/mdlayher/wifi"
	goWireless "github.com/theojulienne/go-wireless"
)

// interfaces handler
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
func scanApHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		muxVars := mux.Vars(req)
		interfaceName := muxVars["interfaceName"]
		if interfaceName == "" {
			errorMessage := v1_common.ErrorMessage{
				Error: "interface name must be specified in path",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		client, err := goWireless.NewClient(interfaceName)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		log.Println(client)
		defer client.Close()

		aps, err := client.Scan()
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		v1_common.JsonResponceHandler(resp, http.StatusOK, aps)

	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// death handler
func deauthHandler(resp http.ResponseWriter, req *http.Request) {

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

		wifiModule, err := wifi_common.NewWiFiModule(interfaceName)
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

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
		err = wifiModule.Configure()
		if err != nil {

			errorMessage := v1_common.ErrorMessage{
				Error: err.Error(),
			}

			v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

			return
		}

		log.Println("kicking out from:", bssid, ", client: ", client)
		wifiModule.SendDeauthPacket(bssid, client)

		resp.WriteHeader(http.StatusOK)
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}

// connect to access point handler
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

// death handler
func cptHandshakeHandler(resp http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	if req.Method == "GET" {

		muxVars := mux.Vars(req)
		interfaceName := muxVars["interfaceName"]
		if interfaceName == "" {
			errorMessage := v1_common.ErrorMessage{
				Error: "interface name must be specified in path",
			}

			v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)

			return
		}

		/*
			wifiModule, err := wifi_common.NewWiFiModule(interfaceName)
			if err != nil {

				errorMessage := v1_common.ErrorMessage{
					Error: err.Error(),
				}

				v1_common.JsonResponceHandler(resp, http.StatusInternalServerError, errorMessage)

				return
			}

			wifiModule.Start()
		*/
	} else {

		errorMessage := v1_common.ErrorMessage{
			Error: "Invalid Request",
		}

		v1_common.JsonResponceHandler(resp, http.StatusBadRequest, errorMessage)
	}
}
