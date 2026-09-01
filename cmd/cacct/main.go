package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/ceems-dev/ceems/internal/common"
	"github.com/ceems-dev/ceems/pkg/api/models"
	"github.com/iancoleman/strcase"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	http_config "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/version"
	"gopkg.in/yaml.v3"
)

// Locations where config file must be found.
var (
	configPaths = []string{
		"/etc/ceems",
	}

	// CLI app.
	cacctApp = kingpin.New(
		filepath.Base(os.Args[0]), "Energy/Emissions/Performance/Usage data for all jobs fetched from CEEMS database.",
	).UsageWriter(os.Stdout)

	// mock user and config paths for test.
	mockCurrentUser, mockConfigPath string

	fieldMap = map[string]*field{
		"jobid": {
			tag:   "uuid",
			name:  "jobID",
			help:  "Job ID",
			title: "Job ID",
			minW:  3,
			maxW:  7,
		},
		"name": {
			tag:   "name",
			name:  "name",
			help:  "Name of the job",
			title: "Name",
			minW:  5,
			maxW:  8,
		},
		"account": {
			tag:   "project",
			name:  "account",
			help:  "Account name",
			title: "Account",
			minW:  5,
			maxW:  8,
		},
		"group": {
			tag:   "groupname",
			name:  "group",
			help:  "Group name",
			title: "Group",
			minW:  5,
			maxW:  5,
		},
		"user": {
			tag:   "username",
			name:  "user",
			help:  "User name",
			title: "User",
			minW:  5,
			maxW:  5,
		},
		"createdat": {
			tag:   "created_at",
			name:  "createdAt",
			help:  "Job creation time",
			title: "Created",
			minW:  5,
			maxW:  8,
		},
		"startedat": {
			tag:   "started_at",
			name:  "startedAt",
			help:  "Job start time",
			title: "Started",
			minW:  5,
			maxW:  12,
		},
		"endedat": {
			tag:   "ended_at",
			name:  "endedAt",
			help:  "Job end time",
			title: "Ended",
			minW:  5,
			maxW:  12,
		},
		"elapsed": {
			tag:   "elapsed",
			name:  "elapsed",
			help:  "Job elapsed time",
			title: "Elapsed",
			minW:  5,
			maxW:  12,
		},
		"totaltime": {
			tag:   "total_time_seconds",
			name:  "totalTime",
			help:  "Wall, CPU and GPU times consumed in seconds over the duration of the job",
			title: "Total Time(s)",
			minW:  5,
			maxW:  12,
		},
		"allocation": {
			tag:   "allocation",
			name:  "allocation",
			help:  "Resource allocation of CPU, GPU, memory, etc",
			title: "Resource Allocation",
			minW:  5,
			maxW:  12,
		},
		"state": {
			tag:   "state",
			name:  "state",
			help:  "Job state",
			title: "State",
			minW:  5,
			maxW:  5,
		},
		"cpuusage": {
			tag:   "avg_cpu_usage",
			name:  "cpuUsage",
			help:  "Average CPU usage over the duration of the job",
			title: "CPU Usage(%)",
			minW:  5,
			maxW:  6,
		},
		"cpumemoryusage": {
			tag:   "avg_cpu_mem_usage",
			name:  "cpuMemoryUsage",
			help:  "Average CPU memory usage over the duration of the job",
			title: "CPU Mem. Usage(%)",
			minW:  5,
			maxW:  6,
		},
		"hostenergy": {
			tag:   "total_cpu_energy_usage_kwh",
			name:  "hostEnergy",
			help:  "Total energy usage by the host duration of the job",
			title: "Host Energy(kWh)",
			minW:  5,
			maxW:  8,
		},
		"hostemissions": {
			tag:   "total_cpu_emissions_gms",
			name:  "hostEmissions",
			help:  "Total eq. emissions due to host energy usage duration of the job",
			title: "Host Emissions(gms)",
			minW:  5,
			maxW:  12,
		},
		"gpuusage": {
			tag:   "avg_gpu_usage",
			name:  "gpuUsage",
			help:  "Average GPU(s) usage over the duration of the job",
			title: "GPU Usage(%)",
			minW:  5,
			maxW:  6,
		},
		"gpumemoryusage": {
			tag:   "avg_gpu_mem_usage",
			name:  "gpuMemoryUsage",
			help:  "Average GPU(s) memory usage over the duration of the job",
			title: "GPU Mem. Usage(%)",
			minW:  5,
			maxW:  6,
		},
		"gpuenergy": {
			tag:   "total_gpu_energy_usage_kwh",
			name:  "gpuEnergy",
			help:  "Total energy usage by the GPU(s) duration of the job",
			title: "GPU Energy(kWh)",
			minW:  5,
			maxW:  8,
		},
		"gpuemissions": {
			tag:   "total_gpu_emissions_gms",
			name:  "gpuEmissions",
			help:  "Total eq. emissions due to GPU(s) energy usage duration of the job",
			title: "GPU Emissions(gms)",
			minW:  5,
			maxW:  12,
		},
	}

	allFields = []string{
		"jobid",
		"name",
		"account",
		"group",
		"user",
		"createdat",
		"startedat",
		"endedat",
		"elapsed",
		"totaltime",
		"allocation",
		"state",
		"cpuusage",
		"cpumemoryusage",
		"hostenergy",
		"hostemissions",
		"gpuusage",
		"gpumemoryusage",
		"gpuenergy",
		"gpuemissions",
	}

	defaultFields = []string{
		"jobid",
		"user",
		"account",
		"elapsed",
		"cpuusage",
		"cpumemoryusage",
		"hostenergy",
		"hostemissions",
		"gpuusage",
		"gpumemoryusage",
		"gpuenergy",
		"gpuemissions",
	}
)

