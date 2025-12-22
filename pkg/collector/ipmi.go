//go:build !noimpi

package collector

// Taken from prometheus-community/ipmi_exporter/blob/master/collector_ipmi.go
// DCMI spec (old) https://www.intel.com/content/dam/www/public/us/en/documents/technical-specifications/dcmi-v1-5-rev-spec.pdf

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ceems-dev/ceems/internal/osexec"
	"github.com/ceems-dev/ceems/internal/security"
	"github.com/ceems-dev/ceems/pkg/ipmi"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ipmiCollectorSubsystem = "ipmi"
	ipmiDcmiLabel          = "ipmi_dcmi_power"
)

// Custom errors.
var (
	ErrIPMIUnavailable = errors.New("ipmi dcmi power readings are not active")
)

// Execution modes.
const (
	sudoMode       = "sudo"
	capabilityMode = "cap"
	nativeMode     = "native" // Native Go implementation of IPMI protocol
	testMode       = "test"
	crayPowerCap   = "capmc"
)

type impiCollector struct {
	logger               *slog.Logger
	hostname             string
	execMode             string
	ipmiCmd              []string
	client               ipmi.Client
	sensorRecords        []*ipmi.FullSensorRecord
	securityContexts     map[string]*security.SecurityContext
	cachedDCMIReadings   map[string]float64
	cachedSensorReadings map[*ipmi.FullSensorRecord]float64
	metricDesc           map[string]*prometheus.Desc
}

/*
	Expected output from DCMI spec
	Ref: https://www.intel.com/content/dam/www/public/us/en/documents/technical-specifications/dcmi-v1-5-rev-spec.pdf

	Different IPMI implementations report stats in different formats:

	**FreeIPMI**:
	Command: `ipmi-dcmi --get-system-power-statistics`
	Output:
	----------------------------------------------------------------------
	Current Power                        : 164 Watts
	Minimum Power over sampling duration : 48 watts
	Maximum Power over sampling duration : 361 watts
	Average Power over sampling duration : 157 watts
	Time Stamp                           : 12/29/2023 - 08:58:00
	Statistics reporting time period     : 1473439000 milliseconds
	Power Measurement                    : Active
	----------------------------------------------------------------------

	**OpenIPMI**:
	Command: `ipmitool dcmi power reading`
	Output:
	----------------------------------------------------------------------

		Instantaneous power reading:                    50 Watts
		Minimum during sampling period:                  6 Watts
		Maximum during sampling period:                304 Watts
		Average power reading over sample period:       49 Watts
		IPMI timestamp:                           Thu Feb 15 08:36:01 2024
		Sampling period:                          00000001 Seconds.
		Power reading state is:                   activated

	----------------------------------------------------------------------

	**IPMIUtils**:
	Command: `ipmiutil dcmi power`
	Output:
	----------------------------------------------------------------------
	ipmiutil dcmi ver 3.17
	-- BMC version 6.10, IPMI version 2.0
	DCMI Version:                   1.5
	DCMI Power Management:          Supported
	DCMI System Interface Access:   Supported
	DCMI Serial TMode Access:       Supported
	DCMI Secondary LAN Channel:     Supported
	Current Power:                   49 Watts
	Min Power over sample duration:  6 Watts
	Max Power over sample duration:  304 Watts
	Avg Power over sample duration:  49 Watts
	Timestamp:                       Thu Feb 15 09:37:32 2024

	Sampling period:                 1000 ms
	Power reading state is:          active
	Exception Action:  OEM defined
	Power Limit:       896 Watts (inactive)
	Correction Time:   62914560 ms
	Sampling period:   1472 sec
	ipmiutil dcmi, completed successfully
	----------------------------------------------------------------------

	**Capmc**: Cray specific
	Command: `capmc get_system_power -w 600` Ref: https://github.com/Cray-HPE/hms-capmc/blob/release/csm-1.0/api/swagger.yaml
	Output:
	----------------------------------------------------------------------
	{
	"start_time":"2015-04-01 17:02:10",
	"avg":5942,
	"min":5748,
	"max":6132,
	"window_len":600,
	"e":0,
	"err_msg":""
	}
	----------------------------------------------------------------------

	NOTE: If the native Go implementation is stable enough, we should remove
	all other modes as it does not add any value to the collector. Currently native
	implementation is only a fallback mode when no other IPMI utils are found on the
	node.
*/

