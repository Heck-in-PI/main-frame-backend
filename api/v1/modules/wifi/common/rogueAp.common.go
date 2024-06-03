package wifi_common

import (
	"errors"
	"log"
	"net"
	"time"

	"github.com/bettercap/bettercap/packets"
)

type RogueAp struct {
	ApName       string
	ApMac        string
	ApChannel    int
	ApEncryption bool
}

func (mod *WiFiModule) ApSettings(rogueAp RogueAp) error {

	bssid, err := net.ParseMAC(rogueAp.ApMac)
	if err != nil {

		return err
	}

	mod.apConfig.SSID = rogueAp.ApName
	mod.apConfig.BSSID = bssid
	mod.apConfig.Channel = rogueAp.ApChannel
	mod.apConfig.Encryption = rogueAp.ApEncryption

	return nil
}

func (mod *WiFiModule) StartAp() error {

	if mod.apRunning {
		return errors.New(mod.apConfig.SSID + " is running")
	}

	go func() {

		mod.apRunning = true
		defer func() {
			mod.apRunning = false
		}()

		for seqn := uint16(0); mod.Running(); seqn++ {
			mod.writes.Add(1)
			defer mod.writes.Done()

			select {
			case <-RogueApChanel:
				return
			default:
			}

			if err, pkt := packets.NewDot11Beacon(mod.apConfig, seqn); err != nil {
				log.Printf("could not create beacon packet: %s\n", err)
			} else {
				mod.injectPacket(pkt)
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	return nil
}
