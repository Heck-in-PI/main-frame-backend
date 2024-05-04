package wifi

import "github.com/mdlayher/wifi"

type WirelessInterface struct {
	Index        int
	Name         string
	HardwareAddr string
	PHY          int
	Device       int
	Type         wifi.InterfaceType
	Frequency    int
}
