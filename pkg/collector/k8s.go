//go:build !nok8s

package collector

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	ceems_k8s "github.com/ceems-dev/ceems/pkg/k8s"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	k8sCollectorSubsystem = "k8s"
)

// CLI opts.
var (
	kubeConfigFileDepr = CEEMSExporterApp.Flag(
		"collector.k8s.kube-config-file",
		"Path to the configuration file to connect to k8s cluster. If left empty, in-cluster config will be used",
	).Default("").Hidden().String()
	kubeConfigFile = CEEMSExporterApp.Flag(
		"collector.k8s.kubeconfig.file",
		"Path to the configuration file to connect to k8s cluster. If left empty, in-cluster config will be used",
	).Default("").String()

	kubeletSocketFileDepre = CEEMSExporterApp.Flag(
		"collector.k8s.kubelet-socket-file",
		"Path to the kubelet pod-resources socket file",
	).Default("/var/lib/kubelet/pod-resources/kubelet.sock").Hidden().String()
	kubeletSocketFile = CEEMSExporterApp.Flag(
		"collector.k8s.kubelet-podresources-socket.file",
		"Path to the kubelet pod-resources socket file",
	).Default("/var/lib/kubelet/pod-resources/kubelet.sock").String()
	k8sCollectPSIStats = CEEMSExporterApp.Flag(
		"collector.k8s.psi-metrics",
		"Enables collection of PSI metrics (default: disabled)",
	).Default("false").Bool()
)

type k8sCollector struct {
	logger                   *slog.Logger
	cgroupManager            *cgroupManager
	cgroupCollector          *cgroupCollector
	perfCollector            *perfCollector
	ebpfCollector            *ebpfCollector
	rdmaCollector            *rdmaCollector
	hostname                 string
	gpuSMI                   *GPUSMI
	k8sClient                *ceems_k8s.Client
	previousPodUIDs          []string
	podDevicesCacheTTL       time.Duration
	podDevicesLastUpdateTime time.Time
}

func init() {
	RegisterCollector(k8sCollectorSubsystem, defaultDisabled, NewK8sCollector)
}

