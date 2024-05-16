package wifi_common

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bettercap/bettercap/modules/wifi"
	"github.com/bettercap/bettercap/network"
	"github.com/bettercap/bettercap/packets"
	"github.com/evilsocket/islazy/data"
	"github.com/evilsocket/islazy/fs"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

func (mod *WiFiModule) Start() {

	src := gopacket.NewPacketSource(mod.Handle, mod.Handle.LinkType())
	pktSourceChan := src.Packets()
	for packet := range pktSourceChan {

		if packet == nil {
			continue
		}

		// perform initial dot11 parsing and layers validation
		if ok, radiotap, dot11 := packets.Dot11Parse(packet); ok {
			// check FCS checksum
			if !dot11.ChecksumValid() {
				log.Println("skipping dot11 packet with invalid checksum.")
				continue
			}

			mod.shakesFile = os.Getenv("SHAKES_FILE")
			if mod.shakesFile == "" {
				log.Println("shakesfile can't be empty, check .env")
				return
			}
			mod.shakesAggregate = false

			mod.discoverAccessPoints(radiotap, dot11, packet)
			for key, value := range mod.aps {
				log.Println("Key:", key, "Value:", value)
			}
			mod.discoverHandshakes(radiotap, dot11, packet)
		}
	}
}

func (mod *WiFiModule) discoverHandshakes(radiotap *layers.RadioTap, dot11 *layers.Dot11, packet gopacket.Packet) {

	isEAPOL := false

	if ok, key, apMac, staMac := packets.Dot11ParseEAPOL(packet, dot11); ok {
		isEAPOL = true

		// first, locate the AP in our list by its BSSID
		/*
			ap, found := mod.Session.WiFi.Get(apMac.String())
			if !found {
				mod.Warning("could not find AP with BSSID %s", apMac.String())
				return
			}
		*/

		mac := network.NormalizeMac(apMac.String())
		ap, found := mod.aps[mac]
		if !found {
			mod.Warning("could not find AP with BSSID %s", apMac.String())
			return
		}

		// locate the client station, if its BSSID is ours, it means we sent
		// an association request via wifi.assoc because we're trying to capture
		// the PMKID from the first EAPOL sent by the AP.
		// (Reference about PMKID https://hashcat.net/forum/thread-7717.html)
		// In this case, we need to add ourselves as a client station of the AP
		// in order to have a consistent association of AP, client and handshakes.
		staIsUs := bytes.Equal(staMac, mod.iface.HW)
		station, found := ap.Get(staMac.String())
		staAdded := false
		if !found {
			station, staAdded = mod.AddClientIfNew(staMac.String(), ap.Frequency, ap.RSSI)
		}

		rawPMKID := []byte(nil)
		if !key.Install && key.KeyACK && !key.KeyMIC {
			// [1] (ACK) AP is sending ANonce to the client
			rawPMKID = station.Handshake.AddAndGetPMKID(packet)
			PMKID := "without PMKID"
			if rawPMKID != nil {
				PMKID = "with PMKID"
			}

			log.Printf("got frame 1/4 of the %s <-> %s handshake (%s) (anonce:%x)\n",
				apMac,
				staMac,
				PMKID,
				key.Nonce)

			//add the ap's station's beacon packet to be saved as part of the handshake cap file
			//https://github.com/ZerBea/hcxtools/issues/92
			//https://github.com/bettercap/bettercap/issues/592

			if ap.Station.Handshake.Beacon != nil {
				log.Printf("adding beacon frame to handshake for %s\n", apMac)
				station.Handshake.AddFrame(1, ap.Station.Handshake.Beacon)
			}

		} else if !key.Install && !key.KeyACK && key.KeyMIC && !allZeros(key.Nonce) {
			// [2] (MIC) client is sending SNonce+MIC to the API
			station.Handshake.AddFrame(1, packet)

			log.Printf("got frame 2/4 of the %s <-> %s handshake (snonce:%x mic:%x)\n",
				apMac,
				staMac,
				key.Nonce,
				key.MIC)
		} else if key.Install && key.KeyACK && key.KeyMIC {
			// [3]: (INSTALL+ACK+MIC) AP informs the client that the PTK is installed
			station.Handshake.AddFrame(2, packet)

			log.Printf("got frame 3/4 of the %s <-> %s handshake (mic:%x)\n",
				apMac,
				staMac,
				key.MIC)
		}

		// if we have unsaved packets as part of the handshake, save them.
		numUnsaved := station.Handshake.NumUnsaved()
		shakesFileName := mod.shakesFile
		if mod.shakesAggregate == false {
			shakesFileName = path.Join(shakesFileName, fmt.Sprintf("%s.pcap", ap.PathFriendlyName()))
		}
		doSave := numUnsaved > 0
		if doSave && shakesFileName != "" {
			log.Printf("(aggregate %v) saving handshake frames to %s\n", mod.shakesAggregate, shakesFileName)
			if err := mod.SaveHandshakesTo(shakesFileName, mod.Handle.LinkType()); err != nil {
				log.Printf("error while saving handshake frames to %s: %s\n", shakesFileName, err)
			}
		}

		validPMKID := rawPMKID != nil
		validHalfHandshake := !staIsUs && station.Handshake.Half()
		validFullHandshake := station.Handshake.Complete()
		// if we have unsaved packets AND
		//   if we captured a PMKID OR
		//   if we captured am half handshake which is not ours OR
		//   if we captured a full handshake
		if doSave && (validPMKID || validHalfHandshake || validFullHandshake) {
			mod.Session.Events.Add("wifi.client.handshake", wifi.HandshakeEvent{
				File:       shakesFileName,
				NewPackets: numUnsaved,
				AP:         apMac.String(),
				Station:    staMac.String(),
				PMKID:      rawPMKID,
				Half:       station.Handshake.Half(),
				Full:       station.Handshake.Complete(),
			})
			// make sure the info that we have key material for this AP
			// is persisted even after stations are pruned due to inactivity
			ap.WithKeyMaterial(true)
		}
		// if we added ourselves as a client station but we didn't get any
		// PMKID, just remove it from the list of clients of this AP.
		if staAdded || (staIsUs && rawPMKID == nil) {
			ap.RemoveClient(staMac.String())
		}
	}

	// quick and dirty heuristic, see thread here https://github.com/bettercap/bettercap/issues/810#issuecomment-805145392
	if isEAPOL || (dot11.Type.MainType() != layers.Dot11TypeData && dot11.Type.MainType() != layers.Dot11TypeCtrl) {
		target := (*network.Station)(nil)
		targetAP := (*network.AccessPoint)(nil)

		// collect target bssids
		bssids := make([]net.HardwareAddr, 0)
		for _, addr := range []net.HardwareAddr{dot11.Address1, dot11.Address2, dot11.Address3, dot11.Address4} {
			if bytes.Equal(addr, network.BroadcastHw) == false {
				bssids = append(bssids, addr)
			}
		}

		// for each AP
		mod.EachAccessPoint(func(mac string, ap *network.AccessPoint) {
			// only check APs we captured handshakes of
			if target == nil && ap.HasKeyMaterial() {
				// search client station
				ap.EachClient(func(mac string, station *network.Station) {
					// any valid key material for this station?
					if station.Handshake.Any() {
						// check if target
						for _, a := range bssids {
							if bytes.Equal(a, station.HW) {
								target = station
								targetAP = ap
								break
							}
						}
					}
				})
			}
		})

		if target != nil {
			log.Printf("saving extra %s frame (%d bytes) for %s\n",
				dot11.Type.String(),
				len(packet.Data()),
				target.String())

			target.Handshake.AddExtra(packet)

			shakesFileName := mod.shakesFile
			if mod.shakesAggregate == false {
				shakesFileName = path.Join(shakesFileName, fmt.Sprintf("%s.pcap", targetAP.PathFriendlyName()))
			}
			if shakesFileName != "" {
				log.Printf("(aggregate %v) saving handshake frames to %s\n", mod.shakesAggregate, shakesFileName)
				if err := mod.SaveHandshakesTo(shakesFileName, mod.Handle.LinkType()); err != nil {
					log.Printf("error while saving handshake frames to %s: %s\n", shakesFileName, err)
				}
			}
		}
	}

}

