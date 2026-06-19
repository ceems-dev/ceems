---
sidebar_position: 1
---

# Energy consumption estimation

## Servers

There are different sources to get the power consumption of CPUs and memory on the servers
and each source has different scope. A quick summary of the power consumption sources supported
by CEEMS are as follows:

- [Intel RAPL](https://greencompute.uk/Measurement/RAPL): Supported by most of the Intel
processors and some newer AMD processors. They report power consumption of CPU cores and
in some cases DRAM.
- [BMC](https://www.supermicro.com/en/glossary/baseboard-management-controller): Every
server comes with a Baseboard Management Controller (BMC) embedded into the motherboard
to have an out-of-the-bound access for controlling the server. Power measurement devices
are included into the BMCs which provides the "Total power consumption" of the server
including CPU cores, DRAM, motherboard, Network Interface Controllers (NICs), storage
devices, PCIe, and any other peripheral components. Depending on the vendor and server
model, sometimes BMCs power measurement include power consumption of GPUs as well. These
power measurements are exposed via [IPMI DCMI](https://www.intel.com/content/dam/www/public/us/en/documents/technical-specifications/dcmi-v1-5-rev-spec.pdf)
or [Redfish](https://www.dmtf.org/standards/redfish) which are supported by CEEMS.
- [Cray PMC](https://support.hpe.com/hpesc/public/docDisplay?docId=dp00007635en_us&page=common/PM_Counters.html): Cray
supercomputers expose Performance Monitoring Counters which include power consumption of
several components of the server like CPU cores, memory, accelerators and total
power consumption. They are exposed in-band in the `/sys` file system like RAPL counters
which are very convenient to read.
- [HWMon](https://www.kernel.org/doc/Documentation/hwmon/sysfs-interface): Certain models,
especially with ARM processors, expose power measurements using HWMon interface. Like
RAPL and Cray PMC, the power and energy reported by these counters are for individual
components like sockets and it is up to the vendor to implement it for different components
of the server.

As it is clear from the above summary, different power sources on the servers have different
scopes and it is important to normalize the counters from different sources before
estimating power consumption of the individual compute units. As stated above, power
consumption from BMC is the most reliable option as every server comes with BMC
equipped. The downside of BMC measurements is that they are not very powerful and it is
not advisable to flood BMC with sub-second power measurement calls. However, CEEMS is
meant to be a monitoring solution and sub-second scrapes is unrealistic and hence, the
limitation of BMC measurements has smaller impact here.

### When BMC and RAPL counters are available

For instance, when both RAPL CPU and DRAM energy counters are available, the power usage
of each individual workload is estimated using the following approach: As most of the
modern servers are diskless, the power consumption of local storage is assumed to be negligible.
Furthermore, 90% of the total energy consumption is attributed to CPU and DRAM and rest
of 10% to other peripheral components based on
[survey study on compute servers](https://ieeexplore.ieee.org/document/7279063).

At node level, power consumed by CPU and DRAM can be estimated as:

Total CPU Power = 0.9 * Total Power * (RAPL Package / (RAPL Package + RAPL DRAM))
Total DRAM Power = 0.9 * Total Power * (RAPL DRAM / (RAPL Package + RAPL DRAM))

The above formulae are self-explanatory where 90% of total power consumption is split
among CPU cores and DRAM based on RAPL package and DRAM counters.

Now we have power usage at node level for CPU and DRAM. We split it further at the
individual workload level using CPU time and DRAM usage by the compute unit. For rest of
of the power usage like network, storage, we split it equally among all workloads
that are running on the node at a given time.

Compute Unit CPU Power = Total CPU Power * (Compute Unit CPU Time / Total Node CPU Time)
Compute Unit Memory Power = Total DRAM Power * (Compute Unit Memory / Total Node Memory)
Misc Power Usage by Compute Unit = 0.1 * Total Power / Number of Compute Units
Total Compute Unit Host Power = Compute Unit CPU Power + Compute Unit Memory Power + Misc Power Usage by Compute Unit

### When Cray PMC are available

As Cray PMC provides power consumption of CPU cores, DRAM and total power consumption, there is
no need to make any assumptions to split the total power consumption. The consumption of
other components are estimated subtracting the power consumption of CPU cores and DRAM
from the total power consumption. Once we have total CPU, DRAM and rest of power consumptions,
we estimate the power consumption of individual compute units using the same approach as
above using CPU time and DRAM of individual compute units and total node CPU time and DRAM.

More details on how power estimation is done for different sources can be consulted
in the [repository](https://github.com/@ceemsOrg@/@ceemsRepo@/tree/main/etc/prometheus).

## GPU

CEEMS leverages [dcgm-exporter](https://github.com/NVIDIA/dcgm-exporter) and
[amd-smi-exporter](https://github.com/amd/amd_smi_exporter) to get power
consumption of GPUs. When the resource manager uses full physical GPU, estimating
the power consumption of each compute unit is straightforward as CEEMS exporter
already exports a metric that maps the compute unit ID to GPU ordinal. However,
NVIDIA GPUs support sharing of one physical GPU among different compute units
using Multi Instance GPU (MIG) and GRID vGPU strategies. Currently, `dcgm-exporter`
does not estimate power consumption of each MIG instance or vGPU. Thus, CEEMS uses the
following approximation to estimate power consumption of shared GPU instances.

### MIG

CEEMS exporter uses an approximation based on number of
Streaming Multiprocessors (SM) for each
MIG instance profile. For instance, in a typical A100 40GB card, a full GPU can be
split into following profiles:

```bash
$ nvidia-smi mig -lgi
+----------------------------------------------------+
| GPU instances:                                     |
| GPU   Name          Profile  Instance   Placement  |
|                       ID       ID       Start:Size |
|====================================================|
|   0  MIG 1g.5gb       19       13          6:1     |
+----------------------------------------------------+
|   0  MIG 2g.10gb      14        5          4:2     |
+----------------------------------------------------+
|   0  MIG 4g.20gb       5        1          0:4     |
+----------------------------------------------------+
```

It means MIG instance `4g.20gb` has 4/7 of SMs, `2g.10gb` has 2/7 of SMs and `1g.5gb` has
1/7 of SMs. Consequently, the power consumed by entire GPU is divided amongst the different
MIG instances in the ratio of their SMs and their usage respectively. For example, if the physical GPU's
power consumption is `P` Watts, and if there are two other compute units using instance `1g.5gb` and `4g.20gb`,
the power consumption of each MIG profile will be estimated as follows:

- `1g.5gb`: P * (1 * SM_USAGE(`1g.5gb`) * SM_OCC(`1g.5gb`))/((1 * SM_USAGE(`1g.5gb`) * SM_OCC(`1g.5gb`) + (4 * SM_USAGE(`4g.20gb`) * SM_OCC(`4g.20gb`))))
- `4g.20gb`: P * (4 * SM_USAGE(`4g.20gb`) * SM_OCC(`4g.20gb`))/((1 * SM_USAGE(`1g.5gb`) * SM_OCC(`1g.5gb`) + (4 * SM_USAGE(`4g.20gb`) * SM_OCC(`4g.20gb`))))

where `SM_USAGE` and `SM_OCC` are SMs usage and occupancy, respectively. The above formula
saying that the total power usage of each instance is the effective usage of SMs by one instance
relative to all the instances that are being used at a given time.

The exporter will export the coefficient for each MIG instance which can be used along with
power consumption metric of `dcgm-exporter` to estimate power consumption of individual MIG
instances.

### vGPU

In the case of Libvirt, besides MIG it supports GRID vGPU time sharing. The following scenarios
are possible when GPUs are present on the compute node:

- PCI pass through of NVIDIA and AMD GPUs to the guest VMs
- Virtualization of full GPUs using NVIDIA Grid vGPU
- Virtualization of MIG GPUs using NVIDIA Grid vGPU

If a GPU is added to VM using PCI pass through, this GPU will not be available
for the hypervisor and hence, it cannot be queried or monitored. This is due to
the fact that the GPU will be unbound from the hypervisor and bound to guest.
Thus, energy consumption and GPU metrics for GPUs using PCI passthrough
**will only be available in the guest**.

NVIDIA's vGPU uses mediated devices to expose GPUs in the guest and thus,
GPUs can be queried and monitored from both hypervisor and guest. However,
CEEMS rely on [dcgm-exporter](https://github.com/NVIDIA/dcgm-exporter) to
export GPU energy consumption and usage metrics and it does not support
usage and energy consumption metrics for vGPUs. Thus, CEEMS exporter uses
the following approximation method to estimate energy consumption of each
vGPU which in-turn gives energy consumption of each guest VM.

NVIDIA Grid vGPU time slicing divides the GPU resources equally among all the
active vGPUs at any given time and schedule the work on the given physical
GPU. Thus, if there are 4 vGPUs active on a given physical GPU, each vGPU
will get 25% of the full GPU compute power. Thus, a reasonable approximation
would be to split the current physical GPU power consumption equally among all
vGPUs. The same applies when using vGPU on the top of MIG partition. MIG
already divides the physical GPU into different profiles by assigning a given
number of SMs for each profile as discussed in SLURM collector above. When
multiple vGPUs are running on the top of MIG instance, this coefficient is
further divided by number of active vGPUs. For instance, if there are 4 vGPUs
scheduled on a MIG profile `4g.20gb`, the power estimated by the above formula
will be further divided by 4 to attribute the power usage by each VM.
