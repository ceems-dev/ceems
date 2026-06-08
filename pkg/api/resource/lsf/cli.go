package lsf

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ceems-dev/ceems/internal/common"
	"github.com/ceems-dev/ceems/internal/osexec"
	"github.com/ceems-dev/ceems/pkg/api/base"
	"github.com/ceems-dev/ceems/pkg/api/helper"
	"github.com/ceems-dev/ceems/pkg/api/models"
)

var (
	// LSF accounting gives memory as 200M, 250.5G and we dont know if it gives without
	// units. However, bjobs return memory as 200 Mbytes.
	// So, regex will capture the number and unit (if exists) and we convert it to bytes.
	// Ref: https://regex101.com/r/ate08j/1
	memRegex = regexp.MustCompile(`([0-9.]+)(?:\s?)([K|M|G|T]?)`)
	toBytes  = map[string]int64{
		"K": 1024,
		"M": 1024 * 1024,
		"G": 1024 * 1024 * 1024,
		"T": 1024 * 1024 * 1024 * 1024,
		"Z": 1024 * 1024 * 1024 * 1024 * 1024,
	}
)

// Run preflights for CLI execution mode.
func preflightsCLI(lsf *lsfScheduler) error {
	// We hit this only when fetch mode is bacct and bjobs commands
	// Assume execMode is always native
	lsf.fetchMode = cliMode
	lsf.logger.Debug("Using LSF CLI commands")

	// If no bacct path is provided, assume it is available on PATH
	if lsf.cluster.CLI.Path == "" {
		path, err := exec.LookPath("bacct")
		if err != nil {
			lsf.logger.Error("Failed to find LSF utility executables on PATH", "err", err)

			return err
		}

		lsf.cluster.CLI.Path = filepath.Dir(path)
	} else {
		// Check if lsf binary directory exists at the given path
		_, err := os.Stat(lsf.cluster.CLI.Path)
		if err != nil {
			lsf.logger.Error("Failed to open LSF bin dir", "path", lsf.cluster.CLI.Path, "err", err)

			return err
		}
	}

	return nil
}