// Custom errors.
var (
	errNoPerm   = errors.New("forbidden response from API server")
	errConfig   = errors.New("unable to get cacct config")
	errLogFile  = errors.New("unable to get open log file")
	errUser     = errors.New("unable to change user context")
	errNoUnits  = errors.New("no jobs found in the selected period")
	errInternal = errors.New("internal server error")
)

// field is a container for each field metadata in the table.
type field struct {
	tag   string
	name  string
	help  string
	title string
	keys  []string
	minW  int
	maxW  int
}

// titles return header titles for the table.
func (f field) titles() []any {
	if len(f.keys) <= 1 {
		return []any{f.title}
	}

	t := make([]any, len(f.keys))
	for i := range len(f.keys) {
		t[i] = f.title
	}

	return t
}

// subtitles return header subtitles for the table.
func (f field) subtitles() []any {
	if len(f.keys) <= 1 {
		return []any{""}
	}

	t := make([]any, len(f.keys))
	for i, k := range f.keys {
		t[i] = k
	}

	return t
}

// // Default TSDB queries.
// var (
// 	defaultQueries = map[string]string{
// 		"cpu_usage":         `uuid:ceems_cpu_usage:ratio_irate{uuid=~"%s"}`,
// 		"cpu_mem_usage":     `uuid:ceems_cpu_memory_usage:ratio{uuid=~"%s"}`,
// 		"host_power_usage":  `uuid:ceems_host_power_watts:pue{uuid=~"%s"}`,
// 		"host_emissions":    `uuid:ceems_host_emissions_g_s:pue{uuid=~"%s"}`,
// 		"avg_gpu_usage":     `uuid:ceems_gpu_usage:ratio{uuid=~"%s"}`,
// 		"avg_gpu_mem_usage": `uuid:ceems_gpu_memory_usage:ratio{uuid=~"%s"}`,
// 		"gpu_power_usage":   `uuid:ceems_gpu_power_watts:pue{uuid=~"%s"}`,
// 		"gpu_emissions":     `uuid:ceems_gpu_emissions_g_s:pue{uuid=~"%s"}`,
// 		"io_read_bytes":     `irate(ceems_ebpf_read_bytes_total{uuid=~"%s"}[1m])`,
// 		"io_write_bytes":    `irate(ceems_ebpf_write_bytes_total{uuid=~"%s"}[1m])`,
// 	}
// )

const (
	instantQuery = "instant"
	rangeQuery   = "range"
)