var (
	ipmiDcmiCmd = CEEMSExporterApp.Flag(
		"collector.ipmi.dcmi.cmd",
		"IPMI DCMI command to get system power statistics. Use full path to executables.",
	).Default("").String()
	ipmiPwrEnergySensors = CEEMSExporterApp.Flag(
		"collector.ipmi.power-energy-sensor-readings",
		"Enables collection of IPMI energy and/or power sensor readings. Sensors will be detected based on units. (default: disabled).",
	).Default("false").Bool()
	ipmiMiscSensors = CEEMSExporterApp.Flag(
		"collector.ipmi.sensor-id",
		"Sensor IDs to monitor and export metrics.",
	).Uint8List()
	ipmiDevNum = CEEMSExporterApp.Flag(
		"collector.ipmi.dev-num",
		"Device number used by OpenIPMI driver. For e.g. if device is found at /dev/ipmi0, device number is 0",
	).Default("0").Int()
	forceNativeMode = CEEMSExporterApp.Flag(
		"collector.ipmi.force-native-mode",
		"Force native mode using OpenIPMI driver.",
	).Default("false").Bool()

	// test flags. Hidden.
	ipmiDcmiTestMode = CEEMSExporterApp.Flag(
		"collector.ipmi.test-mode",
		"Enables IPMI DCMI collector in test mode. Only used in unit and e2e tests.",
	).Default("false").Hidden().Bool()
)

var (
	ipmiDcmiCmds = []string{
		"ipmi-dcmi --get-system-power-statistics",
		"ipmitool dcmi power reading",
		"ipmiutil dcmi power",
		// Cray capmc
		// Ref: https://cug.org/proceedings/cug2015_proceedings/includes/files/pap132.pdf
		"capmc get_system_power -w 600",
	}

	// IPMIUtil Ref: https://sourceforge.net/p/ipmiutil/code-git/ci/master/tree/util/idcmi.c#l343
	// FreeIPMI Ref: https://github.com/chu11/freeipmi-mirror/blob/f4057a93cbc11ecf4bfadb4bcd5d375f65f01f19/ipmi-dcmi/ipmi-dcmi.c#L1828-L1830
	// IPMITool Ref: https://github.com/ipmitool/ipmitool/blob/be11d948f89b10be094e28d8a0a5e8fb532c7b60/lib/ipmi_dcmi.c#L1447-L1451
	ipmiDCMIPowerMeasurementRegex = regexp.MustCompile(
		`^\s*(?:Power Measurement|Power reading state is)\s*:\s*(?P<value>[A|a]ctive|Not\sAvailable|not\sactive|activated|deactivated).*`,
	)
	ipmiDCMIPowerReadingRegexMap = map[string]*regexp.Regexp{
		"current": regexp.MustCompile(
			`^\s*(?:Current Power|Instantaneous power reading)\s*:\s*(?P<value>[0-9.]*)\s*[w|W]atts.*`,
		),
		"min": regexp.MustCompile(
			`^\s*(?:Minimum Power over sampling duration|Minimum during sampling period|Min Power over sample duration)\s*:\s*(?P<value>[0-9.]*)\s*[w|W]atts.*`,
		),
		"max": regexp.MustCompile(
			`^\s*(?:Maximum Power over sampling duration|Maximum during sampling period|Max Power over sample duration)\s*:\s*(?P<value>[0-9.]*)\s*[w|W]atts.*`,
		),
		"avg": regexp.MustCompile(
			`^\s*(?:Average Power over sampling duration|Average power reading over sample period|Avg Power over sample duration)\s*:\s*(?P<value>[0-9.]*)\s*[w|W]atts.*`,
		),
	}
)