// Parse bacct command output and return batchjob slice.
func parseBacctCmdOutput(bacctOutput string, start time.Time, end time.Time, jobsCache *sync.Map) []models.Unit {
	// Get current location
	loc := end.Location()

	// Update period
	intStartTS := start.UnixMilli()
	intEndTS := end.UnixMilli()

	// Start with a default delimiter
	delimiter := "------------------------------------------------------------------------------"
	// Find the line separator
	sepMatches := regexSeparator.FindAllStringSubmatch(bacctOutput, 1)
	if len(sepMatches) > 0 && len(sepMatches[0]) > 0 {
		delimiter = sepMatches[0][0]
	}

	// Split output to get per job record
	bacctOutputRecords := strings.Split(bacctOutput, delimiter)

	jobs := make([]models.Unit, len(bacctOutputRecords))

	wg := &sync.WaitGroup{}
	wg.Add(len(bacctOutputRecords))

	for irecord, record := range bacctOutputRecords {
		go func(i int, r string) {
			var jobStat models.Unit

			matches := regexJobDetails.FindStringSubmatch(r)

			// Ignore if no matches found
			if len(matches) == 0 {
				wg.Done()

				return
			}

			// Get all named captured groups
			components := make(map[string]string)
			for i, name := range regexJobDetails.SubexpNames() {
				components[name] = matches[i]
			}

			// Ignore jobs that never ran
			if components["start_time"] == "" {
				wg.Done()

				return
			}

			// Get user's job group, ncpus, ngpus from cache map that we fetched from bjobs
			// If we managed to parse bacct output, the nnodes, ncpus, ngpus, mem and nodelist
			// will be overwritten by bacct output. This is desirable as jobs can be
			// modified between last scrape and its termination. So, we get last updated
			// job info
			components["group"] = ""

			var (
				nnodes, ncpus, ngpus int
				nodelist             string
				mem                  int64
			)

			if jobsCache != nil {
				if val, ok := jobsCache.LoadAndDelete(components["jobid"]); ok {
					if u, ok := val.(jobAttributes); ok {
						components["group"] = u.userGroup
						nnodes = u.numNodes
						ncpus = u.numCPUs
						ngpus = u.numGPUs
						mem = u.mem
						nodelist = u.nodelist
					}
				}
			}

			// Convert time strings to configured time location
			eventTS := make(map[string]int64, 3)

			for _, c := range []string{"submit_time", "start_time", "end_time"} {
				// bacct does not include tz in the time format. So, always assume the
				// location to be the current one that API server is running on.
				t, err := time.ParseInLocation(bacctTimeLayoout, components[c], loc)
				if err == nil {
					components[c] = t.Format(base.DatetimezoneLayout)
				}

				eventTS[c] = helper.TimeToTimestamp(base.DatetimezoneLayout, components[c])
			}

			// Get CPU allocations
			allocCPUs := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(components["nodelist"], "<", ""), ">", ""))
			if len(allocCPUs) > 0 {
				ncpus = len(allocCPUs)

				// Remove duplicates to get allocated nodes
				slices.Sort(allocCPUs)
				allocNodes := slices.Compact(allocCPUs)
				nnodes = len(allocNodes)
				nodelist = strings.Join(allocNodes, "|")
			}

			// Get memory allocations
			var (
				memString string
				memEff    float64
			)

			for l := range strings.SplitSeq(components["accounting"], "\n") {
				// Split line into columns
				fields := strings.Fields(strings.TrimSpace(l))

				// If there are 7 columns that 5th one is the memory usage
				if len(fields) == 7 && !strings.Contains(l, "MEM") {
					memString = fields[5]
				} else if len(fields) == 2 && !strings.Contains(l, "MEM_EFFICIENCY") {
					memEff, _ = strconv.ParseFloat(strings.TrimSuffix(fields[1], "%"), 64)
				}
			}

			// If mem is not empty string, convert the units [K|M|G|T] into numeric bytes
			// The following logic covers the cases when memory is of form 200M, 250.5G
			// and also without unit eg 20000, 40000. When there is no unit we assume
			// it is already in bytes
			memStringMatches := memRegex.FindStringSubmatch(memString)

			if len(memStringMatches) >= 2 {
				memFloat, err := strconv.ParseFloat(memStringMatches[1], 64)
				if err == nil {
					if len(memStringMatches) == 3 {
						if unitConv, ok := toBytes[memStringMatches[2]]; ok {
							mem = int64(memFloat) * unitConv

							// If memEff is not zero, convert real memory usage
							// into memory allocation
							if memEff > 0 {
								mem = int64(100.0 * float64(mem) / memEff)
							}
						}
					}
				}
			}

			var (
				allocGPUs          []string
				hostname, gpuIndex string
			)

			for l := range strings.SplitSeq(components["gpu_alloc"], "\n") {
				// Ignore empty and header lines
				if l == "" || strings.Contains(l, "HOST") {
					continue
				}

				// Split line into columns
				fields := strings.Fields(strings.TrimSpace(l))

				// If there are 11 columns, there is a hostname in the column and GPU
				// ordinal is at 3rd position.
				// If there is no hostname, we need to use previously found hostname and
				// GPU ordinal will be at 2nd position.
				if len(fields) == 11 {
					hostname = fields[0]
					gpuIndex = fields[2]
				} else if len(fields) == 10 {
					gpuIndex = fields[1]
				}

				allocGPUs = append(allocGPUs, fmt.Sprintf("%s:%s", hostname, gpuIndex))
			}

			// Remove duplicates
			if len(allocGPUs) > 0 {
				slices.Sort(allocGPUs)
				allocGPUs = slices.Compact(allocGPUs)
				ngpus = len(allocGPUs)
			}

			// Assume job's elapsed time during this interval overlaps with interval's
			// boundaries
			startMark := intStartTS
			endMark := intEndTS

			// If job has already finished in the past we need to get boundaries from
			// job's start and end time. This case should not arrive in production as
			// there is no reason LSF gives us the jobs that have finished in the past
			// that do not overlap with interval boundaries
			if eventTS["end_time"] > 0 && eventTS["end_time"] < intStartTS {
				startMark = eventTS["start_time"]
				endMark = eventTS["end_time"]

				goto elapsed
			}

			// If job has started **after** start of interval, we should mark job's start
			// time as start of elapsed time
			if eventTS["start_time"] > intStartTS {
				startMark = eventTS["start_time"]
			}

			// If job has ended before end of interval, we should mark job's end time
			// as elapsed end time.
			if eventTS["end_time"] > 0 && eventTS["end_time"] < intEndTS {
				endMark = eventTS["end_time"]
			}

		elapsed:
			// Get elapsed time of job in this interval in seconds
			elapsedSeconds := (endMark - startMark) / 1000

			// Get cpuSeconds and gpuSeconds of the current interval
			var cpuSeconds, gpuSeconds int64

			cpuSeconds = int64(ncpus) * elapsedSeconds
			gpuSeconds = int64(ngpus) * elapsedSeconds

			// Get cpuMemSeconds and gpuMemSeconds of current interval in MB
			var cpuMemSeconds, gpuMemSeconds int64
			if mem > 0 {
				cpuMemSeconds = mem * elapsedSeconds / toBytes["M"]
			} else {
				cpuMemSeconds = elapsedSeconds
			}

			// Currently we use walltime as GPU mem time. This wont be a correct proxy
			// if MIG is enabled in GPUs where different portions of memory can be
			// allocated
			if ngpus > 0 {
				gpuMemSeconds = elapsedSeconds
			}

			// Allocation
			allocation := models.Allocation{
				"nodes": nnodes,
				"cpus":  ncpus,
				"mem":   mem,
				"gpus":  ngpus,
			}

			// Tags
			tags := models.Tag{
				"queue":       components["queue"],
				"res_req":     components["res_req"],
				"exit_status": components["exit_status"],
				"nodelist":    nodelist,
				"workdir":     components["cwd"],
			}

			// Make jobStats struct for each job and put it in jobs slice
			jobStat = models.Unit{
				ResourceManager: lsfBatchScheduler,
				UUID:            components["jobid"],
				Project:         components["project"],
				User:            components["user"],
				Group:           components["group"],
				CreatedAt:       components["submit_time"],
				StartedAt:       components["start_time"],
				EndedAt:         components["end_time"],
				CreatedAtTS:     eventTS["submit_time"],
				StartedAtTS:     eventTS["start_time"],
				EndedAtTS:       eventTS["end_time"],
				Elapsed:         common.Timespan(time.Duration(eventTS["end_time"]-eventTS["start_time"]) * time.Second / 1e3).Format("15:04:05"),
				State:           components["status"],
				Allocation:      allocation,
				TotalTime: models.MetricMap{
					"walltime":         models.JSONFloat(elapsedSeconds),
					"alloc_cputime":    models.JSONFloat(cpuSeconds),
					"alloc_cpumemtime": models.JSONFloat(cpuMemSeconds),
					"alloc_gputime":    models.JSONFloat(gpuSeconds),
					"alloc_gpumemtime": models.JSONFloat(gpuMemSeconds),
				},
				Tags: tags,
			}

			jobLock.Lock()

			jobs[i] = jobStat

			jobLock.Unlock()
			wg.Done()
		}(irecord, record)
	}

	wg.Wait()

	// Remove empty jobs
	var validJobs []models.Unit

	for _, job := range jobs {
		if job.UUID != "" {
			validJobs = append(validJobs, job)
		}
	}

	return validJobs
}