type TSDBQuery struct {
	Name  string `yaml:"name"`
	Help  string `yaml:"help"`
	Title string `yaml:"title"`
	Query string `yaml:"query"`
	Kind  string `yaml:"kind"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (q *TSDBQuery) UnmarshalYAML(unmarshal func(any) error) error {
	// Set a default config
	*q = TSDBQuery{}

	type plain TSDBQuery

	err := unmarshal((*plain)(q))
	if err != nil {
		return err
	}

	// Check if query is of range or instant
	if q.Kind != rangeQuery && q.Kind != instantQuery {
		return fmt.Errorf("invalid value %s found for kind. Must be one of %s or %s", q.Kind, instantQuery, rangeQuery)
	}

	// Validate config
	if q.Name == "" || q.Query == "" || q.Kind == "" {
		return errors.New("name, query and kind cannot be empty in entry of queries")
	}

	// If title is empty, use same as name
	if q.Title == "" {
		q.Title = q.Name
	}

	// Convert name to camelCase
	q.Name = strcase.ToLowerCamel(q.Name)

	return nil
}

// Config contains the cacct configuration settings.
type Config struct {
	API struct {
		Web            WebConfig `yaml:"web"`
		ClusterID      string    `yaml:"cluster_id"`
		UserHeaderName string    `yaml:"user_header_name"`
	} `yaml:"ceems_api_server"`
	TSDB struct {
		Web                     WebConfig      `yaml:"web"`
		ScrapeInterval          model.Duration `yaml:"scrape_interval"`
		EvaluationInterval      model.Duration `yaml:"evaluation_interval"`
		QueryMaxSeries          int64          `yaml:"query_max_series"`
		QueryMinSamples         float64        `yaml:"query_min_samples"`
		MaxUnitsForRangeQueries int            `yaml:"max_units_for_range_queries"`
		QueryTimeout            model.Duration `yaml:"query_timeout"`
		Queries                 []TSDBQuery    `yaml:"queries"`
		rangeQueries            []TSDBQuery
		instantQueries          []TSDBQuery
	} `yaml:"tsdb"`
	Logging struct {
		Enabled   bool             `yaml:"enabled"`
		Level     *promslog.Level  `yaml:"level"`
		Format    *promslog.Format `yaml:"format"`
		Directory string           `yaml:"directory"`
		File      string
	} `yaml:"logging"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	// Set a default config
	*c = Config{}
	c.API.UserHeaderName = "X-Grafana-User"
	c.Logging.Level = promslog.NewLevel()
	c.TSDB.MaxUnitsForRangeQueries = 10
	c.TSDB.QueryMaxSeries = 20
	c.TSDB.QueryMinSamples = 0.5
	c.TSDB.QueryTimeout = model.Duration(time.Minute)

	err := c.Logging.Level.Set("info")
	if err != nil {
		return fmt.Errorf("failed to set default log level: %w", err)
	}

	c.Logging.Format = promslog.NewFormat()

	err = c.Logging.Format.Set("logfmt")
	if err != nil {
		return fmt.Errorf("failed to set default log format: %w", err)
	}

	c.Logging.Directory = "/var/log/ceems"
	// c.TSDB.Queries = defaultQueries

	type plain Config

	err = unmarshal((*plain)(c))
	if err != nil {
		return err
	}

	// Validate config
	err = c.Validate()
	if err != nil {
		return err
	}

	// Add instant queries to fieldMap and allFields
	for _, q := range c.TSDB.Queries {
		nameKey := strings.ToLower(q.Name)
		switch q.Kind {
		case instantQuery:
			fieldMap[nameKey] = &field{
				name:  q.Name,
				help:  q.Help,
				title: q.Title,
				minW:  2,
				maxW:  10,
			}
			allFields = append(allFields, nameKey)
		}
	}

	return nil
}

// Validate validates the config.
func (c *Config) Validate() error {
	// Check there are no duplicate names in range and instant queries
	var allQueryNames []string
	for _, q := range c.TSDB.Queries {
		if slices.Contains(allQueryNames, q.Name) {
			return fmt.Errorf("name key %s duplicates found in tsdb.queries %s", q.Name, strings.Join(allQueryNames, ","))
		}

		allQueryNames = append(allQueryNames, q.Name)
	}

	// If logging is not enabled, nothing to do here
	if !c.Logging.Enabled {
		return nil
	}

	// Check if logging directory exists
	absPath, err := filepath.Abs(c.Logging.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve abolsute path for logging directory: %w", err)
	}

	_, err = os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to locate logging directory: %w", err)
	}

	// Set absolute path
	c.Logging.Directory = absPath

	// Set logging file path based on format
	switch c.Logging.Format.String() {
	case "json":
		c.Logging.File = filepath.Join(c.Logging.Directory, "cacct.json")
	default:
		c.Logging.File = filepath.Join(c.Logging.Directory, "cacct.log")
	}

	return nil
}

// SetupTSDBQueries sets up TSDB queries based on CLI args.
func (c *Config) SetupTSDBQueries(instantQueries []string, rangeQueries []string) {
	// Add instant queries to fieldMap and allFields
	for _, q := range c.TSDB.Queries {
		switch {
		case q.Kind == instantQuery && slices.Contains(instantQueries, q.Name):
			c.TSDB.instantQueries = append(c.TSDB.instantQueries, q)
		case q.Kind == rangeQuery && slices.Contains(rangeQueries, q.Name):
			c.TSDB.rangeQueries = append(c.TSDB.rangeQueries, q)
		}
	}
}

