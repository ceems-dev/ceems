package ipmi

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// IPMI DCMI related constants.
const (
	IPMI_DCMI           = 0xDC //nolint:stylecheck
	IPMI_DCMI_GETRED    = 0x2  //nolint:stylecheck
	IPMI_NETFN_DCGRP    = 0x2C //nolint:stylecheck
	IPMI_DCMI_ACTIVATED = 0x40 //nolint:stylecheck
)

type PowerReading struct {
	Minimum, Maximum, Average, Current float64
	Activated                          bool
}

// DCMIPowerReading returns the current IPMI DCMI power reading.
func (i *ipmiClient) DCMIPowerReading() (*PowerReading, error) {
	// Request payload
	msgData := [4]uint8{IPMI_DCMI, 0x1, 0x0, 0x0}

	// IPMI Request
	req := Request{
		Addr:    uintptr(unsafe.Pointer(&i.bmcAddr)),
		AddrLen: uint(unsafe.Sizeof(i.bmcAddr)),
		Msgid:   1,
		Msg: ipmiMsg{
			Data:    uintptr(unsafe.Pointer(&msgData[0])),
			DataLen: 4,
			Netfn:   IPMI_NETFN_DCGRP,
			Cmd:     IPMI_DCMI_GETRED,
		},
	}

	// Do request and read response
	resp, err := i.Do(&req)
	if err != nil {
		i.logger.Error("Failed to make IPMI request to get DCMI reading", "err", err)

		return nil, fmt.Errorf("failed to make ipmi request for dcmi reading: %w", err)
	}

	// Get readings
	return &PowerReading{
		Current:   float64(binary.LittleEndian.Uint16(resp.Data[2:4])),
		Minimum:   float64(binary.LittleEndian.Uint16(resp.Data[4:6])),
		Maximum:   float64(binary.LittleEndian.Uint16(resp.Data[6:8])),
		Average:   float64(binary.LittleEndian.Uint16(resp.Data[8:10])),
		Activated: resp.Data[18] == IPMI_DCMI_ACTIVATED,
	}, nil
}
