---
sidebar_position: 8
---

# cacct

`cacct` is a CLI client that can be used instead of Grafana when operators
cannot or do not wish to maintain a Grafana instance. This CLI client communicates
with both the CEEMS API server and the TSDB server to fetch energy, usage,
performance metrics for a given compute unit, project, and/or user. It has been largely
inspired by SLURM's [`sacct`](https://slurm.schedmd.com/sacct.html) tool, and the API
resembles that of `sacct`.

:::important[IMPORTANT]

`cacct` identifies the current username from their Linux UID. Thus, for `cacct`
to work correctly, the user's UID must be the same on the machine where `cacct` is
executed and in the CEEMS API server database.

:::

This tool has been specifically designed for HPC platforms where there is a common
login node that users can access via SSH. The tool must be installed on such login
nodes along with its configuration file. The `cacct` configuration file contains the
HTTP client configuration details needed to connect to the CEEMS API and TSDB servers.
Consequently, this configuration file might contain secrets for communicating with these
servers, making it crucial to protect this file on a multi-tenant system like HPC login
nodes. This will be discussed further in the following sections. First, let's examine
the available configuration sections for `cacct`:

```yaml
# cacct configuration skeleton
logging: <LOGGING CONFIG>

ceems_api_server: <CEEMS API SERVER CONFIG>

tsdb: <TSDB CONFIG>
```

:::important[IMPORTANT]

`cacct` always looks for its configuration file at `/etc/ceems/config.yml` or
`/etc/ceems/config.yaml`. Therefore, the configuration file must be installed in
one of these locations.

:::

A sample configuration file with only the CEEMS API Server configuration is presented below:

```yaml
ceems_api_server:
  cluster_id: slurm-0
  user_header_name: X-Grafana-User
  web:
    url: http://ceems-api-server:9020
    basic_auth:
      username: ceems
      password: supersecretpassword
```

