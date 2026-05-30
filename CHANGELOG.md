# Changelog

## 0.13.0 / 2026-05-30

- [CI] ci: Sign binaries and container images using cosign [#514](https://github.com/ceems-dev/ceems/pull/514) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] chore: Make lack of RAPL power limit error logs debug [#512](https://github.com/ceems-dev/ceems/pull/512) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] chore: Update Grafana dashboard JSON models [#504](https://github.com/ceems-dev/ceems/pull/504) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] fix: Ensure nvidia power metric is detected well for generating rules [#496](https://github.com/ceems-dev/ceems/pull/496) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] feat: Support LSF job arrays [#491](https://github.com/ceems-dev/ceems/pull/491) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Add LSF collector [#488](https://github.com/ceems-dev/ceems/pull/488) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#469](https://github.com/ceems-dev/ceems/pull/469), [#470](https://github.com/ceems-dev/ceems/pull/470), [#471](https://github.com/ceems-dev/ceems/pull/471), [#472](https://github.com/ceems-dev/ceems/pull/472), [#473](https://github.com/ceems-dev/ceems/pull/473), [#474](https://github.com/ceems-dev/ceems/pull/474), [#475](https://github.com/ceems-dev/ceems/pull/475), [#477](https://github.com/ceems-dev/ceems/pull/477), [#478](https://github.com/ceems-dev/ceems/pull/478), [#479](https://github.com/ceems-dev/ceems/pull/479), [#480](https://github.com/ceems-dev/ceems/pull/480), [#481](https://github.com/ceems-dev/ceems/pull/481), [#483](https://github.com/ceems-dev/ceems/pull/483), [#484](https://github.com/ceems-dev/ceems/pull/484), [#485](https://github.com/ceems-dev/ceems/pull/485), [#486](https://github.com/ceems-dev/ceems/pull/486), [#487](https://github.com/ceems-dev/ceems/pull/487), [#489](https://github.com/ceems-dev/ceems/pull/489), [#490](https://github.com/ceems-dev/ceems/pull/490), [#492](https://github.com/ceems-dev/ceems/pull/492), [#493](https://github.com/ceems-dev/ceems/pull/493), [#494](https://github.com/ceems-dev/ceems/pull/494), [#495](https://github.com/ceems-dev/ceems/pull/495), [#497](https://github.com/ceems-dev/ceems/pull/497), [#498](https://github.com/ceems-dev/ceems/pull/498), [#499](https://github.com/ceems-dev/ceems/pull/499), [#500](https://github.com/ceems-dev/ceems/pull/500), [#501](https://github.com/ceems-dev/ceems/pull/501), [#502](https://github.com/ceems-dev/ceems/pull/502), [#503](https://github.com/ceems-dev/ceems/pull/503), [#505](https://github.com/ceems-dev/ceems/pull/505), [#506](https://github.com/ceems-dev/ceems/pull/506), [#507](https://github.com/ceems-dev/ceems/pull/507), [#508](https://github.com/ceems-dev/ceems/pull/508), [#509](https://github.com/ceems-dev/ceems/pull/509), [#511](https://github.com/ceems-dev/ceems/pull/511) ([@dependabot](https://github.com/dependabot))

## 0.12.3 / 2026-02-25

- [BUGFIX] fix(emissions): update eMapsZonesResponse to match current Electricity Maps API [#468](https://github.com/ceems-dev/ceems/pull/468) ([@samoz83](https://github.com/samoz83))
- [MAINT] Bump dependencies [#459](https://github.com/ceems-dev/ceems/pull/459), [#460](https://github.com/ceems-dev/ceems/pull/460), [#461](https://github.com/ceems-dev/ceems/pull/461), [#462](https://github.com/ceems-dev/ceems/pull/462), [#463](https://github.com/ceems-dev/ceems/pull/463), [#464](https://github.com/ceems-dev/ceems/pull/464), [#465](https://github.com/ceems-dev/ceems/pull/465), [#466](https://github.com/ceems-dev/ceems/pull/466) ([@dependabot](https://github.com/dependabot))

## 0.12.2 / 2026-01-19

- [BUGFIX] fix(collector): add index label to hwmon metrics to distinguish devices [#456](https://github.com/ceems-dev/ceems/pull/456) ([@samoz83](https://github.com/samoz83))
- [MAINT] Bump dependencies [#452](https://github.com/ceems-dev/ceems/pull/452), [#453](https://github.com/ceems-dev/ceems/pull/453), [#454](https://github.com/ceems-dev/ceems/pull/454), [#457](https://github.com/ceems-dev/ceems/pull/457), [#458](https://github.com/ceems-dev/ceems/pull/458) ([@dependabot](https://github.com/dependabot))

## 0.12.1 / 2025-12-22

- [BUGFIX] Check group ownership of files while dropping privileges [#451](https://github.com/ceems-dev/ceems/pull/451) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#437](https://github.com/ceems-dev/ceems/pull/437), [#438](https://github.com/ceems-dev/ceems/pull/438), [#439](https://github.com/ceems-dev/ceems/pull/439), [#440](https://github.com/ceems-dev/ceems/pull/440), [#441](https://github.com/ceems-dev/ceems/pull/441), [#442](https://github.com/ceems-dev/ceems/pull/442), [#443](https://github.com/ceems-dev/ceems/pull/443), [#444](https://github.com/ceems-dev/ceems/pull/444), [#445](https://github.com/ceems-dev/ceems/pull/445), [#446](https://github.com/ceems-dev/ceems/pull/446), [#447](https://github.com/ceems-dev/ceems/pull/447), [#448](https://github.com/ceems-dev/ceems/pull/448), [#449](https://github.com/ceems-dev/ceems/pull/449), [#450](https://github.com/ceems-dev/ceems/pull/450) ([@dependabot](https://github.com/dependabot))

## 0.12.0 / 2025-11-08

- [FEAT] System level logging for `cacct` [#436](https://github.com/ceems-dev/ceems/pull/436) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] bug(fix): Use configured time zone when SLURM does not include time offsets [#433](https://github.com/ceems-dev/ceems/pull/433) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Make multiple update calls to eBPF coll in unit tests [#424](https://github.com/ceems-dev/ceems/pull/424) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] chore: Update go to 1.25.x [#420](https://github.com/ceems-dev/ceems/pull/420) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Always return error responses in JSON for CEEMS LB [#414](https://github.com/ceems-dev/ceems/pull/414) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#412](https://github.com/ceems-dev/ceems/pull/412), [#415](https://github.com/ceems-dev/ceems/pull/415), [#416](https://github.com/ceems-dev/ceems/pull/416), [#417](https://github.com/ceems-dev/ceems/pull/417), [#419](https://github.com/ceems-dev/ceems/pull/419), [#421](https://github.com/ceems-dev/ceems/pull/421), [#422](https://github.com/ceems-dev/ceems/pull/422), [#423](https://github.com/ceems-dev/ceems/pull/423), [#425](https://github.com/ceems-dev/ceems/pull/425), [#426](https://github.com/ceems-dev/ceems/pull/426), [#427](https://github.com/ceems-dev/ceems/pull/427), [#428](https://github.com/ceems-dev/ceems/pull/428), [#429](https://github.com/ceems-dev/ceems/pull/429), [#430](https://github.com/ceems-dev/ceems/pull/430), [#434](https://github.com/ceems-dev/ceems/pull/434) ([@dependabot](https://github.com/dependabot))

## 0.11.2 / 2025-09-12

- [MAINT] Update static emission factors for OWID and world average [#411](https://github.com/ceems-dev/ceems/pull/411) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#410](https://github.com/ceems-dev/ceems/pull/410) ([@dependabot](https://github.com/dependabot))

## 0.11.1 / 2025-09-06

- [BUGFIX] Allow GPU fetching to fail for libvirt collector [#409](https://github.com/ceems-dev/ceems/pull/409) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.11.0 / 2025-09-02

### Breaking Changes

#### CEEMS Exporter

- Collector `rapl` is disabled by default now and to enable it add `--collector.rapl` to CLI arguments.
- Collector `ipmi_dcmi` has been renamed to `ipmi` as more functionality beyond DCMI has been added to the collector.
- Following metric labels have been renamed to be more consistent with Prometheus naming convention:
    * `ceems_ipmi_dcmi_current_watts` -> `ceems_ipmi_dcmi_power_current_watts`
    * `ceems_ipmi_dcmi_min_watts` -> `ceems_ipmi_dcmi_power_min_watts`
    * `ceems_ipmi_dcmi_max_watts` -> `ceems_ipmi_dcmi_power_max_watts`
    * `ceems_ipmi_dcmi_avg_watts` -> `ceems_ipmi_dcmi_power_avg_watts`
    * `ceems_redfish_current_watts` -> `ceems_redfish_power_current_watts`
    * `ceems_redfish_min_watts` -> `ceems_redfish_power_min_watts`
    * `ceems_redfish_max_watts` -> `ceems_redfish_power_max_watts`
    * `ceems_redfish_avg_watts` -> `ceems_redfish_power_avg_watts`

#### CEEMS tool

- The relabel configs generated by subcommand `create-relabel-configs` are obsolete as the relabelling of metrics directly handled inside the recording rules. Please
regenerate recording rules with new version and remove existing relabel configs on Prometheus server.
- Several minor bugs in recording rules have been fixed. Please regenerate the recording rules with new version of `ceems_tool`.
- GPU profiling metrics have been renamed to have `prof` in the metric label. For instance, `uuid:ceems_gpu_sm_active:ratio` became
`uuid:ceems_gpu_prof_sm_active:ratio`.
- NVIDIA profiling metrics suffix has been corrected to use `sum` instead of `ratio` for NVLink, PCIe traffic metrics. Thus, metrics
have been renamed as follows:
    * `uuid:ceems_gpu_pcie_tx_bytes:ratio` -> `uuid:ceems_gpu_prof_pcie_tx_bytes:sum`
    * `uuid:ceems_gpu_pcie_rx_bytes:ratio` -> `uuid:ceems_gpu_prof_pcie_rx_bytes:sum`
    * `uuid:ceems_gpu_nvlink_tx_bytes:ratio` -> `uuid:ceems_gpu_prof_nvlink_tx_bytes:sum`
    * `uuid:ceems_gpu_nvlink_rx_bytes:ratio` -> `uuid:ceems_gpu_prof_nvlink_rx_bytes:sum`

### List of PRs

- [FEAT] Add rules for IO and network metrics [#406](https://github.com/ceems-dev/ceems/pull/406) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Support runtime XML directory for libvirt collector [#404](https://github.com/ceems-dev/ceems/pull/404) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump golanglint-ci to 2.4 [#399](https://github.com/ceems-dev/ceems/pull/399) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BREAKING] Updates and fixes to recording rules subcommand of `ceems_tool` [#397](https://github.com/ceems-dev/ceems/pull/397) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BREAKING] Support exporting metrics of IPMI sensors [#395](https://github.com/ceems-dev/ceems/pull/395) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#394](https://github.com/ceems-dev/ceems/pull/394), [#398](https://github.com/ceems-dev/ceems/pull/398), [#400](https://github.com/ceems-dev/ceems/pull/400), [#405](https://github.com/ceems-dev/ceems/pull/405), [#407](https://github.com/ceems-dev/ceems/pull/407), [#408](https://github.com/ceems-dev/ceems/pull/408) ([@dependabot](https://github.com/dependabot))

## 0.10.2 / 2025-08-07

- [BUGFIX] Fix bpf code to work with LLVM 20 [#393](https://github.com/ceems-dev/ceems/pull/393) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Fix k8s resource manager [#392](https://github.com/ceems-dev/ceems/pull/392) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#389](https://github.com/ceems-dev/ceems/pull/389), [#390](https://github.com/ceems-dev/ceems/pull/390), [#387](https://github.com/ceems-dev/ceems/pull/387) ([@dependabot](https://github.com/dependabot))

## 0.10.1 / 2025-07-22

- [BUGFIX] Fix parsing nvidia-smi XML output [#388](https://github.com/ceems-dev/ceems/pull/388) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#387](https://github.com/ceems-dev/ceems/pull/387) ([@dependabot](https://github.com/dependabot))

## 0.10.0 / 2025-07-20

- [CI] Free up disk space for crossbuild jobs [#386](https://github.com/ceems-dev/ceems/pull/386) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Add CONTRIBUTING.md file [#385](https://github.com/ceems-dev/ceems/pull/385) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Migrate repo to ceems-dev org [#384](https://github.com/ceems-dev/ceems/pull/384) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Filter SLURM cgroups to remove stale ones [#382](https://github.com/ceems-dev/ceems/pull/382) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] K8s support for CEEMS API server [#381](https://github.com/ceems-dev/ceems/pull/381) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Add systemd-less mode for Libvirt collector [#377](https://github.com/ceems-dev/ceems/pull/377) ([@wtripp180901](https://github.com/wtripp180901))
- [MAINT] Bump dependencies [#375](https://github.com/ceems-dev/ceems/pull/375), [#376](https://github.com/ceems-dev/ceems/pull/376), [#378](https://github.com/ceems-dev/ceems/pull/378), [#383](https://github.com/ceems-dev/ceems/pull/383) ([@dependabot](https://github.com/dependabot))

## 0.9.1 / 2025-07-02

- [FEAT] Support gzip compression [#374](https://github.com/ceems-dev/ceems/pull/374) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#372](https://github.com/ceems-dev/ceems/pull/372), [#373](https://github.com/ceems-dev/ceems/pull/373) ([@dependabot](https://github.com/dependabot))

## 0.9.0 / 2025-06-27

### Breaking Changes

#### CEEMS LB

- Undocumented Resource-based LB strategy has been removed. Deployments using this strategy must use Prometheus' [remote read](https://prometheus.io/docs/prometheus/latest/querying/remote_read_api/) feature to achieve the same functionality.

#### CEEMS Exporter

- The configuration of Redfish collector must be under the section `redfish_collector` instead of `redfish_web`. More details in [docs](https://mahendrapaipuri.github.io/ceems/docs/configuration/ceems-exporter#redfish-collector).
- CLI flag `--collector.redfish.web-config` has been deprecated in the favour of `--collector.redfish.config.file`.
- CLI flag `--collector.k8s.kube-config-file` has been deprecated in the favour of `--collector.k8s.kubeconfig.file`.
- CLI flag `--collector.k8s.kubelet-socket-file` has been deprecated in the favour of `--collector.k8s.kubelet-podresources-socket.file`.

#### Redfish Proxy

- The configuration of Redfish proxy must be under `redfish_proxy` instead of `redfish_proxy.web`. More details in [docs](https://mahendrapaipuri.github.io/ceems/docs/configuration/ceems-exporter#redfish-collector).

### List of PRs

- [FEAT] Support env vars in config files [#369](https://github.com/ceems-dev/ceems/pull/369) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Add k8s admission controller [#367](https://github.com/ceems-dev/ceems/pull/367) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] refactor: Rename config section names to be consistent across package [#364](https://github.com/ceems-dev/ceems/pull/364) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BREAKING] breaking: Remove resource-based LB strategy [#361](https://github.com/ceems-dev/ceems/pull/361) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Native eBPF profiler [#360](https://github.com/ceems-dev/ceems/pull/360) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#359](https://github.com/ceems-dev/ceems/pull/359), [#362](https://github.com/ceems-dev/ceems/pull/363), [#365](https://github.com/ceems-dev/ceems/pull/365), [#366](https://github.com/ceems-dev/ceems/pull/366), [#368](https://github.com/ceems-dev/ceems/pull/368), [#371](https://github.com/ceems-dev/ceems/pull/371) ([@dependabot](https://github.com/dependabot))

## 0.8.0 / 2025-05-20

- [FEAT] Harden redfish proxy app [#357](https://github.com/ceems-dev/ceems/pull/357) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Several maintenance changes [#354](https://github.com/ceems-dev/ceems/pull/354) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Add k8s collector in the exporter [#349](https://github.com/ceems-dev/ceems/pull/349) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#345](https://github.com/ceems-dev/ceems/pull/345), [#346](https://github.com/ceems-dev/ceems/pull/346), [#347](https://github.com/ceems-dev/ceems/pull/347), [#348](https://github.com/ceems-dev/ceems/pull/348), [#351](https://github.com/ceems-dev/ceems/pull/351), [#353](https://github.com/ceems-dev/ceems/pull/353), [#355](https://github.com/ceems-dev/ceems/pull/355), [#356](https://github.com/ceems-dev/ceems/pull/356), [#358](https://github.com/ceems-dev/ceems/pull/358) ([@dependabot](https://github.com/dependabot))

## 0.7.2 / 2025-04-19

- [FEAT] Make redfish timeout a configurable value [#342](https://github.com/ceems-dev/ceems/pull/342) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] docs: fix typos and improve consistency [#339](https://github.com/ceems-dev/ceems/pull/339) ([@ncreddine](https://github.com/ncreddine))
- [MAINT] Better usage of bpf LRU hash maps [#335](https://github.com/ceems-dev/ceems/pull/335) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#331](https://github.com/ceems-dev/ceems/pull/331), [#332](https://github.com/ceems-dev/ceems/pull/332), [#334](https://github.com/ceems-dev/ceems/pull/334), [#336](https://github.com/ceems-dev/ceems/pull/336), [#337](https://github.com/ceems-dev/ceems/pull/337), [#338](https://github.com/ceems-dev/ceems/pull/338), [#340](https://github.com/ceems-dev/ceems/pull/340), [#343](https://github.com/ceems-dev/ceems/pull/343), [#344](https://github.com/ceems-dev/ceems/pull/344) ([@dependabot](https://github.com/dependabot))

## 0.7.1 / 2025-03-25

- [MAINT] Minor improvements in power usage collectors [#330](https://github.com/ceems-dev/ceems/pull/330) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Update docusaurus.config.ts [#329](https://github.com/ceems-dev/ceems/pull/329) ([@ncreddine](https://github.com/ncreddine))
- [MAINT] Bump dependencies [#328](https://github.com/ceems-dev/ceems/pull/328), [#331](https://github.com/ceems-dev/ceems/pull/331) ([@dependabot](https://github.com/dependabot))

## 0.7.0 / 2025-03-16

- [FEAT] Add Watttime emission factor [#327](https://github.com/ceems-dev/ceems/pull/327) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] `cacct` client tool  [#321](https://github.com/ceems-dev/ceems/pull/321) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] feat: Add netdev and IB collectors [#310](https://github.com/ceems-dev/ceems/pull/310) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Add hwmon collector [#309](https://github.com/ceems-dev/ceems/pull/309) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#320](https://github.com/ceems-dev/ceems/pull/320), [#322](https://github.com/ceems-dev/ceems/pull/322), [#323](https://github.com/ceems-dev/ceems/pull/323), [#324](https://github.com/ceems-dev/ceems/pull/324), [#325](https://github.com/ceems-dev/ceems/pull/325), [#326](https://github.com/ceems-dev/ceems/pull/326) ([@dependabot](https://github.com/dependabot))

## 0.6.0 / 2025-02-24

- [FEAT] Enhancements for CEEMS API server [#304](https://github.com/ceems-dev/ceems/pull/304) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Support label filtering in CEEMS LB responses [#303](https://github.com/ceems-dev/ceems/pull/303) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Add CLI section in docs [#296](https://github.com/ceems-dev/ceems/pull/296) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Deployment guide and minor improvements [#294](https://github.com/ceems-dev/ceems/pull/294) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Support SLURM multiple daemons [#289](https://github.com/ceems-dev/ceems/pull/289) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Ceems Tooling support [#288](https://github.com/ceems-dev/ceems/pull/288) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#283](https://github.com/ceems-dev/ceems/pull/283), [#285](https://github.com/ceems-dev/ceems/pull/285), [#286](https://github.com/ceems-dev/ceems/pull/286), [#287](https://github.com/ceems-dev/ceems/pull/287), [#290](https://github.com/ceems-dev/ceems/pull/290), [#291](https://github.com/ceems-dev/ceems/pull/291), [#292](https://github.com/ceems-dev/ceems/pull/292), [#300](https://github.com/ceems-dev/ceems/pull/300), [#301](https://github.com/ceems-dev/ceems/pull/301), [#305](https://github.com/ceems-dev/ceems/pull/305), [#306](https://github.com/ceems-dev/ceems/pull/306), [#307](https://github.com/ceems-dev/ceems/pull/307), [#308](https://github.com/ceems-dev/ceems/pull/308) ([@dependabot](https://github.com/dependabot))

## 0.5.3 / 2025-01-24

- [BUGFIX] Minor corrections in SLURM fetcher and TSDB updater [#280](https://github.com/ceems-dev/ceems/pull/280) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Set MIG instance in a separate label, when present [#279](https://github.com/ceems-dev/ceems/pull/279) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] More configurability on tsdb updater's query batching [#277](https://github.com/ceems-dev/ceems/pull/277) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Handle running query parameter correctly [#271](https://github.com/ceems-dev/ceems/pull/271) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] TSDB retention period estimation [#270](https://github.com/ceems-dev/ceems/pull/270) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#273](https://github.com/ceems-dev/ceems/pull/273), [#274](https://github.com/ceems-dev/ceems/pull/274), [#276](https://github.com/ceems-dev/ceems/pull/276), [#278](https://github.com/ceems-dev/ceems/pull/278) ([@dependabot](https://github.com/dependabot))

## 0.5.2 / 2025-01-17

- [BUGFIX] Re-establish session when token invalidates for Redfish collector [#268](https://github.com/ceems-dev/ceems/pull/268) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] TSDB estimate batch size dynamically and update OWID data [#262](https://github.com/ceems-dev/ceems/pull/262) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#264](https://github.com/ceems-dev/ceems/pull/264), [#265](https://github.com/ceems-dev/ceems/pull/265), [#269](https://github.com/ceems-dev/ceems/pull/269) ([@dependabot](https://github.com/dependabot))

## 0.5.1 / 2025-01-08

- [FEATURE] Add Cray's pm_counters collector [#261](https://github.com/ceems-dev/ceems/pull/261) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Use total swap as limit when cgroup sets it as max [#260](https://github.com/ceems-dev/ceems/pull/260) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Configurable Timezone for CEEMS DB [#253](https://github.com/ceems-dev/ceems/pull/253) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Support for Pyroscope servers for CEEMS LB [#252](https://github.com/ceems-dev/ceems/pull/252) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#246](https://github.com/ceems-dev/ceems/pull/246), [#247](https://github.com/ceems-dev/ceems/pull/247), [#248](https://github.com/ceems-dev/ceems/pull/248), [#249](https://github.com/ceems-dev/ceems/pull/249), [#250](https://github.com/ceems-dev/ceems/pull/250), [#251](https://github.com/ceems-dev/ceems/pull/251), [#254](https://github.com/ceems-dev/ceems/pull/254), [#255](https://github.com/ceems-dev/ceems/pull/255), [#256](https://github.com/ceems-dev/ceems/pull/256) ([@dependabot](https://github.com/dependabot))

## 0.5.0 / 2024-12-12

- [BUGFIX] Support IPMI package on 32/64 bit platforms [#245](https://github.com/ceems-dev/ceems/pull/245) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Upgrade Go to 1.23.x [#244](https://github.com/ceems-dev/ceems/pull/244) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Update dockerfile to include redfish_proxy [#243](https://github.com/ceems-dev/ceems/pull/243) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add Redfish Collector [#240](https://github.com/ceems-dev/ceems/pull/240) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Pure go IPMI implementation using OpenIPMI interface [#238](https://github.com/ceems-dev/ceems/pull/238) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Embed demo Grafana in iframe in documentation welcome page [#233](https://github.com/ceems-dev/ceems/pull/233) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Report usage statistics by taking running units into account [#232](https://github.com/ceems-dev/ceems/pull/232) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Support automatic token rotation for Openstack [#227](https://github.com/ceems-dev/ceems/pull/227) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Prioritize SLURM_JOB_GPUS env for GPU mapping [#221](https://github.com/ceems-dev/ceems/pull/221) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Migrate to slog logging [#211](https://github.com/ceems-dev/ceems/pull/211) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Implement correct scaling of perf hardware counters [#210](https://github.com/ceems-dev/ceems/pull/210) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#212](https://github.com/ceems-dev/ceems/pull/212), [#213](https://github.com/ceems-dev/ceems/pull/213), [#215](https://github.com/ceems-dev/ceems/pull/215), [#222](https://github.com/ceems-dev/ceems/pull/222), [#225](https://github.com/ceems-dev/ceems/pull/225), [#226](https://github.com/ceems-dev/ceems/pull/226), [#228](https://github.com/ceems-dev/ceems/pull/228), [#229](https://github.com/ceems-dev/ceems/pull/229), [#236](https://github.com/ceems-dev/ceems/pull/236) ([@dependabot](https://github.com/dependabot)), [#237](https://github.com/ceems-dev/ceems/pull/237), [#241](https://github.com/ceems-dev/ceems/pull/241) ([@dependabot](https://github.com/dependabot)), [#242](https://github.com/ceems-dev/ceems/pull/242) ([@dependabot](https://github.com/dependabot))

## 0.5.0-rc.2 / 2024-10-31

- [BUFGIX] Scale perf counters based on times enabled and ran [#209](https://github.com/ceems-dev/ceems/pull/209) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.5.0-rc.1 / 2024-10-29

- [MAINT] Major refactor to improve performance of exporter [#204](https://github.com/ceems-dev/ceems/pull/204) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#205](https://github.com/ceems-dev/ceems/pull/205), [#206](https://github.com/ceems-dev/ceems/pull/206), [#207](https://github.com/ceems-dev/ceems/pull/207) ([@dependabot](https://github.com/dependabot))

## 0.4.1 / 2024-10-25

- [FEATURE] Use custom header to find target cluster [#203](https://github.com/ceems-dev/ceems/pull/203) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.4.0 / 2024-10-23

- [FEATURE] Add support for HTTP alloy discovery [#198](https://github.com/ceems-dev/ceems/pull/198) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add openstack resource manager support to API server [#196](https://github.com/ceems-dev/ceems/pull/196) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add support for MIG and vGPUs in exporter [#193](https://github.com/ceems-dev/ceems/pull/193) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Export power limit from RAPL counters [#189](https://github.com/ceems-dev/ceems/pull/189) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add libvirt collector [#186](https://github.com/ceems-dev/ceems/pull/186) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add RDMA collector [#182](https://github.com/ceems-dev/ceems/pull/182) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Fix cmd execution mode detection [#181](https://github.com/ceems-dev/ceems/pull/181) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Hide test related CLI flags [#180](https://github.com/ceems-dev/ceems/pull/180) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add ebpf support for mips,ppc and risc archs [#179](https://github.com/ceems-dev/ceems/pull/179) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#183](https://github.com/ceems-dev/ceems/pull/183), [#184](https://github.com/ceems-dev/ceems/pull/184), [#185](https://github.com/ceems-dev/ceems/pull/185), [#192](https://github.com/ceems-dev/ceems/pull/192), [#194](https://github.com/ceems-dev/ceems/pull/194), [#199](https://github.com/ceems-dev/ceems/pull/199), [#200](https://github.com/ceems-dev/ceems/pull/200), [#201](https://github.com/ceems-dev/ceems/pull/201), [#202](https://github.com/ceems-dev/ceems/pull/202) ([@dependabot](https://github.com/dependabot))

## 0.3.1 / 2024-10-03

- [BUGFIX] Fix cmd execution mode detection [#181](https://github.com/ceems-dev/ceems/pull/181) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Hide test related CLI flags [#180](https://github.com/ceems-dev/ceems/pull/180) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEAT] Add ebpf support for mips,ppc and risc archs [#179](https://github.com/ceems-dev/ceems/pull/179) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.3.0 / 2024-09-28

- [CI] Move docs workflow to separate file [#178](https://github.com/ceems-dev/ceems/pull/178) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Verify TSDB actual retention period [#177](https://github.com/ceems-dev/ceems/pull/177) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Make CEEMS apps capability aware [#176](https://github.com/ceems-dev/ceems/pull/176) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Remove unnecessary log lines [#167](https://github.com/ceems-dev/ceems/pull/167) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Refactor slurm collector organization [#155](https://github.com/ceems-dev/ceems/pull/155) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Graceful exporter shutdown and misc fixes [#153](https://github.com/ceems-dev/ceems/pull/153) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Use consistent CLI flags for exporter [#144](https://github.com/ceems-dev/ceems/pull/144) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add perf collector that exports perf metrics [#137](https://github.com/ceems-dev/ceems/pull/137) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Bump dependencies [#138](https://github.com/ceems-dev/ceems/pull/138), [#139](https://github.com/ceems-dev/ceems/pull/139), [#140](https://github.com/ceems-dev/ceems/pull/140), [#141](https://github.com/ceems-dev/ceems/pull/141), [#142](https://github.com/ceems-dev/ceems/pull/142), [#143](https://github.com/ceems-dev/ceems/pull/143), [#145](https://github.com/ceems-dev/ceems/pull/145), [#146](https://github.com/ceems-dev/ceems/pull/146), [#147](https://github.com/ceems-dev/ceems/pull/147), [#148](https://github.com/ceems-dev/ceems/pull/148), [#149](https://github.com/ceems-dev/ceems/pull/149), [#150](https://github.com/ceems-dev/ceems/pull/150), [#151](https://github.com/ceems-dev/ceems/pull/151) , [#152](https://github.com/ceems-dev/ceems/pull/152), [#154](https://github.com/ceems-dev/ceems/pull/154), [#157](https://github.com/ceems-dev/ceems/pull/157), [#158](https://github.com/ceems-dev/ceems/pull/158), [#159](https://github.com/ceems-dev/ceems/pull/159), [#160](https://github.com/ceems-dev/ceems/pull/160), [#161](https://github.com/ceems-dev/ceems/pull/161), [#162](https://github.com/ceems-dev/ceems/pull/162), [#163](https://github.com/ceems-dev/ceems/pull/163), [#164](https://github.com/ceems-dev/ceems/pull/164), [#168](https://github.com/ceems-dev/ceems/pull/168), [#169](https://github.com/ceems-dev/ceems/pull/169), [#171](https://github.com/ceems-dev/ceems/pull/171), [#172](https://github.com/ceems-dev/ceems/pull/172), [#173](https://github.com/ceems-dev/ceems/pull/173), [#174](https://github.com/ceems-dev/ceems/pull/174), [#175](https://github.com/ceems-dev/ceems/pull/175) ([@dependabot](https://github.com/dependabot))

## 0.2.1 / 2024-08-17

- [BUGFIX] Fix setting sysprocattr correctly based on command [#136](https://github.com/ceems-dev/ceems/pull/136) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.2.0 / 2024-08-11

- [FEATURE] Pass context to downstream functions [#133](https://github.com/ceems-dev/ceems/pull/133) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Enable more linters [#132](https://github.com/ceems-dev/ceems/pull/132) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] General maintenance [#129](https://github.com/ceems-dev/ceems/pull/129) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Use native JSON functions in aggregate query [#128](https://github.com/ceems-dev/ceems/pull/128) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Stats API endpoint [#127](https://github.com/ceems-dev/ceems/pull/127) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Cache current usage query result [#122](https://github.com/ceems-dev/ceems/pull/122) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.1.1 / 2024-07-24

- [MAINT] DB query performance improvements [#113](https://github.com/ceems-dev/ceems/pull/113) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Fix metric aggregation [#112](https://github.com/ceems-dev/ceems/pull/112) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Incremental improvements on API server [#111](https://github.com/ceems-dev/ceems/pull/111) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Dont cache failed requests for emissions [#110](https://github.com/ceems-dev/ceems/pull/110) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Upgrade to Go 1.22.x [#109](https://github.com/ceems-dev/ceems/pull/109) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [TEST] Migrate to testify for unit tests [#108](https://github.com/ceems-dev/ceems/pull/108) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.1.0 / 2024-07-06

- [BUGFIX] Build swag using native arch in cross build [#107](https://github.com/ceems-dev/ceems/pull/107) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [CI] Avoid building test bins for release workflows [#106](https://github.com/ceems-dev/ceems/pull/106) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Fix tsdb updater [#104](https://github.com/ceems-dev/ceems/pull/104) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Store metrics as map in DB [#102](https://github.com/ceems-dev/ceems/pull/102) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Improve docs on Slurm collector [#101](https://github.com/ceems-dev/ceems/pull/101) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Improve docs on Slurm collector [#101](https://github.com/ceems-dev/ceems/pull/101) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [CI] Test DEB packages in CI [#100](https://github.com/ceems-dev/ceems/pull/100) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [CI] Extract go code for CodeQL analysis [#99](https://github.com/ceems-dev/ceems/pull/99) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Enforce rules on cluster and updater IDs [#98](https://github.com/ceems-dev/ceems/pull/98) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Update Docs [#97](https://github.com/ceems-dev/ceems/pull/97) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [CI] Add CodeQL workflow [#96](https://github.com/ceems-dev/ceems/pull/96) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add user and project tables to DB [#95](https://github.com/ceems-dev/ceems/pull/95) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Multicluster support [#94](https://github.com/ceems-dev/ceems/pull/94) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] General maintenance and enhancements [#92](https://github.com/ceems-dev/ceems/pull/92) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Add swagger docs [#90](https://github.com/ceems-dev/ceems/pull/90) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Setup docs website [#88](https://github.com/ceems-dev/ceems/pull/88) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [DOCS] Publish README to registries [#87](https://github.com/ceems-dev/ceems/pull/87) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Use weighted mean for agg stats [#86](https://github.com/ceems-dev/ceems/pull/86) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [CI] Make and publish container images [#85](https://github.com/ceems-dev/ceems/pull/85) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add demo end points [#84](https://github.com/ceems-dev/ceems/pull/84) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Support DB and API modes for access control [#83](https://github.com/ceems-dev/ceems/pull/83) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Enhancement api server [#78](https://github.com/ceems-dev/ceems/pull/78) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add `cpu_per_core_count` metric to CPU collector [#76](https://github.com/ceems-dev/ceems/pull/76) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add `last_updated_at` col in usage table [#75](https://github.com/ceems-dev/ceems/pull/75) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [REFACTOR] Use auth middleware for LB [#74](https://github.com/ceems-dev/ceems/pull/74) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add recording rules for Prometheus [#67](https://github.com/ceems-dev/ceems/pull/67) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Ensure non-negative values in agg metrics [#66](https://github.com/ceems-dev/ceems/pull/66) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.1.0-rc.6 / 2024-04-04

- [REFACTOR] Use generic name in metric names [#65](https://github.com/ceems-dev/ceems/pull/65) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Use custom float64 type [#62](https://github.com/ceems-dev/ceems/pull/62) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Configurable TSDB updater queries and DB migrations [#64](https://github.com/ceems-dev/ceems/pull/64) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Use custom float64 type [#62](https://github.com/ceems-dev/ceems/pull/62) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [TEST] Add unit tests [#61](https://github.com/ceems-dev/ceems/pull/61) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [CI] Fix go coverage badge in README [#60](https://github.com/ceems-dev/ceems/pull/60) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [CI] Add coverage badge to README [#59](https://github.com/ceems-dev/ceems/pull/59) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Debian and RPM packaging  [#58](https://github.com/ceems-dev/ceems/pull/58) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add a default resource manager [#57](https://github.com/ceems-dev/ceems/pull/57) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Auto detect IPMI command and add support for capmc [#56](https://github.com/ceems-dev/ceems/pull/44) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] chore: Several enhancements for CEEMS LB [#54](https://github.com/ceems-dev/ceems/pull/54) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Incremental metrics aggregation [#53](https://github.com/ceems-dev/ceems/pull/53) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Backend Auth for CEEMS LB  [#52](https://github.com/ceems-dev/ceems/pull/52) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.1.0-rc.5 / 2024-03-02

- [FEATURE] feat: Support RDMA stats in exporter [#45](https://github.com/ceems-dev/ceems/pull/45) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Rename stats pkg to api [#44](https://github.com/ceems-dev/ceems/pull/44) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] TSDB Load Balancer [#43](https://github.com/ceems-dev/ceems/pull/43) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] DB migrations support [#42](https://github.com/ceems-dev/ceems/pull/42) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [MAINT] Refactor DB schema [#41](https://github.com/ceems-dev/ceems/pull/41) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.1.0-rc.4 / 2024-02-18

- [BUGFIX] Misc bugfixes [#40](https://github.com/ceems-dev/ceems/pull/40) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Support different IPMI implementations [#39](https://github.com/ceems-dev/ceems/pull/39) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [REFACTOR] Rename pkg to ceems [#38](https://github.com/ceems-dev/ceems/pull/38) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Cache job props for SLURM collector [#37](https://github.com/ceems-dev/ceems/pull/37) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Extend DB schema to add new fields [#36](https://github.com/ceems-dev/ceems/pull/36) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Backup DB at configured interval [#35](https://github.com/ceems-dev/ceems/pull/35) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.1.0-rc.3 / 2024-01-22

- [REFACTOR] refactor: Remove support for job steps [#34](https://github.com/ceems-dev/ceems/pull/34) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Fetch admin users from grafana [#33](https://github.com/ceems-dev/ceems/pull/33) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [REFACTOR] Rename pkg [#32](https://github.com/ceems-dev/ceems/pull/32) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Enhancements in collector [#31](https://github.com/ceems-dev/ceems/pull/31) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Fix tsdb cleanup [#30](https://github.com/ceems-dev/ceems/pull/30) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [REFACTOR] Split node metrics into separate collectors [#29](https://github.com/ceems-dev/ceems/pull/29) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add total procs cputime metric [#28](https://github.com/ceems-dev/ceems/pull/28) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add support for TSDB vacuuming [#27](https://github.com/ceems-dev/ceems/pull/27) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Use a separate time series for each job for mapping GPU [#26](https://github.com/ceems-dev/ceems/pull/26) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Use query builder [#25](https://github.com/ceems-dev/ceems/pull/25) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Job stats server enhancements [#24](https://github.com/ceems-dev/ceems/pull/24) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [REFACTOR] Use cgroups v2 pkg [#23](https://github.com/ceems-dev/ceems/pull/23) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [REFACTOR] Rename emissions factory from source to provider [#22](https://github.com/ceems-dev/ceems/pull/22) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Export min and max power readings from ipmi [#21](https://github.com/ceems-dev/ceems/pull/21) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add hostname label to exporter metrics [#20](https://github.com/ceems-dev/ceems/pull/20) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] Correct env var name for getting gpu index [#19](https://github.com/ceems-dev/ceems/pull/19) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.1.0-rc.2 / 2023-12-26

- [REFACTOR] Refactor jobstats pkg [#18](https://github.com/ceems-dev/ceems/pull/18) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [REFACTOR] Use default http client for requests for emissions collector [#16](https://github.com/ceems-dev/ceems/pull/16) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [REFACTOR] Refactor emissions pkg [#16](https://github.com/ceems-dev/ceems/pull/16) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [BUGFIX] bugfix: Correctly parse SLURM nodelist range string [#15](https://github.com/ceems-dev/ceems/pull/15) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))

## 0.1.0-rc.1 / 2023-12-20

- [FEATURE] Bug fixes and refactoring [#14](https://github.com/ceems-dev/ceems/pull/14) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Misc improvements [#13](https://github.com/ceems-dev/ceems/pull/13) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Merge job stats DB and server commands [#12](https://github.com/ceems-dev/ceems/pull/12) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Support GPU jobID map from /proc [#11](https://github.com/ceems-dev/ceems/pull/11) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add Runtime pkg [#10](https://github.com/ceems-dev/ceems/pull/10) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Misc features [#9](https://github.com/ceems-dev/ceems/pull/9) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add API server to serve job stats [#8](https://github.com/ceems-dev/ceems/pull/8) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add jobstats pkg [#7](https://github.com/ceems-dev/ceems/pull/7) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Use pkg structure [#6](https://github.com/ceems-dev/ceems/pull/6) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Use UID and GID to job labels [#5](https://github.com/ceems-dev/ceems/pull/5) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Reorganise repo [#4](https://github.com/ceems-dev/ceems/pull/4) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add unique jobid label for SLURM jobs [#3](https://github.com/ceems-dev/ceems/pull/3) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] Add Emission collector [#2](https://github.com/ceems-dev/ceems/pull/2) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
- [FEATURE] CircleCI setup [#1](https://github.com/ceems-dev/ceems/pull/1) ([@mahendrapaipuri](https://github.com/mahendrapaipuri))
