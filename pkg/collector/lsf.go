//go:build !nolsf

package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ceems-dev/ceems/internal/common"
	"github.com/ceems-dev/ceems/internal/osexec"
	"github.com/ceems-dev/ceems/internal/security"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

const (
	lsfCollectorSubsystem = "lsf"
)

// CLI opts.
var (
	// cgroup opts.
	lsfCollectSwapMemoryStats = CEEMSExporterApp.Flag(
		"collector.lsf.swap-memory-metrics",
		"Enables collection of swap memory metrics (default: disabled)",
	).Default("false").Bool()
	lsfCollectPSIStats = CEEMSExporterApp.Flag(
		"collector.lsf.psi-metrics",
		"Enables collection of PSI metrics (default: disabled)",
	).Default("false").Bool()
)

// Used for e2e tests.
var (
	bjobsPath = CEEMSExporterApp.Flag(
		"collector.lsf.bjobs-path",
		"Absolute path to bjobs binary. Use only for testing.",
	).Hidden().Default("").String()
)

// Security context names.
const (
	lsfReadProcCtx = "lsf_read_procs"
)

// Regex to get physical GPU UUID and GPU_I_ID, GPU_C_ID from MIG device IDs.
var migDeviceIDRegex = regexp.MustCompile("^MIG-(?P<GPU_UUID>.*?)/(?P<GPU_I_ID>[0-9]+?)/(?P<GPU_C_ID>[0-9]+)$")

// Cache interval.
var (
	lsfCacheTTL = 15 * time.Minute
)

type lsfCollector struct {
	logger                     *slog.Logger
	cgroupManager              *cgroupManager
	cgroupCollector            *cgroupCollector
	perfCollector              *perfCollector
	ebpfCollector              *ebpfCollector
	rdmaCollector              *rdmaCollector
	hostname                   string
	gpuSMI                     *GPUSMI
	bjobsExePath               string
	previousJobIDs             []string
	jobResourcesLastUpdateTime time.Time
	jobResourcesCacheTTL       time.Duration
	procFS                     procfs.FS
	securityContexts           map[string]*security.SecurityContext
}

func init() {
	RegisterCollector(lsfCollectorSubsystem, defaultDisabled, NewLSFCollector)
}

