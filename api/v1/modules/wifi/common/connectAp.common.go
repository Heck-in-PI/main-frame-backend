package wifi_common

import (
	v1_common "mf-backend/api/v1/v1Common"
)

type ConnectAp struct {
	ApName string
	ApPass string
}

func ConnectNetwork(iface string, ssid string, apPass string) (string, error) {

	command := "nmcli device wifi connect " + ssid + " password " + apPass + " ifname " + iface

	return v1_common.CommandExec(command)
}
