# Prometheus config

The [rules](https://github.com/ceems-dev/ceems/tree/main/etc/prometheus/rules)
directory contain sample recording rules files that can be used to estimate the
energy and emissions time series metrics of each compute unit derived from different
sources.

## Using `ceems_tool`

There is a utility tool `ceems_tool` that can be used to generate recording rules
from Prometheus server once the CEEMS exporter targets have been configured and
being scrapped by Prometheus successfully. We recommend to this tool to generate
recording rules for your deployment. More details on how to generate rules can be
found in [docs](https://ceems-dev.github.io/ceems/docs/usage/ceems-tool).

> [!IMPORTANT]
> ALWAYS USE `evaluation_interval` THAT IS SAME AS THE `scrape_interval` FOR BEST RESULTS.

## Rules

The following recording rules files are provided for reference purposes and `ceems_tool`
must be preferred to generate recording rules.

### [`host-usage.rules`](./rules/host-usage.rules)

The rules in this file estimate the host CPU and CPU memory usage for each
compute unit and also average usage aggregated over Prometheus jobs.

### [`io-usage.rules`](./rules/io-usage.rules)

The rules in this file estimate the I/O read/write bandwidths for each
compute unit and also total usage aggregated over Prometheus jobs. These
metrics are available only when
[eBPF](https://ceems-dev.github.io/ceems/docs/components/ceems-exporter#ebpf-sub-collector)
collector is enabled on CEEMS exporter.

### [`network-usage.rules`](./rules/network-usage.rules)

The rules in this file estimate the network ingress and egress bandwidths for each
compute unit and also total usage aggregated over Prometheus jobs. These
metrics are available only when
[eBPF](https://ceems-dev.github.io/ceems/docs/components/ceems-exporter#ebpf-sub-collector)
collector is enabled on CEEMS exporter.

### [`host-power-cray-pmc.rules`](./rules/host-power-cray-pmc.rules)

The rules defined in this file estimate host power usage for the nodes where Cray
PM counters are available. The rules make the following assumptions:

- Total host power is reported by Cray PM counters.

The provided rules estimate the power usage of individual compute units and total
power usage of nodes aggregated over Prometheus jobs.

### [`host-power-redfish.rules`](./rules/host-power-redfish.rules)

The rules defined in this file estimate host power usage for the nodes where Redfish
reports power usage. The rules make the following assumptions:

- Total host power is reported by Redfish. Chassis that reports host power usage
must be used.

The provided rules estimate the power usage of individual compute units and total
power usage of nodes aggregated over Prometheus jobs.

### [`host-power-hwmon.rules`](./rules/host-power-hwmon.rules)

The rules defined in this file estimate host power usage for the nodes where HWMon
counters report power usage. The rules make the following assumptions:

- Total host power is reported by HWMon counters. Chip that reports host power usage
must be used.

The provided rules estimate the power usage of individual compute units and total
power usage of nodes aggregated over Prometheus jobs.

### [`host-power-ipmi.rules`](./rules/host-power-ipmi.rules)

The rules defined in this file estimate host power usage for the nodes where in-band
IPMI DCMI reports power usage. The rules make the following assumptions:

- Total host power is reported by in-band IPMI DCMI. If power usage reported by
IPMI DCMI includes GPU power usage (if present on the node), appropriate rule
must be selected from the rule file to exclude GPU power usage

The provided rules estimate the power usage of individual compute units and total
power usage of nodes aggregated over Prometheus jobs.

### [`host-power-rapl.rules`](./rules/host-power-rapl.rules)

The rules defined in this file estimate host power usage for the nodes that exposes
only RAPL counters to get host power usage. The rules make the following assumptions:

- RAPL counters are available for both CPU and DRAM packages

The provided rules estimate the power usage of individual compute units and total
power usage of nodes aggregated over Prometheus jobs.

<!-- ### `host-power-ipmi-with-nvidia-gpus.rules`

The rules defined in this file are meant to be used for group of nodes have NVIDIA
GPUs and host power reported by IPMI DCMI **includes** GPUs power usage.
The rules make the following assumptions:

- Total host power (with GPUs power usage) is reported by IPMI DCMI
- RAPL counters are **not available** for the host

As power usage reported by IPMI DCMI contains both host and GPU, we need to remove
power usage by GPU to get the power usage by host alone. To do so, we leverage the
power usage reported by [NVIDIA DCGM exporter](https://github.com/NVIDIA/dcgm-exporter).

The provided rules estimate the power usage of individual compute units based on
compute unit CPU and total node's CPU usage. More details
are provided in the comments of the rules file.

### `host-power-redfish-with-amd-gpus.rules`

The rules defined in this file are meant to be used for group of nodes have AMD
GPUs and host power reported by Redfish **includes** GPUs power usage.
The rules make the following assumptions:

- Total host power (with GPUs power usage) is reported by Redfish
- RAPL counters are available for both CPU and DRAM packages

As power usage reported by IPMI DCMI contains both host and GPU, we need to remove
power usage by GPU to get the power usage by host alone. To do so, we leverage the
power usage reported by [AMD SMI exporter](https://github.com/amd/amd_smi_exporter).

The provided rules estimate the power usage of individual compute units based on
compute unit CPU and total node's CPU usage. More details
are provided in the comments of the rules file. -->

### [`nvidia-dcgm-gpu.rules`](./rules/nvidia-dcgm-gpu.rules)

The rules defined in this file estimate GPU usage, power usage and profiling metrics
for the nodes that have NVIDIA GPUs monitored by
[NVIDIA DCGM exporter](https://github.com/NVIDIA/dcgm-exporter).

The provided rules estimate different GPU metrics of individual compute units and sum/average
over all nodes aggregated over Prometheus jobs.

### [`amd-device-metrics-gpu.rules`](./rules/amd-device-metrics-gpu.rules)

The rules defined in this file estimate GPU usage, power usage and profiling metrics
for the nodes that have AMD GPUs monitored by
[AMD Device metrics exporter](https://github.com/ROCm/device-metrics-exporter).

The provided rules estimate different GPU metrics of individual compute units and sum/average
over all nodes aggregated over Prometheus jobs.

### [`amd-smi-gpu.rules`](./rules/amd-smi-gpu.rules)

The rules defined in this file estimate GPU usage and power usage
for the nodes that have AMD GPUs monitored by
[AMD SMI exporter](https://github.com/amd/amd_smi_exporter).

The provided rules estimate different GPU metrics of individual compute units and sum/average
over all nodes aggregated over Prometheus jobs.

## Installing rules

<!-- The rules files must be modified appropriately by using correct job names and installed
to Prometheus deployment. For instance, imagine a target cluster can be grouped as follows:

- `cpu-partition-1`: A group of nodes with only CPUs
- `cpu-partition-2`: Another group of nodes with only CPUs
- `v100-partition-1`: A group of nodes with V100 GPUs
- `a100-partition-1`: A group of nodes with A100 GPUs

And operators defined a prometheus job for each group using the same names as used above.
CEEMS exporter must be deployed on all the nodes and
[NVIDIA DCGM exporter](https://github.com/NVIDIA/dcgm-exporter) on groups `v100-partition-1`
and `a100-partition-1`. Assume DCGM targets are placed in Prometheus job with `dcgm-` as suffix
to the group name. For example, DCGM targets on group `v100-partition-1` will be in a job
`dcgm-v100-partition-1`. Moreover imagine that the IPMI DCMI reports only CPU power usage
for the group `v100-partition-1` where as it reports both CPU and GPU for the group
`a100-partition-1`. In this case, rules files can be generated as follows:

```bash
# Create a folder to keep all created rules files
mkdir -p cluster_rules

# Create rules files for cpu-partition-1 and cpu-partition-2
sed 's/<sample-cpu>/<cpu-partition-1>/g' cpu-only-nodes.rules > cluster_rules/cpu-partition-1.rules
sed 's/<sample-cpu>/<cpu-partition-2>/g' cpu-only-nodes.rules > cluster_rules/cpu-partition-2.rules

# Create rules files for v100-partition-1
sed 's/<sample-gpu>/<v100-partition-1>/g' cpu-only-nodes.rules > cluster_rules/v100-partition-1.rules
sed 's/<sample-dcgm>/<dcgm-v100-partition-1>/g' gpu.rules > cluster_rules/dcgm-v100-partition-1.rules

# Create rules files for a100-partition-1
sed -e 's/<sample-gpu>/<a100-partition-1>/g' -e 's/<sample-dcgm>/<dcgm-a100-partition-1>/g' cpu-gpu-nodes.rules > cluster_rules/a100-partition-1.rules
sed 's/<sample-dcgm>/<dcgm-a100-partition-1>/g' gpu.rules > cluster_rules/dcgm-a100-partition-1.rules
``` -->

After generating rules using `ceems_tool` or appropriately modifying the references rule files
provided in this repository, we need to make sure they are valid. This can be done using
[`promtool`](https://prometheus.io/docs/prometheus/latest/command-line/promtool/). Assuming generated
rule files are placed in `myrules` folder:

```bash
find myrules -name "*.rules" | xargs -I {} promtool check rules {}
```

Finally, all the rules files must be placed under the folder provided to `rules_files` key
in Prometheus [config](https://prometheus.io/docs/prometheus/latest/configuration/configuration/).

Once the rules have been installed, restart/reload Prometheus.