// NewLSFCollector returns a new Collector exposing a summary of cgroups.
func NewLSFCollector(logger *slog.Logger) (Collector, error) {
	// Check if bjobs binary is available on PATH
	bjobsExePath, err := lookupSmiCmd(*bjobsPath, "bjobs")
	if err != nil {
		logger.Error("Failed to lookup bjobs command. Make sure bjobs binary is on PATH", "err", err)

		return nil, err
	}

	// Get LSF's cgroup details
	cgroupManager, err := NewCgroupManager(lsf, logger)
	if err != nil {
		logger.Error("Failed to create cgroup manager", "err", err)

		return nil, err
	}

	logger.Info("cgroup: " + cgroupManager.String())

	// Set cgroup options
	opts := cgroupOpts{
		collectSwapMemStats: *lsfCollectSwapMemoryStats,
		collectPSIStats:     *lsfCollectPSIStats,
		collectBlockIOStats: false, // LSF does not support blkio controller.
	}

	// Start new instance of cgroupCollector
	cgCollector, err := NewCgroupCollector(logger.With("sub_collector", "cgroup"), cgroupManager, opts)
	if err != nil {
		logger.Info("Failed to create cgroup collector", "err", err)

		return nil, err
	}

	// Start new instance of perfCollector
	var perfCollector *perfCollector

	if perfCollectorEnabled() {
		perfCollector, err = NewPerfCollector(logger.With("sub_collector", "perf"), cgroupManager)
		if err != nil {
			logger.Info("Failed to create perf collector", "err", err)

			return nil, err
		}
	}

	// Start new instance of ebpfCollector
	var ebpfCollector *ebpfCollector

	if ebpfCollectorEnabled() {
		ebpfCollector, err = NewEbpfCollector(logger.With("sub_collector", "ebpf"), cgroupManager)
		if err != nil {
			logger.Info("Failed to create ebpf collector", "err", err)

			return nil, err
		}
	}

	// Start new instance of rdmaCollector
	var rdmaCollector *rdmaCollector

	if rdmaCollectorEnabled() {
		rdmaCollector, err = NewRDMACollector(logger.With("sub_collector", "rdma"), cgroupManager)
		if err != nil {
			logger.Info("Failed to create RDMA collector", "err", err)

			return nil, err
		}
	}

	// Instantiate a new instance of GPUSMI struct
	gpuSMI, err := NewGPUSMI(nil, logger)
	if err != nil {
		logger.Error("Error creating GPU SMI instance", "err", err)

		return nil, err
	}

	// Attempt to get GPU devices
	err = gpuSMI.Discover()
	if err != nil {
		// If we failed to fetch GPUs that are from supported
		// vendor, return with error
		logger.Error("Error fetching GPU devices", "err", err)

		return nil, err
	}

	// For unit testing, we need to setup hostname
	if *forceHostname != "" {
		hostname = *forceHostname
	}

	c := &lsfCollector{
		cgroupManager:              cgroupManager,
		cgroupCollector:            cgCollector,
		perfCollector:              perfCollector,
		ebpfCollector:              ebpfCollector,
		rdmaCollector:              rdmaCollector,
		hostname:                   hostname,
		gpuSMI:                     gpuSMI,
		bjobsExePath:               bjobsExePath,
		jobResourcesCacheTTL:       lsfCacheTTL,
		jobResourcesLastUpdateTime: time.Now(),
		logger:                     logger,
	}

	// If MIG devices found, setup security context
	if c.gpuSMI.InstancedEnabled() {
		// Instantiate a new Proc FS
		c.procFS, err = procfs.NewFS(*procfsPath)
		if err != nil {
			logger.Error("Unable to open procfs", "path", *procfsPath, "err", err)

			return nil, err
		}

		// Setup necessary capabilities. These are the caps we need to read
		// env vars in /proc file system to get MIG GPU indices
		caps, err := setupAppCaps([]string{"cap_sys_ptrace", "cap_dac_read_search"})
		if err != nil {
			logger.Warn("Failed to parse capability name(s)", "err", err)
		}

		// Setup new security context(s)
		cfg := &security.SCConfig{
			Name:         lsfReadProcCtx,
			Caps:         caps,
			Func:         readProcEnvirons,
			Logger:       logger,
			ExecNatively: disableCapAwareness,
		}

		securityCtx, err := security.NewSecurityContext(cfg)
		if err != nil {
			logger.Error("Failed to create a security context", "err", err)

			return nil, err
		}

		c.securityContexts = map[string]*security.SecurityContext{lsfReadProcCtx: securityCtx}
	}

	return c, nil
}

// Update implements Collector and update job metrics.
func (c *lsfCollector) Update(ch chan<- prometheus.Metric) error {
	// Initialise job metrics
	cgroups, err := c.jobCgroups()
	if err != nil {
		return err
	}

	// Start a wait group
	wg := sync.WaitGroup{}

	wg.Go(func() {
		// Update cgroup metrics
		err := c.cgroupCollector.Update(ch, cgroups, c.gpuSMI)
		if err != nil {
			c.logger.Error("Failed to update cgroup stats", "err", err)
		}
	})

	if perfCollectorEnabled() {
		wg.Go(func() {
			// Update perf metrics
			err := c.perfCollector.Update(ch, cgroups, lsfCollectorSubsystem)
			if err != nil {
				c.logger.Error("Failed to update perf stats", "err", err)
			}
		})
	}

	if ebpfCollectorEnabled() {
		wg.Go(func() {
			// Update ebpf metrics
			err := c.ebpfCollector.Update(ch, cgroups, lsfCollectorSubsystem)
			if err != nil {
				c.logger.Error("Failed to update IO and/or network stats", "err", err)
			}
		})
	}

	if rdmaCollectorEnabled() {
		wg.Go(func() {
			// Update RDMA metrics
			err := c.rdmaCollector.Update(ch, cgroups, lsfCollectorSubsystem)
			if err != nil {
				c.logger.Error("Failed to update RDMA stats", "err", err)
			}
		})
	}

	// Wait for all go routines
	wg.Wait()

	return nil
}

// Stop releases system resources used by the collector.
func (c *lsfCollector) Stop(ctx context.Context) error {
	c.logger.Debug("Stopping", "collector", lsfCollectorSubsystem)

	// Stop all sub collectors
	// Stop cgroupCollector
	err := c.cgroupCollector.Stop(ctx)
	if err != nil {
		c.logger.Error("Failed to stop cgroup collector", "err", err)
	}

	// Stop perfCollector
	if perfCollectorEnabled() {
		err := c.perfCollector.Stop(ctx)
		if err != nil {
			c.logger.Error("Failed to stop perf collector", "err", err)
		}
	}

	// Stop ebpfCollector
	if ebpfCollectorEnabled() {
		err := c.ebpfCollector.Stop(ctx)
		if err != nil {
			c.logger.Error("Failed to stop ebpf collector", "err", err)
		}
	}

	// Stop rdmaCollector
	if rdmaCollectorEnabled() {
		err := c.rdmaCollector.Stop(ctx)
		if err != nil {
			c.logger.Error("Failed to stop RDMA collector", "err", err)
		}
	}

	return nil
}

