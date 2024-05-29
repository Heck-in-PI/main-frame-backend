package wifi_common

import (
	"log"
	"net"

	"github.com/bettercap/bettercap/network"
	"github.com/bettercap/bettercap/packets"
)

type Prober struct {
	ApMac  string
	ApName string
}

func (mod *WiFiModule) SendProbePacket(staMac net.HardwareAddr, ssid string) {

	for seq := uint16(0); seq < 5; seq++ {
		if err, pkt := packets.NewDot11ProbeRequest(staMac, seq, ssid, network.GetInterfaceChannel(mod.iface.Name())); err != nil {
			log.Printf("could not create probe packet: %s\n", err)
			continue
		} else {
			mod.injectPacket(pkt)
		}
	}

	log.Println("sent probe frames")
}
