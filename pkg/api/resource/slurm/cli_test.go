package slurm

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ceems-dev/ceems/internal/security"
	"github.com/ceems-dev/ceems/pkg/api/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreflightsCLI(t *testing.T) {
	manager := slurmScheduler{
		logger:           noOpLogger,
		securityContexts: make(map[string]*security.SecurityContext),
	}
	err := preflightsCLI(&manager)
	require.Error(t, err)

	// Add sacct command to PATH
	sacctPath, _ := filepath.Abs("../../testdata")
	t.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), sacctPath))

	err = preflightsCLI(&manager)
	require.NoError(t, err)
	assert.Equal(t, sacctPath, manager.cluster.CLI.Path)
}

func TestParseSacctCmdOutput(t *testing.T) {
	tests := []struct {
		name        string
		sacctOutput string
		walltime    float64
	}{
		{
			name:        "Job finished in past",
			sacctOutput: `1479763|part1|qos1|acc1|grp|1000|usr|1000|2023-02-20T14:37:02+0100|2023-02-20T14:37:07+0100|2023-02-20T15:37:07+0100|01:49:22|3000|0:0|RUNNING|billing=80,cpu=160,energy=1439089,gres/gpu=8,mem=320G,node=2|compute-0|test_script1|/home/usr`,
			walltime:    3600,
		},
		{
			name:        "Job created but not started",
			sacctOutput: `1479763|part1|qos1|acc1|grp|1000|usr|1000|2023-02-21T14:37:02+0100|NA|NA|01:49:22|3000|0:0|PENDING|billing=80,cpu=160,energy=1439089,gres/gpu=8,mem=320G,node=2|compute-0|test_script1|/home/usr`,
			walltime:    0,
		},
		{
			name:        "Job started inside current interval",
			sacctOutput: `1479763|part1|qos1|acc1|grp|1000|usr|1000|2023-02-21T15:10:00+0100|2023-02-21T15:10:00+0100|NA|01:49:22|3000|0:0|RUNNING|billing=80,cpu=160,energy=1439089,gres/gpu=8,mem=320G,node=2|compute-0|test_script1|/home/usr`,
			walltime:    300,
		},
		{
			name:        "Job ended inside current interval",
			sacctOutput: `1479763|part1|qos1|acc1|grp|1000|usr|1000|2023-02-21T14:10:00+0100|2023-02-21T14:10:00+0100|2023-02-21T15:10:00+0100|01:49:22|3000|0:0|COMPLETED|billing=80,cpu=160,energy=1439089,gres/gpu=8,mem=320G,node=2|compute-0|test_script1|/home/usr`,
			walltime:    600,
		},
		{
			name:        "Job started and ended inside current interval",
			sacctOutput: `1479763|part1|qos1|acc1|grp|1000|usr|1000|2023-02-21T15:10:00+0100|2023-02-21T15:10:00+0100|2023-02-21T15:12:00+0100|01:49:22|3000|0:0|COMPLETED|billing=80,cpu=160,energy=1439089,gres/gpu=8,mem=320G,node=2|compute-0|test_script1|/home/usr`,
			walltime:    120,
		},
		{
			name:        "Job times do not have TZ offset",
			sacctOutput: `1479763|part1|qos1|acc1|grp|1000|usr|1000|2023-02-21T15:10:00|2023-02-21T15:10:00|2023-02-21T15:12:00|01:49:22|3000|0:0|COMPLETED|billing=80,cpu=160,energy=1439089,gres/gpu=8,mem=320G,node=2|compute-0|test_script1|/home/usr`,
			walltime:    120,
		},
	}

	// Check units
	units, numUnits := parseSacctCmdOutput(sacctCmdOutput, start, end)
	require.ElementsMatch(t, units, expectedBatchJobs)
	require.Equal(t, 2, numUnits)

	for _, test := range tests {
		units, _ = parseSacctCmdOutput(test.sacctOutput, start, end)
		if test.walltime == 0 {
			assert.Equal(t, 0, int(units[0].TotalTime["walltime"]), test.name)
		} else {
			assert.InEpsilon(t, test.walltime, float64(units[0].TotalTime["walltime"]), 0, test.name)
		}
	}
}

func TestParseSacctMgrCmdOutput(t *testing.T) {
	users, projects := parseSacctMgrCmdOutput(sacctMgrCmdOutput, current.Format(base.DatetimezoneLayout))
	require.ElementsMatch(t, expectedUsers, users)
	require.ElementsMatch(t, expectedProjects, projects)
}