// jobResources updates resources with job IDs.
func (c *lsfCollector) jobResources(cgroups []cgroup) {
	// Get current job IDs on the node
	currentJobIDs := make([]string, len(cgroups))
	for icgroup, cgroup := range cgroups {
		currentJobIDs[icgroup] = cgroup.uuid
	}

	// Check if there are any new/deleted jobs between current and previous scrape
	if areEqual(currentJobIDs, c.previousJobIDs) && time.Since(c.jobResourcesLastUpdateTime) < c.jobResourcesCacheTTL {
		return
	}

	// LSF supports dynamic MIG and hence, every new job might potentially create
	// a new GPU device. Hence, we need to discover new devices, if any, when new job
	// is found
	err := c.gpuSMI.Discover()
	if err != nil {
		c.logger.Error("Failed to (re)discover GPU devices. Job devices will not be updated", "err", err)

		return
	}

	// Check if MIG instances enabled
	var nvidiaMIGEnabled bool

	if c.gpuSMI.InstancedEnabled() {
		for _, vendor := range c.gpuSMI.vendors {
			if vendor.id == nvidia {
				nvidiaMIGEnabled = true

				break
			}
		}
	}

	// Make a timeout of 1 sec
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Get job resources from bjobs command
	cmdOut, err := osexec.ExecuteContext(
		ctx, c.bjobsExePath,
		[]string{"-u", "all", "-m", c.hostname, "-json", "-o", "jobid jobindex alloc_slot nalloc_slot gpu_alloc"},
		nil,
	)
	if err != nil {
		c.logger.Error("Failed to execute bjobs command. Job devices will not be updated", "err", err)

		return
	}

	// Read command output into var
	var bjobs common.LSFJobsList

	err = json.Unmarshal(cmdOut, &bjobs)
	if err != nil {
		c.logger.Error("Failed to failed bjobs command output. Job devices will not be updated", "err", err)

		return
	}

	// Make a map from job ID to devices and ncpus
	jobDeviceMapper := make(map[string][]string)
	jobNumCPUsMapper := make(map[string]int)

	var jobsWithMIGDevice []string

	for _, job := range bjobs.Records {
		// Job GPU_SLOT is of form <hostname>:<gpu_id_proc1>,<gpu_id_proc2>..;<hostname>:<gpu_id_proc1>,<gpu_id_proc2>...
		// Split by ";" to get GPU indices for each host and check if current hostname
		// is in string
		for gpuSlot := range strings.SplitSeq(job.GPUSlot, ";") {
			if strings.Contains(gpuSlot, c.hostname) {
				// In the case of MIG devices, GPU IDs will be of format <hostname>:<gpu_id_proc1>:<mig_gpu_inst_size>\/<mig_compute_inst_size>,<gpu_id_proc2>:<mig_gpu_inst_size>\/<mig_compute_inst_size>
				// We cannot realiably track back GPU_I_ID and GPU_C_ID from the above string as they present only
				// sizes of GPU and compute instances.
				// This also means that we need to split the hostname and GPU devices at first occurrence of ":" as the MIG
				// devices also include ":" in their spec.
				// If we find, "/" in the GPU spec string, ignore as we will not get any meaningful data from
				// parsing that GPU spec. We will get MIG devices by reading env vars later.
				//
				// Turns out to be for MPS based jobs, ordinals are correctly setup in bjobs output.
				// In addition they export CUDA_VISIBLE_DEVICES_ORIG environment variable that has
				// global GPU minor values and we can rely on that as well in case if bjobs output does not work as intended.
				if _, hostGPUSlots, found := strings.Cut(gpuSlot, ":"); found {
					// We should check CUDA_VISIBLE_DEVICES env var only when MIG devices are attached to the job.
					// We can check it using the presence of "/" in the job GPU spec
					if strings.Contains(hostGPUSlots, "/") {
						jobsWithMIGDevice = append(jobsWithMIGDevice, job.ID)

						continue
					}

					gpuMinors := strings.Split(hostGPUSlots, ",")
					// gpuMinors will be a slice for each process in the job. So, there will
					// be repeations. We need to get only unique values
					//
					// More importantly, gpuMinors are the GPU MINORS and not indices. The global
					// indices that we use for GPUs are for interal representation. We should replace
					// these minors with our internal representation indices.
					slices.Sort(gpuMinors)
					gpuMinors = slices.Compact(gpuMinors)

					for _, gpuMinor := range gpuMinors {
						for _, dev := range c.gpuSMI.Devices {
							if dev.Minor == strings.TrimSpace(gpuMinor) {
								jobDeviceMapper[dev.Index] = append(jobDeviceMapper[dev.Index], job.ID)

								break
							}
						}
					}
				}
			}
		}

		// Parse ALLOC_SLOT which is in form <hostname1>:<hostname1>..:<hostname2>:<hostname2>..
		// Each hostname will repeated based on number of processes on that host.
		for hostSlot := range strings.SplitSeq(job.AllocSlot, ":") {
			if strings.Contains(hostSlot, c.hostname) {
				jobNumCPUsMapper[job.ID]++
			}
		}
	}

	// If no MIG devices found, nothing to do here
	if nvidiaMIGEnabled && len(c.securityContexts) == 0 {
		c.logger.Warn("GPU Instances enabled but collector does not have enough permissions to fetch job MIG devices. Please restart the exporter")
	}

	// First update cgroups with num cpus and also update jobDeviceMappers with any
	// MIG devices, when enabled
	for icgroup, cgroup := range cgroups {
		if ncpus, ok := jobNumCPUsMapper[cgroup.uuid]; ok {
			cgroups[icgroup].ncpus = ncpus
		}

		// Find MIG devices only for the jobs with MIG devices attached
		if nvidiaMIGEnabled && slices.Contains(jobsWithMIGDevice, cgroup.uuid) && len(c.securityContexts) > 0 {
			c.jobGPUInstanceResources(cgroup.uuid, cgroup.procs, jobDeviceMapper)
		}
	}

	// Iterate over devices to find which device corresponds to this id
	for igpu, gpu := range c.gpuSMI.Devices {
		// If device is physical GPU
		if uids, ok := jobDeviceMapper[gpu.Index]; ok {
			for handle, count := range elementCounts(uids) {
				c.gpuSMI.Devices[igpu].ComputeUnits = append(
					c.gpuSMI.Devices[igpu].ComputeUnits, ComputeUnit{UUID: handle.Value(), NumShares: count},
				)
			}

			c.gpuSMI.Devices[igpu].CurrentShares += uint64(len(uids))
		}

		// If device is instance GPU
		for iinst, inst := range gpu.Instances {
			if uids, ok := jobDeviceMapper[inst.Index]; ok {
				for handle, count := range elementCounts(uids) {
					c.gpuSMI.Devices[igpu].Instances[iinst].ComputeUnits = append(
						c.gpuSMI.Devices[igpu].Instances[iinst].ComputeUnits,
						ComputeUnit{UUID: handle.Value(), NumShares: count},
					)
				}

				c.gpuSMI.Devices[igpu].Instances[iinst].CurrentShares += uint64(len(uids))
			}
		}
	}

	// Update job IDs state variable
	c.previousJobIDs = currentJobIDs
	c.jobResourcesLastUpdateTime = time.Now()
}

