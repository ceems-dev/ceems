//go:build !noipmi
// +build !noipmi

package collector

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ceems-dev/ceems/internal/security"
	"github.com/ceems-dev/ceems/pkg/ipmi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	ipmidcmiStdout = map[string]string{
		"freeipmi": `
Current Power                        : 332 Watts
Minimum Power over sampling duration : 68 watts
Maximum Power over sampling duration : 504 watts
Average Power over sampling duration : 348 watts
Time Stamp                           : 11/03/2023 - 08:36:29
Statistics reporting time period     : 2685198000 milliseconds
Power Measurement                    : Active
`, "freeipmiAlt": `
Current Power                        : 332 watts
Minimum Power over sampling duration : 68 Watts
Maximum Power over sampling duration : 504 Watts
Average Power over sampling duration : 348 Watts
Time Stamp                           : 11/03/2023 - 08:36:29
Statistics reporting time period     : 2685198000 milliseconds
Power Measurement                    : Active
`, "ipmitutil": `
ipmiutil dcmi ver 3.17
-- BMC version 6.10, IPMI version 2.0
DCMI Version:                   1.5
DCMI Power Management:          Supported
DCMI System Interface Access:   Supported
DCMI Serial TMode Access:       Supported
DCMI Secondary LAN Channel:     Supported
Current Power:                   332 Watts
Min Power over sample duration:  68 Watts
Max Power over sample duration:  504 Watts
Avg Power over sample duration:  348 Watts
Timestamp:                       Thu Feb 15 09:37:32 2024

Sampling period:                 1000 ms
Power reading state is:          active
Exception Action:  OEM defined
Power Limit:       896 Watts (inactive)
Correction Time:   62914560 ms
Sampling period:   1472 sec
ipmiutil dcmi, completed successfully
`, "ipmitool": `

	Instantaneous power reading:                   332 Watts
	Minimum during sampling period:                 68 Watts
	Maximum during sampling period:                504 Watts
	Average power reading over sample period:      348 Watts
	IPMI timestamp:                           Thu Feb 15 08:36:01 2024
	Sampling period:                          00000001 Seconds.
	Power reading state is:                   activated

`, "capmc": `{
"start_time":"2015-04-01 17:02:10",
"avg":348,
"min":68,
"max":504,
"window_len":600,
"e":0,
"err_msg":""
}`,
	}
	ipmidcmiStdoutDisactive = map[string]string{
		"freeipmi":   "Power Measurement                    : Not Available",
		"ipmitutil":  "Power reading state is:          not active",
		"ipmitool":   "Power reading state is:                   deactivated",
		crayPowerCap: `{"e":1,"err_msg":"failed"}`,
	}
	expectedPower = map[string]float64{
		"dcmi_current": 332,
		"dcmi_min":     68,
		"dcmi_max":     504,
		"dcmi_avg":     348,
	}
	expectedCapmcPower = map[string]float64{
		"dcmi_current": 348,
		"dcmi_min":     68,
		"dcmi_max":     504,
		"dcmi_avg":     348,
	}
	testSensorRecords = []*ipmi.FullSensorRecord{
		{Identity: "Sensor 1"},
		{Identity: "Sensor 2"},
	}
	expectedSensorReading = map[*ipmi.FullSensorRecord]float64{
		testSensorRecords[0]: 123,
		testSensorRecords[1]: 223,
	}
)

type mockIPMIClient struct {
	dcmiCounter, sensorCounter int
}

func newMockIPMIClient() ipmi.Client {
	return &mockIPMIClient{}
}

func (c *mockIPMIClient) Close() error {
	return nil
}

func (c *mockIPMIClient) Do(r *ipmi.Request) (*ipmi.Response, error) {
	return nil, nil //nolint:nilnil
}

func (c *mockIPMIClient) DCMIPowerReading() (*ipmi.PowerReading, error) {
	if c.dcmiCounter == 2 {
		return nil, errors.New("some error")
	}

	c.dcmiCounter++

	return &ipmi.PowerReading{
		Minimum: expectedPower["dcmi_min"],
		Maximum: expectedPower["dcmi_max"],
		Current: expectedPower["dcmi_current"],
		Average: expectedPower["dcmi_avg"],
	}, nil
}