// WebConfig contains HTTP related config.
type WebConfig struct {
	URL              string                       `yaml:"url"`
	HTTPClientConfig http_config.HTTPClientConfig `yaml:",inline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (w *WebConfig) UnmarshalYAML(unmarshal func(any) error) error {
	// Set a default config
	*w = WebConfig{}
	w.HTTPClientConfig = http_config.DefaultHTTPClientConfig

	type plain WebConfig

	err := unmarshal((*plain)(w))
	if err != nil {
		return err
	}

	// The UnmarshalYAML method of HTTPClientConfig is not being called because it's not a pointer.
	// We cannot make it a pointer as the parser panics for inlined pointer structs.
	// Thus we just do its validation here.
	err = w.HTTPClientConfig.Validate()
	if err != nil {
		return err
	}

	return nil
}

// Response defines the response model of CEEMSAPIServer.
type Response[T any] struct {
	Status   string   `json:"status"`
	Data     []T      `json:"data"`
	Error    string   `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func main() {
	var (
		helpFormat, longFormat            bool
		htmlOut, csvOut, mdOut            bool
		summaryStats                      bool
		tsDataOut, tsMetrics              string
		accountsFlag, jobsFlag, usersFlag string
		formatFlag                        string
		startTime, endTime                string
	)

	cacctApp.Version(version.Print("caact"))
	cacctApp.HelpFlag.Short('h')

	// CLI flags
	cacctApp.Flag(
		"account", "Comma separated list of account to select jobs to display. By default, all accounts are selected.",
	).StringVar(&accountsFlag)
	cacctApp.Flag(
		"starttime", "Select jobs eligible after this time. Valid format is YYYY-MM-DD[THH:MM[:SS]] (default: 00:00:00 of the current day).",
	).Default(time.Now().Format("2006-01-02") + "T00:00:00").StringVar(&startTime)
	cacctApp.Flag(
		"endtime", "Select jobs eligible before this time. Valid format is YYYY-MM-DD[THH:MM[:SS]] (default: now).",
	).Default(time.Now().Format("2006-01-02T15:04:05")).StringVar(&endTime)
	cacctApp.Flag(
		"job", "Comma separated list of jobs to display information. Default is all jobs in the period.",
	).StringVar(&jobsFlag)
	cacctApp.Flag(
		"user", "Comma separated list of user names to select jobs to display. A special value `all` can be used to fetch jobs of all users when querying user has enough privileges. By default, the running user is used.",
	).StringVar(&usersFlag)
	cacctApp.Flag(
		"format", "Comma separated list of fields (Use --helpformat for list of available fields).",
	).StringVar(&formatFlag)
	cacctApp.Flag(
		"helpformat", "List of available fields.",
	).Default("false").BoolVar(&helpFormat)
	cacctApp.Flag(
		"long", fmt.Sprintf("Equivalent to specifying --format=\"%s\".", strings.Join(allFields, ",")),
	).Default("false").BoolVar(&longFormat)
	cacctApp.Flag(
		"summary", "Include summary statistics at the end in the results.",
	).Default("true").BoolVar(&summaryStats)
	cacctApp.Flag(
		"ts.metrics", "Comma separated list of time series metrics. Check available metrics using --helpformat flag.",
	).StringVar(&tsMetrics)
	cacctApp.Flag(
		"ts.out-dir", "Directory to save time series data.",
	).Default("out").StringVar(&tsDataOut)
	cacctApp.Flag(
		"csv", "Produce CSV output (default: false).",
	).Default("false").BoolVar(&csvOut)
	cacctApp.Flag(
		"html", "Produce HTML output (default: false).",
	).Default("false").BoolVar(&htmlOut)
	cacctApp.Flag(
		"markdown", "Produce markdown output (default: false).",
	).Default("false").BoolVar(&mdOut)

	_, err := cacctApp.Parse(os.Args[1:])
	if err != nil {
		kingpin.Fatalf("failed to parse CLI flags: %v", err)
	}

	// First read the config file to get the queries supplied for TSDB so that we
	// can add them to helpformat flag.
	// Either setuid or setgid bits must be applied on the app so that
	// the config file can be read as the owner of this app
	config, err := readConfig(mockConfigPath)
	if err != nil {
		os.Exit(checkErr(fmt.Errorf("%w: %w", errConfig, err)))
	}

	// If helpformat, print available fields and return
	if helpFormat {
		// First collect keys and sort them
		keys := sortedKeys(fieldMap)

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Field", "Description"})

		for _, k := range keys {
			t.AppendRow(table.Row{fieldMap[k].name, fieldMap[k].help})
		}

		// Append available time series metrics
		t.AppendSeparator()

		t.AppendRow(table.Row{"Available Time Series"})
		t.AppendSeparator()

		for _, q := range config.TSDB.Queries {
			if q.Kind == rangeQuery {
				t.AppendRow(table.Row{q.Name, q.Help})
			}
		}

		t.Render()

		os.Exit(0)
	}

	// Convert flags to slices
	accounts := splitString(accountsFlag, ",")
	jobs := splitString(jobsFlag, ",")
	userNames := splitString(usersFlag, ",")

	// Get format fields
	formatFields := splitString(formatFlag, ",")
	if len(formatFields) == 0 {
		formatFields = defaultFields
	}

	// If long format is asked, use all fields
	if longFormat {
		formatFields = allFields
	}

	var (
		activeInstantQueries []string
		activeRangeQueries   []string
	)

	// Get active range query names based on CLI args
	for _, t := range splitString(tsMetrics, ",") {
		for _, q := range config.TSDB.Queries {
			if strings.EqualFold(q.Name, t) {
				activeRangeQueries = append(activeRangeQueries, q.Name)
			}
		}
	}

	// Get active instant query names based on CLI args
	// ALWAYS include uuid in fields
	fields := []string{"uuid"}

	for _, f := range formatFields {
		nameKey := strings.ToLower(f)
		if field, ok := fieldMap[nameKey]; ok && nameKey != "jobid" {
			tag := field.tag
			if tag != "" {
				fields = append(fields, tag)
			}

			for _, q := range config.TSDB.Queries {
				if q.Name == field.name && q.Kind == instantQuery {
					activeInstantQueries = append(activeInstantQueries, q.Name)
				}
			}
		}
	}

	// Setup queries on config struct
	config.SetupTSDBQueries(activeInstantQueries, activeRangeQueries)

	// Always add started and ended ts fields as we will need them for TSDB data retrieval
	fields = append(fields, []string{"started_at_ts", "ended_at_ts"}...)

	// Convert start and end times to time.Time
	var start, end time.Time

	start, err = parseTime(startTime)
	if err != nil {
		kingpin.Fatalf("failed to parse --starttime flag: %v", err)
	}

	end, err = parseTime(endTime)
	if err != nil {
		kingpin.Fatalf("failed to parse --endtime flag: %v", err)
	}

	// Setup logger
	promslogConfig := &promslog.Config{
		Level:  config.Logging.Level,
		Format: config.Logging.Format,
		Writer: io.Discard,
	}

	// Open logging file
	if config.Logging.Enabled {
		logFile, err := os.OpenFile(config.Logging.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o660)
		if err != nil {
			os.Exit(checkErr(fmt.Errorf("%w: %w", errLogFile, err)))
		}
		defer logFile.Close()

		promslogConfig.Writer = logFile
	}

	// Create a new logger
	logger := promslog.New(promslogConfig)

	// Get current user and add user's config dir to slice of config
	// dirs.
	// If current user is root and mockCurrentUser and/or mockConfigPath
	// are set, we override the actual with mock ones. Only used in testing
	// and it should not affect production cases.
	currentUser, err := getCurrentUser(mockCurrentUser)
	if err != nil {
		logger.Error("Failed to get current user executing cacct", "err", err)

		os.Exit(checkErr(fmt.Errorf("failed to get current user: %w", err)))
	}

	// Check if currentUser is only user in userNames and if so, set userNames to nil
	if len(userNames) == 1 && userNames[0] == currentUser.Username {
		userNames = nil
	}

	logger = logger.With("username", currentUser.Username)
	logger.Info("Current user identified")

	// Now time to drop privileges so that rest of app will be run as regular user
	// who invoked it. It is necessary so to be able to create directories and files
	// to user's space.
	// The condition ensures that it will be executed only in production and not in e2e
	// test cases
	if mockCurrentUser == "" && mockConfigPath == "" {
		// Convert UID anf GID to int
		uid, err := strconv.Atoi(currentUser.Uid)
		if err != nil {
			logger.Error("Failed to parse user UID", "err", err)

			os.Exit(checkErr(fmt.Errorf("%w: %w", errUser, err)))
		}

		// gid, err := strconv.Atoi(currentUser.Gid)
		// if err != nil {
		// 	os.Exit(checkErr(fmt.Errorf("%w: failed to get current user gid: %w", errUser, err)))
		// }

		// Set UID and GID to current user
		err = syscall.Setuid(uid)
		if err != nil {
			logger.Error("Failed to set UID", "username", currentUser.Username, "uid", currentUser.Uid, "err", err)

			os.Exit(checkErr(fmt.Errorf("%w: failed to set current user uid: %w", errUser, err)))
		}

		// err = syscall.Setgid(gid)
		// if err != nil {
		// 	os.Exit(checkErr(fmt.Errorf("%w: failed to set current user gid: %w", errUser, err)))
		// }
	}

	logger = logger.With("uid", syscall.Getuid(), "euid", syscall.Geteuid(), "gid", syscall.Getgid(), "egid", syscall.Getegid())
	logger.Info("User context changed after setuid syscall")

	// Get stats
	units, usages, err := stats(logger, config, currentUser.Username, start, end, accounts, jobs, userNames, fields, summaryStats)
	if err != nil {
		os.Exit(checkErr(err))
	}

	// If no units found, exit, nothing more to do
	if len(units) == 0 {
		os.Exit(checkErr(errNoUnits))
	}

	// If instant queries have been configured, get results
	var instantQueryResults map[string]map[string]string

	if len(config.TSDB.instantQueries) > 0 {
		logger.Debug("Fetching instant queries results from TSDB")

		instantQueryResults, err = executeInstantQueries(logger, config, units)
		if err != nil {
			logger.Error("failed to fetch instant query results from TSDB", "err", err)
			fmt.Fprintln(os.Stderr, "failed to fetch metrics data")
		}
	}

	// If tsData is enabled, get time series data
	if len(config.TSDB.rangeQueries) > 0 {
		// If found jobs are more than 10, print a warning
		if len(units) > config.TSDB.MaxUnitsForRangeQueries {
			logger.Warn("Too many jobs to fetch time series data. Ignoring --ts.metrics flag", "num_units", len(units), "max_allowed_units", config.TSDB.MaxUnitsForRangeQueries)
			msg := fmt.Sprintf("too many jobs to fetch time series data. Please provide explicit job IDs (less than %d at a time) using --job when --ts.metrics is set", config.TSDB.MaxUnitsForRangeQueries)
			fmt.Fprintln(os.Stderr, msg)

			goto print_table
		}

		logger.Debug("Fetching time series data from TSDB")

		err := executeRangeQueries(logger, config, units, tsDataOut)
		if err != nil {
			logger.Error("failed to fetch time series data", "err", err)
			fmt.Fprintln(os.Stderr, "failed to fetch time series data")
		}
	}

print_table:
	// Print stats as table
	t := newTable(currentUser.Username, userNames, units, usages, instantQueryResults, activeInstantQueries, summaryStats)

	// Based on request rendering format
	switch {
	case htmlOut:
		t.RenderHTML()
	case csvOut:
		t.RenderCSV()
	case mdOut:
		t.RenderMarkdown()
	default:
		t.Render()
	}
}

// newTable returns a new table with data.
func newTable(currentUser string, users []string, units []models.Unit, usages []models.Usage, instantQueryResults map[string]map[string]string, activeInstantQueries []string, includeSummaryStats bool) table.Writer {
	// // Get current width
	// currentWidth := 180
	// // If we are in terminal override the default width with current width of terminal
	// if term.IsTerminal(0) {
	// 	width, _, err := term.GetSize(0)
	// 	if err == nil {
	// 		currentWidth = width
	// 	}
	// }

	// Make a new writer
	t := table.NewWriter()

	// Row config
	rowConfig := table.RowConfig{AutoMerge: true}

	// Table style
	style := table.Style{
		Name:    "CustomStyleLight",
		Box:     table.StyleBoxLight,
		Color:   table.ColorOptionsDefault,
		HTML:    table.DefaultHTMLOptions,
		Options: table.OptionsDefault,
		// Size: table.SizeOptions{
		// 	WidthMax: currentWidth,
		// 	WidthMin: 10,
		// },
		Title: table.TitleOptionsDefault,
		Format: table.FormatOptions{
			Footer: text.FormatDefault,
			Header: text.FormatUpper,
			Row:    text.FormatDefault,
		},
	}

	// Configure table
	var columnConfigs []table.ColumnConfig
	for _, field := range fieldMap {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:     field.title,
			WidthMin: field.minW,
			WidthMax: field.maxW,
		})
	}

	t.SuppressEmptyColumns()
	t.SuppressTrailingSpaces()
	t.SetStyle(style)
	t.SetOutputMirror(os.Stdout)
	t.SetColumnConfigs(columnConfigs)

	// Collect metric map's keys for each metric
	for _, unit := range units {
		updateField(unit.AveCPUUsage.Keys(), fieldMap["cpuusage"])
		updateField(unit.AveCPUMemUsage.Keys(), fieldMap["cpumemoryusage"])
		updateField(unit.TotalCPUEnergyUsage.Keys(), fieldMap["hostenergy"])
		updateField(unit.TotalCPUEmissions.Keys(), fieldMap["hostemissions"])
		updateField(unit.AveGPUUsage.Keys(), fieldMap["gpuusage"])
		updateField(unit.AveGPUMemUsage.Keys(), fieldMap["gpumemoryusage"])
		updateField(unit.TotalGPUEnergyUsage.Keys(), fieldMap["gpuenergy"])
		updateField(unit.TotalGPUEmissions.Keys(), fieldMap["gpuemissions"])
	}

	// Setup headers
	headers := table.Row{}
	subHeaders := table.Row{}

	for _, h := range allFields {
		headers = append(headers, fieldMap[h].titles()...)
		subHeaders = append(subHeaders, fieldMap[h].subtitles()...)
	}

	t.AppendHeader(headers, rowConfig)
	t.AppendHeader(subHeaders)

	// Append rows
	rows := make([]table.Row, len(units))

	for iunit, unit := range units {
		// Marshal total time and allocation
		var totalTime, allocation string

		if len(unit.TotalTime) > 0 {
			val, err := json.Marshal(unit.TotalTime)
			if err == nil {
				totalTime = string(val)
			}
		}

		if len(unit.Allocation) > 0 {
			val, err := json.Marshal(unit.Allocation)
			if err == nil {
				allocation = string(val)
			}
		}

		row := table.Row{
			unit.UUID, unit.Name, unit.Project, unit.Group, unit.User, unit.CreatedAt,
			unit.StartedAt, unit.EndedAt, unit.Elapsed, totalTime, allocation, unit.State,
		}
		row = append(row, unit.AveCPUUsage.Values("%.2f", len(fieldMap["cpuusage"].keys))...)
		row = append(row, unit.AveCPUMemUsage.Values("%.2f", len(fieldMap["cpumemoryusage"].keys))...)
		row = append(row, unit.TotalCPUEnergyUsage.Values("%f", len(fieldMap["hostenergy"].keys))...)
		row = append(row, unit.TotalCPUEmissions.Values("%f", len(fieldMap["hostemissions"].keys))...)
		row = append(row, unit.AveGPUUsage.Values("%.2f", len(fieldMap["gpuusage"].keys))...)
		row = append(row, unit.AveGPUMemUsage.Values("%.2f", len(fieldMap["gpumemoryusage"].keys))...)
		row = append(row, unit.TotalGPUEnergyUsage.Values("%f", len(fieldMap["gpuenergy"].keys))...)
		row = append(row, unit.TotalGPUEmissions.Values("%f", len(fieldMap["gpuemissions"].keys))...)

		// Add instant Query results to row
		for _, query := range activeInstantQueries {
			row = append(row, instantQueryResults[query][unit.UUID])
		}

		rows[iunit] = row
	}

	t.AppendRows(rows)

	if includeSummaryStats {
		// Append summary row
		t.AppendSeparator()

		summaryRow := table.Row{"Summary"}
		for range headers {
			summaryRow = append(summaryRow, "")
		}

		t.AppendRow(summaryRow, rowConfig)
		t.AppendSeparator()

		for _, usage := range usages {
			if usage.User == currentUser || slices.Contains(users, usage.User) || slices.Contains(users, "all") {
				// Check if elapsed time in non zero
				var totalElapsedTime string
				if usage.TotalTime["walltime"] > 0 {
					totalElapsedTime = common.Timespan(time.Duration(usage.TotalTime["walltime"]) * time.Second).Format("15:04:05")
				}

				// Usage row
				row := table.Row{
					usage.NumUnits, "", usage.Project, usage.Group, usage.User, "", "", "", totalElapsedTime, "", "", "",
				}
				row = append(row, usage.AveCPUUsage.Values("%.2f", len(fieldMap["cpuusage"].keys))...)
				row = append(row, usage.AveCPUMemUsage.Values("%.2f", len(fieldMap["cpumemoryusage"].keys))...)
				row = append(row, usage.TotalCPUEnergyUsage.Values("%f", len(fieldMap["hostenergy"].keys))...)
				row = append(row, usage.TotalCPUEmissions.Values("%f", len(fieldMap["hostemissions"].keys))...)
				row = append(row, usage.AveGPUUsage.Values("%.2f", len(fieldMap["gpuusage"].keys))...)
				row = append(row, usage.AveGPUMemUsage.Values("%.2f", len(fieldMap["gpumemoryusage"].keys))...)
				row = append(row, usage.TotalGPUEnergyUsage.Values("%f", len(fieldMap["gpuenergy"].keys))...)
				row = append(row, usage.TotalGPUEmissions.Values("%f", len(fieldMap["gpuemissions"].keys))...)

				// Append instant query results columns as "N/A"
				for range activeInstantQueries {
					row = append(row, "N/A")
				}

				// Append row to table
				t.AppendFooter(row)
			}
		}
	}

	return t
}

