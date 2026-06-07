package lsf

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreflightsCLI(t *testing.T) {
	manager := lsfScheduler{
		logger: noOpLogger,
	}
	err := preflightsCLI(&manager)
	require.Error(t, err)

	// Add bacct command to PATH
	bacctPath, _ := filepath.Abs("../../testdata")
	t.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), bacctPath))

	err = preflightsCLI(&manager)
	require.NoError(t, err)
	assert.Equal(t, bacctPath, manager.cluster.CLI.Path)
}

func TestParseBacctCmdOutput(t *testing.T) {
	tests := []struct {
		name                 string
		bacctOutput          string
		walltime             float64
		nnodes, ncpus, ngpus int
		mem                  int64
	}{
		{
			name: "Job finished in past with GPUs",
			bacctOutput: `
Job <3>, User <usr3>, Project <prj3>, Status <DONE>, Queue <interactive>, Interactive pseudo-terminal shell mode, Command <bash>, Share group charged </usr3>
Sun Apr 12 17:23:38 2026: Submitted from host <nodes-1.example.com>, CWD <$HOME>;
Sun Apr 12 17:23:39 2026: Dispatched 4 Task(s) on Host(s) <nodes-2.example.com> <nodes-2.example.com> <nodes-1.example.com> <nodes-1.example.com>, Allocated 4 Slot(s) on Host(s) <nodes-2.example.com> <nodes-2.example.com> <nodes-1.example.com> <nodes-1.example.com>, Effective RES_REQ <select[(type == any ) && (ngpus>0)] order[r15s:pg] rusage[ngpus_physical=2.00/host] span[ptile=2] >;
Sun Apr 12 17:24:14 2026: Completed <done>.

GPU_ALLOCATION:
 HOST             TASK GPU_ID  GI_PLACEMENT/SIZE    CI_PLACEMENT/SIZE    MODEL        MTOTAL  FACTOR MRSV    SOCKET NVLINK/XGMI
 nodes-1.example  0    0       -                    -                    RadeonInstin 31.9G   0.0    0M      0      -
                  1    1       -                    -                    RadeonInstin 31.9G   0.0    0M      0      -
 nodes-2.example  0    0       -                    -                    RadeonInstin 31.9G   0.0    0M      0      -
                  1    1       -                    -                    RadeonInstin 31.9G   0.0    0M      0      -

Accounting information about this job:
     Share group charged </usr3>
     CPU_T     WAIT     TURNAROUND   STATUS     HOG_FACTOR    MEM    SWAP
      0.22        1             36     done         0.0060    11M      0M
     CPU_PEAK     CPU_PEAK_DURATION     CPU_PEAK_EFFICIENCY
         0.00           0 second(s)                   0.00%
     CPU_AVERAGE_EFFICIENCY      MEM_EFFICIENCY
                      0.00%               0.00%
------------------------------------------------------------------------------`,
			walltime: 35,
			nnodes:   2,
			ncpus:    4,
			ngpus:    4,
			mem:      11534336,
		},
		{
			name: "Job started inside current interval without GPUs",
			bacctOutput: `
Job <3>, User <usr3>, Project <prj3>, Status <DONE>, Queue <interactive>, Interactive pseudo-terminal shell mode, Command <bash>, Share group charged </usr3>
Sun Apr 12 18:23:38 2026: Submitted from host <nodes-1.example.com>, CWD <$HOME>;
Sun Apr 12 18:23:39 2026: Dispatched 4 Task(s) on Host(s) <nodes-2.example.com> <nodes-2.example.com> <nodes-1.example.com> <nodes-1.example.com>, Allocated 4 Slot(s) on Host(s) <nodes-2.example.com> <nodes-2.example.com> <nodes-1.example.com> <nodes-1.example.com>, Effective RES_REQ <select[(type == any ) && (ngpus>0)] order[r15s:pg] rusage[ngpus_physical=2.00/host] span[ptile=2] >;
Sun Apr 12 19:24:14 2026: Completed <done>.

Accounting information about this job:
     Share group charged </usr3>
     CPU_T     WAIT     TURNAROUND   STATUS     HOG_FACTOR    MEM    SWAP
      0.22        1             36     done         0.0060    11M      0M
     CPU_PEAK     CPU_PEAK_DURATION     CPU_PEAK_EFFICIENCY
         0.00           0 second(s)                   0.00%
     CPU_AVERAGE_EFFICIENCY      MEM_EFFICIENCY
                      0.00%               0.00%
------------------------------------------------------------------------------`,
			walltime: 381,
			ncpus:    4,
			ngpus:    0,
			mem:      11534336,
			nnodes:   2,
		},
		{
			name: "Job ended inside current interval without GPUs and non zero memory efficiency",
			bacctOutput: `
Job <3>, User <usr3>, Project <prj3>, Status <DONE>, Queue <interactive>, Interactive pseudo-terminal shell mode, Command <bash>, Share group charged </usr3>
Sun Apr 12 17:23:38 2026: Submitted from host <nodes-1.example.com>, CWD <$HOME>;
Sun Apr 12 17:23:39 2026: Dispatched 4 Task(s) on Host(s) <nodes-2.example.com> <nodes-2.example.com> <nodes-1.example.com> <nodes-1.example.com>, Allocated 4 Slot(s) on Host(s) <nodes-2.example.com> <nodes-2.example.com> <nodes-1.example.com> <nodes-1.example.com>, Effective RES_REQ <select[(type == any ) && (ngpus>0)] order[r15s:pg] rusage[ngpus_physical=2.00/host] span[ptile=2] >;
Sun Apr 12 18:24:14 2026: Completed <done>.

Accounting information about this job:
     Share group charged </usr3>
     CPU_T     WAIT     TURNAROUND   STATUS     HOG_FACTOR    MEM    SWAP
      0.22        1             36     done         0.0060    11M      0M
     CPU_PEAK     CPU_PEAK_DURATION     CPU_PEAK_EFFICIENCY
         0.00           0 second(s)                   0.00%
     CPU_AVERAGE_EFFICIENCY      MEM_EFFICIENCY
                      0.00%               5.00%
------------------------------------------------------------------------------`,
			walltime: 554,
			ncpus:    4,
			ngpus:    0,
			mem:      230686720,
			nnodes:   2,
		},
		{
			name: "Job started and ended inside current interval without GPUs and non zero memory efficiency",
			bacctOutput: `
Job <3>, User <usr3>, Project <prj3>, Status <DONE>, Queue <interactive>, Interactive pseudo-terminal shell mode, Command <bash>, Share group charged </usr3>
Sun Apr 12 18:23:38 2026: Submitted from host <nodes-1.example.com>, CWD <$HOME>;
Sun Apr 12 18:23:39 2026: Dispatched 2 Task(s) on Host(s) <nodes-2.example.com> <nodes-1.example.com>, Allocated 2 Slot(s) on Host(s) <nodes-2.example.com> <nodes-1.example.com>, Effective RES_REQ <select[(type == any ) && (ngpus>0)] order[r15s:pg] rusage[ngpus_physical=2.00/host] span[ptile=2] >;
Sun Apr 12 18:24:14 2026: Completed <done>.

Accounting information about this job:
     Share group charged </usr3>
     CPU_T     WAIT     TURNAROUND   STATUS     HOG_FACTOR    MEM    SWAP
      0.22        1             36     done         0.0060    11M      0M
     CPU_PEAK     CPU_PEAK_DURATION     CPU_PEAK_EFFICIENCY
         0.00           0 second(s)                   0.00%
     CPU_AVERAGE_EFFICIENCY      MEM_EFFICIENCY
                      0.00%               5.00%
------------------------------------------------------------------------------`,
			walltime: 35,
			ncpus:    2,
			ngpus:    0,
			mem:      230686720,
			nnodes:   2,
		},
		{
			name: "Job started and ended inside current interval and GPUs are fetched from jobCache",
			bacctOutput: `
Job <33>, User <usr3>, Project <prj3>, Status <DONE>, Queue <interactive>, Interactive pseudo-terminal shell mode, Command <bash>, Share group charged </usr3>
Sun Apr 12 18:23:38 2026: Submitted from host <nodes-1.example.com>, CWD <$HOME>;
Sun Apr 12 18:23:39 2026: Dispatched 2 Task(s) on Host(s) <nodes-2.example.com> <nodes-1.example.com>, Allocated 2 Slot(s) on Host(s) <nodes-2.example.com> <nodes-1.example.com>, Effective RES_REQ <select[(type == any ) && (ngpus>0)] order[r15s:pg] rusage[ngpus_physical=2.00/host] span[ptile=2] >;
Sun Apr 12 18:24:14 2026: Completed <done>.

Accounting information about this job:
     Share group charged </usr3>
     CPU_T     WAIT     TURNAROUND   STATUS     HOG_FACTOR    MEM    SWAP
      0.22        1             36     done         0.0060    11M      0M
     CPU_PEAK     CPU_PEAK_DURATION     CPU_PEAK_EFFICIENCY
         0.00           0 second(s)                   0.00%
     CPU_AVERAGE_EFFICIENCY      MEM_EFFICIENCY
                      0.00%               5.00%
------------------------------------------------------------------------------`,
			walltime: 35,
			ncpus:    2,
			ngpus:    2,
			mem:      230686720,
			nnodes:   2,
		},
	}

	// Setup jobCache
	var jobCache sync.Map
	jobCache.Store("33", jobAttributes{
		numNodes: 2,
		numCPUs:  2,
		numGPUs:  2,
	})

	for _, test := range tests {
		units := parseBacctCmdOutput(test.bacctOutput, start, end, &jobCache)
		if test.walltime == 0 {
			assert.Equal(t, 0, int(units[0].TotalTime["walltime"]), test.name)
		} else {
			assert.InEpsilon(t, test.walltime, float64(units[0].TotalTime["walltime"]), 0, test.name)
		}

		assert.Equal(t, test.nnodes, units[0].Allocation["nodes"], test.name+" nodes")
		assert.Equal(t, test.ncpus, units[0].Allocation["cpus"], test.name+" ncpus")
		assert.Equal(t, test.ngpus, units[0].Allocation["gpus"], test.name+" ngpus")
		assert.Equal(t, test.mem, units[0].Allocation["mem"], test.name+" mem")
	}
}