func (c *mockIPMIClient) LanIP() (*string, error) {
	ip := "10.0.0.1"

	return &ip, nil
}

func (c *mockIPMIClient) SensorRecords() ([]*ipmi.FullSensorRecord, error) {
	return testSensorRecords, nil
}

func (c *mockIPMIClient) SensorReadings(records []*ipmi.FullSensorRecord) (map[*ipmi.FullSensorRecord]float64, error) {
	if c.sensorCounter == 1 {
		return nil, errors.New("some error")
	}

	c.sensorCounter++

	return expectedSensorReading, nil
}

func TestIPMICollector(t *testing.T) {
	_, err := CEEMSExporterApp.Parse([]string{
		"--collector.ipmi.dcmi.cmd", "testdata/ipmi/capmc/capmc",
		"--collector.ipmi.test-mode",
	})
	require.NoError(t, err)

	collector, err := NewIPMICollector(noOpLogger)
	require.NoError(t, err)

	// Setup background goroutine to capture metrics.
	metrics := make(chan prometheus.Metric)
	defer close(metrics)

	go func() {
		i := 0
		for range metrics {
			i++
		}
	}()

	err = collector.Update(metrics)
	require.NoError(t, err)

	err = collector.Stop(t.Context())
	require.NoError(t, err)
}

func TestIpmiMetrics(t *testing.T) {
	c := impiCollector{logger: noOpLogger}

	for testName, testString := range ipmidcmiStdout {
		var value map[string]float64

		var err error

		expectedOutput := expectedPower
		if testName == crayPowerCap {
			expectedOutput = expectedCapmcPower
			value, err = c.parseCapmcOutput([]byte(testString))
		} else {
			value, err = c.parseIPMIOutput([]byte(testString))
		}

		require.NoError(t, err)
		assert.Equal(t, expectedOutput, value, testName)
	}
}

func TestIpmiMetricsDisactive(t *testing.T) {
	c := impiCollector{logger: noOpLogger}

	for testName, testString := range ipmidcmiStdoutDisactive {
		var value map[string]float64
		if testName == crayPowerCap {
			value, _ = c.parseCapmcOutput([]byte(testString))
		} else {
			value, _ = c.parseIPMIOutput([]byte(testString))
		}

		assert.Empty(t, value, testName)
	}
}

func TestIpmiClientFinder(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "ipmi-dcmi",
			path: "freeipmi",
		},
		{
			name: "ipmitool",
			path: "openipmi",
		},
		{
			name: "ipmiutil",
			path: "ipmiutils",
		},
	}

	// Get PATH
	basePath := os.Getenv("PATH")

	for _, test := range tests {
		ipmiClientPath, err := filepath.Abs(filepath.Join("testdata/ipmi", test.path))
		require.NoError(t, err)

		// Set path
		t.Setenv("PATH", fmt.Sprintf("%s:%s", ipmiClientPath, basePath))

		ipmiClientSlice, err := findIPMICmd()
		require.NoError(t, err, test.name)
		assert.Equal(t, test.name, ipmiClientSlice[0], test.name)
	}
}

