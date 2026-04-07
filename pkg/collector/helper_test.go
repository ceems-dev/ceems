package collector

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type logLine struct {
	Level   string `json:"level"`
	Message string `json:"msg"`
	Source  string `json:"source"`
	AAttr   int    `json:"a"`
	PAttr   string `json:"p"`
}

func TestGokitLogger(t *testing.T) {
	// logfmt
	for _, lvl := range []string{"info", "error", "warn", ""} {
		// slog logger
		buf := &bytes.Buffer{}
		slogger := slog.New(slog.NewTextHandler(buf, nil))
		logger := NewGokitLogger(lvl, slogger)

		// When level is empty string, we default to info
		if lvl == "" {
			lvl = "info"
		}

		kvs := []any{"a", 123}
		lc := log.With(logger, kvs...)

		err := lc.Log("msg", "message")
		require.NoError(t, err)
		assert.Equal(
			t, "msg=message source=helper_test.go:42 a=123",
			strings.TrimSpace(strings.Split(buf.String(), "level="+strings.ToUpper(lvl))[1]),
		)

		buf.Reset()

		lc = log.WithPrefix(lc, "p", "first")
		lc.Log("msg", "message")
		require.NoError(t, err)
		assert.Equal(
			t, "msg=message p=first source=helper_test.go:52 a=123",
			strings.TrimSpace(strings.Split(buf.String(), "level="+strings.ToUpper(lvl))[1]),
		)
	}

	// json format
	for _, lvl := range []string{"info", "error", "warn", ""} {
		// slog logger
		buf := &bytes.Buffer{}
		slogger := slog.New(slog.NewJSONHandler(buf, nil))
		logger := NewGokitLogger(lvl, slogger)

		// When level is empty string, we default to info
		if lvl == "" {
			lvl = "info"
		}

		kvs := []any{"a", 123}
		lc := log.With(logger, kvs...)

		err := lc.Log("msg", "message")
		require.NoError(t, err)

		var got logLine

		err = json.Unmarshal(buf.Bytes(), &got)
		require.NoError(t, err)

		assert.Equal(t, logLine{strings.ToUpper(lvl), "message", "helper_test.go:75", 123, ""}, got)

		buf.Reset()

		lc = log.WithPrefix(lc, "p", "first")
		err = lc.Log("msg", "message")
		require.NoError(t, err)

		err = json.Unmarshal(buf.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, logLine{strings.ToUpper(lvl), "message", "helper_test.go:88", 123, "first"}, got)
	}
}

func TestAreEqual(t *testing.T) {
	testCases := []struct {
		inputA   []string
		inputB   []string
		expected bool
	}{
		{
			inputA:   []string{"a", "b", "c"},
			inputB:   []string{"a", "c", "b"},
			expected: true,
		},
		{
			inputA:   []string{"a1", "b", "c2"},
			inputB:   []string{"a1", "c2", "b"},
			expected: true,
		},
		{
			inputA:   []string{"a", "b", "c2"},
			inputB:   []string{"a1", "c2", "b"},
			expected: false,
		},
		{
			inputA:   []string{"a", "b"},
			inputB:   []string{"a1", "c2", "b"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		got := areEqual(tc.inputA, tc.inputB)
		assert.Equal(t, tc.expected, got)
	}
}

func TestElementsCount(t *testing.T) {
	testCases := []struct {
		input    []string
		expected map[string]uint64
	}{
		{
			input: []string{"a", "a", "b", "c", "c"},
			expected: map[string]uint64{
				"a": 2,
				"b": 1,
				"c": 2,
			},
		},
		{
			input: []string{"a", "b", "c", "c", "a"},
			expected: map[string]uint64{
				"a": 2,
				"b": 1,
				"c": 2,
			},
		},
	}

	for _, tc := range testCases {
		got := elementCounts(tc.input)
		for h, v := range got {
			assert.Equal(t, tc.expected[h.Value()], v)
		}
	}
}

func TestParseRange(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "0-2",
			expected: []string{"0", "1", "2"},
		},
		{
			input:    "0-2,7-9",
			expected: []string{"0", "1", "2", "7", "8", "9"},
		},
		{
			input:    "0-2,5",
			expected: []string{"0", "1", "2", "5"},
		},
		{
			input:    "0,5",
			expected: []string{"0", "5"},
		},
	}

	for _, tc := range testCases {
		got, err := parseRange(tc.input)
		require.NoError(t, err)

		assert.Equal(t, tc.expected, got)
	}
}