// updateField updates field keys with the ones found struct field keys.
func updateField(structKeys []string, f *field) {
	for _, k := range structKeys {
		if !slices.Contains(f.keys, k) {
			f.keys = append(f.keys, k)
		}
	}
}

// readConfig returns config struct from first found config file.
func readConfig(mockConfigPath string) (*Config, error) {
	var config Config

	// If mockConfigPath is set as well, add to configPaths
	// Do not override configPaths because if there is an existing config
	// sitting somewhere, we should give priority to it rather than the
	// mock config
	if mockConfigPath != "" {
		configPaths = append(configPaths, mockConfigPath)
	}

	// Look for config.yml or config.yaml or cacct.yml or cacct.yaml files
	for _, configPath := range configPaths {
		for _, file := range []string{"config.yml", "config.yaml", "cacct.yml", "cacct.yaml"} {
			configFile := filepath.Join(configPath, file)

			_, err := os.Stat(configFile)
			if err == nil {
				// Read config file
				cfg, err := os.ReadFile(configFile)
				if err != nil {
					return nil, err
				}

				err = yaml.Unmarshal(cfg, &config) //nolint: musttag
				if err != nil {
					return nil, err
				}

				return &config, nil
			}
		}
	}

	return nil, errConfig
}

// getCurrentUser returns the actual user executing the cacct. If --current-user
// CLI flag is passed, that user will be returned as current user.
func getCurrentUser(mockUserName string) (*user.User, error) {
	// Get current user is who is executing cacct
	var currentUser *user.User

	// Get effective UID as cacct is a setuid binary
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	} else {
		// Check if mockUserName is set. This will be always empty string
		// for production builds as we do not compile flags for production
		// builds
		if mockUserName != "" {
			currentUser = &user.User{Username: mockUserName}
		} else {
			currentUser = u
		}
	}

	// // Add user HOME to configPaths
	// userConfigDir, err := os.UserConfigDir()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get config file: %w", err)
	// }

	// configPaths = append(configPaths, filepath.Join(userConfigDir, "ceems"))

	return currentUser, nil
}