// NewK8sCollector returns a new Collector exposing a summary of cgroups.
func NewK8sCollector(logger *slog.Logger) (Collector, error) {
	// Log deprecation notices
	if *kubeConfigFileDepr != "" {
		logger.Warn("flag --collector.k8s.kube-config-file has been deprecated. Use --collector.k8s.kubeconfig.file instead")

		*kubeConfigFile = *kubeConfigFileDepr
	}

	if *kubeletSocketFileDepre != "" {
		logger.Warn("flag --collector.k8s.kubelet-socket-file has been deprecated. Use --collector.k8s.kubelet-podresources-socket.file instead")

		*kubeletSocketFile = *kubeletSocketFileDepre
	}

	// Get SLURM's cgroup details
	cgroupManager, err := NewCgroupManager(k8s, logger)
	if err != nil {
		logger.Info("Failed to create cgroup manager", "err", err)

		return nil, err
	}

	logger.Info("cgroup: " + cgroupManager.String())

	// Set cgroup options
	opts := cgroupOpts{
		collectPSIStats:     *k8sCollectPSIStats,
		collectSwapMemStats: true,
		collectBlockIOStats: true,
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

	// Create a new k8s client
	client, err := ceems_k8s.New(*kubeConfigFile, *kubeletSocketFile, logger.With("subsystem", "k8s_client"))
	if err != nil {
		logger.Error("Failed to create k8s client", "err", err)

		return nil, err
	}

	// Instantiate a new instance of GPUSMI struct
	gpuSMI, err := NewGPUSMI(client, logger)
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

	// Instantiate a collector
	coll := &k8sCollector{
		cgroupManager:            cgroupManager,
		cgroupCollector:          cgCollector,
		perfCollector:            perfCollector,
		ebpfCollector:            ebpfCollector,
		rdmaCollector:            rdmaCollector,
		hostname:                 hostname,
		k8sClient:                client,
		gpuSMI:                   gpuSMI,
		podDevicesCacheTTL:       15 * time.Minute,
		podDevicesLastUpdateTime: time.Now(),
		logger:                   logger,
	}

	// Set up file permissions on kubelet socket when GPU devices are found
	// We are using ACL on socket file instead of using capability inside a
	// security context as the go-grpc package uses go routines inside
	// its implementation to make requests to unix socket file. This means
	// there will be additional go routines inside the security context which
	// will not be scheduled on the kernel thread that has privilege to reach
	// socket. Thus, we end up with permission denied errors.
	if len(gpuSMI.Devices) > 0 {
		// Get absolute path of kubelet socket
		kubeletSocketFilePath, err := filepath.Abs(*kubeletSocketFile)
		if err != nil {
			logger.Error("Failed to get absolute path of kubelet socket", "err", err)

			return nil, err
		}

		// Evaluate symlinks
		kubeletSocketFilePath, err = filepath.EvalSymlinks(kubeletSocketFilePath)
		if err != nil {
			logger.Error("Failed to resolve path of kubelet socket", "err", err)

			return nil, err
		}

		// We need to have write permissions on this socket to make gRPC requests
		readWritePaths := []string{kubeletSocketFilePath}

		// Now loop over all the directories to ensure that we have rx permissions
		var readPaths []string

		dir := filepath.Dir(kubeletSocketFilePath)

		for dir != "/" {
			// Add path to read paths
			readPaths = append(readPaths, dir)

			// Get next directory
			dir = filepath.Dir(dir)
		}

		// Setup necessary path permissions
		setupAppPathPerms(readPaths, readWritePaths)
	}

	return coll, nil
}

// Update implements Collector and update job metrics.
func (c *k8sCollector) Update(ch chan<- prometheus.Metric) error {
	// Get pod cgroups and update devices
	cgroups, err := c.podCgroups()
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
			err := c.perfCollector.Update(ch, cgroups, k8sCollectorSubsystem)
			if err != nil {
				c.logger.Error("Failed to update perf stats", "err", err)
			}
		})
	}

	if ebpfCollectorEnabled() {
		wg.Go(func() {
			// Update ebpf metrics
			err := c.ebpfCollector.Update(ch, cgroups, k8sCollectorSubsystem)
			if err != nil {
				c.logger.Error("Failed to update IO and/or network stats", "err", err)
			}
		})
	}

	if rdmaCollectorEnabled() {
		wg.Go(func() {
			// Update RDMA metrics
			err := c.rdmaCollector.Update(ch, cgroups, k8sCollectorSubsystem)
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
func (c *k8sCollector) Stop(ctx context.Context) error {
	c.logger.Debug("Stopping", "collector", k8sCollectorSubsystem)

	// Stop k8s client
	err := c.k8sClient.Close()
	if err != nil {
		c.logger.Error("Failed to stop k8s client", "err", err)
	}

	// Stop all sub collectors
	// Stop cgroupCollector
	err = c.cgroupCollector.Stop(ctx)
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

// podDevices updates devices with pod UIDs.
func (c *k8sCollector) podDevices(cgroups []cgroup) {
	// If there are no GPU devices, nothing to do here. Return
	if len(c.gpuSMI.Devices) == 0 {
		return
	}

	// Get current pod UIDs on the node
	currentPodUIDs := make([]string, len(cgroups))
	for icgroup, cgroup := range cgroups {
		currentPodUIDs[icgroup] = cgroup.uuid
	}

	// Check if there are any new/deleted pods between current and previous
	if areEqual(currentPodUIDs, c.previousPodUIDs) && time.Since(c.podDevicesLastUpdateTime) < c.podDevicesCacheTTL {
		return
	}

	// As DRA is in GA now, MIG profiles can be created dynamically. This means
	// we need to always rediscover the GPU devices everytime a new pod has been
	// created to account for new MIG partitions, if any
	// Attempt to get GPU devices
	err := c.gpuSMI.Discover()
	if err != nil {
		c.logger.Error("Failed to (re)discover GPU devices. Pod to device mappings will not be available", "err", err)

		return
	}

	// Make a timeout of 1 sec
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Get pod devices from pod resource API
	pods, err := c.k8sClient.ListPodsWithDevs(ctx)
	if err != nil {
		c.logger.Error("Failed to fetch pod resources. Pod to device mappings will not be available", "err", err)

		return
	}

	// Make a map from pod UUID to devices
	podDeviceMapper := make(map[string][]string)

	for _, pod := range pods {
		for _, cont := range pod.Containers {
			for _, dev := range cont.Devices {
				for _, id := range dev.IDs {
					// For NVIDIA GPUs, when time slicing is enabled, deviceID will
					// be of form GPU-<uuid>::1 or MIG-<uuid>::1.
					// So we need to split by :: and take first part
					id = strings.ToLower(strings.Split(id, "::")[0])

					podDeviceMapper[id] = append(podDeviceMapper[id], pod.UID)
				}
			}
		}
	}

	// Iterate over devices to find which device corresponds to this id
	for igpu, gpu := range c.gpuSMI.Devices {
		// If device is physical GPU
		if uids, ok := podDeviceMapper[strings.ToLower(gpu.ID())]; ok {
			for handle, count := range elementCounts(uids) {
				c.gpuSMI.Devices[igpu].ComputeUnits = append(
					c.gpuSMI.Devices[igpu].ComputeUnits, ComputeUnit{UUID: handle.Value(), NumShares: count},
				)
			}

			c.gpuSMI.Devices[igpu].CurrentShares += uint64(len(uids))
		}

		// If device is instance GPU
		for iinst, inst := range gpu.Instances {
			if uids, ok := podDeviceMapper[strings.ToLower(inst.ID())]; ok {
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

	// Update pod UIDs state variable
	c.previousPodUIDs = currentPodUIDs
}

// podCgroups returns cgroups for active pods.
func (c *k8sCollector) podCgroups() ([]cgroup, error) {
	// Get active cgroups
	cgroups, err := c.cgroupManager.discover()
	if err != nil {
		return nil, fmt.Errorf("failed to discover cgroups: %w", err)
	}

	// Update devices
	c.podDevices(cgroups)

	return cgroups, nil
}
