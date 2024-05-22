/*
*	The code in the following file is an extraction from the bettercap projct
*	ALL CREDID GOES TO THEM
*	Some functions received little modification to fit the need
*	Including creating new function
 */

package wifi_common

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/bettercap/bettercap/modules/wifi"
	"github.com/bettercap/bettercap/network"
	"github.com/bettercap/bettercap/packets"
	"github.com/bettercap/bettercap/session"
	"github.com/evilsocket/islazy/tui"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

type WiFiModule struct {
	session.SessionModule
	iface               *network.Endpoint
	Handle              *pcap.Handle
	shakesFile          string
	shakesAggregate     bool
	minRSSI             int
	aps                 map[string]*AccessPoint
	pktSourceChan       chan gopacket.Packet
	pktSourceChanClosed bool

	sync.Mutex
}

type AccessPoint struct {
	*network.Station

	clients         map[string]*network.Station
	withKeyMaterial bool
}

func NewWiFiModule(ifaceName string) (*WiFiModule, error) {

	sess, err := session.New()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var iface *network.Endpoint

	iface, err = network.FindInterface(ifaceName)
	if err != nil {
		return nil, err
	}

	mod := &WiFiModule{
		SessionModule:   session.NewSessionModule("wifi", sess),
		iface:           iface,
		minRSSI:         -200,
		aps:             make(map[string]*AccessPoint),
		shakesAggregate: false,
	}

	mod.shakesFile = os.Getenv("SHAKES_FILE")
	if mod.shakesFile == "" {
		return nil, errors.New("shakesfile can't be empty, check .env")
	}

	return mod, nil
}

func (mod *WiFiModule) Configure() error {

	opts := network.CAPTURE_DEFAULTS
	opts.Timeout = 500 * time.Millisecond
	opts.Monitor = true

	var err error

	for retry := 0; ; retry++ {
		if mod.Handle, err = network.CaptureWithOptions(mod.iface.String(), opts); err == nil {
			// we're done
			break
		} else if retry == 0 && err.Error() == wifi.ErrIfaceNotUp {
			// try to bring interface up and try again
			log.Printf("interface %s is down, bringing it up ...", mod.iface.String())
			if err := network.ActivateInterface(mod.iface.String()); err != nil {
				return err

			}
			continue
		} else if !opts.Monitor {
			// second fatal error, just bail
			log.Printf("error while activating handle: %s\n", err)
			return err
		} else {
			// first fatal error, try again without setting the interface in monitor mode
			log.Printf("error while activating handle: %s, %s\n", err, tui.Bold("interface might already be monitoring. retrying!"))
			opts.Monitor = false
		}
	}

	return nil
}

func (mod *WiFiModule) Start(selector string) error {

	if err := mod.Configure(); err != nil {
		return err
	}

	mod.SetRunning(true, func() {

		src := gopacket.NewPacketSource(mod.Handle, mod.Handle.LinkType())
		mod.pktSourceChan = src.Packets()
		for packet := range mod.pktSourceChan {

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

				if selector == "discoverAP" {

					mod.discoverAccessPoints(radiotap, dot11, packet)
				} else if selector == "discoverClient" {

					mod.discoverClients(radiotap, dot11)
				} else if selector == "discoverHS" {

					mod.discoverHandshakes(dot11, packet)
				}
			}
		}

		mod.pktSourceChanClosed = true

	})
	return nil
}

func (mod *WiFiModule) forcedStop() error {
	return mod.SetRunning(false, func() {
		// signal the main for loop we want to exit
		if !mod.pktSourceChanClosed {
			mod.pktSourceChan <- nil
		}
		// close the pcap handle to make the main for exit
		mod.Handle.Close()
	})
}