// jobCgroups returns cgroups of active jobs.
func (c *lsfCollector) jobCgroups() ([]cgroup, error) {
	// Get current cgroups
	cgroups, err := c.cgroupManager.discover()
	if err != nil {
		return nil, fmt.Errorf("failed to discover cgroups: %w", err)
	}

	// Sometimes LSF daemon fails to clean up cgroups for
	// terminated jobs. In that case our current cgroup slice will
	// contain terminated jobs and it is not desirable. We clean
	// up current cgroups by looking at number of procs inside each
	// cgroup. When there are no procs associated with cgroup, it is
	// terminated job
	var activeCgroups []cgroup

	var staleCgroupIDs []string

	for _, cgroup := range cgroups {
		if len(cgroup.procs) > 0 {
			activeCgroups = append(activeCgroups, cgroup)
		} else {
			staleCgroupIDs = append(staleCgroupIDs, cgroup.uuid)
		}
	}

	// If stale cgroups found, emit a warning log
	if len(staleCgroupIDs) > 0 {
		c.logger.Warn(
			"Stale cgroups without any processes found", "ids", strings.Join(staleCgroupIDs, ","),
			"num_cgroups", len(staleCgroupIDs),
		)
	}

	// Update resources
	c.jobResources(activeCgroups)

	return activeCgroups, nil
}

