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
	"net"
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
	PktSourceChan       chan gopacket.Packet
	pktSourceChanClosed bool
	hopPeriod           time.Duration
	stickChan           int
	frequencies         []int
	channel             int
	apTTL               int
	staTTL              int
	hopChanges          chan bool
	reads               *sync.WaitGroup
	chanLock            *sync.Mutex

	sync.Mutex
}

type AccessPoint struct {
	*network.Station

	clients         map[string]*network.Station
	withKeyMaterial bool
}

func NewWiFiModule(ifaceName string) (*WiFiModule, error) {

	sess, err := NewSession()
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
		apTTL:           300,
		staTTL:          300,
		channel:         0,
		stickChan:       0,
		hopPeriod:       250 * time.Millisecond,
		hopChanges:      make(chan bool),
		chanLock:        &sync.Mutex{},
		reads:           &sync.WaitGroup{},
	}

	mod.InitState("channels")

	mod.shakesFile = os.Getenv("SHAKES_FILE")
	if mod.shakesFile == "" {
		return nil, errors.New("shakesfile can't be empty, check .env")
	}

	freqs, err := network.GetSupportedFrequencies(mod.iface.Name())
	if err != nil {
		return nil, err
	}
	mod.setFrequencies(freqs)
	//mod.hopChanges <- true

	return mod, nil
}