// runBacctCmd executes bacct command and return output.
func (s *lsfScheduler) runBacctCmd(ctx context.Context, start, end time.Time) ([]byte, error) {
	// bacct path
	bacctPath := filepath.Join(s.cluster.CLI.Path, "bacct")

	// Setup any provided env vars
	var env []string
	for name, value := range s.cluster.CLI.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", name, value))
	}

	// Get jobs of ALL users in machine readable format
	args := []string{
		"-u", "all", "-l", "-gpu", "-UF",
		"-C", fmt.Sprintf("%s,%s", start.Format(lsfCommandsTimeLayout), end.Format(lsfCommandsTimeLayout)),
	}

	return osexec.ExecuteContext(ctx, bacctPath, args, env)
}

// Parse bjobs command output and return batchjob slice.
func parseBjobsCmdOutput(bjobsOutput []byte, start time.Time, end time.Time, jobsCache *sync.Map) ([]models.Unit, int, error) {
	// Get current location
	loc := end.Location()

	// Update period
	intStartTS := start.UnixMilli()
	intEndTS := end.UnixMilli()

	// Unmarshal command output into LSFJobsList struct
	var bjobs common.LSFJobsList

	err := json.Unmarshal(bjobsOutput, &bjobs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal bjobs output: %w", err)
	}

	jobs := make([]models.Unit, len(bjobs.Records))
	numJobs := 0

	wg := &sync.WaitGroup{}
	wg.Add(len(bjobs.Records))

	for irecord, record := range bjobs.Records {
		go func(i int, r common.LSFJobRecord) {
			var jobStat models.Unit

			// If job state is not running, ignore this job
			// Job states: https://www.ibm.com/docs/en/spectrum-lsf/10.1.0?topic=execution-about-job-states
			if r.Stat != "RUN" {
				wg.Done()

				return
			}

			// Convert time strings to configured time location
			eventTS := make(map[string]int64, 2)

			// Components map to save all the attributes
			components := map[string]string{
				"submit_time": r.SubmitTime,
				"start_time":  r.StartTime,
				"end_time":    r.FinishTime,
			}

			for c, ts := range components {
				// bjobs does not include tz in the time format. So, always assume the
				// location to be the current one that API server is running on.
				t, err := time.ParseInLocation(bjobsTimeLayoout, ts, loc)
				if err == nil {
					components[c] = t.Format(base.DatetimezoneLayout)
				} else {
					components[c] = "N/A"
				}

				eventTS[c] = helper.TimeToTimestamp(base.DatetimezoneLayout, components[c])
			}

			// If job has already finished in the past bjobs should not return this job.
			// In case if it does, just ignore it
			if eventTS["end_time"] > 0 && eventTS["end_time"] < intStartTS {
				wg.Done()

				return
			}

			// Get allocated nodes
			allocCPUs := strings.Split(r.AllocSlot, ":")
			slices.Sort(allocCPUs)
			allocNodes := slices.Compact(allocCPUs)
			ncpus := r.NumAllocSlot

			// If mem is not empty string, convert the units [K|M|G|T] into numeric bytes
			// The following logic covers the cases when memory is of form 200 Mbytes, 250.5 Gbtyes
			// and also without unit eg 20000, 40000. When there is no unit we assume
			// it is already in bytes
			var mem int64

			memStringMatches := memRegex.FindStringSubmatch(r.Mem)
			if len(memStringMatches) >= 2 {
				// Get used memory
				memUsed, err := strconv.ParseFloat(memStringMatches[1], 64)
				if err == nil {
					// If units are present, convert to bytes
					if len(memStringMatches) == 3 {
						if unitConv, ok := toBytes[memStringMatches[2]]; ok {
							memUsed = memUsed * float64(unitConv)
						}
					}

					// Get memory efficiency
					memEff, _ := strconv.ParseFloat(strings.TrimSuffix(r.MemEfficiency, "%"), 64)

					// If memory is greater than zero, we can multiply used memory with
					// efficiency to get reserved memory
					if memEff > 0 {
						mem = int64(100 * memUsed / memEff)
					} else {
						mem = int64(memUsed)
					}
				}
			}

			// Get number of GPUs
			ngpus := r.NumGPU

			// Get elapsed time
			// bjobs output returns in format 50 second(s). Need to strip second(s)
			// from string before converting to int
			components["elapsed_time"] = ""

			v, err := strconv.ParseInt(strings.TrimSpace(strings.TrimSuffix(r.Runtime, "second(s)")), 10, 64)
			if err == nil {
				components["elapsed_time"] = common.Timespan(time.Duration(v) * time.Second).Format("15:04:05")
			}

			// Assume job's elapsed time during this interval overlaps with interval's
			// boundaries
			startMark := intStartTS
			endMark := intEndTS

			// If job has started **after** start of interval, we should mark job's start
			// time as start of elapsed time
			if eventTS["start_time"] > intStartTS {
				startMark = eventTS["start_time"]
			}

			// If job has ended before end of interval, we should mark job's end time
			// as elapsed end time.
			if eventTS["end_time"] > 0 && eventTS["end_time"] < intEndTS {
				endMark = eventTS["end_time"]
			}

			// Get elapsed time of job in this interval in seconds
			elapsedSeconds := (endMark - startMark) / 1000

			// Get cpuSeconds and gpuSeconds of the current interval
			var cpuSeconds, gpuSeconds int64

			cpuSeconds = int64(ncpus) * elapsedSeconds
			gpuSeconds = int64(ngpus) * elapsedSeconds

			// Get cpuMemSeconds and gpuMemSeconds of current interval in MB
			var cpuMemSeconds, gpuMemSeconds int64
			if mem > 0 {
				cpuMemSeconds = mem * elapsedSeconds / toBytes["M"]
			} else {
				cpuMemSeconds = elapsedSeconds
			}

			// Currently we use walltime as GPU mem time. This wont be a correct proxy
			// if MIG is enabled in GPUs where different portions of memory can be
			// allocated
			if ngpus > 0 {
				gpuMemSeconds = elapsedSeconds
			}

			// Allocation
			allocation := models.Allocation{
				"nodes": len(allocNodes),
				"cpus":  ncpus,
				"mem":   mem,
				"gpus":  ngpus,
			}

			// Tags
			tags := models.Tag{
				"queue":    r.Queue,
				"res_req":  r.EffectiveResReq,
				"nodelist": strings.Join(allocNodes, "|"),
				"workdir":  r.ExecCWD,
			}

			// Make jobStats struct for each job and put it in jobs slice
			jobStat = models.Unit{
				ResourceManager: lsfBatchScheduler,
				UUID:            r.ID,
				Project:         r.Project,
				User:            r.User,
				Group:           r.UserGroup,
				CreatedAt:       components["submit_time"],
				StartedAt:       components["start_time"],
				EndedAt:         components["end_time"],
				CreatedAtTS:     eventTS["submit_time"],
				StartedAtTS:     eventTS["start_time"],
				EndedAtTS:       eventTS["end_time"],
				Elapsed:         components["elapsed_time"],
				State:           r.Stat,
				Allocation:      allocation,
				TotalTime: models.MetricMap{
					"walltime":         models.JSONFloat(elapsedSeconds),
					"alloc_cputime":    models.JSONFloat(cpuSeconds),
					"alloc_cpumemtime": models.JSONFloat(cpuMemSeconds),
					"alloc_gputime":    models.JSONFloat(gpuSeconds),
					"alloc_gpumemtime": models.JSONFloat(gpuMemSeconds),
				},
				Tags: tags,
			}

			// Push jobStat struct into jobsCache to be used in bacct parsing
			jobAttr := jobAttributes{
				numNodes:  len(allocNodes),
				numCPUs:   ncpus,
				numGPUs:   ngpus,
				userGroup: r.UserGroup,
				nodelist:  strings.Join(allocNodes, "|"),
				mem:       mem,
			}
			if _, exists := jobsCache.LoadOrStore(r.ID, jobAttr); exists {
				jobsCache.Swap(r.ID, jobAttr)
			}

			jobLock.Lock()

			jobs[i] = jobStat
			numJobs += 1

			jobLock.Unlock()
			wg.Done()
		}(irecord, record)
	}

	wg.Wait()

	return jobs, numJobs, nil
}

// runBjobsCmd executes bjobs command and return output.
func (s *lsfScheduler) runBjobsCmd(ctx context.Context) ([]byte, error) {
	// bjobs path
	bjobsPath := filepath.Join(s.cluster.CLI.Path, "bjobs")

	// Setup any provided env vars
	var env []string
	for name, value := range s.cluster.CLI.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", name, value))
	}

	// Get jobs of ALL users in machine readable format
	args := []string{
		"-u", "all", "-json", "-o", "jobid jobindex user user_group queue job_name proj_name stat alloc_slot nalloc_slot submit_time start_time finish_time run_time effective_resreq gpu_num gpu_alloc mem memlimit mem_efficiency exec_cwd",
	}

	return osexec.ExecuteContext(ctx, bjobsPath, args, env)
}

// Run preflight checks on provided config.
func preflightChecks(s *lsfScheduler) error {
	// // Always prefer REST API mode if configured
	// if clusterConfig.Web.URL != "" {
	// 	return checkRESTAPI(clusterConfig, logger)
	// }
	return preflightsCLI(s)
}