The above configuration assumes that the target cluster has `slurm-0` as its cluster
ID, as configured in the [CEEMS API server configuration](./ceems-api-server.md#clusters-configuration).
By default, the CEEMS API server expects the username in the `X-Grafana-User` header,
so `cacct` sets the value for this header with the username making the request.
Finally, the `web` section contains the HTTP client configuration for the CEEMS API
server. In this example, the CEEMS API server is reachable at host `ceems-api-server`
on port `9020`, and basic authentication is configured.

`cacct` can pull time series data from the TSDB server for the requested compute units.
This is possible only when the `tsdb` section is configured. A sample configuration file
including both CEEMS API server and TSDB server configurations is shown below:

```yaml
ceems_api_server:
  cluster_id: slurm-0
  user_header_name: X-Grafana-User
  web:
    url: http://ceems-api-server:9020
    basic_auth:
      username: ceems
      password: supersecretpassword

tsdb:
  web:
    url: http://tsdb:9090
    basic_auth:
      username: prometheus
      password: anothersupersecretpassword
  max_units: 5
  scrape_interval: 10s
  evaluation_interval: 10s
  range_queries:
    # CPU utilization
    - name: cpu_usage 
      title: "CPU Usage (%)"
      help: "Usage of CPUs in %"
      query: uuid:ceems_cpu_usage:ratio_irate{uuid=~`{{.UUIDs}}`}
    
    # CPU Memory utilization
    - name: cpu_mem_usage
      title: "CPU Memory Usage (%)" 
      help: "Ratio of memory used to memory reserved of CPU in %"
      query: uuid:ceems_cpu_memory_usage:ratio{uuid=~`{{.UUIDs}}`}
      
    # Host power usage in Watts
    - name: host_power_usage
      title: "Host Power usage (W)" 
      help: "Instanteous power usage of host in Watts"
      query: uuid:ceems_host_power_watts:pue{uuid=~`{{.UUIDs}}`}

    # Host emissions in g/s
    - name: host_emissions 
      title: "Host Eq. Emissions Rate (g/s)"
      help: "Instanteous emissions rate of host in g/s"
      query: uuid:ceems_host_emissions_g_s:pue{uuid=~`{{.UUIDs}}`}

    # GPU utilization
    - name: avg_gpu_usage
      title: "GPU Usage (%)" 
      help: "Usage of GPU in %"
      query: uuid:ceems_gpu_usage:ratio{uuid=~`{{.UUIDs}}`}

    # GPU memory utilization
    - name: avg_gpu_mem_usage
      title: "GPU Memory Usage (%)" 
      help: "Ratio of memory used to memory reserved of GPU in %"
      query: uuid:ceems_gpu_memory_usage:ratio{uuid=~`{{.UUIDs}}`}

    # GPU power usage in Watts
    - name: gpu_power_usage
      title: "GPU Power Usage (W)"
      help: "Instanteous Power Usage of GPU in Watts"
      query: uuid:ceems_gpu_power_watts:pue{uuid=~`{{.UUIDs}}`}

    # GPU emissions in g/s
    - name: gpu_emissions 
      title: "GPU Eq. Emission Rate (g/s)"
      help: "Instanteous emissions rate of GPU in g/s"
      query: uuid:ceems_gpu_emissions_g_s:pue{uuid=~`{{.UUIDs}}`}

    # Read IO bytes/s
    - name: io_read_bytes 
      title: "IO Read Bandwidth (b/s)"
      help: "Instanteous IO read bandwidth in bytes/s"
      query: irate(ceems_ebpf_read_bytes_total{uuid=~`{{.UUIDs}}`}[1m])

    # Write IO bytes/s
    - name: io_write_bytes 
      title: "IO Write Bandwidth (b/s)"
      help: "Instanteous IO write bandwidth in bytes/s"
      query: irate(ceems_ebpf_write_bytes_total{uuid=~`{{.UUIDs}}`}[1m])
```

The keys in the `tsdb` section are explained as below:

- `tsdb.max_units`: Maximum number of units' time series to be fetched in a single
CLI execution. Use an appropriate number based on the deployment as making too many
range queries to TSDB can spike the memory usage of TSDB server. Default value is 10.
- `tsdb.scrape_interval`: The scrape interval of the TSDB jobs where queries are being
executed
- `tsdb.evaluation_interval`: The evaluation interval of the TSDB recording rules
- `tsdb.range_queries`: A list of queries to be executed to fetch the time series data.
    - `tsdb.range_queries.name`: An **unique** short name for the query
    - `tsdb.range_queries.title`: A human readable short title for the query. It will be included in the
    output, so choose a name easy to understand for the end users.
    - `tsdb.range_queries.help`: A small help text to explain what query metrics means
    - `tsdb.range_queries.query`: It is the TSDB query that will be executed. Available
    template variales are `{{.UUIDs}}`, `{{.ScrapeInterval}}`, `{{.RateInterval}}` and `{{.Range}}`.

Similar to the CEEMS API server configuration, this example assumes the TSDB server is
reachable at `tsdb:9090` and basic authentication is configured on the HTTP server. The
`tsdb.range_queries` section is where operators configure the queries to pull time series data
for each metric. If operators used [`ceems_tool`](../usage/ceems-tool.md) to generate
recording rules for the TSDB, the queries in the sample configuration above will work
out-of-the-box.

:::note[NOTE]

There is no risk of injection here, as the UUID values provided by the end-user are first
sanitized and then verified with the CEEMS API server to check if the user is the owner
of the compute unit before passing them to the TSDB server.

:::

As `cacct` is user facing application, any errors encountered while making requests to
CEEMS API server and/or TSDB will not be shown to end user to avoid leaking any sensitive
information. Thus, the errors messages from `cacct` are very generic and will not aide for
debugging the errors. To address this shortcoming, `cacct` supports a system level
logging where a comprehensive logging of all `cacct` invocations from end users are maintained.
This system level logging is meant to be used only by system operators and administrators
and thus, it will be created with strict permissions. This logging file can also be used
for auditing purposes as user activity will be logged along with the runtime configurations
thus allow operators to detect users that might be abusing the service. This logging can be
enabled using `logging` section in the configuration file.

```yaml
logging:
  enabled: true
  format: json
  level: debug

ceems_api_server:
  cluster_id: slurm-0
  user_header_name: X-Grafana-User
  web:
    url: http://ceems-api-server:9020
    basic_auth:
      username: ceems
      password: supersecretpassword

tsdb:
  web:
    url: http://tsdb:9090
    basic_auth:
      username: prometheus
      password: anothersupersecretpassword
  max_units: 5
  scrape_interval: 10s
  evaluation_interval: 10s
  range_queries:
    # CPU utilization
    - name: cpu_usage 
      title: "CPU Usage (%)"
      help: "Usage of CPUs in %"
      query: uuid:ceems_cpu_usage:ratio_irate{uuid=~`{{.UUIDs}}`}
    
    # CPU Memory utilization
    - name: cpu_mem_usage
      title: "CPU Memory Usage (%)" 
      help: "Ratio of memory used to memory reserved of CPU in %"
      query: uuid:ceems_cpu_memory_usage:ratio{uuid=~`{{.UUIDs}}`}
      
    # Host power usage in Watts
    - name: host_power_usage
      title: "Host Power usage (W)" 
      help: "Instanteous power usage of host in Watts"
      query: uuid:ceems_host_power_watts:pue{uuid=~`{{.UUIDs}}`}

    # Host emissions in g/s
    - name: host_emissions 
      title: "Host Eq. Emissions Rate (g/s)"
      help: "Instanteous emissions rate of host in g/s"
      query: uuid:ceems_host_emissions_g_s:pue{uuid=~`{{.UUIDs}}`}

    # GPU utilization
    - name: avg_gpu_usage
      title: "GPU Usage (%)" 
      help: "Usage of GPU in %"
      query: uuid:ceems_gpu_usage:ratio{uuid=~`{{.UUIDs}}`}

    # GPU memory utilization
    - name: avg_gpu_mem_usage
      title: "GPU Memory Usage (%)" 
      help: "Ratio of memory used to memory reserved of GPU in %"
      query: uuid:ceems_gpu_memory_usage:ratio{uuid=~`{{.UUIDs}}`}

    # GPU power usage in Watts
    - name: gpu_power_usage
      title: "GPU Power Usage (W)"
      help: "Instanteous Power Usage of GPU in Watts"
      query: uuid:ceems_gpu_power_watts:pue{uuid=~`{{.UUIDs}}`}

    # GPU emissions in g/s
    - name: gpu_emissions 
      title: "GPU Eq. Emission Rate (g/s)"
      help: "Instanteous emissions rate of GPU in g/s"
      query: uuid:ceems_gpu_emissions_g_s:pue{uuid=~`{{.UUIDs}}`}

    # Read IO bytes/s
    - name: io_read_bytes 
      title: "IO Read Bandwidth (b/s)"
      help: "Instanteous IO read bandwidth in bytes/s"
      query: irate(ceems_ebpf_read_bytes_total{uuid=~`{{.UUIDs}}`}[1m])

    # Write IO bytes/s
    - name: io_write_bytes 
      title: "IO Write Bandwidth (b/s)"
      help: "Instanteous IO write bandwidth in bytes/s"
      query: irate(ceems_ebpf_write_bytes_total{uuid=~`{{.UUIDs}}`}[1m])
```

A complete reference can be found in the [Reference](./config-reference.md) section. A valid
sample configuration file can be found in the
[repository](https://github.com/@ceemsOrg@/@ceemsRepo@/blob/main/build/config/cacct/cacct.yml).

## Securing configuration file

As evident from the previous section, the `cacct` configuration file contains secrets that
should not be accessible to end-users. At the same time, the `cacct` executable must be
accessible to end-users so they can fetch their usage statistics. This means `cacct` must
be able to read the configuration file at runtime, but the user executing it should not.
This can be achieved using the [Sticky bit](https://www.redhat.com/en/blog/suid-sgid-sticky-bit).

By using the SETUID bit on the executable, the binary will have privileges of the user
that owns the file. Thus, a SETUID `ceems` owned file can read config file owned by `ceems`.
Once the config file has been read, `cacct` will drop privileges and executes rest of code as the
user who invoked it. This way the privileges are only kept for a minimal time to read config file
and dropped after fetching config. The SETGID sticky bit
can be set on `cacct` as follows:

```bash
chown ceems:ceems /usr/local/bin/cacct
chmod u+s /usr/local/bin/cacct
# Ensure others can execute cacct
chmod o+x /usr/local/bin/cacct

# Use the same user/group as owner:group for the cacct configuration file
chown ceems:ceems /etc/ceems/config.yml
# Revoke all permissions for others
chmod o-rwx /etc/ceems/config.yml
```

Now, every time `cacct` is invoked, it will have privileges of the `ceems` user/group instead to read
`/etc/ceems/config.yml` and drop privileges to user who invoked the program later.

Similarly, if system logging is desired, the `cacct` binary should keep the group ownership of `ceems`
while end user launches the application. This way the log file will be owned by `ceems` user preventing
regular end users to access it.

```bash
# Create a directory to place system level log file
mkdir -p /var/log/ceems
chown ceems:ceems /var/log/ceems
# Revoke all permissions for others
chmod o-rwx /var/log/ceems
```

When `cacct` is installed using the RPM/DEB file provided by the
[CEEMS Releases](https://github.com/@ceemsOrg@/@ceemsRepo@/releases), `cacct` is already installed with
the sticky bits set. Operators only need to populate the configuration file at `/etc/ceems/config.yml`.
