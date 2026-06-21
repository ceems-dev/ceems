//go:build !noopenpbs

package collector

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ceems-dev/ceems/internal/security"
	"github.com/containerd/cgroups/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenPBSCollector(t *testing.T) {
	_, err := CEEMSExporterApp.Parse(
		[]string{
			"--path.cgroupfs", "testdata/sys/fs/cgroup",
			"--path.procfs", "testdata/proc",
			"--path.sysfs", "testdata/sys",
			"--collector.force-hostname", "testhost-1",
			"--collector.openpbs.swap-memory-metrics",
			"--collector.openpbs.psi-metrics",
			"--collector.perf.hardware-events",
			"--collector.rdma.stats",
			"--collector.gpu.type", "nvidia",
			"--collector.gpu.nvidia-smi-path", "testdata/nvidia-smi",
			"--collector.cgroups.force-version", "v1",
		},
	)
	require.NoError(t, err)

	collector, err := NewOpenPBSCollector(noOpLogger)
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

func TestOpenPBSJobDevices(t *testing.T) {
	_, err := CEEMSExporterApp.Parse(
		[]string{
			"--path.cgroupfs", "testdata/sys/fs/cgroup",
			"--path.procfs", "testdata/proc",
			"--collector.cgroups.force-version", "v1",
			"--collector.gpu.type", "nvidia",
			"--collector.gpu.nvidia-smi-path", "testdata/nvidia-smi",
		},
	)
	require.NoError(t, err)

	// cgroup manager
	cgManager, err := NewCgroupManager(openpbs, noOpLogger)
	require.NoError(t, err)

	// GPU SMI
	gpuSMI, err := NewGPUSMI(nil, noOpLogger)
	require.NoError(t, err)

	err = gpuSMI.Discover()
	require.NoError(t, err)

	c := openpbsCollector{
		cgroupManager:    cgManager,
		gpuSMI:           gpuSMI,
		logger:           noOpLogger,
		hostname:         "testhost-1",
		securityContexts: make(map[string]*security.SecurityContext),
	}

	// Add dummy security context
	cfg := &security.SCConfig{
		Name:   openpbsReadProcCtx,
		Caps:   nil,
		Func:   readProcEnvirons,
		Logger: c.logger,
	}
	c.securityContexts[openpbsReadProcCtx], err = security.NewSecurityContext(cfg)
	require.NoError(t, err)

	expectedJobIDs := []string{
		"1009248.testhost-1", "1009249[0].testhost-1", "1009249[1].testhost-1",
	}

	expectedJobDeviceMappers := map[string][]ComputeUnit{
		"10": {{UUID: "1009249[0].testhost-1", NumShares: 1}, {UUID: "1009249[1].testhost-1", NumShares: 1}},
		"1":  {{UUID: "1009249[0].testhost-1", NumShares: 1}, {UUID: "1009249[1].testhost-1", NumShares: 1}},
		"2":  {{UUID: "1009248.testhost-1", NumShares: 1}},
		"6":  {{UUID: "1009248.testhost-1", NumShares: 1}},
	}

	_, err = c.jobCgroups()
	require.NoError(t, err)

	assert.Equal(t, expectedJobIDs, c.previousJobIDs)

	for gpuIndex := range expectedJobDeviceMappers {
		for _, gpu := range c.gpuSMI.Devices {
			if gpu.Index == gpuIndex {
				assert.ElementsMatch(t, expectedJobDeviceMappers[gpu.Index], gpu.ComputeUnits, "GPU %s", gpu.Index)
			}

			for _, inst := range gpu.Instances {
				if inst.Index == gpuIndex {
					assert.ElementsMatch(t, expectedJobDeviceMappers[inst.Index], inst.ComputeUnits, "MIG %s", inst.Index)
				}
			}
		}
	}
}

func writeOpenPBSEnvironFile(procFS string, jobid string, val string, arrayJob bool) error {
	var (
		envs []string
		path string
	)

	if arrayJob {
		envs = []string{fmt.Sprintf("PBS_JOBID=100[%s].testhost-1", jobid), "CUDA_VISIBLE_DEVICES=" + val}
		path = procFS + "/100" + jobid + "/environ"
	} else {
		envs = []string{fmt.Sprintf("PBS_JOBID=%s.testhost-1", jobid), "CUDA_VISIBLE_DEVICES=" + val}
		path = procFS + "/" + jobid + "/environ"
	}

	return os.WriteFile(
		path,
		[]byte(strings.Join(envs, "\000")+"\000"),
		0o600,
	)
}

