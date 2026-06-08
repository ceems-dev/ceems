// Package lsf implements the fetcher interface to fetch compute units from LSF
// resource manager
package lsf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/ceems-dev/ceems/pkg/api/base"
	"github.com/ceems-dev/ceems/pkg/api/models"
	"github.com/ceems-dev/ceems/pkg/api/resource"
)

// Fetch modes.
const (
	cliMode = "cli"
)

// lsfScheduler is the struct containing the configuration of a given lsf cluster.
type lsfScheduler struct {
	logger          *slog.Logger
	cluster         models.Cluster
	fetchMode       string // Whether to fetch from REST API or CLI commands
	jobsCache       sync.Map
	lastFetchedJobs []models.Unit
}

const lsfBatchScheduler = "lsf"

var jobLock = sync.RWMutex{}

// Regular expressions for parsing bacct output.
// Ref: https://regex101.com/r/u6Fn6r/2
// Seems hacky but might just work.
var (
	regexSeparator  = regexp.MustCompile(`[-]{3,}`)
	regexJobDetails = regexp.MustCompile(`(?m)^Job <(?P<jobid>.*?)>(?:.*?)User <(?P<user>.*?)>(?:.*?)Project <(?P<project>.*?)>(?:.*?)Status <(?P<status>.*?)>(?:.*?)Queue <(?P<queue>.*?)>(?:.*?)Command <(?P<command>.*?)>(?:.*?)(Share group charged <(?P<share_group>.*?)>(?:.*?))?$\n^(?P<submit_time>.*?): Submitted from host <(?P<submit_host>.*?)>(?:.*?)CWD <(?P<cwd>.*?)>(?:.*?)$\n(^(?P<start_time>.*?): (?:.*?)[D|d]ispatched (?P<ntasks>[\d]+?) Task\(s\) on Host\(s\) (?:.*?), Allocated (?P<nslots>[\d]+?) Slot\(s\) on Host\(s\) (?P<nodelist>.*?), Effective RES_REQ (?P<res_req>.*)(?:.*?)$\n)?^(?P<end_time>.*?): Completed <(?P<exit_status>.*?)>(?P<exit_msg>.*?).$([\r\n\s]+GPU_ALLOCATION:[\r\n\s]+(?P<gpu_alloc>(.|\n)*)$)?[\r\n\s]+Accounting information about this job:[\r\n\s]+(?P<accounting>(.|\n)*)`)
)

var (
	lsfCommandsTimeLayout = "2006/01/02/15:04"
	bacctTimeLayoout      = "Mon Jan 2 15:04:05 2006"
	bjobsTimeLayoout      = "Jan 2 15:04:05 2006"
)

type jobAttributes struct {
	numNodes  int
	numCPUs   int
	numGPUs   int
	nodelist  string
	mem       int64
	userGroup string
}

func init() {
	// Register batch scheduler
	resource.Register(lsfBatchScheduler, New)
}

// New returns a new lsfScheduler that returns batch job stats.
func New(cluster models.Cluster, logger *slog.Logger) (resource.Fetcher, error) {
	// Make lsfCluster configs from clusters
	lsfScheduler := lsfScheduler{
		logger:  logger,
		cluster: cluster,
	}

	err := preflightChecks(&lsfScheduler)
	if err != nil {
		return nil, err
	}

	logger.Info("Batch jobs from LSF cluster will be fetched", "id", cluster.ID)

	return &lsfScheduler, nil
}

// FetchUnits fetches jobs from lsf.
func (s *lsfScheduler) FetchUnits(
	ctx context.Context,
	start time.Time,
	end time.Time,
) ([]models.ClusterUnits, error) {
	// Fetch each cluster one by one to reduce memory footprint
	var jobs []models.Unit

	var err error
	if s.fetchMode == cliMode {
		jobs, err = s.fetchFromBacctAndBjobs(ctx, start, end)
		if err != nil && len(jobs) == 0 {
			s.logger.Error("Failed to execute LSF bacct and bjobs commands", "cluster_id", s.cluster.ID, "err", err)

			return nil, err
		}

		return []models.ClusterUnits{{Cluster: s.cluster, Units: jobs}}, nil
	}

	return nil, fmt.Errorf("unknown fetch mode for compute units LSF cluster %s", s.cluster.ID)
}