// Security context names.
const (
	ipmiExecCmdCtx = "ipmi_exec_cmd"
	openIPMICtx    = "open_ipmi"
)

type ipmiReadings struct {
	dcmiPower map[string]float64
	sensors   map[*ipmi.FullSensorRecord]float64
}

type ipmiClientSecurityCtxData struct {
	client        ipmi.Client
	sensorRecords []*ipmi.FullSensorRecord
	readings      *ipmiReadings
}

func init() {
	RegisterCollector(ipmiCollectorSubsystem, defaultDisabled, NewIPMICollector)
}

// NewIPMICollector returns a new Collector exposing IMPI DCMI power metrics.
func NewIPMICollector(logger *slog.Logger) (Collector, error) {
	// Check if native mode is enabled when sensors are provided
	if (*ipmiPwrEnergySensors || len(*ipmiMiscSensors) > 0) && !*forceNativeMode {
		return nil, errors.New("fetching ipmi sensor readings is only supported when --collector.ipmi.force-native-mode is enabled")
	}

	var execMode string

	// Initialize metricDesc map
	metricDesc := make(map[string]*prometheus.Desc, 5)

	metricDesc["dcmi_current"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, ipmiDcmiLabel, "current_watts"),
		"Current power consumption reported by DCMI in watts", []string{"hostname"}, nil,
	)
	metricDesc["dcmi_min"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, ipmiDcmiLabel, "min_watts"),
		"Minimum power consumption reported by DCMI in watts", []string{"hostname"}, nil,
	)
	metricDesc["dcmi_max"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, ipmiDcmiLabel, "max_watts"),
		"Maximum power consumption reported by DCMI in watts", []string{"hostname"}, nil,
	)
	metricDesc["dcmi_avg"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, ipmiDcmiLabel, "avg_watts"),
		"Average power consumption reported by DCMI in watts", []string{"hostname"}, nil,
	)
	metricDesc["sensors"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, "ipmi_sensor_reading", "current"),
		"Current reading of IPMI sensor", []string{"hostname", "sensorname", "sensorunits"}, nil,
	)

	// If no IPMI command is provided, try to find one
	var cmdSlice []string

	var err error

	// If native mode is forced set execMode and goto outside
	if *forceNativeMode {
		execMode = nativeMode

		goto outside
	}

	if *ipmiDcmiCmd == "" {
		cmdSlice, err = findIPMICmd()
		if err != nil {
			logger.Info("None of ipmitool,ipmiutil,ipmi-dcmi commands found. Using native implementation using OpenIPMI interface")

			execMode = nativeMode

			goto outside
		}
	} else {
		cmdSlice = strings.Split(*ipmiDcmiCmd, " ")
	}

	logger.Debug("Using IPMI command", "ipmi", strings.Join(cmdSlice, " "))

	// Append to cmdSlice an empty string if it has len of 1
	// We dont want nil pointer references when we execute command
	if len(cmdSlice) == 1 {
		cmdSlice = append(cmdSlice, "")
	}

	// If we are in test mode, set execMode to test
	if *ipmiDcmiTestMode {
		execMode = testMode

		goto outside
	}

	// Verify if running ipmiDcmiCmd works
	// If it works natively it means the current process is running as root.
	// Note that the collectors are initiated before dropping privileges and so
	// this command should succeed.
	// Eventually we drop all the privileges and use cap_setuid and cap_setgid to
	// execute ipmi command in subprocess as root.
	// So we set execMode as capabilityMode here too.
	_, err = osexec.Execute(cmdSlice[0], cmdSlice[1:], nil)
	if err == nil {
		execMode = capabilityMode

		goto outside
	}

	// If ipmiDcmiCmd failed to run and if sudo is not already present in command,
	// add sudo to command and execute. If current user has sudo rights it will be a success
	if cmdSlice[0] != sudoMode {
		_, err := osexec.ExecuteWithTimeout(sudoMode, cmdSlice, 1, nil)
		if err == nil {
			execMode = sudoMode

			goto outside
		}
	}

	// As last attempt, run the command as root user by forking subprocess
	// as root. If there is setuid cap on the process, it will be a success
	_, err = osexec.ExecuteAs(cmdSlice[0], cmdSlice[1:], 0, 0, nil)
	if err == nil {
		execMode = capabilityMode

		goto outside
	}

	// By this point, if we still did not find a execMode return error
	if execMode == "" {
		logger.Error("Failed to execute IPMI commands. Ensure enough privileges are set on exporter process")

		return nil, errors.New("failed to execute ipmi commands due to lack of privileges")
	}

