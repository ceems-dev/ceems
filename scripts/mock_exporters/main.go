package main

import (
	"context"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var minNvGPUPower, maxNvGPUPower, minAMDGPUPower, maxAMDGPUPower float64

type Device struct {
	ID      string
	UUID    string
	IID     string
	PCIAddr string
}

// DCGM collector.
type dcgmCollector struct {
	devices        []Device
	gpuUtil        *prometheus.Desc
	gpuMemFree     *prometheus.Desc
	gpuMemUsed     *prometheus.Desc
	gpuPower       *prometheus.Desc
	gpuPowerInst   *prometheus.Desc
	gpuSMActive    *prometheus.Desc
	gpuSMOcc       *prometheus.Desc
	gpuGREngActive *prometheus.Desc
	gpuPipeActive  *prometheus.Desc
	gpuFP64Active  *prometheus.Desc
	gpuFP32Active  *prometheus.Desc
	gpuFP16Active  *prometheus.Desc
	gpuNVLTX       *prometheus.Desc
	gpuNVLRX       *prometheus.Desc
	gpuDRAMActive  *prometheus.Desc
	gpuPCIeRX      *prometheus.Desc
	gpuPCIeTX      *prometheus.Desc
}

func randFloat(minVal, maxVal float64) float64 {
	return minVal + rand.Float64()*(maxVal-minVal) //nolint:gosec
}

func newDCGMCollector() *dcgmCollector {
	devices := []Device{
		{"0", "GPU-f124aa59-d406-d45b-9481-8fcd694e6c9e", "", "00000000:10:00.0"},
		{"1", "GPU-61a65011-6571-a6d2-5ab8-66cbb6f7f9c3", "", "00000000:15:00.0"},
		{"2", "GPU-956348bc-d43d-23ed-53d4-857749fa2b67", "1", "00000000:21:00.0"},
		{"2", "GPU-956348bc-d43d-23ed-53d4-857749fa2b67", "5", "00000000:21:00.0"},
		{"2", "GPU-956348bc-d43d-23ed-53d4-857749fa2b67", "13", "00000000:21:00.0"},
		{"3", "GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7", "1", "00000000:81:00.0"},
		{"3", "GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7", "5", "00000000:81:00.0"},
		{"3", "GPU-feba7e40-d724-01ff-b00f-3a439a28a6c7", "6", "00000000:81:00.0"},
		{"4", "GPU-61a65011-6571-a6d2-5th8-66cbb6f7f9c3", "", "00000000:83:00.0"},
		{"5", "GPU-61a65011-6571-a64n-5ab8-66cbb6f7f9c3", "", "00000000:85:00.0"},
		{"6", "GPU-1d4d0f3e-b51a-4040-96e3-bf380f7c5728", "", "00000000:87:00.0"},
		{"7", "GPU-6cc98505-fdde-461e-a93c-6935fba45a27", "", "00000000:89:00.0"},
	}

	return &dcgmCollector{
		devices: devices,
		gpuUtil: prometheus.NewDesc("DCGM_FI_DEV_GPU_UTIL",
			"GPU utilization",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuMemUsed: prometheus.NewDesc("DCGM_FI_DEV_FB_USED",
			"GPU memory used",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuMemFree: prometheus.NewDesc("DCGM_FI_DEV_FB_FREE",
			"GPU memory free",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuPower: prometheus.NewDesc("DCGM_FI_DEV_POWER_USAGE",
			"GPU power",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuPowerInst: prometheus.NewDesc("DCGM_FI_DEV_POWER_USAGE_INSTANT",
			"GPU power",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuSMActive: prometheus.NewDesc("DCGM_FI_PROF_SM_ACTIVE",
			"GPU SM active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuSMOcc: prometheus.NewDesc("DCGM_FI_PROF_SM_OCCUPANCY",
			"GPU SM occupancy",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuGREngActive: prometheus.NewDesc("DCGM_FI_PROF_GR_ENGINE_ACTIVE",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuPipeActive: prometheus.NewDesc("DCGM_FI_PROF_PIPE_TENSOR_ACTIVE",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuFP64Active: prometheus.NewDesc("DCGM_FI_PROF_PIPE_FP64_ACTIVE",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuFP32Active: prometheus.NewDesc("DCGM_FI_PROF_PIPE_FP32_ACTIVE",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuFP16Active: prometheus.NewDesc("DCGM_FI_PROF_PIPE_FP16_ACTIVE",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuDRAMActive: prometheus.NewDesc("DCGM_FI_PROF_DRAM_ACTIVE",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuNVLTX: prometheus.NewDesc("DCGM_FI_PROF_NVLINK_TX_BYTES",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuNVLRX: prometheus.NewDesc("DCGM_FI_PROF_NVLINK_RX_BYTES",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuPCIeTX: prometheus.NewDesc("DCGM_FI_PROF_PCIE_TX_BYTES",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
		gpuPCIeRX: prometheus.NewDesc("DCGM_FI_PROF_PCIE_RX_BYTES",
			"GPU GR engien active",
			[]string{"Hostname", "UUID", "GPU_I_ID", "device", "gpu", "pci_bus_id", "modelName"}, nil,
		),
	}
}

// Describe writes all descriptors to the prometheus desc channel.
func (collector *dcgmCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.gpuUtil

	ch <- collector.gpuMemUsed

	ch <- collector.gpuMemFree

	ch <- collector.gpuPower

	ch <- collector.gpuSMActive

	ch <- collector.gpuSMOcc

	ch <- collector.gpuGREngActive

	ch <- collector.gpuPipeActive

	ch <- collector.gpuFP64Active

	ch <- collector.gpuFP32Active

	ch <- collector.gpuFP16Active

	ch <- collector.gpuDRAMActive

	ch <- collector.gpuNVLRX

	ch <- collector.gpuNVLTX

	ch <- collector.gpuPCIeRX

	ch <- collector.gpuPCIeTX
}

// Collect implements required collect function for all promehteus collectors.
func (collector *dcgmCollector) Collect(ch chan<- prometheus.Metric) {
	// Generate random power consumptions for physical devices
	powerUsage := make(map[string]float64)
	for _, dev := range collector.devices {
		powerUsage[dev.ID] = randFloat(minNvGPUPower, maxNvGPUPower)
	}

	for _, dev := range collector.devices {
		ch <- prometheus.MustNewConstMetric(
			collector.gpuUtil, prometheus.GaugeValue, 100*rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuMemUsed, prometheus.GaugeValue, 100*rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuMemFree, prometheus.GaugeValue, 100*rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuPower, prometheus.GaugeValue, powerUsage[dev.ID], "host", dev.UUID, dev.IID,
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuPowerInst, prometheus.GaugeValue, powerUsage[dev.ID], "host", dev.UUID, dev.IID,
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuSMActive, prometheus.GaugeValue, rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuSMOcc, prometheus.GaugeValue, rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuGREngActive, prometheus.GaugeValue, rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuPipeActive, prometheus.GaugeValue, rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuFP64Active, prometheus.GaugeValue, rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuFP32Active, prometheus.GaugeValue, rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuFP16Active, prometheus.GaugeValue, rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuDRAMActive, prometheus.GaugeValue, rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuNVLRX, prometheus.GaugeValue, 1024*1024*1024*rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuNVLTX, prometheus.GaugeValue, 1024*1024*1024*rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuPCIeTX, prometheus.GaugeValue, 1024*1024*rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuPCIeRX, prometheus.GaugeValue, 1024*1024*rand.Float64(), "host", dev.UUID, dev.IID, //nolint:gosec
			"nvidia"+dev.ID, dev.ID, dev.PCIAddr, "NVIDIA A100 80GiB",
		)
	}
}

// AMD SMI collector.
type amdSMICollector struct {
	devices    []Device
	gpuUtil    *prometheus.Desc
	gpuMemUtil *prometheus.Desc
	gpuPower   *prometheus.Desc
}

func newAMDSMICollector() *amdSMICollector {
	devices := []Device{
		{"0", "20170000800c", "", "00000000:15:00.0"},
		{"1", "20170003580c", "", "00000000:16:00.0"},
		{"2", "20180003050c", "", "00000000:17:00.0"},
		{"3", "20170005280c", "", "00000000:18:00.0"},
	}

	return &amdSMICollector{
		devices: devices,
		gpuUtil: prometheus.NewDesc("amd_gpu_use_percent",
			"GPU utilization",
			[]string{"gpu_use_percent", "productname"}, nil,
		),
		gpuMemUtil: prometheus.NewDesc("amd_gpu_memory_use_percent",
			"GPU memory used",
			[]string{"gpu_memory_use_percent", "productname"}, nil,
		),
		gpuPower: prometheus.NewDesc("amd_gpu_power",
			"GPU power",
			[]string{"gpu_power", "productname"}, nil,
		),
	}
}

// Describe writes all descriptors to the prometheus desc channel.
func (collector *amdSMICollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.gpuUtil

	ch <- collector.gpuMemUtil

	ch <- collector.gpuPower
}

// Collect implements required collect function for all promehteus collectors.
func (collector *amdSMICollector) Collect(ch chan<- prometheus.Metric) {
	for idev := range collector.devices {
		ch <- prometheus.MustNewConstMetric(
			collector.gpuUtil, prometheus.GaugeValue, 100*rand.Float64(), strconv.Itoa(idev), //nolint:gosec
			"Advanced Micro Devices Inc",
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuMemUtil, prometheus.GaugeValue, 100*rand.Float64(), strconv.Itoa(idev), //nolint:gosec
			"Advanced Micro Devices Inc",
		)
		// GPU power reported in micro Watts
		ch <- prometheus.MustNewConstMetric(
			collector.gpuPower, prometheus.GaugeValue, 1e6*randFloat(minAMDGPUPower, maxAMDGPUPower), strconv.Itoa(idev),
			"Advanced Micro Devices Inc",
		)
	}
}

// AMD Device Metrics collector.
type amdDeviceMetricsCollector struct {
	devices            []Device
	gpuUtil            *prometheus.Desc
	gpuVRAMTotal       *prometheus.Desc
	gpuVRAMUsed        *prometheus.Desc
	gpuGTTTotal        *prometheus.Desc
	gpuGTTUsed         *prometheus.Desc
	gpuVisibleRAMTotal *prometheus.Desc
	gpuVisibleRAMUsed  *prometheus.Desc
	gpuPower           *prometheus.Desc
	gpuSMActive        *prometheus.Desc
	gpuProfOccupancy   *prometheus.Desc
	gpuTensorActive    *prometheus.Desc
	gpuFP64Ops         *prometheus.Desc
	gpuFP32Ops         *prometheus.Desc
	gpuFP16Ops         *prometheus.Desc
	gpuWriteSize       *prometheus.Desc
	gpuReadSize        *prometheus.Desc
}

func newAMDDeviceMetricsCollector(prefix string) *amdDeviceMetricsCollector {
	devices := []Device{
		{"0", "20170000800c", "0", "00000000:15:00.0"},
		{"1", "20170003580c", "0", "00000000:16:00.0"},
		{"2", "20170003580c", "1", "00000000:16:00.0"},
		{"3", "20170003580c", "2", "00000000:16:00.0"},
		{"4", "20170003580c", "3", "00000000:16:00.0"},
		{"5", "20170003580c", "4", "00000000:16:00.0"},
		{"6", "20170003580c", "5", "00000000:16:00.0"},
		{"7", "20170003580c", "6", "00000000:16:00.0"},
		{"8", "20170003580c", "7", "00000000:16:00.0"},
		{"9", "20180003050c", "0", "00000000:17:00.0"},
		{"10", "20180003050c", "1", "00000000:17:00.0"},
		{"11", "20170005280c", "0", "00000000:18:00.0"},
	}

	return &amdDeviceMetricsCollector{
		devices: devices,
		gpuUtil: prometheus.NewDesc(prefix+"gpu_gfx_activity",
			"GPU utilization",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuVRAMTotal: prometheus.NewDesc(prefix+"gpu_total_vram",
			"GPU VRAM total",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuVRAMUsed: prometheus.NewDesc(prefix+"gpu_used_vram",
			"GPU VRAM used",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuGTTTotal: prometheus.NewDesc(prefix+"gpu_total_gtt",
			"GTT memory total",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuGTTUsed: prometheus.NewDesc(prefix+"gpu_used_gtt",
			"GTT memory used",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuVisibleRAMTotal: prometheus.NewDesc(prefix+"gpu_total_visible_vram",
			"Visible RAM memory total",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuVisibleRAMUsed: prometheus.NewDesc(prefix+"gpu_used_visible_vram",
			"Visible RAM memory used",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuPower: prometheus.NewDesc(prefix+"gpu_package_power",
			"GPU power",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuSMActive: prometheus.NewDesc(prefix+"gpu_prof_sm_active",
			"GPU SM active",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuTensorActive: prometheus.NewDesc(prefix+"gpu_prof_tensor_active_percent",
			"GPU SM active",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuProfOccupancy: prometheus.NewDesc(prefix+"gpu_prof_occupancy_percent",
			"GPU occupancy",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuFP64Ops: prometheus.NewDesc(prefix+"gpu_prof_total_64_ops",
			"GPU FP64 ops",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuFP32Ops: prometheus.NewDesc(prefix+"gpu_prof_total_32_ops",
			"GPU FP64 ops",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuFP16Ops: prometheus.NewDesc(prefix+"gpu_prof_total_16_ops",
			"GPU FP64 ops",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuWriteSize: prometheus.NewDesc(prefix+"gpu_prof_write_size",
			"GPU FP64 ops",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
		gpuReadSize: prometheus.NewDesc(prefix+"gpu_prof_fetch_size",
			"GPU FP64 ops",
			[]string{"gpu_id", "gpu_partition_id", "serial_number"}, nil,
		),
	}
}

// Describe writes all descriptors to the prometheus desc channel.
func (collector *amdDeviceMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.gpuUtil

	ch <- collector.gpuVRAMTotal

	ch <- collector.gpuVRAMTotal

	ch <- collector.gpuGTTTotal

	ch <- collector.gpuGTTUsed

	ch <- collector.gpuVisibleRAMTotal

	ch <- collector.gpuVisibleRAMUsed

	ch <- collector.gpuPower

	ch <- collector.gpuSMActive

	ch <- collector.gpuProfOccupancy

	ch <- collector.gpuTensorActive

	ch <- collector.gpuFP16Ops

	ch <- collector.gpuFP32Ops

	ch <- collector.gpuFP64Ops

	ch <- collector.gpuWriteSize

	ch <- collector.gpuReadSize
}

// Collect implements required collect function for all promehteus collectors.
func (collector *amdDeviceMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	for _, dev := range collector.devices {
		ch <- prometheus.MustNewConstMetric(
			collector.gpuUtil, prometheus.GaugeValue, 100*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuVRAMTotal, prometheus.GaugeValue, 1024*1024*1024*24, dev.ID,
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuVRAMUsed, prometheus.GaugeValue, 1024*1024*1024*24*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuGTTTotal, prometheus.GaugeValue, 1024*1024*24, dev.ID,
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuGTTUsed, prometheus.GaugeValue, 1024*1024*24*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuVisibleRAMTotal, prometheus.GaugeValue, 1024*1024*24, dev.ID,
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuVisibleRAMUsed, prometheus.GaugeValue, 1024*1024*24*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		var power float64 = 0
		if dev.IID == "0" {
			power = randFloat(minAMDGPUPower, maxAMDGPUPower)
		}

		ch <- prometheus.MustNewConstMetric(
			collector.gpuPower, prometheus.GaugeValue, power, dev.ID,
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuSMActive, prometheus.GaugeValue, 100*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuTensorActive, prometheus.GaugeValue, 100*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuProfOccupancy, prometheus.GaugeValue, 100*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuFP64Ops, prometheus.GaugeValue, 1e8*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuFP32Ops, prometheus.GaugeValue, 1e6*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuFP16Ops, prometheus.GaugeValue, 1e3*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuWriteSize, prometheus.GaugeValue, 5e6*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			collector.gpuReadSize, prometheus.GaugeValue, 3e6*rand.Float64(), dev.ID, //nolint:gosec
			dev.IID, dev.UUID,
		)
	}
}

func dcgmExporter(ctx context.Context) {
	dcgm := newDCGMCollector()
	dcgmRegistry := prometheus.NewRegistry()
	dcgmRegistry.MustRegister(dcgm)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", promhttp.HandlerFor(dcgmRegistry, promhttp.HandlerOpts{}).ServeHTTP)

	// Start server
	server := &http.Server{
		Addr:              ":9400",
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           mux,
	}

	defer func() {
		err := server.Shutdown(ctx)
		if err != nil {
			log.Println("Failed to shutdown fake NVIDIA DCGM exporter server", err)
		}
	}()

	// Spinning up the server.
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func amdSMIExporter(ctx context.Context) {
	amdSMI := newAMDSMICollector()
	amdRegistry := prometheus.NewRegistry()
	amdRegistry.MustRegister(amdSMI)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", promhttp.HandlerFor(amdRegistry, promhttp.HandlerOpts{}).ServeHTTP)

	// Start server
	server := &http.Server{
		Addr:              ":9500",
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           mux,
	}

	defer func() {
		err := server.Shutdown(ctx)
		if err != nil {
			log.Println("Failed to shutdown fake AMD SMI exporter server", err)
		}
	}()

	// Spinning up the server.
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func amdDeviceMetricsExporter(ctx context.Context, prefix string) {
	amdDeviceMetrics := newAMDDeviceMetricsCollector(prefix)
	amdDeviceMetricsRegistry := prometheus.NewRegistry()
	amdDeviceMetricsRegistry.MustRegister(amdDeviceMetrics)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", promhttp.HandlerFor(amdDeviceMetricsRegistry, promhttp.HandlerOpts{}).ServeHTTP)

	// Start server
	server := &http.Server{
		Addr:              ":9600",
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           mux,
	}

	defer func() {
		err := server.Shutdown(ctx)
		if err != nil {
			log.Println("Failed to shutdown fake AMD device metrics exporter server", err)
		}
	}()

	// Spinning up the server.
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.Println("Starting fake exporters")

	args := os.Args[1:]

	// Registering our handler functions, and creating paths.
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)

	// For e2e tests use constant power usage for reproducibility
	if slices.Contains(args, "test-mode") {
		minNvGPUPower = 200.0
		maxNvGPUPower = 200.0
		minAMDGPUPower = 100.0
		maxAMDGPUPower = 100.0
	} else {
		minNvGPUPower = 60.0
		maxNvGPUPower = 700.0
		minAMDGPUPower = 30.0
		maxAMDGPUPower = 500.0
	}

	if slices.Contains(args, "dcgm") {
		go func() {
			dcgmExporter(ctx)
		}()
	}

	if slices.Contains(args, "amd-smi") {
		go func() {
			amdSMIExporter(ctx)
		}()
	}

	if slices.Contains(args, "amd-device-metrics") {
		go func() {
			amdDeviceMetricsExporter(ctx, "amd_")
		}()
	}

	sig := <-sigs
	log.Println(sig)

	cancel()

	log.Println("Fake exporters have been stopped")
}