func TestCachedPowerReadings(t *testing.T) {
	tmpDir := t.TempDir()
	tmpIPMIPath := tmpDir + "/ipmiutil"

	// Set path
	t.Setenv("PATH", fmt.Sprintf("%s:%s", tmpDir, os.Getenv("PATH")))

	// Expected values
	expected := map[string]float64{"dcmi_avg": 49, "dcmi_current": 304, "dcmi_max": 304, "dcmi_min": 6}

	// When collector is being instantiated
	d1 := []byte(`#!/bin/bash
exit 1`)
	err := os.WriteFile(tmpIPMIPath, d1, 0o700) //nolint:gosec
	require.NoError(t, err)

	_, err = CEEMSExporterApp.Parse([]string{
		"--collector.ipmi.dcmi.cmd", tmpIPMIPath,
		"--collector.ipmi.test-mode",
	})
	require.NoError(t, err)

	collector, err := NewIPMICollector(noOpLogger)
	require.NoError(t, err)

	c := collector.(*impiCollector) //nolint:forcetypeassert

	// Setup background goroutine to capture metrics.
	metrics := make(chan prometheus.Metric)
	defer close(metrics)

	go func() {
		i := 0
		for range metrics {
			i++
		}
	}()

	// Get readings
	err = collector.Update(metrics)
	require.Error(t, err, "first scrape should result in error")

	// Now command should pass
	d1 = []byte(`#!/bin/bash

echo """ipmiutil dcmi ver 3.17
-- BMC version 6.10, IPMI version 2.0 
DCMI Version:                   1.5
DCMI Power Management:          Supported
DCMI System Interface Access:   Supported
DCMI Serial TMode Access:       Supported
DCMI Secondary LAN Channel:     Supported
  Current Power:                   304 Watts
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
ipmiutil dcmi, completed successfully"""`)
	err = os.WriteFile(tmpIPMIPath, d1, 0o700) //nolint:gosec
	require.NoError(t, err)

	// Get readings
	err = collector.Update(metrics)
	require.NoError(t, err)

	assert.Equal(t, expected, c.cachedDCMIReadings)

	// Modify script again to return error
	d1 = []byte(`#!/bin/bash
exit 1`)
	err = os.WriteFile(tmpIPMIPath, d1, 0o700) //nolint:gosec
	require.NoError(t, err)

	// Get readings
	got, err := c.update()
	require.NoError(t, err)

	assert.Equal(t, expected, got.dcmiPower)

	// Modify IPMI command to give 0 current usage
	d1 = []byte(`#!/bin/bash

echo """ipmiutil dcmi ver 3.17
-- BMC version 6.10, IPMI version 2.0 
DCMI Version:                   1.5
DCMI Power Management:          Supported
DCMI System Interface Access:   Supported
DCMI Serial TMode Access:       Supported
DCMI Secondary LAN Channel:     Supported
  Current Power:                   0 Watts
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
ipmiutil dcmi, completed successfully"""`)
	err = os.WriteFile(tmpIPMIPath, d1, 0o700) //nolint:gosec
	require.NoError(t, err)

	// Get readings again and we should get last cached values
	got, err = c.update()
	require.NoError(t, err)

	assert.Equal(t, expected, got.dcmiPower)
}

func TestIpmiNativeMode(t *testing.T) {
	_, err := CEEMSExporterApp.Parse([]string{
		"--collector.ipmi.dcmi.cmd", "testdata/ipmi/capmc/capmc",
		"--collector.ipmi.test-mode",
	})
	require.NoError(t, err)

	collector, err := NewIPMICollector(noOpLogger)
	require.NoError(t, err)

	c := collector.(*impiCollector) //nolint:forcetypeassert

	// Set native mode
	c.execMode = nativeMode
	c.client = newMockIPMIClient()
	c.sensorRecords = testSensorRecords

	// Setup security context
	cfg := &security.SCConfig{
		Name:         openIPMICtx,
		Logger:       noOpLogger,
		Func:         doIPMIRequests,
		ExecNatively: true,
	}
	secuCtx, err := security.NewSecurityContext(cfg)
	require.NoError(t, err)

	c.securityContexts[openIPMICtx] = secuCtx

	// Setup background goroutine to capture metrics.
	metrics := make(chan prometheus.Metric)
	defer close(metrics)

	go func() {
		i := 0
		for range metrics {
			i++
		}
	}()

	// Make first scrape and should get expected values
	err = c.Update(metrics)
	require.NoError(t, err)

	assert.Equal(t, expectedPower, c.cachedDCMIReadings)
	assert.Equal(t, expectedSensorReading, c.cachedSensorReadings)

	// Make second scrape where sensors should fail but should get from
	// cached
	got, err := c.update()
	require.NoError(t, err)

	assert.Equal(t, expectedSensorReading, got.sensors)
	assert.Equal(t, expectedSensorReading, c.cachedSensorReadings)

	// Make third scrape where DCMI should fail but should get from
	// cached
	got, err = c.update()
	require.NoError(t, err)

	assert.Equal(t, expectedPower, got.dcmiPower)
	assert.Equal(t, expectedPower, c.cachedDCMIReadings)
}