func TestParseBjobsCmdOutput(t *testing.T) {
	tests := []struct {
		name                 string
		bjobsOutput          string
		walltime             float64
		nnodes, ncpus, ngpus int
		mem                  int64
		group                string
	}{
		{
			name: "Job started in past with GPUs",
			bjobsOutput: `{
  "COMMAND":"bjobs",
  "JOBS":1,
  "RECORDS":[
    {
      "JOBID":"321",
      "JOBINDEX":"0",
      "USER":"ubuntu",
      "USER_GROUP":"ubuntugrp",
      "QUEUE":"interactive",
      "JOB_NAME":"bash",
      "PROJ_NAME":"default",
      "STAT":"RUN",
      "ALLOC_SLOT":"nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com",
      "NALLOC_SLOT":"32",
      "SUBMIT_TIME":"Apr 12 18:10:00 2026",
      "START_TIME":"Apr 12 18:10:55 2026",
      "FINISH_TIME":"Apr 12 22:27:01 2026 L",
      "RUN_TIME":"34 second(s)",
      "EFFECTIVE_RESREQ":"select[type == local] order[r15s:pg] ",
      "GPU_NUM":"4",
      "GPU_ALLOC":"nodes-1.example.com:0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1;nodes-2.example.com:0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1",
      "MEM":"9 Mbytes",
      "MEMLIMIT":"",
      "MEM_EFFICIENCY":"0.00%",
      "EXEC_CWD":""
    }
  ]
}`,
			walltime: 900,
			nnodes:   2,
			ncpus:    32,
			ngpus:    4,
			mem:      9437184,
			group:    "ubuntugrp",
		},
		{
			name: "Job started in the interval with GPUs",
			bjobsOutput: `{
  "COMMAND":"bjobs",
  "JOBS":1,
  "RECORDS":[
    {
      "JOBID":"321",
      "JOBINDEX":"0",
      "USER":"ubuntu",
      "USER_GROUP":"ubuntugrp1",
      "QUEUE":"interactive",
      "JOB_NAME":"bash",
      "PROJ_NAME":"default",
      "STAT":"RUN",
      "ALLOC_SLOT":"nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-1.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com:nodes-2.example.com",
      "NALLOC_SLOT":"32",
      "SUBMIT_TIME":"Apr 12 18:20:00 2026",
      "START_TIME":"Apr 12 18:20:55 2026",
      "FINISH_TIME":"Apr 12 22:27:01 2026 L",
      "RUN_TIME":"34 second(s)",
      "EFFECTIVE_RESREQ":"select[type == local] order[r15s:pg] ",
      "MEM":"9 Gbytes",
      "MEMLIMIT":"",
      "MEM_EFFICIENCY":"5.20%",
      "EXEC_CWD":""
    }
  ]
}`,
			walltime: 545,
			nnodes:   2,
			ncpus:    32,
			ngpus:    0,
			mem:      185839931076,
			group:    "ubuntugrp1",
		},
	}

	// Setup jobCache
	var jobCache sync.Map

	for _, test := range tests {
		units, _, err := parseBjobsCmdOutput([]byte(test.bjobsOutput), start, end, &jobCache)
		require.NoError(t, err, test.name)

		if test.walltime == 0 {
			assert.Equal(t, 0, int(units[0].TotalTime["walltime"]), test.name)
		} else {
			assert.InEpsilon(t, test.walltime, float64(units[0].TotalTime["walltime"]), 0, test.name)
		}

		assert.Equal(t, test.nnodes, units[0].Allocation["nodes"], test.name+" nodes")
		assert.Equal(t, test.ncpus, units[0].Allocation["cpus"], test.name+" ncpus")
		assert.Equal(t, test.ngpus, units[0].Allocation["gpus"], test.name+" ngpus")
		assert.Equal(t, test.mem, units[0].Allocation["mem"], test.name+" mem")

		// Check if jobCache is populated
		if v, ok := jobCache.Load("321"); ok {
			if u, ok := v.(jobAttributes); ok {
				assert.Equal(t, test.nnodes, u.numNodes, test.name+" jobCache nnodes")
				assert.Equal(t, test.ncpus, u.numCPUs, test.name+" jobCache ncpus")
				assert.Equal(t, test.ngpus, u.numGPUs, test.name+" jobCache ngpus")
				assert.Equal(t, test.mem, u.mem, test.name+" jobCache mem")
				assert.Equal(t, test.group, u.userGroup, test.name+" jobCache group")
			}
		}
	}
}
