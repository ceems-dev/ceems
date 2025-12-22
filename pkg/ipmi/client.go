// Package ipmi implements in-band communication with BMC using IPMI commands
// using `/dev/ipmi*` device.
package ipmi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// IPMI related constants.
const (
	IPMICTL_SET_GETS_EVENTS_CMD     = 0x80046910
	IPMICTL_SEND_COMMAND            = 0x8028690d
	IPMICTL_RECEIVE_MSG_TRUNC       = 0xc030690b
	IPMI_SYSTEM_INTERFACE_ADDR_TYPE = 0xC
	IPMI_BMC_CHANNEL                = 0xF
)

type Client interface {
	Do(r *Request) (*Response, error)
	Close() error
	DCMIPowerReading() (*PowerReading, error)
	LanIP() (*string, error)
	SensorRecords() ([]*FullSensorRecord, error)
	SensorReadings(records []*FullSensorRecord) (map[*FullSensorRecord]float64, error)
}

type Config struct {
	Logger  *slog.Logger
	DevNum  int
	Timeout time.Duration
}

type timeout struct {
	value time.Duration
}

type ipmiClient struct {
	logger  *slog.Logger
	devFile *os.File
	bmcAddr ipmiSystemInterfaceAddr
	timeout time.Duration
}

// NewClient returns a new instance of Client struct.
func NewClient(c *Config) (Client, error) {
	if c.DevNum < 0 {
		return nil, errors.New("device number for IPMI must be greater than zero")
	}

	// List of devices to verify in the order of preference
	ipmiDevs := []string{"/dev/ipmi%d", "/dev/ipmi/%d", "/dev/ipmidev/%d"}

	// Attempt to open device file
	var devFile *os.File

	for _, d := range ipmiDevs {
		f, err := os.Open(fmt.Sprintf(d, c.DevNum))
		if err == nil {
			c.Logger.Debug("IPMI device found", "device", fmt.Sprintf(d, c.DevNum))

			devFile = f

			break
		}
	}

	// If no device is found, return error
	if devFile == nil {
		return nil, errors.New("no IPMI device found on the host")
	}

	// Setup event receiver
	recvEvents := 1

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, devFile.Fd(), IPMICTL_SET_GETS_EVENTS_CMD, uintptr(unsafe.Pointer(&recvEvents)))
	if errno != 0 {
		return nil, fmt.Errorf("failed to enable IPMI event receiver: %w", errno)
	}

	// Set a valid timeout
	if c.Timeout == 0 {
		c.Timeout = time.Second
	}

	// Instantitate client
	client := &ipmiClient{
		logger:  c.Logger,
		devFile: devFile,
		bmcAddr: ipmiSystemInterfaceAddr{
			AddrType: IPMI_SYSTEM_INTERFACE_ADDR_TYPE,
			Channel:  IPMI_BMC_CHANNEL,
			Lun:      0x0,
		},
		timeout: c.Timeout,
	}

	return client, nil
}

// Do sends IPMI request and returns the response.
func (i *ipmiClient) Do(req *Request) (*Response, error) {
	// Device file descriptor
	fd := i.devFile.Fd()

	// Send request
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, IPMICTL_SEND_COMMAND, uintptr(unsafe.Pointer(req)))
	if errno != 0 {
		i.logger.Error("Failed to send IPMI request", "err", errno)

		return nil, fmt.Errorf("failed to send IPMI request: %w", errno)
	}

	var activeFdSet unix.FdSet

	var serverFD int
	if fd < math.MaxInt {
		serverFD = int(fd)
	} else {
		serverFD = math.MaxInt - 1
	}

	FDZero(&activeFdSet)
	FDSet(fd, &activeFdSet)

	resp := Response{}
	addr := ipmiAddr{}
	recv := ipmiRecv{
		Addr:    uintptr(unsafe.Pointer(&addr)),
		AddrLen: uint(unsafe.Sizeof(addr)),
		Msg: ipmiMsg{
			Data:    uintptr(unsafe.Pointer(&resp.Data)),
			DataLen: uint16(unsafe.Sizeof(resp.Data)),
		},
	}

	// Set timeout for select
	timeout := timeout{i.timeout}

	_, err := unix.Select(serverFD+1, &activeFdSet, nil, nil, timeout.timeval())
	if err != nil {
		i.logger.Error("Failed to receive response from IPMI device interface", "err", err)

		return nil, fmt.Errorf("failed to receive response from IPMI device interface: %w", err)
	}

	// Check if fd is ready to read
	if !FDIsSet(fd, &activeFdSet) {
		i.logger.Error("No response received from IPMI device interface")

		return nil, errors.New("no response received from IPMI device interface")
	}

	// Read data into recv struct
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, fd, IPMICTL_RECEIVE_MSG_TRUNC, uintptr(unsafe.Pointer(&recv)))
	if errno != 0 {
		i.logger.Error("Failed to read response from IPMI device interface", "err", errno)

		return nil, fmt.Errorf("failed to read response from IPMI device interface: %w", errno)
	}

	// If Msgids match between response and request break
	if req.Msgid != recv.Msgid {
		i.logger.Error("Received response with unexpected ID", "req_id", req.Msgid, "resp_id", recv.Msgid)

		return nil, fmt.Errorf("received response with unexpected id: %d", recv.Msgid)
	}

	// Read response data
	resp.DataLen = int32(recv.Msg.DataLen)
	// i.logger.Debug("IPMI response data", "data", resp.Data[0:resp.DataLen])

	// Check completion code
	err = binary.Read(bytes.NewReader(resp.Data[0:1]), binary.BigEndian, &resp.Ccode)
	if err == nil && resp.Ccode != 0 {
		return nil, errors.New("received non zero completion code in IPMI response")
	}

	return &resp, nil
}

// Close IPMI device file.
func (i *ipmiClient) Close() error {
	return i.devFile.Close()
}