outside:

	logger.Debug("IPMI collector", "execution_mode", execMode)

	collector := impiCollector{
		logger:               logger,
		hostname:             hostname,
		execMode:             execMode,
		metricDesc:           metricDesc,
		cachedDCMIReadings:   make(map[string]float64),
		cachedSensorReadings: make(map[*ipmi.FullSensorRecord]float64),
		securityContexts:     make(map[string]*security.SecurityContext),
	}

	// Setup necessary capabilities.
	// For capabilityMode we need cap_setuid and cap_setgid.
	// For sudoMode we wont need any more extra capabilities.
	// For nativeMode we need cap_dac_override capability to talk to device.
	switch execMode {
	case capabilityMode:
		// Setup command
		collector.ipmiCmd = cmdSlice

		caps, err := setupAppCaps([]string{"cap_setuid", "cap_setgid"})
		if err != nil {
			logger.Warn("Failed to parse capability name(s)", "err", err)
		}

		// Setup security context
		cfg := &security.SCConfig{
			Name:         ipmiExecCmdCtx,
			Caps:         caps,
			Func:         security.ExecAsUser,
			Logger:       logger,
			ExecNatively: disableCapAwareness,
		}

		collector.securityContexts[ipmiExecCmdCtx], err = security.NewSecurityContext(cfg)
		if err != nil {
			logger.Error("Failed to create a security context for IPMI collector", "err", err)

			return nil, err
		}
	case nativeMode:
		// Capability to be able to talk to /dev/ipmi0
		caps, err := setupAppCaps([]string{"cap_dac_override"})
		if err != nil {
			logger.Warn("Failed to parse capability name(s)", "err", err)
		}

		// IPMI config
		ipmiConfig := &ipmi.Config{
			Logger:  logger.With("subsystem", "ipmi_client"),
			DevNum:  *ipmiDevNum,
			Timeout: time.Second,
		}

		// Setup IPMI client
		collector.client, err = ipmi.NewClient(ipmiConfig)
		if err != nil {
			logger.Error("Failed to create a IPMI client", "err", err)

			return nil, err
		}

		if *ipmiPwrEnergySensors || len(*ipmiMiscSensors) > 0 {
			// Get all sensor records
			sensorRecords, err := collector.client.SensorRecords()
			if err != nil {
				logger.Error("Failed to get sensor records", "err", err)

				return nil, err
			}

			// Filter records
			for _, record := range sensorRecords {
				// When power and energy sensors are requested, filter based on units
				if *ipmiPwrEnergySensors {
					if record.BaseUnit == ipmi.SensorUnitWatts || record.BaseUnit == ipmi.SensorUnitJoules {
						collector.sensorRecords = append(collector.sensorRecords, record)
					}
				}

				// When sensor IDs are requested
				if slices.Contains(*ipmiMiscSensors, record.Number) {
					collector.sensorRecords = append(collector.sensorRecords, record)
				}
			}

			// Remove duplicates
			slices.SortFunc(collector.sensorRecords, func(a, b *ipmi.FullSensorRecord) int {
				return cmp.Compare(a.Number, b.Number)
			})

			collector.sensorRecords = slices.CompactFunc(collector.sensorRecords, func(a, b *ipmi.FullSensorRecord) bool {
				return a.Number == b.Number
			})

			// Get sensor names
			sensorNames := make([]string, len(collector.sensorRecords))
			for irecord, record := range collector.sensorRecords {
				sensorNames[irecord] = record.Identity
			}

			logger.Debug("Sensor to monitor", "sensors", strings.Join(sensorNames, ","), "num_sensors", len(collector.sensorRecords))
		}

		// Setup security context
		cfg := &security.SCConfig{
			Name:         openIPMICtx,
			Caps:         caps,
			Func:         doIPMIRequests,
			Logger:       logger,
			ExecNatively: disableCapAwareness,
		}

		collector.securityContexts[openIPMICtx], err = security.NewSecurityContext(cfg)
		if err != nil {
			logger.Error("Failed to create a security context for IPMI collector", "err", err)

			return nil, err
		}
	case testMode:
		collector.ipmiCmd = cmdSlice
	}

	return &collector, nil
}

