/*
*	The code in the following file is an extraction from the bettercap projct
*	ALL CREDID GOES TO THEM
*	Some functions received little modification to fit the need
*	Including creating new function
 */

package wifi_common

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"

	"github.com/bettercap/bettercap/network"
	"github.com/bettercap/bettercap/packets"
	"github.com/evilsocket/islazy/fs"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

func (mod *WiFiModule) discoverHandshakes(dot11 *layers.Dot11, packet gopacket.Packet) {

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

		// locate the client station, if its BSSID is ours, it means we sent
		// an association request via wifi.assoc because we're trying to capture
		// the PMKID from the first EAPOL sent by the AP.
		// (Reference about PMKID https://hashcat.net/forum/thread-7717.html)
		// In this case, we need to add ourselves as a client station of the AP
		// in order to have a consistent association of AP, client and handshakes.

		mod.Lock()
		mac := network.NormalizeMac(apMac.String())
		ap, found := mod.aps[mac]
		if !found {
			log.Printf("could not find AP with BSSID %s\n", apMac.String())
			return
		}
		staIsUs := bytes.Equal(staMac, mod.iface.HW)
		station, found := ap.Get(staMac.String())
		staAdded := false
		if !found {
			station, staAdded = ap.AddClientIfNew(staMac.String(), ap.Frequency, ap.RSSI)
		}
		mod.Unlock()

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
		if !mod.shakesAggregate {
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

			// make sure the info that we have key material for this AP
			// is persisted even after stations are pruned due to inactivity
			mod.Lock()
			ap.withKeyMaterial = true
			mod.Unlock()
		}
		// if we added ourselves as a client station but we didn't get any
		// PMKID, just remove it from the list of clients of this AP.
		if staAdded || (staIsUs && rawPMKID == nil) {
			mod.Lock()
			ap.RemoveClient(staMac.String())
			mod.Unlock()
		}
	}

	// quick and dirty heuristic, see thread here https://github.com/bettercap/bettercap/issues/810#issuecomment-805145392
	if isEAPOL || (dot11.Type.MainType() != layers.Dot11TypeData && dot11.Type.MainType() != layers.Dot11TypeCtrl) {
		target := (*network.Station)(nil)
		targetAP := (*AccessPoint)(nil)

		// collect target bssids
		bssids := make([]net.HardwareAddr, 0)
		for _, addr := range []net.HardwareAddr{dot11.Address1, dot11.Address2, dot11.Address3, dot11.Address4} {
			if !bytes.Equal(addr, network.BroadcastHw) {
				bssids = append(bssids, addr)
			}
		}

		// for each AP
		mod.EachAccessPoint(func(mac string, ap *AccessPoint) {
			// only check APs we captured handshakes of
			if target == nil && mod.HasKeyMaterial(ap) {
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
			if !mod.shakesAggregate {
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

	mod.Lock()
	defer mod.Unlock()

	for _, ap := range mod.aps {
		for _, station := range ap.Clients {
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
	}

	return nil
}

func (mod *WiFiModule) EachAccessPoint(cb func(mac string, ap *AccessPoint)) {

	mod.Lock()
	defer mod.Unlock()

	for m, ap := range mod.aps {
		cb(m, ap)
	}
}