// jobGPUInstanceResources returns MIG devices bound to current job.
func (c *lsfCollector) jobGPUInstanceResources(uuid string, procs []procfs.Proc, jobDeviceMapper map[string][]string) {
	// Read env vars in a security context that raises necessary capabilities
	// Apparently it is possible to disable setting CUDA_VISIBLE_DEVICES
	// during job submission. However, docs say that CUDA_VISIBLE_DEVICES<proc_number>
	// will be still set in this case which should be identical. Use it as fallback.
	// Ref: https://www.ibm.com/docs/en/spectrum-lsf/10.1.0?topic=10-gpu-enhancements
	dataPtr := &readProcSecurityCtxData{
		procs: procs,
		ignoreProc: func(envs []string) bool {
			// LSB_JOBID WILL NOT HAVE job index. LSB_BATCH_JID is more realiable
			return !slices.Contains(envs, "LSB_BATCH_JID="+uuid)
		},
		// In the case of AMD GPUs, GPU_DEVICE_ORDINAL and GPU_DEVICE_ORDINAL1 are
		// exported with "real" GPU minor numbers. But at the same time, bjobs
		// output has correct ordinals already so, there wont be a need for this.
		targetEnvVars: []string{"CUDA_VISIBLE_DEVICES", "CUDA_VISIBLE_DEVICES1"},
	}

	// Get env var values
	if securityCtx, ok := c.securityContexts[lsfReadProcCtx]; ok {
		err := securityCtx.Exec(dataPtr)
		if err != nil {
			c.logger.Error(
				"Failed to run inside security contxt", "jobid", uuid, "err", err,
			)

			return
		}
	} else {
		c.logger.Error(
			"Security context not found", "name", lsfReadProcCtx, "jobid", uuid,
		)

		return
	}

	// Process the found env vars
	var cudaVisibleDevices string

	// Get MIG GPU UUIDs. They will be of form:
	// CUDA_VISIBLE_DEVICES=MIG-GPU-c3d90686-d5f4-6940-6fff-4ed682431d64/11/0,MIG-GPU-c3d90686-d5f4-6940-6fff-4ed682431d64/8/0
	// where MIG- prefix is added to physical GPU UUID. The format is as follows:
	// MIG-<Physical GPU UUID>/GPU_I_ID/GPU_C_ID
	for _, envVar := range dataPtr.targetEnvVars {
		if val, ok := dataPtr.targetEnvVarValues[envVar]; ok {
			cudaVisibleDevices = val

			break
		}
	}

	// If cudaVisibleDevices is not found, nothing to do here. Log a warning and return
	if cudaVisibleDevices == "" {
		c.logger.Warn("Failed to get MIG GPU ordinals or job does not request MIG GPU resources", "jobid", uuid)

		return
	}

	// By this time, we found CUDA_VISIBLE_DEVICES
	for migID := range strings.SplitSeq(cudaVisibleDevices, ",") {
		match := migDeviceIDRegex.FindStringSubmatch(migID)
		// If no matches found, emit a warning and skip
		if len(match) == 0 {
			c.logger.Warn("Failed to parse CUDA_VISIBLE_DEVICE environment variable", "jobid", uuid, "value", migID)

			continue
		}

		// Get all regex captured groups
		var (
			gpuUUID        string
			gpuGID, gpuCID uint64
		)

		for i, name := range migDeviceIDRegex.SubexpNames() {
			// Normalize UUID to ensure comparison
			if name == "GPU_UUID" {
				gpuUUID = strings.ToLower(strings.TrimSpace(match[i]))
			}

			if name == "GPU_I_ID" {
				m, err := strconv.ParseUint(match[i], 10, 64)
				if err == nil {
					gpuGID = m
				}
			}

			if name == "GPU_C_ID" {
				m, err := strconv.ParseUint(match[i], 10, 64)
				if err == nil {
					gpuCID = m
				}
			}
		}

		// Add job to jobDeviceMapper
		for _, gpu := range c.gpuSMI.Devices {
			if strings.ToLower(gpu.UUID) == gpuUUID {
				for _, inst := range gpu.Instances {
					if inst.ComputeInstID == gpuCID && inst.GPUInstID == gpuGID {
						jobDeviceMapper[inst.Index] = append(jobDeviceMapper[inst.Index], uuid)
					}
				}
			}
		}
	}

	// Emit logs with found GPU ordinals and shards and/or MPS shares
	c.logger.Debug("GPU MIG devices", "jobid", uuid, "devices", cudaVisibleDevices)
}
