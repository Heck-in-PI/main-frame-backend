/*
*	The code in the following file is an extraction from the bettercap projct
*	ALL CREDID GOES TO THEM
*	Some functions received little modification to fit the need
*	Including creating new function
 */

package wifi_common

import (
	"bytes"
	"log"
	"strconv"
	"time"

	"github.com/bettercap/bettercap/network"
	"github.com/bettercap/bettercap/packets"
	"github.com/evilsocket/islazy/data"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func (mod *WiFiModule) discoverAccessPoints(radiotap *layers.RadioTap, dot11 *layers.Dot11, packet gopacket.Packet) {

	// search for Dot11InformationElementIDSSID
	if ok, ssid := packets.Dot11ParseIDSSID(packet); ok {
		from := dot11.Address3

		// skip stuff we're sending
		if bytes.Equal(from, mod.apConfig.BSSID) {
			return
		}

		if !network.IsZeroMac(from) && !network.IsBroadcastMac(from) {
			if int(radiotap.DBMAntennaSignal) >= mod.minRSSI {
				var frequency int
				bssid := from.String()

				if found, channel := packets.Dot11ParseDSSet(packet); found {
					frequency = network.Dot11Chan2Freq(channel)
				} else {
					frequency = int(radiotap.ChannelFrequency)
				}

				if ap, isNew := mod.AddIfNew(ssid, bssid, frequency, radiotap.DBMAntennaSignal); !isNew {
					//set beacon packet on the access point station.
					//This is for it to be included in the saved handshake file for wifi.assoc
					ap.Station.Handshake.Beacon = packet
					mod.Lock()
					ap.EachClient(func(mac string, station *network.Station) {
						station.Handshake.SetBeacon(packet)
					})
					mod.Unlock()
				}
			} else {
				log.Printf("skipping %s with %d dBm\n", from.String(), radiotap.DBMAntennaSignal)
			}
		}
	}
}

func (mod *WiFiModule) AddIfNew(ssid, mac string, frequency int, rssi int8) (*AccessPoint, bool) {

	mod.Lock()
	defer mod.Unlock()

	mac = network.NormalizeMac(mac)

	if ap, found := mod.aps[mac]; found {
		ap.LastSeen = time.Now()
		if rssi != 0 {
			ap.RSSI = rssi
		}
		// always get the cleanest one
		if !isBogusMacESSID(ssid) {
			ap.Hostname = ssid
		}

		return ap, false
	}

	var unsortedKV *data.UnsortedKV
	networkAp := network.NewAccessPoint(ssid, mac, frequency, rssi, unsortedKV)
	newAp := AccessPoint{
		Station: networkAp.Station,
		clients: make(map[string]*network.Station),
	}

	mod.aps[mac] = &newAp

	return &newAp, true
}

func (ap *AccessPoint) EachClient(cb func(mac string, station *network.Station)) {

	for m, station := range ap.clients {
		cb(m, station)
	}
}

func isBogusMacESSID(essid string) bool {
	for _, c := range essid {
		if !strconv.IsPrint(c) {
			return true
		}
	}
	return false
}

func (ap *AccessPoint) Get(bssid string) (*network.Station, bool) {

	bssid = network.NormalizeMac(bssid)
	if s, found := ap.clients[bssid]; found {
		return s, true
	}
	return nil, false
}

func (ap *AccessPoint) RemoveClient(mac string) {

	bssid := network.NormalizeMac(mac)
	delete(ap.clients, bssid)
}

func (mod *WiFiModule) HasKeyMaterial(ap *AccessPoint) bool {
	mod.Lock()
	defer mod.Unlock()

	return ap.withKeyMaterial
}