func (mod *WiFiModule) Configure() error {

	opts := network.CAPTURE_DEFAULTS
	opts.Timeout = 500 * time.Millisecond
	opts.Monitor = true

	var err error

	for retry := 0; ; retry++ {
		if mod.Handle, err = network.CaptureWithOptions(mod.iface.Name(), opts); err == nil {
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

func (mod *WiFiModule) Start() error {

	if err := mod.Configure(); err != nil {
		return err
	}

	mod.SetRunning(true, func() {

		// start channel hopper if needed
		if mod.channel == 0 {
			go mod.channelHopper()
		}

		// start the pruner
		go mod.stationPruner()

		mod.reads.Add(1)

		src := gopacket.NewPacketSource(mod.Handle, mod.Handle.LinkType())
		mod.PktSourceChan = src.Packets()

		mod.reads.Wait()
	})

	return nil
}

func (mod *WiFiModule) AccessPointPacketAnalyzer() {

	for packet := range mod.PktSourceChan {

		if !mod.Running() {
			break
		} else if packet == nil {
			continue
		}

		// perform initial dot11 parsing and layers validation
		if ok, radiotap, dot11 := packets.Dot11Parse(packet); ok {
			// check FCS checksum
			if !dot11.ChecksumValid() {
				log.Println("skipping dot11 packet with invalid checksum.")
				continue
			}

			mod.discoverAccessPoints(radiotap, dot11, packet)
		}
	}

	mod.Lock()
	mod.pktSourceChanClosed = true
	mod.Unlock()
}

func (mod *WiFiModule) DiscoverClientAnalyzer() {

	for packet := range mod.PktSourceChan {

		if !mod.Running() {
			break
		} else if packet == nil {
			continue
		}

		// perform initial dot11 parsing and layers validation
		if ok, radiotap, dot11 := packets.Dot11Parse(packet); ok {
			// check FCS checksum
			if !dot11.ChecksumValid() {
				log.Println("skipping dot11 packet with invalid checksum.")
				continue
			}

			mod.discoverClients(radiotap, dot11)
		}
	}

	mod.Lock()
	mod.pktSourceChanClosed = true
	mod.Unlock()
}

func (mod *WiFiModule) ForcedStop() error {
	return mod.SetRunning(false, func() {
		// signal the main for loop we want to exit
		if !mod.pktSourceChanClosed {
			mod.PktSourceChan <- nil
		}
		// close the pcap handle to make the main for exit
		mod.Handle.Close()
		mod.reads.Done()
	})
}

func NewSession() (*session.Session, error) {

	s := &session.Session{
		Prompt: session.NewPrompt(),
		Env:    nil,
		Active: false,
		Queue:  nil,
	}

	return s, nil
}

func (m *WiFiModule) SetRunning(running bool, cb func()) error {

	if running == m.Running() {
		if m.Started {
			return fmt.Errorf("module %s is already running", m.Name)

		} else {
			return fmt.Errorf("module %s is not running", m.Name)
		}
	}

	m.StatusLock.Lock()
	m.Started = running
	m.StatusLock.Unlock()

	if cb != nil {
		if running {
			// this is the worker, start async
			go cb()
		} else {
			// stop callback, this is sync with a 10 seconds timeout
			done := make(chan bool, 1)
			go func() {
				cb()
				done <- true
			}()

			select {
			case <-done:
				return nil
			case <-time.After(10 * time.Second):
				fmt.Printf("%s: Stopping module %s timed out.\n", tui.Yellow(tui.Bold("WARNING")), m.Name)
			}
		}
	}

	return nil
}

func (mod *WiFiModule) channelHopper() {
	mod.reads.Add(1)
	defer mod.reads.Done()

	log.Println("channel hopper started.")

	for mod.Running() {
		delay := mod.hopPeriod
		// if we have both 2.4 and 5ghz capabilities, we have
		// more channels, therefore we need to increase the time
		// we hop on each one otherwise me lose information
		if len(mod.frequencies) > 14 {
			delay = delay * 2
		}

		frequencies := mod.frequencies

	loopCurrentChannels:
		for _, frequency := range frequencies {
			channel := network.Dot11Freq2Chan(frequency)
			// stick to the access point channel as long as it's selected
			// or as long as we're deauthing on it
			if mod.stickChan != 0 {
				channel = mod.stickChan
			}

			if stop := mod.hop(channel); stop {
				mod.ForcedStop()
				return
			}

			select {
			case <-mod.hopChanges:
				log.Println("hop changed")
				break loopCurrentChannels
			case <-time.After(delay):
				if !mod.Running() {
					return
				}
			}
		}
	}
}

func (mod *WiFiModule) hop(channel int) (mustStop bool) {
	mod.chanLock.Lock()
	defer mod.chanLock.Unlock()

	return mod.hopUnlocked(channel)
}

func (mod *WiFiModule) hopUnlocked(channel int) (mustStop bool) {
	// log.Printf("hopping on channel %d", channel)

	if err := network.SetInterfaceChannel(mod.iface.Name(), channel); err != nil {
		// check if the device has been disconnected
		if !mod.isInterfaceConnected() {
			log.Printf("interface %s disconnected, stopping module\n", mod.iface.Name())
			mustStop = true
		} else {
			log.Printf("error while hopping to channel %d: %s\n", channel, err)
		}
	}

	return
}

func (mod *WiFiModule) isInterfaceConnected() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		mod.Error("error while enumerating interfaces: %s", err)
		return false
	}

	for _, iface := range ifaces {
		if mod.iface.HwAddress == network.NormalizeMac(iface.HardwareAddr.String()) {
			return true
		}
	}

	return false
}

func (mod *WiFiModule) stationPruner() {

	mod.reads.Add(1)
	defer mod.reads.Done()

	maxApTTL := time.Duration(mod.apTTL) * time.Second
	maxStaTTL := time.Duration(mod.staTTL) * time.Second

	log.Printf("wifi stations pruner started (ap.ttl:%v sta.ttl:%v).\n", maxApTTL, maxStaTTL)
	for mod.Running() {
		// loop every AP
		for _, ap := range mod.List() {
			sinceLastSeen := time.Since(ap.LastSeen)
			if sinceLastSeen > maxApTTL {
				log.Printf("station %s not seen in %s, removing.\n", ap.BSSID(), sinceLastSeen)
				mod.Session.WiFi.Remove(ap.BSSID())
				continue
			}
			// loop every AP client
			mod.Lock()
			clients := make([]*network.Station, 0, len(ap.clients))
			for _, c := range ap.clients {
				clients = append(clients, c)
			}
			mod.Unlock()
			for _, c := range clients {
				sinceLastSeen := time.Since(c.LastSeen)
				if sinceLastSeen > maxStaTTL {
					log.Printf("client %s of station %s not seen in %s, removing.\n", c.String(), ap.BSSID(), sinceLastSeen)
					mod.Lock()
					ap.RemoveClient(c.BSSID())
					mod.Unlock()
				}
			}
		}
		time.Sleep(1 * time.Second)
		// refresh
		maxApTTL = time.Duration(mod.apTTL) * time.Second
		maxStaTTL = time.Duration(mod.staTTL) * time.Second
	}
}

func (mod *WiFiModule) setFrequencies(freqs []int) {
	log.Printf("new frequencies: %v\n", freqs)

	mod.frequencies = freqs
	channels := []int{}
	for _, freq := range freqs {
		channels = append(channels, network.Dot11Freq2Chan(freq))
	}

	mod.State.Store("channels", channels)
}

func (mod *WiFiModule) List() (list []*AccessPoint) {
	mod.Lock()
	defer mod.Unlock()

	list = make([]*AccessPoint, 0, len(mod.aps))

	for _, ap := range mod.aps {
		list = append(list, ap)
	}
	return
}
