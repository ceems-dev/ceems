//go:build !nolsf

package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/ceems-dev/ceems/internal/security"
	"github.com/containerd/cgroups/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLSFCollector(t *testing.T) {
	_, err := CEEMSExporterApp.Parse(
		[]string{
			"--path.cgroupfs", "testdata/sys/fs/cgroup",
			"--path.procfs", "testdata/proc",
			"--path.sysfs", "testdata/sys",
			"--collector.force-hostname", "testhost-1",
			"--collector.lsf.bjobs-path", "testdata/bjobs",
			"--collector.lsf.swap-memory-metrics",
			"--collector.lsf.psi-metrics",
			"--collector.perf.hardware-events",
			"--collector.rdma.stats",
			"--collector.gpu.type", "nvidia",
			"--collector.gpu.nvidia-smi-path", "testdata/nvidia-smi",
			"--collector.cgroups.force-version", "v2",
		},
	)
	require.NoError(t, err)

	collector, err := NewLSFCollector(noOpLogger)
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

func TestLSFJobDevices(t *testing.T) {
	_, err := CEEMSExporterApp.Parse(
		[]string{
			"--path.cgroupfs", "testdata/sys/fs/cgroup",
			"--path.procfs", "testdata/proc",
			"--collector.cgroups.force-version", "v1",
			"--collector.gpu.type", "nvidia",
			"--collector.gpu.nvidia-smi-path", "testdata/nvidia-smi",
			"--collector.lsf.bjobs-path", "testdata/bjobs",
		},
	)
	require.NoError(t, err)

	// cgroup manager
	cgManager, err := NewCgroupManager(lsf, noOpLogger)
	require.NoError(t, err)

	// GPU SMI
	gpuSMI, err := NewGPUSMI(nil, noOpLogger)
	require.NoError(t, err)

	err = gpuSMI.Discover()
	require.NoError(t, err)

	c := lsfCollector{
		cgroupManager:    cgManager,
		gpuSMI:           gpuSMI,
		logger:           noOpLogger,
		hostname:         "testhost-1",
		bjobsExePath:     "testdata/bjobs",
		securityContexts: make(map[string]*security.SecurityContext),
	}

	// Add dummy security context
	cfg := &security.SCConfig{
		Name:   lsfReadProcCtx,
		Caps:   nil,
		Func:   readProcEnvirons,
		Logger: c.logger,
	}
	c.securityContexts[lsfReadProcCtx], err = security.NewSecurityContext(cfg)
	require.NoError(t, err)

	expectedJobIDs := []string{
		"1009248", "1009249", "1009250",
	}

	expectedJobDeviceMappers := map[string][]ComputeUnit{
		"0": {{UUID: "1009249", NumShares: 1}},
		"1": {{UUID: "1009250", NumShares: 1}, {UUID: "1009249", NumShares: 1}},
		"2": {{UUID: "1009248", NumShares: 1}},
		"5": {{UUID: "1009248", NumShares: 1}},
	}

	expectedCgroupNumCPUs := map[string]int{
		"1009248": 6,
		"1009249": 4,
		"1009250": 4,
	}

	cgroups, err := c.jobCgroups()
	require.NoError(t, err)

	for _, cgroup := range cgroups {
		assert.Equal(t, expectedCgroupNumCPUs[cgroup.uuid], cgroup.ncpus, cgroup.uuid)
	}

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

func writeLSFEnvironFile(procFS string, jobid string, val string) error {
	envs := []string{"LSB_BATCH_JID=" + jobid, "CUDA_VISIBLE_DEVICES=" + val}

	return os.WriteFile(
		procFS+"/"+strings.ReplaceAll(strings.ReplaceAll(jobid, "[", ""), "]", "")+"/environ",
		[]byte(strings.Join(envs, "\000")+"\000"),
		0o600,
	)
}

func writeGPUSMI(path string, content string) {
	bjobsContent := fmt.Sprintf(`#!/bin/bash
echo """%s"""`, content,
	)

	// Write content to file
	os.WriteFile(path, []byte(bjobsContent), 0o700)
}

func writeBjobsExe(path string, bjobs LSFJobsList) error {
	// Marhsall bjobs
	bjobsOut, err := json.Marshal(bjobs)
	if err != nil {
		return err
	}

	bjobsContent := fmt.Sprintf(`#!/bin/bash
echo -e '%s'`, bjobsOut,
	)

	// Write content to file
	os.WriteFile(path+"/bjobs", []byte(bjobsContent), 0o700)

	return nil
}

func TestLSFJobDevicesCaching(t *testing.T) {
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
		idRegex:     lsfCgroupV1PathRegex,
		mountPoints: []string{cgroupsPath + "/cpuacct/lsf"},
		isChild: func(p string) bool {
			return false
		},
	}

	// Setup mock GPU vendors
	nvidiaSMIPath := filepath.Join(path, "nvidia-smi")
	mockVendors := []vendor{devVendors[nvidia]}
	mockVendors[0].smiCmd = nvidiaSMIPath
	mockVendors[0].smiQueryCmd = nvidiaSMIQueryCmd

	// Setup SMI command
	nvidiaSmiLog := `<?xml version="1.0" ?>
<!DOCTYPE nvidia_smi_log SYSTEM "nvsmi_device_v12.dtd">
<nvidia_smi_log>
	<timestamp>Fri Oct 11 18:24:09 2024</timestamp>
	<driver_version>535.129.03</driver_version>
	<cuda_version>12.2</cuda_version>
	<attached_gpus>4</attached_gpus>
	<gpu id=\"00000000:15:00.0\">
		<mig_mode>
				<current_mig>N/A</current_mig>
				<pending_mig>N/A</pending_mig>
		</mig_mode>
		<mig_devices>
				None
		</mig_devices>
		<uuid>GPU-f124aa59-d406-d45b-9481-8fcd694e6c9e</uuid>
	</gpu>
	<gpu id=\"00000000:10:00.0\">
		<mig_mode>
				<current_mig>Enabled</current_mig>
				<pending_mig>Enabled</pending_mig>
		</mig_mode>
		<mig_devices>
				<mig_device>
					<index>0</index>
					<gpu_instance_id>1</gpu_instance_id>
					<compute_instance_id>0</compute_instance_id>
				</mig_device>
				<mig_device>
					<index>1</index>
					<gpu_instance_id>5</gpu_instance_id>
					<compute_instance_id>0</compute_instance_id>
				</mig_device>
				<mig_device>
					<index>2</index>
					<gpu_instance_id>13</gpu_instance_id>
					<compute_instance_id>0</compute_instance_id>
				</mig_device>
		</mig_devices>
		<uuid>GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7</uuid>
	</gpu>
	<gpu id=\"00000000:18:00.0\">
		<mig_mode>
				<current_mig>N/A</current_mig>
				<pending_mig>N/A</pending_mig>
		</mig_mode>
		<mig_devices>
				None
		</mig_devices>
		<uuid>GPU-61a65011-6571-a6d2-5ab8-66cbb6f7f9c3</uuid>
	</gpu>
	<gpu id=\"00000000:20:00.0\">
		<mig_mode>
				<current_mig>Enabled</current_mig>
				<pending_mig>Enabled</pending_mig>
		</mig_mode>
		<mig_devices>
				<mig_device>
					<index>0</index>
					<gpu_instance_id>1</gpu_instance_id>
					<compute_instance_id>0</compute_instance_id>
				</mig_device>
				<mig_device>
					<index>1</index>
					<gpu_instance_id>5</gpu_instance_id>
					<compute_instance_id>0</compute_instance_id>
				</mig_device>
				<mig_device>
					<index>1</index>
					<gpu_instance_id>6</gpu_instance_id>
					<compute_instance_id>0</compute_instance_id>
				</mig_device>
		</mig_devices>
		<uuid>GPU-61a65011-6571-a6d2-5th8-66cbb6f7f9c3</uuid>
	</gpu>
</nvidia_smi_log>`
	writeGPUSMI(nvidiaSMIPath, nvidiaSmiLog)

	c := lsfCollector{
		cgroupManager:    cgManager,
		logger:           noOpLogger,
		gpuSMI:           &GPUSMI{logger: noOpLogger, vendors: mockVendors},
		hostname:         "testhost-1",
		bjobsExePath:     path + "/bjobs",
		securityContexts: make(map[string]*security.SecurityContext),
	}

	// Add dummy security context
	cfg := &security.SCConfig{
		Name:   lsfReadProcCtx,
		Caps:   nil,
		Func:   readProcEnvirons,
		Logger: c.logger,
	}
	c.securityContexts[lsfReadProcCtx], err = security.NewSecurityContext(cfg)
	require.NoError(t, err)

	// Add cgroups
	for i := range 20 {
		dir := fmt.Sprintf("%s/cpuacct/lsf/cluster1/job.%d.12345.123443", cgroupsPath, i)

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

	// Fake jobs
	bjobs := LSFJobsList{
		NumJobs: 8,
		Records: []LSFJobRecord{
			{
				ID:        "0",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com:testhost-1.example.com:testhost-1.example.com",
				GPUSlot:   "testhost-1.example.com:0,2,0,2",
			},
			{
				ID:        "1",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com:testhost-1.example.com:testhost-1.example.com",
				GPUSlot:   `testhost-1.example.com:1:1\/0,3:1\/0`,
			},
			{
				ID:        "2",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com:testhost-2.example.com:testhost-2.example.com",
				GPUSlot:   "testhost-1.example.com:2,2;testhost-2.example.com:0,0",
			},
			{
				ID:        "3",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com",
				GPUSlot:   `testhost-1.example.com:1:1\/0,3:1\/0`,
			},
			{
				ID:        "4",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com",
				GPUSlot:   `testhost-1.example.com:3:1\/0`,
			},
			{
				ID:        "5",
				AllocSlot: "testhost-1.example.com",
				GPUSlot:   "testhost-1.example.com:0",
			},
			{
				ID:        "6",
				AllocSlot: "testhost-1.example.com",
				GPUSlot:   `testhost-1.example.com:3:6\/0`,
			},
			{
				ID:        "7",
				AllocSlot: "testhost-1.example.com",
				GPUSlot:   `testhost-1.example.com:1:5\/0,3:5\/0`,
			},
		},
	}

	// Make CUDA_VISIBLE_DEVICES env string
	for _, job := range bjobs.Records {
		if _, hostGPUSlot, found := strings.Cut(job.GPUSlot, ":"); found && strings.Contains(hostGPUSlot, "/") {
			hostGPUSlot = strings.ReplaceAll(hostGPUSlot, "1:", "MIG-GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7/")
			hostGPUSlot = strings.ReplaceAll(hostGPUSlot, "3:", "MIG-GPU-61a65011-6571-a6d2-5th8-66cbb6f7f9c3/")
			hostGPUSlot = strings.ReplaceAll(hostGPUSlot, `\/`, "/")
			err = writeLSFEnvironFile(procFS, job.ID, hostGPUSlot)
			require.NoError(t, err)
		} else {
			gpuOrdinals := strings.Split(hostGPUSlot, ",")
			// Remove duplicates
			slices.Sort(gpuOrdinals)
			gpuOrdinals = slices.Compact(gpuOrdinals)

			var localOrdinals []string
			for i := range gpuOrdinals {
				localOrdinals = append(localOrdinals, strconv.Itoa(i))
			}

			// Write CUDA_VISIBLE_DEVICES env var
			err = writeLSFEnvironFile(procFS, job.ID, strings.Join(localOrdinals, ","))
			require.NoError(t, err)
		}
	}

	// Write bjobs to file
	err = writeBjobsExe(path, bjobs)
	require.NoError(t, err)

	// Now call get metrics which should populate jobPropsCache
	_, err = c.jobCgroups()
	require.NoError(t, err)

	// Check if jobPropsCache has 20 jobs and GPU ordinals are correct
	assert.Len(t, c.previousJobIDs, 20)

	expected := map[string][]ComputeUnit{
		"0": {{UUID: "0", NumShares: 1}, {UUID: "5", NumShares: 1}},
		"1": {{UUID: "1", NumShares: 1}, {UUID: "3", NumShares: 1}},
		"2": {{UUID: "7", NumShares: 1}},
		"3": {},
		"4": {{UUID: "0", NumShares: 1}, {UUID: "2", NumShares: 1}},
		"5": {{UUID: "1", NumShares: 1}, {UUID: "3", NumShares: 1}, {UUID: "4", NumShares: 1}},
		"6": {{UUID: "7", NumShares: 1}},
		"7": {{UUID: "6", NumShares: 1}},
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
		dir := fmt.Sprintf("%s/cpuacct/lsf/cluster1/job.%d.12345.123443", cgroupsPath, i)

		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}

	for i := 1; i <= 20; i++ {
		dir := fmt.Sprintf("%s/cpuacct/lsf/cluster1/job.19[%d].12345.123443", cgroupsPath, i)

		err = os.MkdirAll(dir, 0o750)
		require.NoError(t, err)

		err = os.WriteFile(
			dir+"/cgroup.procs",
			fmt.Appendf(nil, "19%d\n", i),
			0o600,
		)
		require.NoError(t, err)

		procDir := fmt.Sprintf("%s/19%d", procFS, i)

		err = os.MkdirAll(procDir, 0o750)
		require.NoError(t, err)
	}

	// Setup SMI command and change MIG instance for GPU 3 and
	// remove 2 MIG instances on GPU 1.
	// Mocking dynamic MIG here
	nvidiaSmiLog = `<?xml version="1.0" ?>
<!DOCTYPE nvidia_smi_log SYSTEM "nvsmi_device_v12.dtd">
<nvidia_smi_log>
	<timestamp>Fri Oct 11 18:24:09 2024</timestamp>
	<driver_version>535.129.03</driver_version>
	<cuda_version>12.2</cuda_version>
	<attached_gpus>4</attached_gpus>
	<gpu id=\"00000000:15:00.0\">
		<mig_mode>
				<current_mig>N/A</current_mig>
				<pending_mig>N/A</pending_mig>
		</mig_mode>
		<mig_devices>
				None
		</mig_devices>
		<uuid>GPU-f124aa59-d406-d45b-9481-8fcd694e6c9e</uuid>
	</gpu>
	<gpu id=\"00000000:10:00.0\">
		<mig_mode>
				<current_mig>Enabled</current_mig>
				<pending_mig>Enabled</pending_mig>
		</mig_mode>
		<mig_devices>
				<mig_device>
					<index>0</index>
					<gpu_instance_id>1</gpu_instance_id>
					<compute_instance_id>0</compute_instance_id>
				</mig_device>
		</mig_devices>
		<uuid>GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7</uuid>
	</gpu>
	<gpu id=\"00000000:18:00.0\">
		<mig_mode>
				<current_mig>N/A</current_mig>
				<pending_mig>N/A</pending_mig>
		</mig_mode>
		<mig_devices>
				None
		</mig_devices>
		<uuid>GPU-61a65011-6571-a6d2-5ab8-66cbb6f7f9c3</uuid>
	</gpu>
	<gpu id=\"00000000:20:00.0\">
		<mig_mode>
				<current_mig>Enabled</current_mig>
				<pending_mig>Enabled</pending_mig>
		</mig_mode>
		<mig_devices>
				<mig_device>
					<index>0</index>
					<gpu_instance_id>1</gpu_instance_id>
					<compute_instance_id>0</compute_instance_id>
				</mig_device>
				<mig_device>
					<index>1</index>
					<gpu_instance_id>5</gpu_instance_id>
					<compute_instance_id>0</compute_instance_id>
				</mig_device>
				<mig_device>
					<index>1</index>
					<gpu_instance_id>13</gpu_instance_id>
					<compute_instance_id>1</compute_instance_id>
				</mig_device>
		</mig_devices>
		<uuid>GPU-61a65011-6571-a6d2-5th8-66cbb6f7f9c3</uuid>
	</gpu>
</nvidia_smi_log>`
	writeGPUSMI(nvidiaSMIPath, nvidiaSmiLog)

	// Binds GPUs to first jobs 19 to 24
	// Fake jobs
	bjobs = LSFJobsList{
		NumJobs: 4,
		Records: []LSFJobRecord{
			{
				ID:        "19[1]",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com:testhost-1.example.com:testhost-1.example.com",
				GPUSlot:   "testhost-1.example.com:0,0,0,0",
			},
			{
				ID:        "19[2]",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com:testhost-1.example.com:testhost-1.example.com",
				GPUSlot:   "testhost-1.example.com:2,2,2,2",
			},
			{
				ID:        "19[3]",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com:testhost-2.example.com:testhost-2.example.com",
				GPUSlot:   `testhost-1.example.com:1:1\/0;testhost-2.example.com:0:1\/0`,
			},
			{
				ID:        "19[4]",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com",
				GPUSlot:   "testhost-1.example.com:0,2",
			},
			{
				ID:        "19[5]",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com",
				GPUSlot:   `testhost-1.example.com:3:13\/1`,
			},
			{
				ID:        "19[6]",
				AllocSlot: "testhost-1.example.com:testhost-1.example.com",
				GPUSlot:   `testhost-1.example.com:3:13\/1`,
			},
		},
	}

	// Make CUDA_VISIBLE_DEVICES env string
	for _, job := range bjobs.Records {
		if _, hostGPUSlot, found := strings.Cut(job.GPUSlot, ":"); found && strings.Contains(hostGPUSlot, "/") {
			hostGPUSlot = strings.ReplaceAll(hostGPUSlot, "1:", "MIG-GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7/")
			hostGPUSlot = strings.ReplaceAll(hostGPUSlot, "3:", "MIG-GPU-61a65011-6571-a6d2-5th8-66cbb6f7f9c3/")
			hostGPUSlot = strings.ReplaceAll(hostGPUSlot, `\/`, "/")
			hostGPUSlot = strings.Split(hostGPUSlot, ";")[0]
			err = writeLSFEnvironFile(procFS, job.ID, hostGPUSlot)
			require.NoError(t, err)
		} else {
			gpuOrdinals := strings.Split(hostGPUSlot, ",")
			// Remove duplicates
			slices.Sort(gpuOrdinals)
			gpuOrdinals = slices.Compact(gpuOrdinals)

			var localOrdinals []string
			for i := range gpuOrdinals {
				localOrdinals = append(localOrdinals, strconv.Itoa(i))
			}

			// Write CUDA_VISIBLE_DEVICES env var
			err = writeLSFEnvironFile(procFS, job.ID, strings.Join(localOrdinals, ","))
			require.NoError(t, err)
		}
	}

	// Update bjobs to file
	err = writeBjobsExe(path, bjobs)
	require.NoError(t, err)

	// Now call again get metrics which should populate jobPropsCache
	_, err = c.jobCgroups()
	require.NoError(t, err)

	// Check if jobPropsCache has only 30 jobs and GPU ordinals are empty
	assert.Len(t, c.previousJobIDs, 30)

	// New expected jobs
	expected = map[string][]ComputeUnit{
		"0": {{UUID: "19[1]", NumShares: 1}, {UUID: "19[4]", NumShares: 1}},
		"1": {{UUID: "19[3]", NumShares: 1}},
		"2": {{UUID: "19[2]", NumShares: 1}, {UUID: "19[4]", NumShares: 1}},
		"5": {{UUID: "19[5]", NumShares: 1}, {UUID: "19[6]", NumShares: 1}},
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
