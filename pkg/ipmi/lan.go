package ipmi

import (
	"fmt"
	"unsafe"
)

// IPMI DCMI related constants.
const (
	IPMI_LAN             = 0x1
	IPMI_LANP_IP_ADDR    = 0x2
	IPMI_NETFN_TRANSPORT = 0xC
)

// LanIP returns the IP address of BMC.
func (i *ipmiClient) LanIP() (*string, error) {
	// Request payload
	msgData := [4]uint8{IPMI_LAN, 0x3, 0x0, 0x0}

	// IPMI Request
	req := Request{
		Addr:    uintptr(unsafe.Pointer(&i.bmcAddr)),
		AddrLen: uint(unsafe.Sizeof(i.bmcAddr)),
		Msgid:   1,
		Msg: ipmiMsg{
			Data:    uintptr(unsafe.Pointer(&msgData[0])),
			DataLen: 4,
			Netfn:   IPMI_NETFN_TRANSPORT,
			Cmd:     IPMI_LANP_IP_ADDR,
		},
	}

	// Do request and read response
	resp, err := i.Do(&req)
	if err != nil {
		i.logger.Error("Failed to make IPMI request to get LAN IP", "err", err)

		return nil, fmt.Errorf("failed to make ipmi request to get lan ip: %w", err)
	}

	// Get LAN IP
	ip := fmt.Sprintf("%d.%d.%d.%d", resp.Data[2], resp.Data[3], resp.Data[4], resp.Data[5])

	return &ip, nil
}
