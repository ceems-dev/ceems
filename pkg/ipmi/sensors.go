package ipmi

import (
	"errors"
	"fmt"
	"unsafe"
)

// IPMI sensor related constants.
const (
	IPMI_SENSOR_RECORD_CMD    = 0x23
	IPMI_SENSOR_RECORD_NETFN  = 0xa
	IPMI_SENSOR_READING_CMD   = 0x2d
	IPMI_SENSOR_READING_NETFN = 0x4
)

// SensorRecords returns full sensor records info.
func (i *ipmiClient) SensorRecords() ([]*FullSensorRecord, error) {
	// All errors
	var errs error

	var recordID uint8 = 0

	var fullSensorRecords []*FullSensorRecord

	for {
		// Request payload
		msgData := [6]uint8{0x0, 0x0, recordID, 0x0, 0x0, 0xff}

		// IPMI Request
		req := Request{
			Addr:    uintptr(unsafe.Pointer(&i.bmcAddr)),
			AddrLen: uint(unsafe.Sizeof(i.bmcAddr)),
			Msgid:   1,
			Msg: ipmiMsg{
				Data:    uintptr(unsafe.Pointer(&msgData[0])), //nolint:gosec
				DataLen: 6,
				Netfn:   IPMI_SENSOR_RECORD_NETFN,
				Cmd:     IPMI_SENSOR_RECORD_CMD,
			},
		}

		// Do request and read response
		resp, err := i.Do(&req)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to make ipmi request to get sensor record %d: %w", recordID, err))

			continue
		}

		sensorRecord := &FullSensorRecord{}

		err = sensorRecord.DecodeFromBytes(resp.Data[:])
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to decode sensor record %d: %w", recordID, err))

			continue
		}

		i.logger.Debug(
			"Full sensor record", "record_id", recordID, "sensor_number", sensorRecord.Number,
			"description", sensorRecord.Identity, "units", sensorRecord.BaseUnit,
		)

		fullSensorRecords = append(fullSensorRecords, sensorRecord)

		// Next recordID
		recordID = resp.Data[1]

		// If recordID reaches 255, we are at the end of the list
		if recordID == 255 {
			break
		}
	}

	return fullSensorRecords, errs
}

// SensorReadings returns readings of sensors of given IDs.
func (i *ipmiClient) SensorReadings(sensorRecords []*FullSensorRecord) (map[*FullSensorRecord]float64, error) {
	// Initialise sensor readings map
	readings := make(map[*FullSensorRecord]float64, len(sensorRecords))

	var errs error

	// Get reading for every sensor record
	for _, record := range sensorRecords {
		// Request payload
		msgData := [1]uint8{record.Number}

		// IPMI Request
		req := Request{
			Addr:    uintptr(unsafe.Pointer(&i.bmcAddr)),
			AddrLen: uint(unsafe.Sizeof(i.bmcAddr)),
			Msgid:   1,
			Msg: ipmiMsg{
				Data:    uintptr(unsafe.Pointer(&msgData[0])),
				DataLen: 1,
				Netfn:   IPMI_SENSOR_READING_NETFN,
				Cmd:     IPMI_SENSOR_READING_CMD,
			},
		}

		// Do request and read response
		resp, err := i.Do(&req)
		if err != nil {
			i.logger.Error("Failed to make IPMI request to get reading of sensor", "sensor", record.Identity, "err", err)
			errs = errors.Join(errs, fmt.Errorf("failed to make ipmi request to get reading of sensor %s: %w", record.Identity, err))

			continue
		}

		readings[record] = record.ConvertReading(int16(resp.Data[1]))
	}

	return readings, errs
}