func TestSantizeMetricName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "metric-name",
			expected: "metric_name",
		},
		{
			input:    "metric-name$",
			expected: "metric_name_",
		},
		{
			input:    "ns/metric-name",
			expected: "ns_metric_name",
		},
	}

	for _, tc := range testCases {
		got := SanitizeMetricName(tc.input)
		assert.Equal(t, tc.expected, got)
	}
}

func TestInode(t *testing.T) {
	absPath, err := filepath.Abs("testdata")
	require.NoError(t, err)

	inodeValue, err := inode(absPath)
	require.NoError(t, err)

	assert.Positive(t, inodeValue)
}

func TestLookupCgroupsRootError(t *testing.T) {
	// Look for non existent name
	_, err := lookupCgroupRoots("testdata/sys/fs/cgroup/system.slice", "doesnotexit.scope")
	require.Error(t, err)
}

func TestReadProcEnvirons(t *testing.T) {
	// Instantiate a new Proc FS
	procFS, err := procfs.NewFS("testdata/proc")
	require.NoError(t, err)

	// Get all processes
	procs, err := procFS.AllProcs()
	require.NoError(t, err)

	testCases := []struct {
		name     string
		dataPtr  *readProcSecurityCtxData
		expected map[string]string
	}{
		{
			name: "Existing env vars with proc filter",
			dataPtr: &readProcSecurityCtxData{
				procs: procs,
				ignoreProc: func(envs []string) bool {
					return !slices.Contains(envs, "LSB_JOBID=1009248")
				},
				targetEnvVars: []string{"CUDA_VISIBLE_DEVICES", "CUDA_VISIBLE_DEVICES1"},
			},
			expected: map[string]string{
				"CUDA_VISIBLE_DEVICES":  "MIG-GPU-956348bc-d43d-23ed-53d4-857749fa2b67/1/0,MIG-GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7/1/0",
				"CUDA_VISIBLE_DEVICES1": "MIG-GPU-956348bc-d43d-23ed-53d4-857749fa2b67/1/0,MIG-GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7/1/0",
			},
		},
		{
			name: "Existing env vars without proc filter",
			dataPtr: &readProcSecurityCtxData{
				procs:         procs,
				targetEnvVars: []string{"CUDA_VISIBLE_DEVICES", "CUDA_VISIBLE_DEVICES1"},
			},
			expected: map[string]string{
				"CUDA_VISIBLE_DEVICES":  "MIG-GPU-956348bc-d43d-23ed-53d4-857749fa2b67/1/0,MIG-GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7/1/0",
				"CUDA_VISIBLE_DEVICES1": "MIG-GPU-956348bc-d43d-23ed-53d4-857749fa2b67/1/0,MIG-GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7/1/0",
			},
		},
		{
			name: "Non existing env vars",
			dataPtr: &readProcSecurityCtxData{
				procs:         procs,
				targetEnvVars: []string{"NON_EXISTING_ENV_VAR", "NON_EXISTING_ENV_VAR1"},
			},
			expected: map[string]string{},
		},
	}

	for _, tc := range testCases {
		err = readProcEnvirons(tc.dataPtr)
		require.NoError(t, err)

		assert.Equal(t, tc.expected, tc.dataPtr.targetEnvVarValues, tc.name)
	}

	// Error when dataPtr cannot be asserted
	data := []string{"Test"}

	err = readProcEnvirons(data)
	require.Error(t, err)
}