// Update implements Collector and exposes IPMI DCMI power related metrics.
func (c *impiCollector) Update(ch chan<- prometheus.Metric) error {
	// Get IPMI readings
	ipmiReadings, err := c.update()
	if err != nil {
		return ErrNoData
	}

	// Returned value 0 means Power Measurement is not avail
	for rType, rValue := range ipmiReadings.dcmiPower {
		if rValue > 0 {
			ch <- prometheus.MustNewConstMetric(c.metricDesc[rType], prometheus.GaugeValue, float64(rValue), c.hostname)

			c.cachedDCMIReadings[rType] = rValue
		}
	}

	// Returned value 0 means sensor reading is not avail
	for sType, sValue := range ipmiReadings.sensors {
		if sValue > 0 {
			ch <- prometheus.MustNewConstMetric(c.metricDesc["sensors"], prometheus.GaugeValue, float64(sValue), c.hostname, sType.Identity, sType.BaseUnit.String())

			c.cachedSensorReadings[sType] = sValue
		}
	}

	return nil
}

// Stop releases system resources used by the collector.
func (c *impiCollector) Stop(_ context.Context) error {
	c.logger.Debug("Stopping", "collector", ipmiCollectorSubsystem)

	// Close fd when native mode is being used
	if c.execMode == nativeMode {
		err := c.client.Close()
		if err != nil {
			c.logger.Debug("Failed to close OpenIPMI device fd", "err", err)

			return err
		}
	}

	return nil
}

// update returns current power readings or cached ones.
func (c *impiCollector) update() (*ipmiReadings, error) {
	// Get power consumption from DCMI and sensor readings
	// IPMI commands tend to fail frequently. If that happens we use last cached metric
	readings, err := c.getIPMIReadings()
	if err != nil {
		// If there are no current and cached readings, return error
		if readings == nil {
			if len(c.cachedDCMIReadings) == 0 && len(c.cachedSensorReadings) == 0 {
				return nil, ErrNoData
			}

			c.logger.Error("Failed to get readings from IPMI. Using last cached values", "err", err)

			return &ipmiReadings{dcmiPower: c.cachedDCMIReadings, sensors: c.cachedSensorReadings}, nil
		}

		// If DCMI readings are not available and cached readings are available set them
		if len(readings.dcmiPower) == 0 && len(c.cachedDCMIReadings) > 0 {
			c.logger.Error(
				"Failed to get power statistics from IPMI DCMI. Using last cached value",
				"err", err, "cached_value", c.cachedDCMIReadings["dcmi_current"],
			)

			readings.dcmiPower = c.cachedDCMIReadings
		}

		// If sensor readings are not available and cached readings are available set them
		if len(c.sensorRecords) > 0 && len(readings.sensors) == 0 && len(c.cachedSensorReadings) > 0 {
			c.logger.Error("Failed to get readings from IPMI sensors. Using last cached values", "err", err)

			readings.sensors = c.cachedSensorReadings
		}
	} else {
		// Ensure powerReadings are non nil
		// Check only current usage which is more important
		if currentUsage, ok := readings.dcmiPower["dcmi_current"]; !ok || currentUsage == 0 {
			c.logger.Error(
				"IPMI DCMI returned null values. Using last cached value",
				"err", err, "cached_value", c.cachedDCMIReadings["dcmi_current"],
			)

			readings.dcmiPower = c.cachedDCMIReadings
		}
	}

	return readings, nil
}

