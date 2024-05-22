/*
*	The code in the following file is an extraction from the bettercap projct
*	ALL CREDID GOES TO THEM
*	Some functions received little modification to fit the need
*	Including creating new function
 */

package wifi_common

import (
	"time"

	"github.com/bettercap/bettercap/network"
	"github.com/bettercap/bettercap/packets"
	"github.com/google/gopacket/layers"
)

func (mod *WiFiModule) discoverClients(radiotap *layers.RadioTap, dot11 *layers.Dot11) {
	mod.EachAccessPoint(func(bssid string, ap *AccessPoint) {
		// packet going to this specific BSSID?
		if packets.Dot11IsDataFor(dot11, ap.HW) {
			bssid := dot11.Address2.String()
			freq := int(radiotap.ChannelFrequency)
			rssi := radiotap.DBMAntennaSignal

			mod.Lock()
			ap.AddClientIfNew(bssid, freq, rssi)
			mod.Unlock()
		}
	})
}

func (ap *AccessPoint) AddClientIfNew(bssid string, frequency int, rssi int8) (*network.Station, bool) {

	bssid = network.NormalizeMac(bssid)

	if s, found := ap.clients[bssid]; found {
		// update
		s.Frequency = frequency
		s.RSSI = rssi
		s.LastSeen = time.Now()

		return s, false
	}

	s := network.NewStation("", bssid, frequency, rssi)
	ap.clients[bssid] = s

	return s, true
}
