package wifi_common

import (
	"log"
	"net"
	"time"

	"github.com/bettercap/bettercap/packets"
	"github.com/go-random/mac"
)

type Beaconer struct {
	NumberOfAP   int
	ApName       string
	ApChannel    int
	ApEncryption bool
}

func ApGenerator(beaconer Beaconer) ([]packets.Dot11ApConfig, error) {

	var apList []packets.Dot11ApConfig
	for i := 0; i < beaconer.NumberOfAP; i++ {

		rand := mac.NewRandomizer()
		bssid, err := net.ParseMAC(rand.MAC())
		if err != nil {

			return nil, err
		}

		apList = append(apList, packets.Dot11ApConfig{
			SSID:       beaconer.ApName,
			BSSID:      bssid,
			Channel:    beaconer.ApChannel,
			Encryption: beaconer.ApEncryption,
		})
	}

	return apList, nil
}

func (mod *WiFiModule) Beaconer(beaconer []packets.Dot11ApConfig) error {

	err := mod.Pause()
	if err != nil {
		return err
	}

	go func() {

		for seqn := uint16(0); ; seqn++ {

			for _, beacon := range beaconer {

				select {
				case <-BeaconerChanel:
					return
				default:
				}

				if err, pkt := packets.NewDot11Beacon(beacon, seqn); err != nil {
					log.Printf("could not create beacon packet: %s\n", err)
				} else {
					mod.injectPacket(pkt)
				}

				seqn++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return nil
}
