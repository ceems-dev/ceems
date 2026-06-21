//go:build !noopenpbs

package collector

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ceems-dev/ceems/internal/security"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

const (
	openpbsCollectorSubsystem = "openpbs"
)

// CLI opts.
var (
	// cgroup opts.
	openpbsCollectSwapMemoryStats = CEEMSExporterApp.Flag(
		"collector.openpbs.swap-memory-metrics",
		"Enables collection of swap memory metrics (default: disabled)",
	).Default("false").Bool()
	openpbsCollectPSIStats = CEEMSExporterApp.Flag(
		"collector.openpbs.psi-metrics",
		"Enables collection of PSI metrics (default: disabled)",
	).Default("false").Bool()
)

// Security context names.
const (
	openpbsReadProcCtx = "openpbs_read_procs"
)

// Cache interval.
var (
	openpbsCacheTTL = 15 * time.Minute
)

type openpbsCollector struct {
	logger                     *slog.Logger
	cgroupManager              *cgroupManager
	cgroupCollector            *cgroupCollector
	perfCollector              *perfCollector
	ebpfCollector              *ebpfCollector
	rdmaCollector              *rdmaCollector
	hostname                   string
	gpuSMI                     *GPUSMI
	previousJobIDs             []string
	jobResourcesLastUpdateTime time.Time
	jobResourcesCacheTTL       time.Duration
	procFS                     procfs.FS
	securityContexts           map[string]*security.SecurityContext
}

func init() {
	RegisterCollector(openpbsCollectorSubsystem, defaultDisabled, NewOpenPBSCollector)
}