func parseTime(s string) (time.Time, error) {
	// First attempt is to parse as YYYY-MM-DDTHH:MM:SS
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local)
	if err == nil {
		return t, nil
	}

	// Second attempt is to parse as YYYY-MM-DDTHH:MM
	t, err = time.ParseInLocation("2006-01-02T15:04", s, time.Local)
	if err == nil {
		return t, nil
	}

	// Third attempt is to parse as YYYY-MM-DD
	t, err = time.ParseInLocation("2006-01-02", s, time.Local)
	if err == nil {
		return t, nil
	}

	// If nothing works, return error
	return time.Time{}, errors.New("invalid time format")
}

func sortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0

	for k := range m {
		keys[i] = k
		i++
	}

	slices.Sort(keys)

	return keys
}

func splitString(s, d string) []string { //nolint:unparam
	var parts []string

	for p := range strings.SplitSeq(s, d) {
		if p != "" {
			parts = append(parts, p)
		}
	}

	return parts
}

func checkErr(err error) int {
	if err != nil {
		switch {
		case errors.Is(err, errNoPerm):
			fmt.Fprintln(os.Stderr, "error: forbidden. It is likely that the user is attempting to view statistics of others")
		case errors.Is(err, errInternal):
			fmt.Fprintln(os.Stderr, "error: server did not return any data due to unknown error")
		case errors.Is(err, errConfig):
			fmt.Fprintln(os.Stderr, "error: "+errConfig.Error())
		case errors.Is(err, errUser):
			fmt.Fprintln(os.Stderr, "error: "+errUser.Error())
		case errors.Is(err, errNoUnits):
			fmt.Fprintln(os.Stderr, "error: "+errNoUnits.Error())
		default:
			fmt.Fprintln(os.Stderr, "error: internal error")
		}

		return 1
	}

	return 0
}