// FetchUsersProjects fetches current LSF users and accounts.
func (s *lsfScheduler) FetchUsersProjects(
	ctx context.Context,
	current time.Time,
) ([]models.ClusterUsers, []models.ClusterProjects, error) {
	if s.fetchMode == cliMode {
		// Make user and project structs from userProjects map
		users, projects := s.fetchUserProjects(current)
		s.logger.Info("LSF user account data fetched", "cluster_id", s.cluster.ID, "num_users", len(users), "num_accounts", len(projects))

		return []models.ClusterUsers{
				{Cluster: s.cluster, Users: users},
			}, []models.ClusterProjects{
				{Cluster: s.cluster, Projects: projects},
			}, nil
	}

	return nil, nil, fmt.Errorf("unknown fetch mode for projects for LSF cluster %s", s.cluster.ID)
}

// Get jobs from lsf bacct and bjobs commands.
func (s *lsfScheduler) fetchFromBacctAndBjobs(ctx context.Context, start time.Time, end time.Time) ([]models.Unit, error) {
	var (
		jobs []models.Unit
		errs error
	)

	// Execute bacct command between start and end times
	bacctOutput, err := s.runBacctCmd(ctx, start, end)
	if err != nil {
		s.logger.Error("Failed to run bacct command", "cluster_id", s.cluster.ID, "err", err)
		errs = errors.Join(errs, err)
	} else {
		// Parse bacct output and create BatchJob structs slice
		finishedJobs := parseBacctCmdOutput(string(bacctOutput), start, end, &s.jobsCache)
		s.logger.Info("LSF finished jobs fetched", "cluster_id", s.cluster.ID, "start", start, "end", end, "num_jobs", len(finishedJobs))

		// Append finished job to jobs slice
		jobs = append(jobs, finishedJobs...)
	}

	// Execute bjobs command to get currently running jobs
	bjobsOutput, err := s.runBjobsCmd(ctx)
	if err != nil {
		s.logger.Error("Failed to run bjobs command", "cluster_id", s.cluster.ID, "err", err)
		errs = errors.Join(errs, err)
	} else {
		// Parse bjobs output and create BatchJob structs slice
		runningJobs, numRunningJobs, err := parseBjobsCmdOutput(bjobsOutput, start, end, &s.jobsCache)
		if err != nil {
			s.logger.Error("Failed to parse bjobs command output", "err", err)
			errs = errors.Join(errs, err)

			// Replace state variable of jobs
			s.lastFetchedJobs = jobs

			return jobs, errs
		}

		s.logger.Info("LSF running jobs fetched", "cluster_id", s.cluster.ID, "start", start, "end", end, "num_jobs", numRunningJobs)

		// Append running job to jobs slice
		jobs = append(jobs, runningJobs...)
	}

	// Replace state variable of jobs
	s.lastFetchedJobs = jobs

	return jobs, errs
}

// Get user project association from current jobs slice.
func (s *lsfScheduler) fetchUserProjects(current time.Time) ([]models.User, []models.Project) {
	// Current time string
	currentTime := current.Format(base.DatetimezoneLayout)

	// Reset the slices
	var (
		userModels    []models.User
		projectModels []models.Project
	)

	// Use a map to associate users and projects
	var (
		userProjects = make(map[string][]string)
		projectUsers = make(map[string][]string)
	)

	// Loop over all jobs and get project and user for each job

	for _, job := range s.lastFetchedJobs {
		userProjects[job.User] = append(userProjects[job.User], job.Project)
		projectUsers[job.Project] = append(projectUsers[job.Project], job.User)
	}

	// Make user models
	for user, projects := range userProjects {
		slices.Sort(projects)

		var projectList models.List

		for _, project := range slices.Compact(projects) {
			projectList = append(projectList, project)
		}

		u := models.User{
			Name:            user,
			ClusterID:       s.cluster.ID,
			ResourceManager: s.cluster.Manager,
			Projects:        projectList,
			LastUpdatedAt:   currentTime,
		}

		userModels = append(userModels, u)
	}

	// Make project models
	for project, users := range projectUsers {
		slices.Sort(users)

		var userList models.List

		for _, user := range slices.Compact(users) {
			userList = append(userList, user)
		}

		p := models.Project{
			Name:            project,
			ClusterID:       s.cluster.ID,
			ResourceManager: s.cluster.Manager,
			Users:           userList,
			LastUpdatedAt:   currentTime,
		}

		projectModels = append(projectModels, p)
	}

	return userModels, projectModels
}