// NewOpenPBSCollector returns a new Collector exposing a summary of cgroups.
func NewOpenPBSCollector(logger *slog.Logger) (Collector, error) {
	// Get OpenPBS's cgroup details
	cgroupManager, err := NewCgroupManager(openpbs, logger)
	if err != nil {
		logger.Error("Failed to create cgroup manager", "err", err)

		return nil, err
	}

	logger.Info("cgroup: " + cgroupManager.String())

	// Set cgroup options
	opts := cgroupOpts{
		collectSwapMemStats: *openpbsCollectSwapMemoryStats,
		collectPSIStats:     *openpbsCollectPSIStats,
		collectBlockIOStats: false, // OpenPBS does not support blkio controller.
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

	c := &openpbsCollector{
		cgroupManager:              cgroupManager,
		cgroupCollector:            cgCollector,
		perfCollector:              perfCollector,
		ebpfCollector:              ebpfCollector,
		rdmaCollector:              rdmaCollector,
		hostname:                   hostname,
		gpuSMI:                     gpuSMI,
		jobResourcesCacheTTL:       openpbsCacheTTL,
		jobResourcesLastUpdateTime: time.Now(),
		logger:                     logger,
	}

	// If GPU devices found, setup security context
	if len(c.gpuSMI.Devices) > 0 {
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
			Name:         openpbsReadProcCtx,
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

		c.securityContexts = map[string]*security.SecurityContext{openpbsReadProcCtx: securityCtx}
	}

	return c, nil
}

// Update implements Collector and update job metrics.
func (c *openpbsCollector) Update(ch chan<- prometheus.Metric) error {
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
			err := c.perfCollector.Update(ch, cgroups, openpbsCollectorSubsystem)
			if err != nil {
				c.logger.Error("Failed to update perf stats", "err", err)
			}
		})
	}

	if ebpfCollectorEnabled() {
		wg.Go(func() {
			// Update ebpf metrics
			err := c.ebpfCollector.Update(ch, cgroups, openpbsCollectorSubsystem)
			if err != nil {
				c.logger.Error("Failed to update IO and/or network stats", "err", err)
			}
		})
	}

	if rdmaCollectorEnabled() {
		wg.Go(func() {
			// Update RDMA metrics
			err := c.rdmaCollector.Update(ch, cgroups, openpbsCollectorSubsystem)
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
func (c *openpbsCollector) Stop(ctx context.Context) error {
	c.logger.Debug("Stopping", "collector", openpbsCollectorSubsystem)

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
func (c *openpbsCollector) jobResources(cgroups []cgroup) {
	// Get current job IDs on the node
	currentJobIDs := make([]string, len(cgroups))
	for icgroup, cgroup := range cgroups {
		currentJobIDs[icgroup] = cgroup.uuid
	}

	// Check if there are any new/deleted jobs between current and previous scrape
	if areEqual(currentJobIDs, c.previousJobIDs) && time.Since(c.jobResourcesLastUpdateTime) < c.jobResourcesCacheTTL {
		return
	}

	// OpenPBS supports dynamic MIG and hence, every new job might potentially create
	// a new GPU device. Hence, we need to discover new devices, if any, when new job
	// is found
	err := c.gpuSMI.Discover()
	if err != nil {
		c.logger.Error("Failed to (re)discover GPU devices. Job devices will not be updated", "err", err)

		return
	}

	// First update cgroups with num cpus and also update jobDeviceMappers with any
	// MIG devices, when enabled
	for _, cgroup := range cgroups {
		// Get GPU ordinals of the job
		for _, uuid := range c.jobGPUInstanceResources(cgroup.uuid, cgroup.procs) {
			deviceUUID := strings.ToLower(strings.TrimSpace(uuid))

			// Iterate over devices to find which device corresponds to this id
			for igpu, gpu := range c.gpuSMI.Devices {
				// If device is physical GPU
				if strings.ToLower(gpu.UUID) == deviceUUID {
					c.gpuSMI.Devices[igpu].ComputeUnits = append(c.gpuSMI.Devices[igpu].ComputeUnits, ComputeUnit{cgroup.uuid, cgroup.hostname, 1})
					c.gpuSMI.Devices[igpu].CurrentShares += 1
				}

				// If device is instance GPU
				for iinst, inst := range gpu.Instances {
					if strings.ToLower(inst.UUID) == deviceUUID {
						c.gpuSMI.Devices[igpu].Instances[iinst].ComputeUnits = append(c.gpuSMI.Devices[igpu].Instances[iinst].ComputeUnits, ComputeUnit{cgroup.uuid, cgroup.hostname, 1})
						c.gpuSMI.Devices[igpu].Instances[iinst].CurrentShares += 1
					}
				}
			}
		}
	}

	// Update job IDs state variable
	c.previousJobIDs = currentJobIDs
	c.jobResourcesLastUpdateTime = time.Now()
}

// jobCgroups returns cgroups of active jobs.
func (c *openpbsCollector) jobCgroups() ([]cgroup, error) {
	// Get current cgroups
	cgroups, err := c.cgroupManager.discover()
	if err != nil {
		return nil, fmt.Errorf("failed to discover cgroups: %w", err)
	}

	// Sometimes OpenPBS daemon fails to clean up cgroups for
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

// jobGPUInstanceResources returns GPU devices bound to current job.
func (c *openpbsCollector) jobGPUInstanceResources(uuid string, procs []procfs.Proc) []string {
	// Read env vars in a security context that raises necessary capabilities
	// Apparently CUDA_VISIBLE_DEVICES set GPU UUIDs
	// Ref: https://github.com/openpbs/openpbs/blob/cd7ab5edaf03dcac2f1cb9e2f42d75eb117d468a/src/hooks/cgroups/pbs_cgroups.PY#L4147-L4185
	dataPtr := &readProcSecurityCtxData{
		procs: procs,
		ignoreProc: func(envs []string) bool {
			return !slices.Contains(envs, "PBS_JOBID="+uuid)
		},
		targetEnvVars: []string{"CUDA_VISIBLE_DEVICES"},
	}

	// Get env var values
	if securityCtx, ok := c.securityContexts[openpbsReadProcCtx]; ok {
		err := securityCtx.Exec(dataPtr)
		if err != nil {
			c.logger.Error(
				"Failed to run inside security contxt", "jobid", uuid, "err", err,
			)

			return nil
		}
	} else {
		c.logger.Error(
			"Security context not found", "name", openpbsReadProcCtx, "jobid", uuid,
		)

		return nil
	}

	// Process the found env vars
	var cudaVisibleDevices string

	// Get GPU UUIDs. They will be of form:
	// CUDA_VISIBLE_DEVICES=GPU-c3d90686-d5f4-6940-6fff-4ed682431d64,GPU-c3d90686-d5f4-6940-6fff-4ed682431d64
	for _, envVar := range dataPtr.targetEnvVars {
		if val, ok := dataPtr.targetEnvVarValues[envVar]; ok {
			cudaVisibleDevices = val

			break
		}
	}

	// If cudaVisibleDevices is not found, nothing to do here. Log a warning and return
	if cudaVisibleDevices == "" {
		c.logger.Warn("Failed to get GPU ordinals or job does not request GPU resources", "jobid", uuid)

		return nil
	}

	// Emit logs with found GPU ordinals and shards and/or MPS shares
	c.logger.Debug("GPU devices", "jobid", uuid, "devices", cudaVisibleDevices)

	return strings.Split(cudaVisibleDevices, ",")
}