// Get current, min and max DCMI power and sensor readings.
func (c *impiCollector) getIPMIReadings() (*ipmiReadings, error) {
	// If mode is native, make request in security context
	if c.execMode == nativeMode {
		return c.doRequestInSecurityContext()
	}
	// Execute IPMI command
	ipmiOutput, err := c.executeIPMICmd()
	if err != nil {
		return nil, err
	}

	// For capmc, we get JSON output, so treat it differently
	var values map[string]float64
	if filepath.Base(c.ipmiCmd[0]) == crayPowerCap {
		values, err = c.parseCapmcOutput(ipmiOutput)
	} else {
		// Parse IPMI output
		values, err = c.parseIPMIOutput(ipmiOutput)
	}

	if err != nil {
		return nil, err
	}

	return &ipmiReadings{dcmiPower: values}, nil
}

// Parse current, min and max power readings for capmc output.
func (c *impiCollector) parseCapmcOutput(stdOut []byte) (map[string]float64, error) {
	// Unmarshal JSON output
	var data map[string]any

	err := json.Unmarshal(stdOut, &data)
	if err != nil {
		return nil, fmt.Errorf("%s Power readings command failed", crayPowerCap)
	}

	// Check error code
	if errValue, valueOk := data["e"].(float64); valueOk && errValue != 0 {
		if errMsg, msgOk := data["err_msg"].(string); msgOk {
			return nil, fmt.Errorf("capmc Power readings not Active: %s", errMsg)
		}
	}

	// Get power readings
	powerReadings := make(map[string]float64, 3)

	for rType := range ipmiDCMIPowerReadingRegexMap {
		if value, ok := data[rType]; ok {
			if valueFloat, valueOk := value.(float64); valueOk {
				powerReadings["dcmi_"+rType] = valueFloat
			}
		}
	}

	// capmc does not return current power. So we use avg as proxy for current
	if powerReadings["dcmi_avg"] > 0 {
		powerReadings["dcmi_current"] = powerReadings["dcmi_avg"]
	}

	return powerReadings, nil
}

// Parse current, min and max power readings for IPMI commands.
func (c *impiCollector) parseIPMIOutput(stdOut []byte) (map[string]float64, error) {
	// Check for Power Measurement are avail
	value, err := getValue(stdOut, ipmiDCMIPowerMeasurementRegex)
	if err != nil {
		return nil, err
	}

	// When Power Measurement in 'Active' state - we can get watts
	powerReadings := make(map[string]float64, 3)

	if value == "active" || value == "Active" || value == "activated" {
		// Get power readings
		for rType, regex := range ipmiDCMIPowerReadingRegexMap {
			reading, err := getValue(stdOut, regex)
			if err == nil {
				readingValue, err := strconv.ParseFloat(reading, 64)
				if err == nil {
					powerReadings["dcmi_"+rType] = readingValue
				}
			}
		}

		return powerReadings, nil
	}

	return nil, ErrIPMIUnavailable
}