func TestOpenPBSJobDevicesCaching(t *testing.T) {
	path := t.TempDir()

	cgroupsPath := path + "/cgroups"
	err := os.Mkdir(cgroupsPath, 0o750)
	require.NoError(t, err)

	procFS := path + "/proc"
	err = os.Mkdir(procFS, 0o750)
	require.NoError(t, err)

	fs, err := procfs.NewFS(procFS)
	require.NoError(t, err)

	// cgroup Manager
	cgManager := &cgroupManager{
		logger:      noOpLogger,
		fs:          fs,
		mode:        cgroups.Legacy,
		root:        cgroupsPath,
		idRegex:     openpbsCgroupV1PathRegex,
		mountPoints: []string{cgroupsPath + "/cpuacct/pbs_jobs.service"},
		isChild: func(p string) bool {
			return false
		},
	}

	mockGPUDevs := mockGPUDevices(4, []int{1, 3})

	c := openpbsCollector{
		cgroupManager:    cgManager,
		logger:           noOpLogger,
		gpuSMI:           &GPUSMI{logger: noOpLogger, Devices: mockGPUDevs},
		hostname:         "testhost-1",
		securityContexts: make(map[string]*security.SecurityContext),
	}

	// Add dummy security context
	cfg := &security.SCConfig{
		Name:   openpbsReadProcCtx,
		Caps:   nil,
		Func:   readProcEnvirons,
		Logger: c.logger,
	}
	c.securityContexts[openpbsReadProcCtx], err = security.NewSecurityContext(cfg)
	require.NoError(t, err)

	// Add cgroups
	for i := range 20 {
		dir := fmt.Sprintf("%s/cpuacct/pbs_jobs.service/jobid/%d.testhost-1", cgroupsPath, i)

		err = os.MkdirAll(dir, 0o750)
		require.NoError(t, err)

		err = os.WriteFile(
			dir+"/cgroup.procs",
			fmt.Appendf(nil, "%d\n", i),
			0o600,
		)
		require.NoError(t, err)

		procDir := fmt.Sprintf("%s/%d", procFS, i)

		err = os.MkdirAll(procDir, 0o750)
		require.NoError(t, err)
	}

	// Mock job to device ID mappings
	mockDeviceJobMapping := map[string][]string{
		"0": {"GPU-0", "GPU-2"},
		"1": {"GPU-2"},
		"2": {"MIG-1-1", "MIG-1-2"},
		"3": {"MIG-1-2", "MIG-1-3"},
		"4": {"MIG-1-1"},
		"5": {"GPU-2"},
		"6": {"MIG-3-1", "MIG-3-2", "MIG-3-3"},
		"7": {"MIG-3-1"},
		"8": {"MIG-3-2"},
		"9": {"MIG-3-3"},
	}

	// Make CUDA_VISIBLE_DEVICES env string
	for job, uuids := range mockDeviceJobMapping {
		// Write CUDA_VISIBLE_DEVICES env var
		err = writeOpenPBSEnvironFile(procFS, job, strings.Join(uuids, ","), false)
		require.NoError(t, err)
	}

	// Now call get metrics which should populate jobPropsCache
	_, err = c.jobCgroups()
	require.NoError(t, err)

	// Check if jobPropsCache has 20 jobs and GPU ordinals are correct
	assert.Len(t, c.previousJobIDs, 20)

	expected := map[string][]ComputeUnit{
		"0": {{UUID: "0.testhost-1", NumShares: 1}},
		"1": {{UUID: "2.testhost-1", NumShares: 1}, {UUID: "4.testhost-1", NumShares: 1}},
		"2": {{UUID: "2.testhost-1", NumShares: 1}, {UUID: "3.testhost-1", NumShares: 1}},
		"3": {{UUID: "3.testhost-1", NumShares: 1}},
		"4": {{UUID: "0.testhost-1", NumShares: 1}, {UUID: "5.testhost-1", NumShares: 1}},
		"5": {{UUID: "6.testhost-1", NumShares: 1}, {UUID: "7.testhost-1", NumShares: 1}},
		"6": {{UUID: "6.testhost-1", NumShares: 1}, {UUID: "8.testhost-1", NumShares: 1}},
		"7": {{UUID: "6.testhost-1", NumShares: 1}, {UUID: "9.testhost-1", NumShares: 1}},
	}

	for _, gpu := range c.gpuSMI.Devices {
		if gpu.Index != "" {
			assert.ElementsMatch(t, expected[gpu.Index], gpu.ComputeUnits, "GPU %s", gpu.Index)
		} else {
			for _, inst := range gpu.Instances {
				assert.ElementsMatch(t, expected[inst.Index], inst.ComputeUnits, "MIG %s", inst.Index)
			}
		}
	}

	// Remove first 10 jobs and add new 20 more jobs
	for i := range 10 {
		dir := fmt.Sprintf("%s/cpuacct/pbs_jobs.service/jobid/%d.testhost-1", cgroupsPath, i)

		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}

	for i := range 20 {
		dir := fmt.Sprintf("%s/cpuacct/pbs_jobs.service/jobid/100[%d].testhost-1", cgroupsPath, i)

		err = os.MkdirAll(dir, 0o750)
		require.NoError(t, err)

		err = os.WriteFile(
			dir+"/cgroup.procs",
			fmt.Appendf(nil, "100%d\n", i),
			0o600,
		)
		require.NoError(t, err)

		procDir := fmt.Sprintf("%s/100%d", procFS, i)

		err = os.MkdirAll(procDir, 0o750)
		require.NoError(t, err)
	}

	// Updated mock job to device ID mappings
	mockDeviceJobMapping = map[string][]string{
		"0": {"GPU-0", "GPU-2"},
		"1": {"GPU-2"},
		"2": {"MIG-1-1", "MIG-1-2"},
		"3": {"MIG-1-2", "MIG-1-3"},
		"4": {"MIG-1-1"},
		"5": {"GPU-2"},
		"6": {"MIG-3-1", "MIG-3-2", "MIG-3-3"},
		"7": {"MIG-3-1"},
		"8": {"MIG-3-2"},
		"9": {"MIG-3-3"},
	}

	// Make CUDA_VISIBLE_DEVICES env string
	for job, uuids := range mockDeviceJobMapping {
		// Write CUDA_VISIBLE_DEVICES env var
		err = writeOpenPBSEnvironFile(procFS, job, strings.Join(uuids, ","), true)
		require.NoError(t, err)
	}

	// Now call again get metrics which should populate jobPropsCache
	_, err = c.jobCgroups()
	require.NoError(t, err)

	// Check if jobPropsCache has only 30 jobs and GPU ordinals are empty
	assert.Len(t, c.previousJobIDs, 30)

	// New expected jobs
	expected = map[string][]ComputeUnit{
		"0": {{UUID: "100[0].testhost-1", NumShares: 1}},
		"1": {{UUID: "100[2].testhost-1", NumShares: 1}, {UUID: "100[4].testhost-1", NumShares: 1}},
		"2": {{UUID: "100[2].testhost-1", NumShares: 1}, {UUID: "100[3].testhost-1", NumShares: 1}},
		"3": {{UUID: "100[3].testhost-1", NumShares: 1}},
		"4": {{UUID: "100[0].testhost-1", NumShares: 1}, {UUID: "100[5].testhost-1", NumShares: 1}},
		"5": {{UUID: "100[6].testhost-1", NumShares: 1}, {UUID: "100[7].testhost-1", NumShares: 1}},
		"6": {{UUID: "100[6].testhost-1", NumShares: 1}, {UUID: "100[8].testhost-1", NumShares: 1}},
		"7": {{UUID: "100[6].testhost-1", NumShares: 1}, {UUID: "100[9].testhost-1", NumShares: 1}},
	}

	for _, gpu := range c.gpuSMI.Devices {
		if gpu.Index != "" {
			assert.ElementsMatch(t, expected[gpu.Index], gpu.ComputeUnits, gpu.Index)
		} else {
			for _, inst := range gpu.Instances {
				assert.ElementsMatch(t, expected[inst.Index], inst.ComputeUnits, inst.Index)
			}
		}
	}
}