func allZeros(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}

func (mod *WiFiModule) discoverAccessPoints(radiotap *layers.RadioTap, dot11 *layers.Dot11, packet gopacket.Packet) {
	// search for Dot11InformationElementIDSSID
	if ok, ssid := packets.Dot11ParseIDSSID(packet); ok {
		from := dot11.Address3

		/*
			// skip stuff we're sending
			if bytes.Equal(from, mod.apConfig.BSSID) {
				return
			}
		*/

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
					ap.EachClient(func(mac string, station *network.Station) {
						station.Handshake.SetBeacon(packet)
					})
				}
			} else {
				log.Printf("skipping %s with %d dBm\n", from.String(), radiotap.DBMAntennaSignal)
			}
		}
	}
}

func (mod *WiFiModule) AddIfNew(ssid, mac string, frequency int, rssi int8) (*network.AccessPoint, bool) {

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
	newAp := network.NewAccessPoint(ssid, mac, frequency, rssi, unsortedKV)
	mod.aps[mac] = newAp

	return newAp, true
}

func isBogusMacESSID(essid string) bool {
	for _, c := range essid {
		if !strconv.IsPrint(c) {
			return true
		}
	}
	return false
}

func (mod *WiFiModule) AddClientIfNew(bssid string, frequency int, rssi int8) (*network.Station, bool) {

	bssid = network.NormalizeMac(bssid)

	if s, found := mod.ap.clients[bssid]; found {
		// update
		s.Frequency = frequency
		s.RSSI = rssi
		s.LastSeen = time.Now()

		return s, false
	}

	s := network.NewStation("", bssid, frequency, rssi)
	mod.ap.clients[bssid] = s

	return s, true
}

type AccessPoint struct {
	clients map[string]*network.Station
}

func (mod *WiFiModule) SaveHandshakesTo(fileName string, linkType layers.LinkType) error {
	// check if folder exists first
	dirName := filepath.Dir(fileName)
	if _, err := os.Stat(dirName); err != nil {
		if err = os.MkdirAll(dirName, os.ModePerm); err != nil {
			return err
		}
	}

	doHead := !fs.Exists(fileName)
	fp, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer fp.Close()

	writer := pcapgo.NewWriter(fp)

	if doHead {
		if err = writer.WriteFileHeader(65536, linkType); err != nil {
			return err
		}
	}

	for _, station := range mod.ap.clients {
		// if half (which includes also complete) or has pmkid
		if station.Handshake.Any() {

			err = nil
			station.Handshake.EachUnsavedPacket(func(pkt gopacket.Packet) {
				if err == nil {
					err = writer.WritePacket(pkt.Metadata().CaptureInfo, pkt.Data())
				}
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (mod *WiFiModule) EachAccessPoint(cb func(mac string, ap *network.AccessPoint)) {

	for m, ap := range mod.aps {
		cb(m, ap)
	}
}