// Execute IPMI command based.
func (c *impiCollector) executeIPMICmd() ([]byte, error) {
	var stdOut []byte

	var err error

	// Execute found ipmi command
	switch c.execMode {
	case capabilityMode:
		stdOut, err = c.executeCmdInSecurityContext()
	case sudoMode:
		stdOut, err = osexec.ExecuteWithTimeout(sudoMode, c.ipmiCmd, 1, nil)
	// Only used in e2e and unit tests
	case testMode:
		stdOut, err = osexec.Execute(c.ipmiCmd[0], c.ipmiCmd[1:], nil)
	default:
		return nil, ErrNoData
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute IPMI command: %w", err)
	}

	return stdOut, nil
}

// executeCmdInSecurityContext executes IPMI command within a security context.
func (c *impiCollector) executeCmdInSecurityContext() ([]byte, error) {
	// Execute command as root
	dataPtr := &security.ExecSecurityCtxData{
		Cmd:    c.ipmiCmd,
		Logger: c.logger,
		UID:    0,
		GID:    0,
	}

	// Read stdOut of command into data
	if securityCtx, ok := c.securityContexts[ipmiExecCmdCtx]; ok {
		err := securityCtx.Exec(dataPtr)
		if err != nil {
			return nil, err
		}
	} else {
		err := security.ExecAsUser(dataPtr)
		if err != nil {
			return nil, err
		}
	}

	return dataPtr.StdOut, nil
}

// doRequestInSecurityContext makes requests to IPMI device interface within a security context.
func (c *impiCollector) doRequestInSecurityContext() (*ipmiReadings, error) {
	// Execute command as root
	dataPtr := &ipmiClientSecurityCtxData{
		client:        c.client,
		sensorRecords: c.sensorRecords,
	}

	// Read stdOut of command into data
	if securityCtx, ok := c.securityContexts[openIPMICtx]; ok {
		// Always return readings as we might have partial result
		// in readings
		err := securityCtx.Exec(dataPtr)
		if err != nil {
			return dataPtr.readings, err
		}
	} else {
		return nil, security.ErrNoSecurityCtx
	}

	return dataPtr.readings, nil
}

func doIPMIRequests(data any) error {
	// Assert data
	var d *ipmiClientSecurityCtxData

	var ok bool
	if d, ok = data.(*ipmiClientSecurityCtxData); !ok {
		return security.ErrSecurityCtxDataAssertion
	}

	// Initialize readings
	d.readings = &ipmiReadings{}

	// Get current power reading from DCMI
	dcmiReading, err := d.client.DCMIPowerReading()
	if err != nil {
		return err
	}

	// Read power readings into dataPointer
	d.readings.dcmiPower = map[string]float64{
		"dcmi_min":     float64(dcmiReading.Minimum),
		"dcmi_max":     float64(dcmiReading.Maximum),
		"dcmi_avg":     float64(dcmiReading.Average),
		"dcmi_current": float64(dcmiReading.Current),
	}

	// Get sensor readings
	sensorReadings, err := d.client.SensorReadings(d.sensorRecords)
	if err != nil {
		return err
	}

	// Read sensor readings into dataPointer
	d.readings.sensors = sensorReadings

	return nil
}

// Find IPMI command from list of different IPMI implementations.
func findIPMICmd() ([]string, error) {
	for _, cmd := range ipmiDcmiCmds {
		cmdSlice := strings.Split(cmd, " ")

		_, err := exec.LookPath(cmdSlice[0])
		if err == nil {
			return cmdSlice, nil
		}

		// Check if binary exists in /sbin or /usr/sbin
		_, err = lookPath(cmdSlice[0])
		if err == nil {
			return cmdSlice, nil
		}
	}

	return nil, errors.New("no ipmi command found")
}

// Get value based on regex from IPMI output.
func getValue(ipmiOutput []byte, regex *regexp.Regexp) (string, error) {
	for line := range strings.SplitSeq(string(ipmiOutput), "\n") {
		match := regex.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		for i, name := range regex.SubexpNames() {
			if name != "value" {
				continue
			}

			return match[i], nil
		}
	}

	return "", fmt.Errorf("could not find value in output: %s", string(ipmiOutput))
}
