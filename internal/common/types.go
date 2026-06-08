package common

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	config_util "github.com/prometheus/common/config"
)

// Regex to extract job index from job ID.
var jobIndexRegex = regexp.MustCompile(`^(?P<id>[0-9]+)(?:\[(?P<index>[0-9]+)\])?$`)

// GrafanaWebConfig makes HTTP Grafana config.
type GrafanaWebConfig struct {
	URL              string                       `yaml:"url"`
	TeamsIDs         []string                     `yaml:"teams_ids"`
	HTTPClientConfig config_util.HTTPClientConfig `yaml:",inline"`
}

// LSFJobRecord contains job related infor for each job.
type LSFJobRecord struct {
	ID              string `json:"JOBID"`
	User            string `json:"USER"`
	UserGroup       string `json:"USER_GROUP"`
	Queue           string `json:"QUEUE"`
	JobName         string `json:"JOB_NAME"`
	Project         string `json:"PROJ_NAME"`
	Stat            string `json:"STAT"`
	SubmitTime      string `json:"SUBMIT_TIME"`
	StartTime       string `json:"START_TIME"`
	FinishTime      string `json:"FINISH_TIME"`
	Runtime         string `json:"RUN_TIME"`
	EffectiveResReq string `json:"EFFECTIVE_RESREQ"`
	AllocSlot       string `json:"ALLOC_SLOT"`
	GPUSlot         string `json:"GPU_ALLOC"`
	Mem             string `json:"MEM"`
	MemLimit        string `json:"MEMLIMIT"`
	MemEfficiency   string `json:"MEM_EFFICIENCY"`
	ExecCWD         string `json:"EXEC_CWD"`
	NumAllocSlot    int
	NumGPU          int
}

// UnmarshalJSON unmarshals byte array into LSFJobRecord.
func (r *LSFJobRecord) UnmarshalJSON(b []byte) error {
	// Define a temporary type to avoid infinite looping
	type LSFJobRecordTmp LSFJobRecord

	type tmp struct {
		LSFJobRecordTmp

		Index           string `json:"JOBINDEX"`
		NumAllocSlotStr string `json:"NALLOC_SLOT"`
		NumGPUStr       string `json:"GPU_NUM"`
	}

	var s tmp

	err := json.Unmarshal(b, &s) //nolint: musttag
	if err != nil {
		return err
	}

	*r = LSFJobRecord(s.LSFJobRecordTmp)

	// If Index exists, attach it to job ID. So, if id is 6 and index is 2,
	// the final id will be "6[2]" as appaears in cgroup paths
	//
	// For regular jobs, index will be always 0 and we should not add the index when
	// it is "0". Job Arrays must be start with index >= 1.
	if s.Index != "" && s.Index != "0" {
		r.ID = fmt.Sprintf("%s[%s]", r.ID, s.Index)
	}

	// If NumAllocSlotStr is not empty, cast it to int
	if s.NumAllocSlotStr != "" {
		v, err := strconv.ParseInt(s.NumAllocSlotStr, 10, 64)
		if err == nil {
			r.NumAllocSlot = int(v)
		}
	}

	// If NumGPUStr is not empty, cast it to int
	if s.NumGPUStr != "" {
		v, err := strconv.ParseInt(s.NumGPUStr, 10, 64)
		if err == nil {
			r.NumGPU = int(v)
		}
	}

	return nil
}

// MarshalJSON marshals LSFJobRecord into byte array.
func (r LSFJobRecord) MarshalJSON() ([]byte, error) {
	// Define a temporary type to avoid infinite looping
	type LSFJobRecordTmp LSFJobRecord

	// Another tmp struct that contains job index
	type tmp struct {
		LSFJobRecordTmp

		Index string `json:"JOBINDEX"`
	}

	var s tmp

	s.LSFJobRecordTmp = LSFJobRecordTmp(r)

	// If jobID has index inside it, extract it
	match := jobIndexRegex.FindStringSubmatch(r.ID)
	// If no matches found, return
	if len(match) > 0 {
		// Get index of the job
		for i, name := range jobIndexRegex.SubexpNames() {
			if name == "id" {
				s.ID = strings.TrimSpace(match[i])
			}

			if name == "index" {
				s.Index = strings.TrimSpace(match[i])
			}
		}
	}

	return json.Marshal(s) //nolint: musttag
}

// LSFJobsList contains list of all job records.
type LSFJobsList struct {
	Command string         `json:"COMMAND"`
	NumJobs int            `json:"JOBS"`
	Records []LSFJobRecord `json:"RECORDS"`
}
